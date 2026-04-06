FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

ARG JOTTA_VERSION=0.17.159692
RUN curl -fsSL "https://repo.jotta.us/archives/linux/amd64/jotta-cli-${JOTTA_VERSION}_linux_amd64.tar.gz" \
    | tar xz -C /usr/local/bin/ \
    && chmod +x /usr/local/bin/jottad /usr/local/bin/jotta-cli

# jottad stores config and state under $HOME by default
# Mount a PVC at /root to persist credentials and sync state across restarts
VOLUME ["/root"]

# jotta-cli communicates with jottad on this port (localhost only)
EXPOSE 14443

ENTRYPOINT ["jottad"]
