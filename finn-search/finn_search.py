#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["requests", "beautifulsoup4"]
# ///
"""
Crawl a Finn.no search URL and output listings as JSON.

Usage:
    uv run finn_search.py <url> [--all-pages] [--details] [--debug] [--out results.json]

Examples:
    uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=fcc&sub_category=1.67.3901&location=0.20061"
    uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=fcc" --all-pages --out fcc.json
    uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=fcc" --details --out fcc_full.json
    uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=fcc" --debug
"""

import argparse
import json
import re
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from urllib.parse import urlencode, urlparse, parse_qs, urlunparse

import requests
from bs4 import BeautifulSoup

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
    "Accept-Language": "nb-NO,nb;q=0.9,no;q=0.8,en;q=0.7",
}

PRICE_RE = re.compile(r"[\d\s]+kr")

CONDITION_MAP = {
    "NewCondition": "Ny",
    "UsedCondition": "Brukt",
    "RefurbishedCondition": "Renovert",
    "DamagedCondition": "Skadet",
}


def page_url(base_url: str, page: int) -> str:
    parsed = urlparse(base_url)
    params = parse_qs(parsed.query, keep_blank_values=True)
    params["page"] = [str(page)]
    new_query = urlencode({k: v[0] for k, v in params.items()})
    return urlunparse(parsed._replace(query=new_query))


def fetch(url: str) -> BeautifulSoup:
    resp = requests.get(url, headers=HEADERS, timeout=15)
    resp.raise_for_status()
    return BeautifulSoup(resp.text, "html.parser")


def debug_dump(soup: BeautifulSoup) -> None:
    from collections import Counter
    counts: Counter = Counter()
    for tag in soup.find_all(True):
        classes = " ".join(sorted(tag.get("class", [])))
        counts[(tag.name, classes[:80])] += 1
    print("\n=== TOP TAGS (tag, classes, count) ===")
    for (tag, cls), n in counts.most_common(40):
        print(f"  {n:4d}x  <{tag}> {cls}")
    print("\n=== POSSIBLE LISTING CONTAINERS ===")
    for selector in ["article", "li[class]", "div[data-testid]", "a[href*='/item/']"]:
        found = soup.select(selector)
        if found:
            print(f"\n  {selector}: {len(found)} found")
            print(f"  First element:\n{found[0].prettify()[:600]}")


def extract_price(article: BeautifulSoup) -> str:
    for div in article.find_all("div"):
        direct_text = "".join(s for s in div.strings if s.parent == div).strip()
        if not direct_text:
            text = div.get_text(strip=True)
            if len(text) > 30:
                continue
            direct_text = text
        m = PRICE_RE.search(direct_text)
        if m:
            return m.group(0).strip()
    return ""


def fetch_item_details(url: str) -> dict:
    """Fetch an individual listing page and return description, condition, category."""
    try:
        soup = fetch(url)
    except Exception as e:
        return {"_error": str(e)}

    details: dict = {}

    # JSON-LD is the cleanest source
    for script in soup.find_all("script", type="application/ld+json"):
        try:
            data = json.loads(script.string or "")
        except Exception:
            continue
        if data.get("@type") != "Product":
            continue
        details["description"] = data.get("description", "")
        raw_condition = data.get("itemCondition", "")
        # schema.org returns a full URL like "https://schema.org/UsedCondition"
        condition_key = raw_condition.split("/")[-1] if raw_condition else ""
        details["condition"] = CONDITION_MAP.get(condition_key, condition_key)
        for prop in data.get("additionalProperty", []):
            if prop.get("name") == "category":
                details["category"] = prop.get("value", "")
        break

    # Condition as human-readable text from the page (more descriptive than schema.org)
    nok = soup.find("section", attrs={"aria-label": "Nøkkelinfo"})
    if nok:
        for p in nok.find_all("p"):
            text = p.get_text(separator=" ", strip=True)
            if "Tilstand" in text:
                b = p.find("b")
                if b:
                    details["condition"] = b.get_text(strip=True)
                break

    return details


def extract_listings(soup: BeautifulSoup) -> list[dict]:
    listings = []

    for article in soup.select("article.sf-search-ad"):
        link = article.select_one("a.sf-search-ad-link")
        if not link:
            continue
        href = link.get("href", "")
        if href.startswith("/"):
            href = "https://www.finn.no" + href
        item_id = link.get("id", href.rstrip("/").split("/")[-1])

        h2 = article.find("h2")
        title = h2.get_text(strip=True) if h2 else ""

        price = extract_price(article)

        location = ""
        date = ""
        footer = article.select_one("div.s-text-subtle")
        if footer:
            spans = footer.find_all("span")
            if spans:
                location = spans[0].get_text(strip=True)
            if len(spans) >= 2:
                date = spans[-1].get_text(strip=True)

        img = article.select_one("img")
        image_url = img.get("src", "") if img else ""

        listings.append({
            "id": item_id,
            "title": title,
            "price": price,
            "location": location,
            "date": date,
            "url": href,
            "image_url": image_url,
        })

    return listings


def enrich_with_details(listings: list[dict], max_workers: int = 5) -> list[dict]:
    """Fetch item pages concurrently and merge details into each listing."""
    total = len(listings)
    enriched = list(listings)
    url_to_idx = {l["url"]: i for i, l in enumerate(enriched)}

    with ThreadPoolExecutor(max_workers=max_workers) as pool:
        futures = {pool.submit(fetch_item_details, l["url"]): l["url"] for l in enriched}
        done = 0
        for future in as_completed(futures):
            url = futures[future]
            done += 1
            print(f"  details {done}/{total}: {url}", file=sys.stderr)
            try:
                details = future.result()
                enriched[url_to_idx[url]].update(details)
            except Exception as e:
                enriched[url_to_idx[url]]["_error"] = str(e)

    return enriched


def crawl_all_pages(base_url: str) -> list[dict]:
    all_listings: list[dict] = []
    seen_ids: set[str] = set()
    page = 1

    while True:
        url = page_url(base_url, page)
        print(f"Fetching page {page}: {url}", file=sys.stderr)
        soup = fetch(url)
        listings = extract_listings(soup)

        if not listings:
            break

        new = [l for l in listings if l["id"] not in seen_ids]
        if not new:
            break

        for l in new:
            seen_ids.add(l["id"])
        all_listings.extend(new)

        page += 1
        time.sleep(1)

    return all_listings


def main() -> None:
    parser = argparse.ArgumentParser(description="Scrape a Finn.no search page")
    parser.add_argument("url", help="Full Finn.no search URL")
    parser.add_argument("--all-pages", action="store_true", help="Follow pagination and collect all results")
    parser.add_argument("--details", action="store_true", help="Fetch each item page for description, condition, category")
    parser.add_argument("--debug", action="store_true", help="Print tag structure to help tune selectors")
    parser.add_argument("--out", metavar="FILE", help="Write JSON output to file instead of stdout")
    args = parser.parse_args()

    if args.debug:
        debug_dump(fetch(args.url))
        return

    if args.all_pages:
        listings = crawl_all_pages(args.url)
    else:
        listings = extract_listings(fetch(args.url))

    if args.details:
        print(f"Fetching details for {len(listings)} listings...", file=sys.stderr)
        listings = enrich_with_details(listings)

    output = {
        "url": args.url,
        "fetched_at": datetime.now(timezone.utc).isoformat(),
        "count": len(listings),
        "listings": listings,
    }

    json_out = json.dumps(output, ensure_ascii=False, indent=2)

    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(json_out)
        print(f"Wrote {len(listings)} listings to {args.out}", file=sys.stderr)
    else:
        print(json_out)


if __name__ == "__main__":
    main()
