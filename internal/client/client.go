// client package provides a client context for invoking 4bit API endpoints.
package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type ClientHttpContext struct {
	HttpClient     *http.Client
	serverEndpoint string
}

type ClientHttpOptions struct {
	// Constructed host:port server endpoint
	ServerEndpoint string
}

type ClientHttpTLSOptions struct {
	ClientHttpOptions
	ClientCertificatePath string
	ClientKeyPath         string
	TrustedCaPath         string
}

// createBaseHttpTransport is a helper function for creating a base HTTP Transport
// template.
func createBaseHttpTransport() *http.Transport {
	return &http.Transport{
		// Customize the maximum number of idle (keep-alive) connections to keep per host
		MaxIdleConns: 10,

		// Customize the Timeout for idle connections
		IdleConnTimeout: 30 * time.Second,

		// Customize the maximum number of idle (keep-alive) connections to keep globally
		MaxIdleConnsPerHost: 5,
	}
}

// createTlsTransport is a private helper function which wraps an HTTP transport
// layer with TLS, give the credentials used.
// This returns an HTTP Transport instance along with an error reflecting the failure
// state.
func createTlsTransport(clientCertPath string, clientKeyPath string, trustedCaCertPath string) (*http.Transport, error) {
	// Load client's key pair.
	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed creating x509 keypair from client cert file %s and client key file %s: %v", clientCertPath, clientKeyPath, err)
	}

	// Load the CA that authorized the server's certs.
	log.Printf("Using trusted CA Certificate: %s\n", trustedCaCertPath)
	caCrtContent, err := os.ReadFile(trustedCaCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert %s: %v", trustedCaCertPath, err)
	}

	// Create a CA certificate pool, in order for the certificate to be
	// validated.
	caCrtPool := x509.NewCertPool()
	caCrtPool.AppendCertsFromPEM(caCrtContent)

	// Create a base HTTP transport instance.
	t := createBaseHttpTransport()

	// Create the TLS for the client.
	t.TLSClientConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCrtPool,
	}

	return t, nil
}

// NewClientContext creates an insecure Client HTTP Context instance.
// This returns an http client instance along with an error reflecting the failure state.
func NewClientContext(opt ClientHttpOptions) (*ClientHttpContext, error) {
	// Create base HTTP transport
	t := createBaseHttpTransport()
	client := http.Client{
		Transport: t,
	}

	return &ClientHttpContext{
		HttpClient:     &client,
		serverEndpoint: opt.ServerEndpoint,
	}, nil
}

// NewClientContextWithTLS creates a Client HTTP Context instance, wrapped in TLS.
// This returns an http client instance along with an error reflecting the failure state.
func NewClientContextWithTLS(opt ClientHttpTLSOptions) (*ClientHttpContext, error) {
	// Wrap TLS around the HTTP Transport.
	httpTransport, err := createTlsTransport(
		opt.ClientCertificatePath,
		opt.ClientKeyPath,
		opt.TrustedCaPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed client context creation: %v", err)
	}

	// Create the client with the tls transport and invoke a request to the
	// server.
	client := &http.Client{
		Transport: httpTransport,
	}

	return &ClientHttpContext{
		HttpClient:     client,
		serverEndpoint: opt.ServerEndpoint,
	}, nil
}

// NewStream opens a new stream with an endpoint.
// It returns the response instance for which the caller is responsbile for consuming and cleaning
// up, along with an error instance reflecting the failure state.
func (ctx *ClientHttpContext) NewStream(apiEndpoint string, httpMethod string, reqInterface interface{}) (*http.Response, error) {
	// Serialze the generic interface.
	reqBodyBytes, err := json.Marshal(reqInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request body: %v", err)
	}

	// Construct variables used for invoking the request.
	endpoint := fmt.Sprintf("https://%s/%s", ctx.serverEndpoint, apiEndpoint)
	reqBodyBuf := bytes.NewBuffer(reqBodyBytes)
	req, err := http.NewRequest(httpMethod, endpoint, reqBodyBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to construct http request context: %v", err)
	}

	// Set initial headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	log.Println("Establishing a stream with endpoint:", endpoint)
	resp, err := ctx.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke GET request with server")
	}

	// Verify response code.
	if resp.StatusCode != http.StatusOK {
		ctx.HttpClient.CloseIdleConnections()
		return resp, fmt.Errorf("http request resulted in a non-OK response code: %d", resp.StatusCode)
	}

	return resp, nil
}

// Invoke is a Client HTTP Context function which invokes the 4bit API endpoint,
// handling HTTP/HTTPS, URI construction, and arguments.
// This returns the response body along with an error instance reflecting the
// failure state.
func (ctx *ClientHttpContext) Invoke(apiEndpoint string, httpMethod string, reqInterface interface{}) ([]byte, error) {
	// Serialze the generic interface.
	reqBodyBytes, err := json.Marshal(reqInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request body: %v", err)
	}

	// Construct variables used for invoking the request.
	endpoint := fmt.Sprintf("https://%s/%s", ctx.serverEndpoint, apiEndpoint)
	reqBodyBuf := bytes.NewBuffer(reqBodyBytes)
	req, err := http.NewRequest(httpMethod, endpoint, reqBodyBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to construct http request context: %v", err)
	}

	log.Println("Invoking a request to endpoint:", endpoint)
	resp, err := ctx.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke GET request with server")
	}
	defer resp.Body.Close()

	resBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return resBody, fmt.Errorf("http request resulted in a non-OK response code: %d", resp.StatusCode)
	}

	return resBody, nil
}
