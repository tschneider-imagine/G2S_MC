package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/tschneider-imagine/G2S_MC/internal/certs"
)

func main() {
	out := flag.String("out", "certs", "output directory for development certificates")
	flag.Parse()

	paths, err := certs.GenerateDevCerts(certs.DevCertOptions{
		OutputDir: *out,
		HostDNS:   []string{"localhost"},
		HostIPs:   []net.IP{net.ParseIP("127.0.0.1")},
	})
	if err != nil {
		log.Fatalf("generate dev certs: %v", err)
	}

	fmt.Printf("CA certificate: %s\n", paths.CACert)
	fmt.Printf("Host certificate: %s\n", paths.HostCert)
	fmt.Printf("Host key: %s\n", paths.HostKey)
	fmt.Printf("Client certificate: %s\n", paths.ClientCert)
	fmt.Printf("Client key: %s\n", paths.ClientKey)
}
