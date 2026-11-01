package cloudflare

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
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

// GetDomainValue returns the IP the A record currently points to.
//
// The response body is decoded into our own struct: the SDK's (v4.0.0)
// RecordResponse union leaves Content empty for A records.
func (dc *DnsClient) GetDomainValue() (netip.Addr, error) {
	var body struct {
		Result struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"result"`
	}
	_, err := dc.DNSService.Records.Get(
		context.TODO(),
		dc.dnsRecordID,
		dns.RecordGetParams{ZoneID: cloudflare.F(dc.zoneID)},
		option.WithResponseBodyInto(&body),
	)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to get record %s: %w", dc.dnsRecordID, err)
	}
	if body.Result.Type != "A" {
		return netip.Addr{}, fmt.Errorf("record %s is type %q, want A", dc.dnsRecordID, body.Result.Type)
	}
	ip, err := netip.ParseAddr(body.Result.Content)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("record %s has non-IP content %q: %w", dc.dnsRecordID, body.Result.Content, err)
	}
	return ip, nil
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
