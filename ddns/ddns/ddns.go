package ddns

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/sander-skjulsvik/tools/ddns/cloudflare"
)

type config struct {
	dnsProviderClient DNSProviderClient
	Domain            string
	DnsResolver       func(string) (netip.Addr, error) // For checking if the current dns ip is equal to current ip
	PublicIPResolver  func() (netip.Addr, error)
}

func NewDefaultCloudflareConfig(token, ZoneID, dnsRecordID, domain string) config {
	return config{
		dnsProviderClient: cloudflare.New(token, ZoneID, dnsRecordID),
		Domain:            domain,
		PublicIPResolver:  getPublicIPIPIFY,
		DnsResolver:       resolveDNS,
	}
}

func Run(conf config) {

	// Event loop
	SleepingEventLoop(20*time.Second, func() {
		// Get public ip address
		myPublicIP, err := conf.PublicIPResolver()
		if err != nil {
			log.Printf("%s", fmt.Errorf(""))
			return
		}

		// Check pub ip if it differs from current
		currentDNSIP, err := conf.DnsResolver(conf.Domain)
		if err != nil {
			log.Fatalf("Failed to lookup: %s, err: %s", conf.Domain, err)
			return
		}

		log.Printf("Current dns ip: %s", currentDNSIP)
		if currentDNSIP == myPublicIP {
			log.Printf("Current ip equals public ip, no change: %s", currentDNSIP)
			return
		}

		// Set ip for domain
		err = conf.dnsProviderClient.SetDomainValue(myPublicIP.String())
		if err != nil {
			log.Printf("failed to set value: %s", err)
			return
		}
	})
}

type DNSProviderClient interface {
	Info()
	SetDomainValue(ip string) error
}

func SleepingEventLoop(sleepTime time.Duration, f func()) {
	for {
		f()
		time.Sleep(sleepTime)
	}

}

func getPublicIPCustom() (string, error) {
	return "1.1.1.2", nil
}

func getPublicIPIPIFY() (netip.Addr, error) {
	url := "https://api.ipify.org?format=json"
	resp, err := http.Get(url)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to get public ip from: %s, err: %w", url, err)
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
	json.Unmarshal(b, &r)

	return netip.ParseAddr(r.Ip)
}

func resolveDNS(domain string) (netip.Addr, error) {
	res, err := net.LookupIP(domain)
	if err != nil {
		return netip.Addr{}, err
	}
	ips := []string{}
	for _, ip := range res {
		ips = append(ips, ip.String())
	}

	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("Warning: Domain did not resolve any ip addresses, setting one")
	}
	if len(ips) > 1 {
		return netip.Addr{}, fmt.Errorf("Warning: multiple ip addresses found for domain: %s", domain)
	}

	ip, err := netip.ParseAddr(ips[0])
	if err != nil {
		return netip.Addr{}, err
	}

	return ip, err
}
