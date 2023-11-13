// clientcmd package wraps /ping endpoint handling in a client sub-command.
package clientcmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// handleClientPingCommand is a cobra callback handler for invoking /ping
// on a running server instance.
// This returns an error instance reflecting the failure state.
func handleClientPingCommand(cmd *cobra.Command, args []string) error {
	// Use the existing HTTP client instance to invoke the ping endpoint.
	resBody, err := clientContext.Invoke("ping", http.MethodGet, []byte(""))
	if err != nil {
		return fmt.Errorf("/ping failed: %v", err)
	}

	log.Printf("/ping response: %s\n", resBody)
	return nil
}

// NewClientPingCommand creates a new ping sub-command.
func NewClientPingCommand() *cobra.Command {
	clientPingCmd := &cobra.Command{
		Use:   "ping",
		Short: "Invokes /ping endpoint on a running server instance",
		RunE:  handleClientPingCommand,
	}

	return clientPingCmd
}
