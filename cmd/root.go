package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientcmd "4bit.api/v0/cmd/client_cmd"
	"4bit.api/v0/internal/config"
	cobra "github.com/spf13/cobra"
)

// RootContext is a shared cancellation context for which is shared up
// the call hierarchy.
type RootContext struct {
	Context *context.Context
	Cancel  *context.CancelFunc
}

// Shared among all commands.
var (
	rootCtx RootContext
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "4bit",
		Short: "4bit is a REST api that monitors and reports on local network devices",

		// Create a pre-hook to instantiate a root cancellation deadline.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Create a cancellation context.
			ctx, cancel := context.WithCancel(context.Background())
			rootCtx.Context = &ctx
			rootCtx.Cancel = &cancel

			// Register termination signal to clean up.
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT)

			// Spin up clean up listener.
			go func() {
				for {
					sig := <-sigChan
					if sig == syscall.SIGINT {
						log.Println("SIGINT: Cleaning up...")
						cancel()

						// Block for context completion.
						<-ctx.Done()
						<-time.NewTimer(1 * time.Second).C
						os.Exit(0)
					}
				}
			}()

			return nil
		},

		// Create a post-hook to nominally clean up.
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Shutdown the root context so that upstream threads can clean up.
			log.Println("Shutting down root context")
			(*rootCtx.Cancel)()
			<-(*rootCtx.Context).Done()
			<-time.NewTimer(1 * time.Second).C
			return nil
		},
	}

	// Global args.
	verbose := rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode")

	// Set global configuration.
	config.Verbose = *verbose

	rootCmd.AddCommand(NewServerCommand())
	rootCmd.AddCommand(clientcmd.NewClientCommand())
	return rootCmd.Execute()
}
