// cmd package provide a version sub-command for... showing the compiled app
// version :). Version is set during compile time via ldflags.
// ie. go build -ldflags "-X 'cmd.version=1.2.3'"
package cmd

import (
	"fmt"

	cobra "github.com/spf13/cobra"
)

var (
	binVersion = "dev"
)

// NewVersionCommand creates a version sub-command which prints the application version.
func NewVersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the applicatin's version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Version: %s\n", binVersion)
			return nil
		},
	}
	return versionCmd
}
