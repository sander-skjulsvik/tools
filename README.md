# Tools


## Statuses

![Test Status](https://github.com/sander-skjulsvik/tools/actions/workflows/test.yml/badge.svg)

## Readmes

[Github Actions](.github/workflows/README.md)
[autodeploy](autodeploy/README.md)
[DDNS](ddns/README.md)
[finn-search](finn-search/README.md)
[immich-picker](immich-picker/README.md)
[publicip](publicip/README.md)
[tls-check](tls-check/README.md)


## Documentations

### autodeploy

Runs in a Kubernetes cluster and keeps one deployment on the newest `vX.Y.Z` release of its image, by polling the registry. Pull-based, with a service account that can only get and patch that one deployment.

See [autodeploy/README.md](autodeploy/README.md).

### DDNS

![DDNS](ddns/README.md)

### finn-search

Scrapes Finn.no Torget search results to JSON. Supports pagination, full item detail fetching (description, condition, category), and a debug mode. Run with `uv` — no install needed.

See [finn-search/README.md](finn-search/README.md).

### immich-picker

Builds a local catalog of the photos of one person in Immich and shortlists the good ones, using face size, sharpness and near-duplicate checks. Read-only towards Immich.

See [immich-picker/README.md](immich-picker/README.md).

### publicip

Prints the machine's current public IP address (IPv4 by default, `-6` for IPv6) using ipify.

See [publicip/README.md](publicip/README.md).

### tls-check

Checks that one or more domains serve a valid TLS certificate (trusted chain, matching hostname, not expired or expiring soon). Exits non-zero if any check fails.

See [tls-check/README.md](tls-check/README.md).
