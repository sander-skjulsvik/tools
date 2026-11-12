# Keep in sync with go.mod; CI and the Makefile pass it with --build-arg GO_VERSION=...
ARG GO_VERSION=1.27.3
FROM docker.io/library/golang:${GO_VERSION} AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/autodeploy ./autodeploy/

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/autodeploy /bin/autodeploy
# No user database in scratch, so a numeric non-root user
USER 65532:65532
ENTRYPOINT ["/bin/autodeploy"]
