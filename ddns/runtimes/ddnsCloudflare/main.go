package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/sander-skjulsvik/tools/ddns/ddns"
)

var ART string = `
_____ _    _       _           _ _    
  / ____| |  (_)     | |         (_) |   
 | (___ | | ___ _   _| |_____   ___| | __
  \___ \| |/ / | | | | / __\ \ / / | |/ /
  ____) |   <| | |_| | \__ \\ V /| |   < 
 |_____/|_|\_\ |\__,_|_|___/ \_/ |_|_|\_\
            _/ |                         
           |__/
`

// journalctl --user -xeu ddns-cloudflare.service
func main() {
	log.Printf("\n\n %s \n\n", ART)
	token := os.Getenv("TOKEN")
	if token == "" || token == "TOKEN" {
		log.Fatalf("\n\nTOKEN: is not set, stopping\n\n")
	}
	zoneID := os.Getenv("ZONE_ID")
	if zoneID == "" || zoneID == "ZONE_ID" {
		log.Fatalf("\n\nZONE_ID: is not set, stopping\n\n")
	}
	dnsRecordID := os.Getenv("DNS_RECORD_ID")
	if dnsRecordID == "" || dnsRecordID == "DNS_RECORD_ID" {
		log.Fatalf("\n\nDNS_RECORD_ID: is not set, stopping\n\n")
	}
	domain := os.Getenv("DOMAIN")
	if domain == "" || domain == "DOMAIN" {
		log.Fatalf("\n\nDOMAIN: is not set, stopping\n\n")
	}
	ddns.Run(ddns.NewDefaultCloudflareConfig(
		token, zoneID, dnsRecordID, domain,
	))
}
