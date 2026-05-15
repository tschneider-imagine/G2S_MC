package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/fakeegm"
)

func main() {
	hostURL := flag.String("host-url", "http://127.0.0.1:8444/g2s", "G2S host listener URL")
	egmID := flag.String("egm-id", "EGM-01", "fake EGM identifier")
	keepAliveCount := flag.Int("keepalive-count", 1, "number of keepAlive messages to send after commsOnLine")
	keepAliveInterval := flag.Duration("keepalive-interval", 2*time.Second, "delay between keepAlive messages")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request timeout")
	caPath := flag.String("ca", "", "optional CA certificate used to verify an HTTPS G2S host")
	certPath := flag.String("cert", "", "optional client certificate for mutual TLS")
	keyPath := flag.String("key", "", "optional client private key for mutual TLS")
	flag.Parse()

	client, err := fakeegm.NewWithTLSFiles(*hostURL, *egmID, *caPath, *certPath, *keyPath)
	if err != nil {
		log.Fatalf("create fake EGM client: %v", err)
	}
	client.HTTPClient.Timeout = *timeout

	ctx := context.Background()
	response, err := client.CommsOnLine(ctx)
	if err != nil {
		log.Fatalf("commsOnLine failed: %v", err)
	}
	fmt.Printf("commsOnLine -> HTTP %d in %s\n", response.StatusCode, response.Duration.Round(time.Millisecond))

	for i := 0; i < *keepAliveCount; i++ {
		if i > 0 || *keepAliveInterval > 0 {
			time.Sleep(*keepAliveInterval)
		}
		response, err := client.KeepAlive(ctx)
		if err != nil {
			log.Fatalf("keepAlive failed: %v", err)
		}
		fmt.Printf("keepAlive %d -> HTTP %d in %s\n", i+1, response.StatusCode, response.Duration.Round(time.Millisecond))
	}
}
