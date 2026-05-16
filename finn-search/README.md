# finn-search

Scrapes a Finn.no Torget search URL and outputs listings as JSON. No browser required — the search page is server-side rendered.

## Usage

```bash
uv run finn_search.py <url> [--all-pages] [--details] [--out FILE]
```

### Examples

```bash
# Single page, stdout
uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=weber&location=0.20061"

# All pages, write to file
uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=weber" --all-pages --out results.json

# Full details (description, condition, category) fetched from each item page
uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=weber" --details --out results_full.json

# Debug mode — prints tag/class counts to help re-tune selectors
uv run finn_search.py "https://www.finn.no/recommerce/forsale/search?q=weber" --debug
```

## Output fields

| Field | Source | Notes |
|---|---|---|
| `id` | Search page | Finn item ID |
| `title` | Search page | Listing title |
| `price` | Search page | e.g. `"400 kr"`, empty if free |
| `location` | Search page | City/area |
| `date` | Search page | e.g. `"2 dg."` or `"12. mai"` |
| `url` | Search page | Full item URL |
| `image_url` | Search page | First listing image |
| `description` | Item page (`--details`) | Full seller description |
| `condition` | Item page (`--details`) | e.g. `"Ny"`, `"Brukt"`, `"Godt brukt - Synlig brukt"` |
| `category` | Item page (`--details`) | Full category path |

## Options

| Flag | Description |
|---|---|
| `--all-pages` | Follow `&page=N` pagination until results are exhausted |
| `--details` | Fetch each item page for description, condition, and category (5 concurrent requests) |
| `--out FILE` | Write JSON to file instead of stdout |
| `--debug` | Print page tag/class structure to help fix selectors after a Finn markup change |

## Dependencies

Managed inline via `uv` — no manual install needed:
- `requests`
- `beautifulsoup4`
