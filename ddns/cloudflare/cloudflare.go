package cloudflare

import (
	"context"
	"fmt"
	"log/slog"
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
		slog.Error("failed to list dns records", "err", err)
		return
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
	duration := after.Sub(before).Seconds()
	slog.Info("updated", "record_id", dc.dnsRecordID, "ip", ip.String(), "duration_s", duration)

	moddedTime := res.ModifiedOn
	if !moddedTime.After(before.Add(-10 * time.Minute)) {
		slog.Warn("modified timestamp is too old", "modified_on", moddedTime, "age_s", before.Sub(moddedTime).Seconds())
	}
	return nil
}
