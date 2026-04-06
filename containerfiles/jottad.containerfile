FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

ARG JOTTA_VERSION=0.17.159692
# Archive is a full filesystem tarball (./usr/bin/jottad, ./usr/bin/jotta-cli)
RUN curl -fsSL "https://repo.jotta.us/archives/linux/amd64/jotta-cli-${JOTTA_VERSION}_linux_amd64.tar.gz" \
    | tar xz -C / \
    && chmod +x /usr/bin/jottad /usr/bin/jotta-cli

# jottad stores config and state at /var/lib/jottad
# Mount a PVC here to persist credentials and sync state across restarts
VOLUME ["/var/lib/jottad"]

# jotta-cli communicates with jottad on this port (localhost only)
EXPOSE 14443

ENTRYPOINT ["/usr/bin/jottad", "stdoutlog"]
