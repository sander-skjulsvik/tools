package main

import (
	_ "embed"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sander-skjulsvik/tools/ddns/ddns"
	"github.com/sander-skjulsvik/tools/libs/vanity"
)

type config struct {
	Token       string `env:"TOKEN" env-required:"true"`
	ZoneID      string `env:"ZONE_ID" env-required:"true"`
	DNSRecordID string `env:"DNS_RECORD_ID" env-required:"true"`
	Domain      string `env:"DOMAIN" env-required:"true"`
	delay       int    `env:"DELAY" env-default:"20"`
}

// journalctl --user -xeu ddns-cloudflare.service
func main() {
	log.Printf("\n\n %s \n\n", vanity.Skjulsvik)

	var cfg config
	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		log.Fatalf("\n\nError reading environment variables: %v\n\n", err)

	}
	ddns.New(
		cfg.Token, cfg.ZoneID, cfg.DNSRecordID, cfg.Domain, cfg.delay,
	).Run()
}
