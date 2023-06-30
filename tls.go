package openldap_exporter

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ClientConfig represents the standard client TLS config.
type ClientConfig struct {
	TLSCA               string
	TLSCert             string
	TLSKey              string
	TLSKeyPwd           string
	InsecureSkipVerify  bool
	ServerName          string
	RenegotiationMethod string
}

func (c *ClientConfig) TLSConfig() (*tls.Config, error) {
	var renegotiationMethod tls.RenegotiationSupport
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
		Renegotiation:      renegotiationMethod,
	}

	if c.TLSCA != "" {
		pool, err := makeCertPool([]string{c.TLSCA})
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = pool
	}

	if c.ServerName != "" {
		tlsConfig.ServerName = c.ServerName
	}

	return tlsConfig, nil
}

func makeCertPool(certFiles []string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, certFile := range certFiles {
		cert, err := os.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("could not read certificate %q: %w", certFile, err)
		}
		if !pool.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("could not parse any PEM certificates %q: %w", certFile, err)
		}
	}
	return pool, nil
}
