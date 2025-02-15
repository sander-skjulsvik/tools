actions:
	act

win-build:
	go build -o bin\\ .\\...

win-test:
	go test .\\...

win-deps:
	choco install golangci-lint act-cli

DATE := $$(date '+%Y-%m-%d-%H-%M-%S')

ddns-cloudflare-docker-image:
	docker build -t ddnscloudflare-$(DATE) -f containerfiles/ddnsCloudflare.dockerfile .