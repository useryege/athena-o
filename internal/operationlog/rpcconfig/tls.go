package rpcconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

func ServerCredentials(c Server) (credentials.TransportCredentials, error) {
	if c.Transport == "plaintext" {
		return insecure.NewCredentials(), nil
	}
	cert, e := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if e != nil {
		return nil, fmt.Errorf("cannot load operation-log TLS certificate and key")
	}
	return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}), nil
}
func ClientCredentials(c Client) (credentials.TransportCredentials, error) {
	if c.Transport == "plaintext" {
		return insecure.NewCredentials(), nil
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.ServerName}
	if c.CAFile != "" {
		b, e := os.ReadFile(c.CAFile)
		if e != nil {
			return nil, fmt.Errorf("cannot read operation-log TLS CA")
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("invalid operation-log TLS CA")
		}
		cfg.RootCAs = roots
	}
	return credentials.NewTLS(cfg), nil
}
