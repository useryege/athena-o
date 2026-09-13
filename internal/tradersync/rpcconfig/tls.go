package rpcconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ServerCredentials loads the configured transport; plaintext is restricted to
// explicit loopback addresses even when configuration is constructed directly.
func ServerCredentials(c Server) (credentials.TransportCredentials, error) {
	if err := validateBoundary(c.ListenAddress, c.Transport, false); err != nil {
		return nil, err
	}
	if c.Transport == "loopback-insecure" {
		return insecure.NewCredentials(), nil
	}
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("Trader Sync TLS certificate and key files are required")
	}
	certificate, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load Trader Sync TLS certificate and key")
	}
	return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}), nil
}

// ClientCredentials also serves the standard health probe, which needs no token.
func ClientCredentials(c Client) (credentials.TransportCredentials, error) {
	if err := validateBoundary(c.Address, c.Transport, true); err != nil {
		return nil, err
	}
	if c.Transport == "loopback-insecure" {
		return insecure.NewCredentials(), nil
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.ServerName}
	if c.CAFile != "" {
		pem, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read Trader Sync TLS CA")
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("invalid Trader Sync TLS CA")
		}
		config.RootCAs = roots
	}
	return credentials.NewTLS(config), nil
}
