# Tools


## Statuses

![Test Status](https://github.com/sander-skjulsvik/tools/actions/workflows/test.yml/badge.svg)

## Readmes

[Github Actions](.github/workflows/README.md)
[DDNS](ddns/README.md)
[finn-search](finn-search/README.md)
[publicip](publicip/README.md)
[tls-check](tls-check/README.md)


## Documentations

### DDNS

![DDNS](ddns/README.md)

### finn-search

Scrapes Finn.no Torget search results to JSON. Supports pagination, full item detail fetching (description, condition, category), and a debug mode. Run with `uv` — no install needed.

See [finn-search/README.md](finn-search/README.md).

### publicip

Prints the machine's current public IP address (IPv4 by default, `-6` for IPv6) using ipify.

See [publicip/README.md](publicip/README.md).

### tls-check

Checks that one or more domains serve a valid TLS certificate (trusted chain, matching hostname, not expired or expiring soon). Exits non-zero if any check fails.

See [tls-check/README.md](tls-check/README.md).
