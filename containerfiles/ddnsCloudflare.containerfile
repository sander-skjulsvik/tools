# Keep in sync with go.mod; CI and the Makefile pass it with --build-arg GO_VERSION=...
ARG GO_VERSION=1.27.3
FROM docker.io/library/golang:${GO_VERSION} AS build
WORKDIR /app
# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: Ensure a fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service /app/ddns/runtimes/ddnsCloudflare/main.go

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/service /bin/service
ENTRYPOINT ["/bin/service"]
