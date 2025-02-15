FROM golang:1.24 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

ADD ./ ./

RUN GOOS=linux go build -o /bin/service /app/ddns/runtimes/ddnsCloudflare/main.go

FROM scratch
COPY --from=build /bin/service /bin/service
CMD ["/bin/service"]
