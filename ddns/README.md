# DDNS

This package is for running a service which maintains a dns A records IP address to the public ip for the local network where the service is runnign.

You can run the programm in two different ways, either you can use one of the runtimes in the `ddns/runtimes/` folder or you can use the Run function in `ddns/ddns/ddns.go`. Currently only Clouflare DNS is implemented.

:mega: This package is currently loosely tested!

## Packages

Upon release a new podman image and a executable will be built and stored in the repo in github.

