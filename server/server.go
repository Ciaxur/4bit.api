package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"4bit.api/v0/internal/server_crl"
	"4bit.api/v0/internal/utils"
	"4bit.api/v0/server/middleware"
	"4bit.api/v0/server/route"
	fileio "4bit.api/v0/utils/fileIO"
	"github.com/gorilla/mux"
)

type ServerOpts struct {
	ServerName        string
	ServerCertificate string
	ServerKey         string
	CACertificate     string
	CACrl             string
	HostEndpoint      string
	PortEndpoint      uint16
}

func Run(opts *ServerOpts) error {
	// TODO: Expand the CA pool to read from a given CA directory and append
	// mutliple trusted CA's to the pool
	// Create the CA pool which will be used for verifying the client with.
	caCrtContent, err := ioutil.ReadFile(opts.CACertificate)
	if err != nil {
		return fmt.Errorf("failed to read the content of CA %s", opts.CACertificate)
	}

	// Check if the server's certificate & key exists.
	if !fileio.FileExists(opts.ServerCertificate) {
		return fmt.Errorf("server certificate '%s' does not exist", opts.ServerCertificate)
	}
	if !fileio.FileExists(opts.ServerKey) {
		return fmt.Errorf("server key '%s' does not exist", opts.ServerKey)
	}

	// Parse the certificate(s) from PEM bytes.
	caCrt, err := utils.ParseCertificateFromPEMByptes(caCrtContent)
	if err != nil {
		return fmt.Errorf("failed to parse certifacte from PEM bytes: %v", err)
	}

	// Construct a list of parsed trusted CAs.
	trustedCerts := []x509.Certificate{}
	trustedCerts = append(trustedCerts, *caCrt)

	caCertPool := x509.NewCertPool()
	for _, trustedCert := range trustedCerts {
		caCertPool.AddCert(&trustedCert)
	}

	// Create a cache to check revoked certs to save on compute.
	revokedCerts := make(map[string]bool)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", opts.HostEndpoint, opts.PortEndpoint),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{
			ServerName: opts.ServerName,
			ClientCAs:  caCertPool,
			// Require and verify the client's cert against it being signed by the CA.
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,

			// Verify that the client certificate is valid. This function will check
			// whether the cert. has been revoked by the CRL.
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
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
			},
		},
	}

	// Instantiate the CA's Certificate Revocation List (CRL).
	log.Printf("Loading in CA's Certificate Revocation List.")
	if err := server_crl.Init(opts.CACrl); err != nil {
		return err
	}

	router := mux.NewRouter()

	// Add middleware.
	router.Use(middleware.BasicLogger)

	// Add server root endpoints.
	route.InitRootRoute(router)
	http.Handle("/", router)

	log.Printf("Listening on %s:%d.\n", opts.HostEndpoint, opts.PortEndpoint)
	if err := server.ListenAndServeTLS(opts.ServerCertificate, opts.ServerKey); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
