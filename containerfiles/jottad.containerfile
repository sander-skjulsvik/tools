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

# What jottad saw in the ubuntu image: CA certs, root user with /root as home,
# os-release, an empty machine-id and writable /tmp
RUN mkdir -p /rootfs/etc/ssl/certs /rootfs/root /rootfs/tmp /rootfs/var/lib/jottad \
    && chmod 700 /rootfs/root \
    && chmod 1777 /rootfs/tmp \
    && cp /etc/ssl/certs/ca-certificates.crt /rootfs/etc/ssl/certs/ \
    && cp /usr/lib/os-release /rootfs/etc/os-release \
    && touch /rootfs/etc/machine-id \
    && echo 'root:x:0:0:root:/root:/sbin/nologin' > /rootfs/etc/passwd \
    && echo 'root:x:0:' > /rootfs/etc/group

FROM scratch
COPY --from=download /rootfs/ /

# jottad stores config and state at /var/lib/jottad
# Mount a PVC here to persist credentials and sync state across restarts
VOLUME ["/var/lib/jottad"]

# jotta-cli communicates with jottad on this port (localhost only)
EXPOSE 14443

ENTRYPOINT ["/usr/bin/jottad", "stdoutlog"]
