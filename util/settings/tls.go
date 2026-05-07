package settings

import (
	"crypto/tls"
	"crypto/x509"

	log "github.com/sirupsen/logrus"

	tlsutil "github.com/useryege/athena/util/tls"
)

// TLSConfig returns a tls.Config with the configured certificates
func (a *AthenaSettings) TLSConfig() *tls.Config {
	if a.Certificate == nil {
		return nil
	}

	certPool := x509.NewCertPool()
	pemCertBytes, _ := tlsutil.EncodeX509KeyPair(*a.Certificate)
	ok := certPool.AppendCertsFromPEM(pemCertBytes)
	if !ok {
		panic("bad certs")
	}

	return &tls.Config{
		RootCAs: certPool,
	}
}

// OIDCTLSConfig returns the TLS config for the OIDC provider. If an external provider is configured, returns a TLS
// config using the root CAs (if any) specified in the OIDC config. If an external OIDC provider is not configured,
// returns the API server TLS config, because the API server proxies requests to Dex.
func (a *AthenaSettings) OIDCTLSConfig() *tls.Config {
	var tlsConfig *tls.Config

	oidcConfig := a.OIDCConfig()
	if oidcConfig != nil {
		tlsConfig = &tls.Config{}
		if oidcConfig.RootCA != "" {
			certPool := x509.NewCertPool()
			ok := certPool.AppendCertsFromPEM([]byte(oidcConfig.RootCA))
			if !ok {
				log.Warn("failed to append certificates from PEM: proceeding without custom rootCA")
			} else {
				tlsConfig.RootCAs = certPool
			}
		}
	} else {
		tlsConfig = a.TLSConfig()
	}

	if tlsConfig != nil && a.OIDCTLSInsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig
}
