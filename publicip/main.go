package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

func main() {
	ipv6 := flag.Bool("6", false, "get public IPv6 address instead of IPv4")
	timeout := flag.Duration("timeout", 10*time.Second, "request timeout")
	flag.Parse()

	url := "https://api.ipify.org"
	if *ipv6 {
		url = "https://api6.ipify.org"
	}

	ip, err := getPublicIP(url, *timeout)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ip)
}

func getPublicIP(url string, timeout time.Duration) (netip.Addr, error) {
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("request failed: %s, err: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("unexpected status from %s: %s", url, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to read response: %w", err)
	}

	ip, err := netip.ParseAddr(strings.TrimSpace(string(b)))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to parse ip: %w", err)
	}
	return ip, nil
}
