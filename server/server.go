package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"4bit.api/v0/internal/server_crl"
	"4bit.api/v0/internal/utils"
	"4bit.api/v0/server/middleware"
	"4bit.api/v0/server/route"
	fileio "4bit.api/v0/utils/fileIO"
	"github.com/gorilla/mux"
)

type ServerOpts struct {
	ServerName          string
	ServerCertificate   string
	ServerKey           string
	TrustedCASDirectory string
	CACrl               string
	HostEndpoint        string
	PortEndpoint        uint16
}

func createPeerCertificateVerification(trustedCerts []x509.Certificate) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// Create a cache to check revoked certs to save on compute.
	revokedCerts := make(map[string]bool)

	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		// Obtain the parsed cached CRL instance.
		crl := server_crl.CachedCaCrl

		// Verify that the CRL content has not been tampered with, by checking
		// its signature against the CA from the certificate pool.
		// Iterating through each certificate to validate the CRL againts the
		// appropriate cert.
		var checked bool = false
		var crlCheckErr error = nil
		var cert *x509.Certificate
		for _, trustedCert := range trustedCerts {
			if err := trustedCert.CheckCRLSignature(crl); err != nil {
				crlCheckErr = err
				continue
			}

			// Successfuly verified and matched the CRL with a certificate.
			checked = true
			cert = &trustedCert
			break
		}
		if !checked {
			return fmt.Errorf("failed to match a certificate from the cert pool with the CRL: %v", crlCheckErr)
		}
		log.Printf("CRL verified peer[%s] with certificate issuer: %s\n", cert.Subject, cert.Issuer)

		// Check if the peer's certificate is among the revoked ones registered
		// within the CRL.
		for _, rawPeerCert := range rawCerts {
			peerCrt, err := x509.ParseCertificate(rawPeerCert)
			if err != nil {
				return fmt.Errorf("failed to parse peer's certificate: %v", err)
			}

			// Check the cached revoked certs.
			if _, found := revokedCerts[peerCrt.SerialNumber.String()]; found {
				return fmt.Errorf("peer certificate[%s] was revoked by %s", peerCrt.Subject, peerCrt.Issuer)
			}

			for _, revokedCert := range crl.TBSCertList.RevokedCertificates {
				if revokedCert.SerialNumber.Cmp(peerCrt.SerialNumber) == 0 {
					// Cache that sucker.
					revokedCerts[peerCrt.SerialNumber.String()] = true
					return fmt.Errorf("peer certificate[%s] was revoked by %s", peerCrt.Subject, peerCrt.Issuer)
				}
			}
		}

		return nil
	}
}

func Run(ctx *context.Context, opts *ServerOpts) error {
	// Create the CA pool, by iterating over a given directory, which will be used for verifying the client with.
	trustedCasContent := [][]byte{}
	trustedCaFiles, err := ioutil.ReadDir(opts.TrustedCASDirectory)
	if err != nil {
		return fmt.Errorf("failed to read trusted ca directory '%s': %v", opts.TrustedCASDirectory, err)
	}
	for _, caFile := range trustedCaFiles {
		filepath := filepath.Join(opts.TrustedCASDirectory, caFile.Name())
		caCrtContent, err := ioutil.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("failed to read the content of CA %s", filepath)
		}
		trustedCasContent = append(trustedCasContent, caCrtContent)
	}

	// Check if the server's certificate & key exists.
	if !fileio.FileExists(opts.ServerCertificate) {
		return fmt.Errorf("server certificate '%s' does not exist", opts.ServerCertificate)
	}
	if !fileio.FileExists(opts.ServerKey) {
		return fmt.Errorf("server key '%s' does not exist", opts.ServerKey)
	}

	// Construct a list of parsed trusted CAs.
	trustedCerts := []x509.Certificate{}

	// Parse the certificate(s) from PEM bytes.
	for _, caCrtContent := range trustedCasContent {
		caCrt, err := utils.ParseCertificateFromPEMBytes(caCrtContent)
		if err != nil {
			return fmt.Errorf("failed to parse certifacte from PEM bytes: %v", err)
		}
		trustedCerts = append(trustedCerts, *caCrt)
	}

	caCertPool := x509.NewCertPool()
	for _, trustedCert := range trustedCerts {
		caCertPool.AddCert(&trustedCert)
	}

	// Check for optional CRL check.
	verifyPeerCertificateFunc := createPeerCertificateVerification(trustedCerts)
	if opts.CACrl == "" {
		verifyPeerCertificateFunc = nil
	}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", opts.HostEndpoint, opts.PortEndpoint),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		TLSConfig: &tls.Config{
			ServerName: opts.ServerName,
			ClientCAs:  caCertPool,
			// Require and verify the client's cert against it being signed by the CA.
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,

			// Verify that the client certificate is valid. This function will check
			// whether the cert. has been revoked by the CRL.
			VerifyPeerCertificate: verifyPeerCertificateFunc,
		},
	}

	// Instantiate the CA's Certificate Revocation List (CRL).
	if opts.CACrl != "" {
		log.Printf("Loading in CA's Certificate Revocation List.")
		if err := server_crl.Init(opts.CACrl); err != nil {
			return err
		}
	}

	router := mux.NewRouter()

	// Add middleware.
	router.Use(middleware.BasicLogger)

	// Add server root endpoints.
	if err := route.InitRootRoute(ctx, router); err != nil {
		return fmt.Errorf("failed to create root server routes: %v", err)
	}
	http.Handle("/", router)

	log.Printf("Listening on %s:%d.\n", opts.HostEndpoint, opts.PortEndpoint)
	if err := server.ListenAndServeTLS(opts.ServerCertificate, opts.ServerKey); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
