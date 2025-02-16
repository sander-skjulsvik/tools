# Github Workflows

## podman-release

This runs on release published, and builds a podman image for cloudflareddns. Published images can be found at: https://github.com/sander-skjulsvik/tools/pkgs/container/tools%2Fddnscloudflare



## Locally testing github actions

For this we can use `act`

Installation: <https://nektosact.com/installation/index.html>



## Running on manuall trigger

use 
```yaml
on:
  workflow_dispatch:

```

