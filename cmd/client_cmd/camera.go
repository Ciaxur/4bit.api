// clientcmd package provides an client API to invoke camera server endpoints.
package clientcmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	rootContext *context.Context
)

var (
	imageOutputFilepath *string

	// Actions
	isSnapshot     *bool
	isListCameras  *bool
	isCameraStream *bool

	// Filtering
	cameraIp    *string
	resultLimit *uint64
)

// handleCameraCommand is a sub-command callback for handling invoking /camera endpoint
// on a running server instance.
// This returns an error instance reflecting the state of failure.
func handleCameraCommand(cmd *cobra.Command, args []string) error {
	// Handle request actions.
	if *isListCameras {
		return handleListCameraCommand()
	} else if *isSnapshot {
		return handleCameraSnapshotCommand()
	} else if *isCameraStream {
		return handleStreamCamerasCommand()
	} else {
		return fmt.Errorf("unknown camera action")
	}
}

func NewCameraCommand() *cobra.Command {
	camCmd := &cobra.Command{
		Use:   "camera",
		Short: "Invokes the /camera API",
		RunE:  handleCameraCommand,
	}

	// Action flags.
	imageOutputFilepath = camCmd.PersistentFlags().String("out", "", "(Optional) Filepath to saved snapshot image. Prints base64-encoded image to stdout if empty")
	isSnapshot = camCmd.PersistentFlags().Bool("snapshot", false, "Takes a snapshot from existing cameras")
	isListCameras = camCmd.PersistentFlags().BoolP("list", "l", false, "Lists available cameras")
	cameraIp = camCmd.PersistentFlags().String("ip", "", "(Optional) IP Address of a camera")
	resultLimit = camCmd.PersistentFlags().Uint64("limit", 10, "Pagination limit from HTTP GET requests")
	isCameraStream = camCmd.PersistentFlags().Bool("stream", false, "Toggles streaming from an available Camera")

	return camCmd
}
