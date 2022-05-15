package cmd

import (
	cobra "github.com/spf13/cobra"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:           "4bit",
		Short:         "4bit is a REST api that monitors and reports on local network devices",
		SilenceErrors: true,
	}
	rootCmd.AddCommand(NewServerCommand())
	return rootCmd.Execute()
}
