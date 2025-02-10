package cloudflare

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

type DnsClient struct {
	dns.DNSService
	dnsRecordID string
	zoneID      string
}

func New(token, zoneID, dnsRecordID string) *DnsClient {
	dnsServiceClient := dns.NewDNSService(
		option.WithAPIToken(token),
		option.WithBaseURL("https://api.cloudflare.com/client/v4/"),
	)

	return &DnsClient{
		dnsRecordID: dnsRecordID,
		zoneID:      zoneID,
		DNSService:  *dnsServiceClient,
	}
}

func (dc *DnsClient) Info() {

	res, err := dc.DNSService.Records.List(
		context.TODO(),
		dns.RecordListParams{
			ZoneID: cloudflare.F(dc.zoneID),
		},
	)
	if err != nil {
		log.Fatalf("Help: %v", err)
	}

	fmt.Printf("%s", res.JSON.RawJSON())
}

func (dc *DnsClient) SetDomainValue(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("SetDomainValue got none valid ip: %s", value)
	}
	before := time.Now()
	res, err := dc.DNSService.Records.Edit(
		context.TODO(),
		dc.dnsRecordID,
		dns.RecordEditParams{
			ZoneID: cloudflare.F(dc.zoneID),
			Record: dns.ARecordParam{
				Content: cloudflare.F(ip.String()),
			},
		},
	)
	after := time.Now()
	if err != nil {
		return fmt.Errorf("failed to set value: %s, err: %w", value, err)
	}
	log.Printf("updated: %s to point to: %s, it took: %fs", dc.dnsRecordID, ip.String(), after.Sub(before).Seconds())

	moddedTime := res.ModifiedOn
	if !moddedTime.After(before.Add(-10 * time.Minute)) {
		log.Printf("Warning: Modded time is too long ago: %fs", before.Sub(moddedTime).Seconds())
	}
	return nil
}
