// cmd package provides a client sub-command for interacting directly with
// the API. This allows for an interactive synergy between server & client.
package clientcmd

import (
	"fmt"
	"log"

	"4bit.api/v0/internal/client"
	"github.com/spf13/cobra"
)

// Shared client variables.
var (
	clientContext *client.ClientHttpContext
)

// Client flags
var (
	// Running Server API instance
	serverHost *string
	serverPort *uint

	// Optional TLS configurations
	clientCertificatePath *string
	clientKeyPath         *string
	clientTrustedCaPath   *string
)

// setupClient configures a client instance with TLS for which to be used
// within client sub-commands.
// This returns an error instance reflecting the state of failure for
// configuring a client instance.
func setupClient(cmd *cobra.Command, args []string) error {
	var err error = nil
	svrEndpoint := fmt.Sprintf("%s:%d", *serverHost, *serverPort)

	// Check client construction with TLS.
	if *clientCertificatePath != "" && *clientKeyPath != "" && *clientTrustedCaPath != "" {
		log.Println("Constructing client instance with TLS")
		clientContext, err = client.NewClientContextWithTLS(client.ClientHttpTLSOptions{
			ClientHttpOptions: client.ClientHttpOptions{
				ServerEndpoint: svrEndpoint,
			},

			ClientCertificatePath: *clientCertificatePath,
			ClientKeyPath:         *clientKeyPath,
			TrustedCaPath:         *clientTrustedCaPath,
		})

	} else {
		log.Println("Constructing insecure client instance")
		clientContext, err = client.NewClientContext(
			client.ClientHttpOptions{
				ServerEndpoint: svrEndpoint,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to create client context: %v", err)
	}

	return nil
}

// NewClientCommand creates a client sub-command, returning a pointer to
// the command instance.
func NewClientCommand() *cobra.Command {
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "Client API interface with a running 4bit server instance",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup a client instance to be shared with the client sub-commands.
			if err := setupClient(cmd, args); err != nil {
				return err
			}

			return nil
		},
	}

	// Server API flags.
	serverHost = clientCmd.PersistentFlags().String("server", "localhost", "Host endpoint for a running 4bit server")
	serverPort = clientCmd.PersistentFlags().Uint("port", 3000, "Listening port on a running 4bit server")

	// Optional TLS flags.
	clientCertificatePath = clientCmd.PersistentFlags().String("certificate", "", "(Optional) Client TLS Certificate file path")
	clientKeyPath = clientCmd.PersistentFlags().String("key", "", "(Optional) Client TLS Key file path")
	clientTrustedCaPath = clientCmd.PersistentFlags().String("trustedCa", "", "(Optional) Client trusted CA bundle file path")

	// Add client sub-commands.
	clientCmd.AddCommand(NewClientPingCommand())

	return clientCmd
}
