package cmd

import (
	"fmt"
	"log"
	"strconv"

	"4bit.api/v0/database"
	"4bit.api/v0/server"
	"github.com/go-pg/pg/v10"
	"github.com/spf13/cobra"
)

// Database flags
var (
	postgres_host     *string
	postgres_port     *uint16
	postgres_database *string
	postgres_username *string
	postgres_password *string
)

func handleServerCmd(cmd *cobra.Command, args []string) error {
	// Establish a connection with the postgres database.
	if _, err := database.NewConnection(&pg.Options{
		Addr:     fmt.Sprintf("%s:%d", *postgres_host, *postgres_port),
		Database: *postgres_database,
		User:     *postgres_username,
		Password: *postgres_password,
	}); err != nil {
		return fmt.Errorf(
			"failed to establish a new connection with %s:%d",
			*postgres_host,
			*postgres_port,
		)
	}
	log.Printf("Postgres connection successful")

	// Extract & construct server options.
	port, err := strconv.ParseUint(cmd.PersistentFlags().Lookup("port").Value.String(), 10, 16)
	if err != nil {
		return fmt.Errorf("failed to parse port: %v", err)
	}

	opts := &server.ServerOpts{
		ServerName:        cmd.PersistentFlags().Lookup("name").Value.String(),
		ServerCertificate: cmd.PersistentFlags().Lookup("srvCrt").Value.String(),
		ServerKey:         cmd.PersistentFlags().Lookup("srvKey").Value.String(),
		CACertificate:     cmd.PersistentFlags().Lookup("caCrt").Value.String(),
		CACrl:             cmd.PersistentFlags().Lookup("caCrl").Value.String(),
		HostEndpoint:      cmd.PersistentFlags().Lookup("host").Value.String(),
		PortEndpoint:      uint16(port),
	}
	if err := server.Run(opts); err != nil {
		return fmt.Errorf("failed server command: %v", err)
	}
	return nil
}

func NewServerCommand() *cobra.Command {
	srvCmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the server on given endpoint with options.",
		RunE:  handleServerCmd,
	}

	// Server hosting flags.
	srvCmd.PersistentFlags().String("srvCrt", "", "Path to server's certificate.")
	srvCmd.MarkPersistentFlagRequired("srvCrt")
	srvCmd.PersistentFlags().String("srvKey", "", "Path to server's key.")
	srvCmd.MarkPersistentFlagRequired("srvKey")
	srvCmd.PersistentFlags().String("caCrt", "", "Path to the CA Certificate.")
	srvCmd.MarkPersistentFlagRequired("caCrt")
	srvCmd.PersistentFlags().String("caCrl", "", "Path to the CA Certificate Revocation List (CRL).")
	srvCmd.MarkPersistentFlagRequired("caCrl")
	srvCmd.PersistentFlags().String("name", "localhost", "Server's name'.")
	srvCmd.PersistentFlags().String("host", "localhost", "Server hostname to serve on.")
	srvCmd.PersistentFlags().Uint("port", 3000, "Server port to serve on.")

	// Database flags.
	postgres_host = srvCmd.PersistentFlags().StringP("postgres_host", "", "localhost", "Postgres database hostname.")
	postgres_port = srvCmd.PersistentFlags().Uint16P("postgres_port", "", 5432, "Postgres database port.")
	postgres_database = srvCmd.PersistentFlags().StringP("postgres_database", "", "4bit", "Postgres database to use.")
	postgres_username = srvCmd.PersistentFlags().StringP("postgres_username", "", "admin", "Postgres username.")
	postgres_password = srvCmd.PersistentFlags().StringP("postgres_password", "", "example", "Postgres password.")

	return srvCmd
}
