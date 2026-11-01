actions:
	act

.PHONY: isodate-nest
isodate-nest:
	go build -o bin/isodate-nest ./isodate-nest/

.PHONY: publicip
publicip:
	go build -o bin/publicip ./publicip/

.PHONY: tls-check
tls-check:
	go build -o bin/tls-check ./tls-check/

.PHONY: install install-publicip install-tls-check
install: install-publicip install-tls-check

# Installs into $(go env GOPATH)/bin (or GOBIN if set)
install-publicip:
	go install ./publicip/

install-tls-check:
	go install ./tls-check/

win-build:
	go build -o bin\\ .\\...

win-test:
	go test .\\...

win-deps:
	choco install golangci-lint act-cli

DATE := $$(date '+%Y-%m-%d-%H-%M-%S')

ddns-cloudflare-docker-image:
	docker build -t ddnscloudflare-$(DATE) --build-arg GO_VERSION=$$(grep -m1 '^go ' go.mod | cut -d' ' -f2) -f containerfiles/ddnsCloudflare.containerfile .

test-release:
	gh release create test-$(DATE) --notes "This is a test release $(DATE)"