#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""
Update the "hus til salgs" table on rugdeberget.no.

Fetches the live page, pulls out the editor content (the intro paragraph and
the table), applies the change, and prints the new page source to paste into
the borettslag.net editor. Only the changed row differs; everything else
stays byte-identical to what is live.

Usage:
    uv run listings.py list [--for-sale]
    uv run listings.py add <finn-url|finnkode> "<address>" <price>           # Prisantydning
    uv run listings.py add <finn-url|finnkode> "<address>" [price] --sold    # SOLGT, never listed as for sale
    uv run listings.py sold <finn-url|finnkode> [price]                      # existing row → SOLGT

Options (before or after the subcommand):
    --source URL|FILE  where to read the page (default: the live page)
    -o, --out FILE     write the new page source to FILE instead of stdout
    --copy             also copy the new page source to the Windows clipboard (WSL)

Several changes before pasting: chain them through a file.
    uv run listings.py add 480000001 "Rugdeberget 22" 4590000 -o new.html
    uv run listings.py sold 477062154 5950000 --source new.html -o new.html
"""

import argparse
import html
import re
import shutil
import subprocess
import sys
import urllib.request
from pathlib import Path

LIVE_URL = "https://www.rugdeberget.no/oppslagstavle-hus-til-salgs-osv/info-om-hus-til-salgs/"
USER_AGENT = "rugdeberget-listings/1.0 (styret, Rugdeberget borettslag)"

WIDGET_RE = re.compile(r'<div\s+class="u-widget"\s*>', re.I)
DIV_TAG_RE = re.compile(r"<div\b|</div\s*>", re.I)
ROW_RE = re.compile(r"<tr\b[^>]*>.*?</tr>", re.S | re.I)
TD_RE = re.compile(r"(<td\b[^>]*>)(.*?)(</td>)", re.S | re.I)
HREF_RE = re.compile(r'href="([^"]*)"', re.I)
FINNKODE_RE = re.compile(r"(?:finnkode=|finn\.no/)(\d{6,})")
TAG_RE = re.compile(r"<[^>]+>")
TBODY_RE = re.compile(r"<tbody\b[^>]*>(?:\r?\n)?", re.I)

AD_URL = "https://www.finn.no/realestate/homes/ad.html?finnkode={}"

NEW_ROW = """<tr style="height: 37px;">
<td style="width: 454px; height: 37px;"><a href="{url}">{address}</a></td>
<td style="width: 325px; height: 37px;"><em><strong>{status}</strong></em></td>
</tr>
"""


def die(msg: str) -> None:
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def info(msg: str) -> None:
    print(msg, file=sys.stderr)


# --- reading ---------------------------------------------------------------


def fetch(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            charset = resp.headers.get_content_charset() or "utf-8"
            return resp.read().decode(charset)
    except OSError as e:
        die(f"could not fetch {url}: {e}")


def extract_widget(page: str) -> str:
    """Return the inner HTML of <div class="u-widget">, i.e. what the editor holds."""
    m = WIDGET_RE.search(page)
    if not m:
        die('no <div class="u-widget"> on the page; has the site layout changed?')
    depth = 1
    for tag in DIV_TAG_RE.finditer(page, m.end()):
        depth += -1 if tag.group(0).startswith("</") else 1
        if depth == 0:
            return page[m.end() : tag.start()].strip()
    die("unterminated u-widget div")


def load(source: str) -> str:
    """Load editor content from a URL (live page) or a file (earlier output or a saved page)."""
    if source.startswith(("http://", "https://")):
        info(f"Fetched {source}")
        return extract_widget(fetch(source))
    doc = Path(source).read_bytes().decode("utf-8")  # bytes, so CRLF survives
    # A full saved page still has the site chrome around it; an earlier output does not.
    return extract_widget(doc) if WIDGET_RE.search(doc) else doc.strip()


# --- parsing and editing ----------------------------------------------------


def parse_finnkode(arg: str) -> str:
    if arg.isdigit():
        return arg
    m = FINNKODE_RE.search(arg)
    if not m:
        die(f"no finnkode found in {arg!r}")
    return m.group(1)


def format_price(raw: str) -> str:
    digits = re.sub(r"\D", "", raw)
    if not digits:
        die(f"not a price: {raw!r}")
    return f"{int(digits):,}".replace(",", " ") + " kr"


def sold_status(price: str | None) -> str:
    return f"SOLGT: {format_price(price)}" if price else "SOLGT"


def text(fragment: str) -> str:
    return " ".join(html.unescape(TAG_RE.sub(" ", fragment)).split())


def rows(doc: str):
    """Yield (match, finnkode|None, address, status) for every table row."""
    for m in ROW_RE.finditer(doc):
        cells = TD_RE.findall(m.group(0))
        if len(cells) < 2:
            continue
        first, second = cells[0][1], cells[1][1]
        href = HREF_RE.search(first)
        code = FINNKODE_RE.search(href.group(1)) if href else None
        address = re.sub(r"\s*\(+link\)\s*|:\s*$", "", text(first), flags=re.I).rstrip(": ")
        yield m, code.group(1) if code else None, address, text(second)


def find_row(doc: str, finnkode: str):
    hits = [r for r in rows(doc) if r[1] == finnkode]
    if len(hits) > 1:
        die(f"finnkode {finnkode} appears in {len(hits)} rows; fix the page by hand")
    return hits[0] if hits else None


def set_status(doc: str, row: re.Match, status: str) -> str:
    """Replace the contents of the row's second <td>, keeping its tag and attributes."""
    row_html = row.group(0)
    td = list(TD_RE.finditer(row_html))[1]
    new_row = (
        row_html[: td.start(2)]
        + f"<em><strong>{html.escape(status, quote=False)}</strong></em>"
        + row_html[td.end(2) :]
    )
    return doc[: row.start()] + new_row + doc[row.end() :]


def add_row(doc: str, code: str, address: str, status: str) -> str:
    """Insert a row keeping the table newest-first by finnkode.

    A new listing has the highest finnkode and lands at the top; a sold one added
    after the fact lands among its contemporaries.
    """
    tbody = TBODY_RE.search(doc)
    if not tbody:
        die("no <tbody> in the page")
    pos = next((m.start() for m, c, _, _ in rows(doc) if c and int(c) < int(code)), None)
    if pos is None:  # older than everything with a finnkode: put it last
        pos = doc.lower().index("</tbody>", tbody.end())
    eol = "\r\n" if "\r\n" in doc else "\n"
    row = NEW_ROW.format(
        url=AD_URL.format(code),
        address=html.escape(address, quote=False),
        status=html.escape(status, quote=False),
    ).replace("\n", eol)
    return doc[:pos] + row + doc[pos:]


# --- output -----------------------------------------------------------------


def copy_to_clipboard(doc: str) -> None:
    clip = shutil.which("clip.exe")
    if not clip:
        die("clip.exe not found (only works under WSL/Windows)")
    try:
        # clip.exe reads UTF-16 correctly when a BOM is present; plain UTF-8 mangles æøå.
        subprocess.run([clip], input=doc.encode("utf-16"), check=True)
    except OSError as e:
        die(f"could not run clip.exe ({e}); is WSL interop enabled? Use -o and copy the file by hand")
    info("Copied new page source to clipboard.")


def emit(doc: str, out: str | None, copy: bool) -> None:
    eol = "\r\n" if "\r\n" in doc else "\n"
    doc = doc + eol
    if out:
        Path(out).write_bytes(doc.encode("utf-8"))
        info(f"Wrote new page source to {out}")
    else:
        sys.stdout.write(doc)
    if copy:
        copy_to_clipboard(doc)


# --- cli --------------------------------------------------------------------


def main() -> None:
    # Options are accepted both before and after the subcommand. SUPPRESS keeps the
    # subparser from overwriting a value given before the subcommand.
    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--source", default=argparse.SUPPRESS, help=f"URL or file to read (default: {LIVE_URL})")
    common.add_argument("-o", "--out", default=argparse.SUPPRESS, help="write new page source here (default: stdout)")
    common.add_argument("--copy", action="store_true", default=argparse.SUPPRESS, help="copy to Windows clipboard")

    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter, parents=[common]
    )
    sub = p.add_subparsers(dest="cmd", required=True)

    ls = sub.add_parser("list", help="show rows", parents=[common])
    ls.add_argument("--for-sale", action="store_true", help="only rows not marked SOLGT")

    add = sub.add_parser("add", help="add a new listing at the top (or update its price)", parents=[common])
    add.add_argument("finn")
    add.add_argument("address")
    add.add_argument("price", nargs="?", help="prisantydning, or sale price with --sold (optional then)")
    add.add_argument("--sold", action="store_true", help="add as SOLGT instead of Prisantydning")

    sold = sub.add_parser("sold", help="mark a listing as sold", parents=[common])
    sold.add_argument("finn")
    sold.add_argument("price", nargs="?")

    args = p.parse_args()
    source = getattr(args, "source", LIVE_URL)
    out = getattr(args, "out", None)
    copy = getattr(args, "copy", False)

    doc = load(source)

    if args.cmd == "list":
        for _, code, address, status in rows(doc):
            if args.for_sale and status.upper().startswith("SOLGT"):
                continue
            print(f"{code or '-':>10}  {address:<32}  {status}")
        return

    code = parse_finnkode(args.finn)
    existing = find_row(doc, code)

    if args.cmd == "add":
        if args.sold:
            status = sold_status(args.price)
        elif args.price:
            status = f"Prisantydning: {format_price(args.price)}"
        else:
            die("add needs a price (prisantydning), or --sold")
        if existing:
            doc = set_status(doc, existing[0], status)
            info(f"Updated {existing[2]} ({code}): {status}")
        else:
            doc = add_row(doc, code, args.address, status)
            info(f"Added {args.address} ({code}): {status}")

    elif args.cmd == "sold":
        if not existing:
            die(f"finnkode {code} not in the table; add it first")
        status = sold_status(args.price)
        doc = set_status(doc, existing[0], status)
        info(f"Updated {existing[2]} ({code}): {status}")

    emit(doc, out, copy)


if __name__ == "__main__":
    main()
