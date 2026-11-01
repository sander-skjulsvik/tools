# Github Workflows

## go-release

This runs on release created, and uploads a separate archive per tool and target (`<tool>-<tag>-<os>-<arch>`, `.zip` for windows, `.tar.gz` otherwise) with a `.sha256` file. Targets are `linux`, `windows`, `darwin` and `freebsd` on `amd64`, `arm64` and `arm` (ARMv7), except `windows/arm` and `darwin/arm`. Go is the version in `go.mod`. To release a new tool, add it to the `tool` matrix.

## podman-release

This runs on release published, and builds a Docker image for cloudflareddns. Published images can be found at: https://github.com/sander-skjulsvik/tools/pkgs/container/tools%2Fddnscloudflare



## Locally testing github actions

For this we can use `act`

Installation: <https://nektosact.com/installation/index.html>



## Running on manuall trigger

use 
```yaml
on:
  workflow_dispatch:

```

