package ddns

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/sander-skjulsvik/tools/ddns/cloudflare"
)

type config struct {
	dnsProviderClient DNSProviderClient
	Domain            string
	PublicIPResolver  func() (netip.Addr, error)
}

func NewDefaultCloudflareConfig(token, ZoneID, dnsRecordID, domain string) config {
	return config{
		dnsProviderClient: cloudflare.New(token, ZoneID, dnsRecordID),
		Domain:            domain,
		PublicIPResolver:  getPublicFromIPIFY,
	}
}

func Run(conf config) {

	// Event loop
	SleepingEventLoop(20*time.Second, func() {
		// Get public ip address
		myPublicIP, err := conf.PublicIPResolver()
		if err != nil {
			slog.Error("failed to get public ip", "err", err)
			return
		}

		// Compare with the record at the provider, not with a DNS lookup: the
		// local resolver may answer split-horizon (the LAN IP), which never
		// matches the public IP and would rewrite the record on every tick.
		currentDNSIP, err := conf.dnsProviderClient.GetDomainValue()
		if err != nil {
			slog.Error("failed to get dns record", "domain", conf.Domain, "err", err)
			return
		}

		slog.Info("checked", "dns_ip", currentDNSIP.String(), "public_ip", myPublicIP.String())
		if currentDNSIP == myPublicIP {
			slog.Info("no change", "ip", currentDNSIP.String())
			return
		}

		// Set ip for domain
		err = conf.dnsProviderClient.SetDomainValue(myPublicIP.String())
		if err != nil {
			slog.Error("failed to set dns value", "err", err)
			return
		}
	})
}

type DNSProviderClient interface {
	Info()
	GetDomainValue() (netip.Addr, error)
	SetDomainValue(ip string) error
}

func SleepingEventLoop(sleepTime time.Duration, f func()) {
	for {
		f()
		time.Sleep(sleepTime)
	}
}

func getPublicFromIPIFY() (netip.Addr, error) {
	url := "https://api.ipify.org?format=json"
	resp, err := http.Get(url)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("request failed: %s, err: %w", url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to parse response: %w", err)
	}

	type IP struct {
		Ip string `json:"ip"`
	}
	var r IP
	if err = json.Unmarshal(b, &r); err != nil {
		return netip.Addr{}, fmt.Errorf("failed to parse ip: %s", err)
	}
	return netip.ParseAddr(r.Ip)
}
