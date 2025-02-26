package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/sander-skjulsvik/tools/ddns/ddns"
)

func main() {

	token := os.Getenv("TOKEN")
	if token == "" || token == "TOKEN" {
		log.Panicf("TOKEN: is not set, stopping")
	}
	zoneID := os.Getenv("ZONE_ID")
	if zoneID == "" || zoneID == "ZONE_ID" {
		log.Panicf("ZONE_ID: is not set, stopping")
	}
	dnsRecordID := os.Getenv("DNS_RECORD_ID")
	if dnsRecordID == "" || dnsRecordID == "DNS_RECORD_ID" {
		log.Panicf("DNS_RECORD_ID: is not set, stopping")
	}
	domain := os.Getenv("DOMAIN")
	if domain == "" || domain == "DOMAIN" {
		log.Panicf("DOMAIN: is not set, stopping")
	}
	ddns.Run(ddns.NewDefaultCloudflareConfig(
		token, zoneID, dnsRecordID, domain,
	))
}
