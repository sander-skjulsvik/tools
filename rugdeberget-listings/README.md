# rugdeberget-listings

Updates the "Info om hus til salgs" table on
[rugdeberget.no](https://www.rugdeberget.no/oppslagstavle-hus-til-salgs-osv/info-om-hus-til-salgs/).

Each run fetches the **live page**, extracts the editor content (`<div class="u-widget">`: intro paragraph + table), applies your change, and prints the **new page source**. Only the changed row differs from what is live.

Python stdlib only, run with `uv`. Nothing is fetched from Finn (its real estate pages are behind bot protection); you give the values.

```bash
uv run listings.py list [--for-sale]                                  # finnkode, address, status
uv run listings.py add <finn-url|finnkode> "Rugdeberget 22" 4590000          # for sale → "Prisantydning: 4 590 000 kr"
uv run listings.py add <finn-url|finnkode> "Rugdeberget 30" 4800000 --sold   # already sold → "SOLGT: 4 800 000 kr" (price optional)
uv run listings.py sold <finn-url|finnkode> [4700000]                        # existing row → "SOLGT: 4 700 000 kr" or "SOLGT"
```

Every row is either `Prisantydning: …` or `SOLGT[: …]`. `add` on a finnkode already in the table updates that row's status instead of duplicating it.
New rows are placed newest-first by finnkode, so a new listing goes to the top and a sold one added after the fact lands among listings from the same period.

Options (before or after the subcommand):

| Option | |
|---|---|
| `-o, --out FILE` | write the new page source to a file instead of stdout |
| `--copy` | also copy it to the Windows clipboard (`clip.exe`, WSL) |
| `--source URL\|FILE` | read from somewhere other than the live page: an earlier output, or a saved copy of the page |

Status messages go to stderr, so `> new.html` captures only the HTML.

Several changes before pasting: chain through a file, because each run starts from the live page otherwise.

```bash
uv run listings.py add 480000001 "Rugdeberget 22" 4590000 -o new.html
uv run listings.py sold 477062154 5950000 --source new.html -o new.html --copy
```

Then in the borettslag.net editor: open the page, switch to source view, select everything, paste, and save.
