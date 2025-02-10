package main

import (
	_ "embed"

	"github.com/sander-skjulsvik/tools/ddns/ddns"
)

const ()

func main() {
	ddns.Run(ddns.NewDefaultCloudflareConfig())
}
