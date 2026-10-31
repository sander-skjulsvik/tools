package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"
)

func main() {
	port := flag.String("port", "443", "port to connect to when the domain has no port")
	warnDays := flag.Int("warn", 14, "fail if the certificate expires within this many days")
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	ip := flag.String("ip", "", "connect to this IP instead of resolving the domain via DNS")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("usage: tls-check [-ip <addr>] [-port 443] [-warn 14] [-timeout 10s] <domain> [domain...]")
	}
	if *ip != "" {
		if _, err := netip.ParseAddr(*ip); err != nil {
			log.Fatalf("invalid -ip %q: %v", *ip, err)
		}
	}

	failed := false
	for _, arg := range flag.Args() {
		host, port := parseTarget(arg, *port)
		dialHost := host
		if *ip != "" {
			dialHost = *ip
		}
		if !check(host, port, dialHost, *warnDays, *timeout) {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// parseTarget accepts "example.com", "example.com:8443" or "https://example.com/path"
// and returns the hostname (for SNI and verification) and the port.
func parseTarget(arg, defaultPort string) (string, string) {
	s := arg
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		host, port = s, defaultPort
	}
	return host, port
}

// check connects to dialHost:port, verifies the served certificate chain
// against the system roots for host, prints a report and returns whether the
// cert is OK. dialHost differs from host when checking a specific server IP.
func check(host, port, dialHost string, warnDays int, timeout time.Duration) bool {
	addr := net.JoinHostPort(dialHost, port)
	label := net.JoinHostPort(host, port)
	if dialHost != host {
		label += " via " + dialHost
	}

	dialer := &net.Dialer{Timeout: timeout}
	// Verification is done manually below so the certificate details can be
	// shown even when it is invalid.
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		fmt.Printf("%s: FAIL\n  error:   %v\n", label, err)
		return false
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		fmt.Printf("%s: FAIL\n  error:   server presented no certificate\n", label)
		return false
	}
	leaf := certs[0]

	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	_, verifyErr := leaf.Verify(x509.VerifyOptions{
		DNSName:       host,
		Intermediates: intermediates,
	})

	daysLeft := int(time.Until(leaf.NotAfter).Hours() / 24)
	expiringSoon := verifyErr == nil && daysLeft < warnDays

	status := "OK"
	if verifyErr != nil {
		status = "FAIL"
	} else if expiringSoon {
		status = "WARN"
	}

	fmt.Printf("%s: %s\n", label, status)
	fmt.Printf("  subject: %s\n", leaf.Subject)
	fmt.Printf("  issuer:  %s\n", leaf.Issuer)
	if len(leaf.DNSNames) > 0 {
		fmt.Printf("  names:   %s\n", formatNames(leaf.DNSNames, 5))
	}
	fmt.Printf("  valid:   %s -> %s (%d days left)\n",
		leaf.NotBefore.Format(time.DateOnly), leaf.NotAfter.Format(time.DateOnly), daysLeft)
	if verifyErr != nil {
		fmt.Printf("  error:   %v\n", verifyErr)
	}
	if expiringSoon {
		fmt.Printf("  error:   expires within %d days\n", warnDays)
	}

	return status == "OK"
}

func formatNames(names []string, max int) string {
	if len(names) <= max {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(names[:max], ", "), len(names)-max)
}
