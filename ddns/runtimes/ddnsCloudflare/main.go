package main

import (
	"log/slog"
	"os"

	"github.com/sander-skjulsvik/tools/ddns/ddns"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	token := os.Getenv("TOKEN")
	if token == "" || token == "TOKEN" {
		slog.Error("TOKEN is not set, stopping")
		os.Exit(1)
	}
	zoneID := os.Getenv("ZONE_ID")
	if zoneID == "" || zoneID == "ZONE_ID" {
		slog.Error("ZONE_ID is not set, stopping")
		os.Exit(1)
	}
	dnsRecordID := os.Getenv("DNS_RECORD_ID")
	if dnsRecordID == "" || dnsRecordID == "DNS_RECORD_ID" {
		slog.Error("DNS_RECORD_ID is not set, stopping")
		os.Exit(1)
	}
	domain := os.Getenv("DOMAIN")
	if domain == "" || domain == "DOMAIN" {
		slog.Error("DOMAIN is not set, stopping")
		os.Exit(1)
	}
	ddns.Run(ddns.NewDefaultCloudflareConfig(
		token, zoneID, dnsRecordID, domain,
	))
}
