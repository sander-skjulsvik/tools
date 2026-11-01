package cloudflare

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

// Shaped like a real GET /zones/{zone}/dns_records/{id} response for an A record.
const aRecordResponse = `{
  "result": {
    "id": "rec123",
    "name": "example.com",
    "type": "A",
    "content": "203.0.113.7",
    "proxiable": true,
    "proxied": false,
    "ttl": 1,
    "settings": {},
    "meta": {},
    "comment": null,
    "tags": [],
    "created_on": "2025-04-01T10:00:00.000000Z",
    "modified_on": "2026-10-28T02:01:16.299582Z"
  },
  "success": true,
  "errors": [],
  "messages": []
}`

func TestGetDomainValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/zones/zone123/dns_records/rec123" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(aRecordResponse))
	}))
	defer srv.Close()

	dc := &DnsClient{
		DNSService:  *dns.NewDNSService(option.WithAPIToken("t"), option.WithBaseURL(srv.URL+"/")),
		dnsRecordID: "rec123",
		zoneID:      "zone123",
	}
	ip, err := dc.GetDomainValue()
	if err != nil {
		t.Fatal(err)
	}
	if ip.String() != "203.0.113.7" {
		t.Fatalf("got %s, want 203.0.113.7", ip)
	}
}
