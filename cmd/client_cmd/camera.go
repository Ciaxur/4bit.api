// clientcmd package provides an client API to invoke camera server endpoints.
package clientcmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"4bit.api/v0/server/route/camera/interfaces"
	"github.com/spf13/cobra"
)

var (
	imageOutputFilepath *string
	isSnapshot          *bool
	isListCameras       *bool
)

// listCameras is a helper function which invokes listing available cameras
// returning a structured list of camera responses on success.
// It returns a camera list instance along with an error reflecting the
// failure state.
func listCameras() (*interfaces.ListCameraResponse, error) {
	// Construct and serialize the request body.
	listCamReq := interfaces.ListCamerasRequest{
		Limit: 10,
	}
	reqBody, err := json.Marshal(listCamReq)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request body: %v", err)
	}

	// Ship that request!
	resBody, err := clientContext.Invoke("camera/list", http.MethodGet, reqBody)
	if err != nil {
		return nil, err
	}

	// Deserialize response into a known response struct.
	listCamResp := interfaces.ListCameraResponse{}
	if err := json.Unmarshal(resBody, &listCamResp); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %v", err)
	}

	return &listCamResp, nil
}

// handleCameraCommand is a sub-command callback for handling invoking /camera endpoint
// on a running server instance.
// This returns an error instance reflecting the state of failure.
func handleCameraCommand(cmd *cobra.Command, args []string) error {
	// TODO: SnapCameraRequest

	// Handle listing available cameras.
	if *isListCameras {
		camList, err := listCameras()
		if err != nil {
			return err
		}

		log.Printf("Found %d cameras:", len(camList.Cameras))
		for _, cam := range camList.Cameras {
			log.Printf("== %s ==\n", cam.Name)
			log.Printf("- IP: %s\n", cam.IP)
			log.Printf("- Port: %d\n", cam.Port)
			log.Printf("- ModifiedAt: %s\n", cam.ModifiedAt.Local())
			log.Printf("- CreatedAt: %s\n", cam.CreatedAt.Local())
		}
	} else {
		return fmt.Errorf("unknown camera action")
	}

	return nil
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

	return camCmd
}
