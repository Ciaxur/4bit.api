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

// GetRootContext simply returns the constructed root context instance.
func GetRootContext() RootContext {
	return rootCtx
}

// initRootContext instantiates a root context for which to be
// used in sub-commands.
// This returns an error instance reflecting the failure state.
func initRootContext() error {
	// Create a cancellation context.
	log.Println("Instantiating root context")
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
}

// Execute initializes all of the commands, then runs the main cobra command
// execution function.
// The application version is passed into the execution from the main package.
// This returns an error instance reflecting the failure state of any sub-command.
func Execute(version string) error {
	// Set the version.
	binVersion = version

	rootCmd := &cobra.Command{
		Use:   "4bit",
		Short: "4bit is a REST api that monitors and reports on local network devices",

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

	// Instantiate a root cancellation deadline.
	if err := initRootContext(); err != nil {
		return err
	}

	// Global args.
	verbose := rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode")

	// Set global configuration.
	config.Verbose = *verbose

	rootCmd.AddCommand(NewServerCommand())
	rootCmd.AddCommand(clientcmd.NewClientCommand(rootCtx.Context))
	rootCmd.AddCommand(NewVersionCommand())
	return rootCmd.Execute()
}
