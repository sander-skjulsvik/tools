package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/sander-skjulsvik/tools/ddns/ddns"
)

func main() {

	type conf struct {
		Token       string
		ZoneID      string
		DnsRecordID string
		Domain      string
	}
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Panicf("TOKEN: is not set, stopping")
	}
	zoneID := os.Getenv("ZONE_ID")
	if zoneID == "" {
		log.Panicf("ZONE_ID: is not set, stopping")
	}
	dnsRecordID := os.Getenv("DNS_RECORD_ID")
	if dnsRecordID == "" {
		log.Panicf("DNS_RECORD_ID: is not set, stopping")
	}
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		log.Panicf("DOMAIN: is not set, stopping")
	}
	c := conf{
		Token:       token,
		ZoneID:      zoneID,
		DnsRecordID: dnsRecordID,
		Domain:      domain,
	}

	log.Printf("Conf: %+v", c)
	ddns.Run(ddns.NewDefaultCloudflareConfig(
		token, zoneID, dnsRecordID, domain,
	))
}
