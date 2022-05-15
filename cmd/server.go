package cmd

import (
	"fmt"
	"strconv"

	"4bit.api/v0/server"
	"github.com/spf13/cobra"
)

func handleServerCmd(cmd *cobra.Command, args []string) error {
	// Extract & construct server options.
	port, err := strconv.ParseUint(cmd.PersistentFlags().Lookup("port").Value.String(), 10, 16)
	if err != nil {
		return fmt.Errorf("failed to parse port: %v", err)
	}

	opts := &server.ServerOpts{
		ServerCertificate: cmd.PersistentFlags().Lookup("srvCrt").Value.String(),
		ServerKey:         cmd.PersistentFlags().Lookup("srvKey").Value.String(),
		CACertificate:     cmd.PersistentFlags().Lookup("caCrt").Value.String(),
		HostEndpoint:      cmd.PersistentFlags().Lookup("host").Value.String(),
		PortEndpoint:      uint16(port),
	}
	server.Run(opts)
	return nil
}

func NewServerCommand() *cobra.Command {
	srvCmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the server on given endpoint with options.",
		RunE:  handleServerCmd,
	}

	srvCmd.PersistentFlags().String("srvCrt", "", "Path to server's certificate.")
	srvCmd.MarkPersistentFlagRequired("srvCrt")
	srvCmd.PersistentFlags().String("srvKey", "", "Path to server's key.")
	srvCmd.MarkPersistentFlagRequired("srvKey")
	srvCmd.PersistentFlags().String("caCrt", "", "Path to the CA Certificate.")
	srvCmd.MarkPersistentFlagRequired("caCrt")
	srvCmd.PersistentFlags().String("host", "localhost", "Server hostname to serve on.")
	srvCmd.PersistentFlags().Uint("port", 3000, "Server port to serve on.")

	return srvCmd
}
