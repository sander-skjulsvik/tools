FROM ubuntu:24.04 AS download

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

ARG JOTTA_VERSION=0.17.159692
# Archive is a full filesystem tarball; jottad and jotta-cli are static Go binaries
RUN mkdir -p /rootfs \
    && curl -fsSL "https://repo.jotta.us/archives/linux/amd64/jotta-cli-${JOTTA_VERSION}_linux_amd64.tar.gz" \
    | tar xz -C /rootfs ./usr/bin/jottad ./usr/bin/jotta-cli ./etc/jottad/config.ini \
    && chmod +x /rootfs/usr/bin/jottad /rootfs/usr/bin/jotta-cli

# Not scratch: jottad shells out to coreutils `stat -f --format="%T %i" <path>`
# to get the filesystem type and id of every backup root. Without it every
# scan fails with "Unable to get filesystemid for path".
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=download /rootfs/ /

# jottad stores config and state at /var/lib/jottad
# Mount a PVC here to persist credentials and sync state across restarts
VOLUME ["/var/lib/jottad"]

# jotta-cli communicates with jottad on this port (localhost only)
EXPOSE 14443

ENTRYPOINT ["/usr/bin/jottad", "stdoutlog"]
