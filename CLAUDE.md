# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Go monorepo (module: `github.com/sander-skjulsvik/tools`) containing CLI utilities and a DDNS service. All tools share the same `go.mod` at the repo root.

## Common Commands

```bash
# Build all binaries
go build -v ./...

# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./dupes/...
go test -v ./dupesCompareDirs/...
go test -v ./ddns/...

# Build container image for ddns-cloudflare (podman)
make ddns-cloudflare-podman-image

# Build container image for ddns-cloudflare (docker)
make ddns-cloudflare-docker-image
```

Windows-specific build/test targets are `make win-build` and `make win-test`.

## Architecture

### Tools

**`dupes/`** — CLI that finds duplicate files within a single directory tree. Supports multiple concurrency strategies selectable at runtime via `-method` flag: `single` (single-threaded) and `producerConsumer` (concurrent). Core logic lives in `dupes/lib/common/`; `dupes/lib/singleThread/` and `dupes/lib/producerConsumer/` each implement the `common.Run` function signature.

**`dupesCompareDirs/`** — CLI that compares duplicate files across two directory trees. Similar concurrency model (`-runMode` flag). Supports three comparison modes via `-mode`: `onlyInBoth`, `onlyInFirst`, `all`. Core logic is in `dupesCompareDirs/lib/`.

**`ddns/`** — Dynamic DNS service that keeps a Cloudflare DNS A record in sync with the host's public IP (polled every 20 seconds via ipify.org). Structured as:
- `ddns/ddns/` — core loop, `DNSProviderClient` interface, IP resolution helpers
- `ddns/cloudflare/` — Cloudflare implementation of `DNSProviderClient` using `cloudflare-go/v4`
- `ddns/runtimes/ddnsCloudflare/` — `main` package; reads credentials from env vars `TOKEN`, `ZONE_ID`, `DNS_RECORD_ID`, `DOMAIN`

### Shared Libraries (`libs/`)

- `libs/files/` — filesystem helpers (walk, file count, directory size)
- `libs/collections/` — generic collection utilities
- `libs/progressbar/` — progress bar abstraction wrapping `gosuri/uiprogress`

### Releases

On GitHub release creation, CI builds binaries for `linux/windows/freebsd` × `amd64/arm` and uploads them as release assets. A container image for `ddnsCloudflare` is also built from `containerfiles/ddnsCloudflare.containerfile`.
