# tls-check

Checks that a domain serves a valid TLS certificate. For each domain it connects, verifies the certificate chain against the system's trusted roots, checks that the certificate matches the hostname and isn't expired, and prints the certificate details.

## Usage

```
tls-check [-ip <addr>] [-port 443] [-warn 14] [-timeout 10s] <domain> [domain...]
```

- `-ip`: connect to this IP instead of looking the domain up in DNS (see below)
- `-port`: port to use when the domain has no port (default `443`)
- `-warn`: fail if the certificate expires within this many days (default `14`, `0` to disable)
- `-timeout`: connection timeout (default `10s`)

Domains can be given as `example.com`, `example.com:8443` or `https://example.com/path`.

Each domain is reported as one of:

- `OK`: the certificate is valid
- `WARN`: the certificate is valid but expires within `-warn` days
- `FAIL`: the connection failed or the certificate is invalid (untrusted, wrong host, expired)

The exit code is `0` if every domain is `OK` and `1` otherwise, so it can be used in scripts and cron jobs.

## Example

```
$ tls-check example.com wrong.host.badssl.com
example.com:443: OK
  subject: CN=example.com
  issuer:  CN=Cloudflare TLS Issuing ECC CA 3,O=SSL Corporation,C=US
  names:   example.com, *.example.com
  valid:   2026-09-26 -> 2026-12-25 (55 days left)
wrong.host.badssl.com:443: FAIL
  subject: CN=*.badssl.com
  issuer:  CN=YR1,O=Let's Encrypt,C=US
  names:   *.badssl.com, badssl.com
  valid:   2026-09-29 -> 2026-12-28 (58 days left)
  error:   x509: certificate is valid for *.badssl.com, badssl.com, not wrong.host.badssl.com
```

## Checking a server before DNS points to it

Use `-ip` to check the certificate on a specific server, for example before moving DNS to a new host or when DNS isn't set up yet. tls-check connects to that IP but still asks for and verifies the certificate for the domain you give, the same way a browser would once DNS points there.

```
$ tls-check -ip 203.0.113.10 example.com
example.com:443 via 203.0.113.10: OK
  ...
```

When you give several domains, all of them are checked against the same IP. That's useful for a server that hosts multiple sites:

```
$ tls-check -ip 203.0.113.10 example.com www.example.com api.example.com
```

IPv6 addresses work too (`-ip 2001:db8::10`).

## Build

```
make tls-check
```

## Install

Installs to `$(go env GOPATH)/bin` (or `$GOBIN` if set):

```
make install-tls-check
```
