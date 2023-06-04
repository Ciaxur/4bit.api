package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// Helper function that parses a certificate from a raw PEM bytes array.
func ParseCertificateFromPEMBytes(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemBytes))
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate from PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	return cert, nil
}
