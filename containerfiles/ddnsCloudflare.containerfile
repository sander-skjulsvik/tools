FROM docker.io/library/golang:1.24 AS build
WORKDIR /app
# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: Ensure a fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service /app/ddns/runtimes/ddnsCloudflare/main.go

FROM scratch
COPY --from=build /bin/service /bin/service
ENTRYPOINT ["/bin/service"]
