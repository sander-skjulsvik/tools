DATE := $$(date '+%Y-%m-%d-%H-%M-%S')

ddns-cloudflare-docker-image:
	docker build -t ddnscloudflare-$(DATE) -f containerfiles/ddnsCloudflare.dockerfile .