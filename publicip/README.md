# publicip

Prints the current public IP address of this machine, as seen by [ipify](https://www.ipify.org/).

## Usage

```
publicip [-6] [-timeout 10s]
```

- `-6` — get the public IPv6 address instead of IPv4
- `-timeout` — request timeout (default `10s`)

## Example

```
$ publicip
203.0.113.42
```

## Build

```
make publicip
```

## Install

Installs to `$(go env GOPATH)/bin` (or `$GOBIN` if set):

```
make install-publicip
```
