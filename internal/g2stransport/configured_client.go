package g2stransport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const configuredClientDefaultTimeoutMS = 5000

type HTTPClientConfig struct {
	TLSRequired      bool
	RootCAPath       string
	ClientCertPath   string
	ClientKeyPath    string
	DefaultTimeoutMS int
}

func (c HTTPClientConfig) normalize() HTTPClientConfig {
	normalized := HTTPClientConfig{
		TLSRequired:      c.TLSRequired,
		RootCAPath:       strings.TrimSpace(c.RootCAPath),
		ClientCertPath:   strings.TrimSpace(c.ClientCertPath),
		ClientKeyPath:    strings.TrimSpace(c.ClientKeyPath),
		DefaultTimeoutMS: c.DefaultTimeoutMS,
	}
	if normalized.DefaultTimeoutMS <= 0 {
		normalized.DefaultTimeoutMS = configuredClientDefaultTimeoutMS
	}
	return normalized
}

type HTTPClientFactory struct {
	config    HTTPClientConfig
	once      sync.Once
	transport *http.Transport
	cachedErr error
}

func NewConfiguredHTTPSender(config HTTPClientConfig) *HTTPSender {
	factory := NewHTTPClientFactory(config)
	return &HTTPSender{
		ClientFactory: factory.NewClient,
	}
}

func NewHTTPClientFactory(config HTTPClientConfig) *HTTPClientFactory {
	return &HTTPClientFactory{config: config.normalize()}
}

func (f *HTTPClientFactory) NewClient(timeoutMS int) (*http.Client, error) {
	f.once.Do(func() {
		f.transport, f.cachedErr = buildConfiguredTransport(f.config)
	})
	if f.cachedErr != nil {
		return nil, f.cachedErr
	}
	effectiveTimeoutMS := timeoutMS
	if effectiveTimeoutMS <= 0 {
		effectiveTimeoutMS = f.config.DefaultTimeoutMS
	}
	clone := f.transport.Clone()
	return &http.Client{
		Transport: clone,
		Timeout:   time.Duration(effectiveTimeoutMS) * time.Millisecond,
	}, nil
}

func buildConfiguredTransport(config HTTPClientConfig) (*http.Transport, error) {
	normalized := config.normalize()
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	if normalized.TLSRequired && normalized.RootCAPath == "" {
		return nil, fmt.Errorf("ca trust path is required when tls is required")
	}
	if normalized.RootCAPath != "" {
		rawCA, err := os.ReadFile(normalized.RootCAPath)
		if err != nil {
			return nil, fmt.Errorf("load ca trust material %q: %w", normalized.RootCAPath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(rawCA) {
			return nil, fmt.Errorf("ca trust material %q does not contain a valid certificate", normalized.RootCAPath)
		}
		tlsConfig.RootCAs = pool
	}

	clientCertPath := normalized.ClientCertPath
	clientKeyPath := normalized.ClientKeyPath
	if (clientCertPath == "") != (clientKeyPath == "") {
		return nil, fmt.Errorf("both client certificate and key paths are required together")
	}
	if clientCertPath != "" && clientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client certificate/key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}
