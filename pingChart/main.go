package main

import (
	"log"
	"os"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {

}

func RunPingChart() {

}

func GetPing() {
	url := "google.com"

	c, err := icmp.ListenPacket(
		"ip4:icmp",
		"0.0.0.0",
	)
	if err != nil {
		log.Fatalf("Failed to create icmp listener: %w", err)
	}
	defer c.Close()

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.ExtendedEchoRequest{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,,
			Data: []byte("Hei-hvordan-gaar-det"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatalf("Failed to marshal icmp message: %w", err)
	}

	// if _, err := c.WriteTo(
	// 	wb,
	// )
}
