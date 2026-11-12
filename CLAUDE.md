# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

This repository is public. Follow the rules on personal data in @AGENTS.md before every commit and pull request.

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

**`autodeploy/`** — Service that runs in a Kubernetes cluster and keeps one deployment on the newest `vX.Y.Z` tag of its image. Every interval it lists the tags through the registry API (token flow, credentials from a mounted image pull secret, re-read each time) and strategic-merge-patches the container image when a newer release exists. It never downgrades, refuses containers running another image, and skips deployments annotated `autodeploy/paused=true`. Talks to the Kubernetes API with plain `net/http` and the service account token (re-read per request), no client-go. Image from `containerfiles/autodeploy.containerfile`; not in the binary release matrix, since it only runs in a cluster.

**`immich-picker/`** — CLI that builds a local catalog of one person's photos in Immich and shortlists them. It must stay read-only towards Immich: every request has to be listed in `allowedRequests` in `immich.go`, and the tool is not allowed to add write requests there without explicit review.

**`location-timeline/`** — CLI that works out where the user has been from the location metadata of their Immich photos: stays, days and countries per period, plus JSON for other tools. Only photos from the user's own cameras or with the user in them count (`config.json` in the data folder). Periods without photos are shown as gaps, never filled in. Each data source implements `Source` in `source.go` and has its own section in the config; reports only see the merged `[]Point`. Immich is the first source; Google location history is planned as a second. The same read-only rule as `immich-picker` applies to its `allowedRequests` in `immich.go`.

### Shared Libraries (`libs/`)

- `libs/files/` — filesystem helpers (walk, file count, directory size)
- `libs/collections/` — generic collection utilities
- `libs/progressbar/` — progress bar abstraction wrapping `gosuri/uiprogress`
- `libs/immich/` — read-only Immich client: each tool passes the only requests it may make, and anything else is refused before it is sent. Redirects are never followed, so the API key stays on the server it was meant for

### Releases

On GitHub release creation, CI (`.github/workflows/go-release.yaml`) builds every tool for `linux/windows/darwin/freebsd` × `amd64/arm64/arm` (minus `windows/arm` and `darwin/arm`) and uploads one archive per tool and target as release assets, plus a `checksums.txt` covering all of them. Releases are versioned `vX.Y.Z`: `make release` bumps the latest such tag on GitHub (`BUMP=minor` by default, or `major`/`patch`, or an explicit `VERSION=v1.2.3`), starting at `v0.1.0`; it refuses when the default branch on GitHub has nothing new since the latest version. See `scripts/release.sh`. Container images are tagged with the release tag, and real releases also move `:latest`. `make test-release` creates a `test-<datetime>` pre-release that runs the same builds without becoming the latest release or image. Stay below `v2`: Go requires a `/v2` module path suffix from there. New tools must be added to its matrix.

The Go version comes from `go.mod` everywhere: the release and test workflows read it directly, and container builds get it as the `GO_VERSION` build arg. A container image for `ddnsCloudflare` is also built from `containerfiles/ddnsCloudflare.containerfile`.
