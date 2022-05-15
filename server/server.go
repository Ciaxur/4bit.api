package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"4bit.api/v0/server/route"
)

type ServerOpts struct {
	ServerCertificate string
	ServerKey         string
	CACertificate     string
	HostEndpoint      string
	PortEndpoint      uint16
}

func Run(opts *ServerOpts) error {
	// Create the CA pool which will be used for verifying the client with.
	caCrtContent, err := ioutil.ReadFile(opts.CACertificate)
	if err != nil {
		return fmt.Errorf("failed to read the content of CA %s", opts.CACertificate)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCrtContent)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", opts.HostEndpoint, opts.PortEndpoint),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{
			ServerName: "localhost",
			ClientCAs:  caCertPool,
			// Require and verify the client's cert against it being signed by the CA.
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,
		},
	}

	// Add server root endpoints.
	http.Handle("/", route.InitRootRoute())

	log.Printf("Listening on %s:%d.\n", opts.HostEndpoint, opts.PortEndpoint)
	if err := server.ListenAndServeTLS(opts.ServerCertificate, opts.ServerKey); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
