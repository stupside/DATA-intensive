package crypto

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadCACertificate loads and parses a CA certificate from disk
func LoadCACertificate(certPath string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM block containing CA certificate")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return caCert, nil
}

// LoadCAWithKey loads both the CA certificate and private key from disk
func LoadCAWithKey(certPath, keyPath string) (*x509.Certificate, ed25519.PrivateKey, error) {
	caCert, err := LoadCACertificate(certPath)
	if err != nil {
		return nil, nil, err
	}

	caKey, err := loadCAPrivateKey(keyPath)
	if err != nil {
		return nil, nil, err
	}

	return caCert, caKey, nil
}

// loadCAPrivateKey loads and parses the CA private key from disk
func loadCAPrivateKey(keyPath string) (ed25519.PrivateKey, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing CA private key (expected PKCS8)")
	}

	caKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	ed25519Key, ok := caKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA private key is not Ed25519")
	}

	return ed25519Key, nil
}
