// clientcmd package provides an client API to invoke camera server endpoints.
package clientcmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"4bit.api/v0/server/route/camera/interfaces"
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

// listCameras is a helper function which invokes listing available cameras
// returning a structured list of camera responses on success.
// It returns a camera list instance along with an error reflecting the
// failure state.
func listCameras() (*interfaces.ListCameraResponse, error) {
	// Construct and serialize the request body.
	listCamReq := interfaces.ListCamerasRequest{
		Limit: *resultLimit,
	}

	// Ship that request!
	resBody, err := clientContext.Invoke("camera/list", http.MethodGet, listCamReq)
	if err != nil {
		return nil, err
	}

	// Deserialize response into a known response struct.
	listCamResp := &interfaces.ListCameraResponse{}
	if err := json.Unmarshal(resBody, listCamResp); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %v", err)
	}

	return listCamResp, nil
}

// snapCameras is a helper function which invokes a GET request snapshot
// on all Cameras.
// It returns a deserialized response from the server along with an error instance
// reflecting the failure state.
func snapCameras() (*interfaces.SnapCameraResponse, error) {
	resBytes, err := clientContext.Invoke(
		"camera/snap",
		http.MethodGet,
		interfaces.SnapCameraRequest{},
	)
	if err != nil {
		return nil, err
	}

	// Deserialize the response to an expected interface.
	snapCamRes := &interfaces.SnapCameraResponse{}
	if err := json.Unmarshal(resBytes, snapCamRes); err != nil {
		return nil, err
	}
	return snapCamRes, nil
}

// streamCamera is a helper function which handles opening a stream to a running
// server API.
// It returns an error instance reflecting the failure state.
func streamCamera(wg *sync.WaitGroup) error {
	// Root context to clean up on.
	ctx := *rootContext

	// Construct the stream request.
	streamReq := &interfaces.StreamCameraRequest{
		IP: *cameraIp,
	}

	// Open HTTP stream.
	resp, err := clientContext.NewStream("camera/subscribe", http.MethodGet, streamReq)
	if err != nil {
		return err
	}

	// Instantiate a reader instance for which to consume data from the response body.
	// We expect responses to be JSON and so use a json decoder.
	jsonDecoder := json.NewDecoder(resp.Body)

	// Start stream consumption.
	go func() {
		// Create a channel for which to signal termination
		sigTerm := make(chan struct{})

		for {
			select {
			case <-ctx.Done():
				log.Println("Closing stream...")
				wg.Done()
				resp.Body.Close()
				return

			case <-sigTerm:
				log.Println("Terminating stream...")
				wg.Done()
				resp.Body.Close()
				return

			default:
				// Consume the payload.
				streamResp := &interfaces.StreamCameraResponse{}
				if err := jsonDecoder.Decode(streamResp); err != nil {
					log.Printf("failed to decode response body: %v", err)
					close(sigTerm)
					continue
				}

				log.Printf("Recieved %d cameras:", len(streamResp.Cameras))
				for camIp, cam := range streamResp.Cameras {
					log.Printf("== %s[%s] ==", cam.Name, camIp)
					log.Printf("- Data: %dB", len(cam.Data))
				}
			}
		}
	}()

	return nil
}

// handleCameraCommand is a sub-command callback for handling invoking /camera endpoint
// on a running server instance.
// This returns an error instance reflecting the state of failure.
func handleCameraCommand(cmd *cobra.Command, args []string) error {
	// Handle request actions.
	if *isListCameras {
		camList, err := listCameras()
		if err != nil {
			return err
		}

		log.Printf("Found %d cameras:", len(camList.Cameras))
		for _, cam := range camList.Cameras {
			// Filter on camera ip, if one was supplied.
			if *cameraIp != "" && *cameraIp != cam.IP {
				continue
			}

			log.Printf("== %s ==\n", cam.Name)
			log.Printf("- IP: %s\n", cam.IP)
			log.Printf("- Port: %d\n", cam.Port)
			log.Printf("- ModifiedAt: %s\n", cam.ModifiedAt.Local())
			log.Printf("- CreatedAt: %s\n", cam.CreatedAt.Local())
			log.Printf("- Adjustment")
			log.Printf("  - ModifiedAt: %s", cam.Adjustment.Timestamp.Local())
			log.Printf("  - CropFrameHeight: %.2f", cam.Adjustment.CropFrameHeight)
			log.Printf("  - CropFrameWidth: %.2f", cam.Adjustment.CropFrameWidth)
			log.Printf("  - CropFrameX: %d", cam.Adjustment.CropFrameX)
			log.Printf("  - CropFrameY: %d", cam.Adjustment.CropFrameY)
		}
	} else if *isSnapshot {
		snapCams, err := snapCameras()
		if err != nil {
			return err
		}

		// List the results of the snapshot response.
		log.Printf("Received %d cameras", len(snapCams.Cameras))
		for camIp, cam := range snapCams.Cameras {
			// Filter on camera ip, if one was supplied.
			if *cameraIp != "" && *cameraIp != camIp {
				continue
			}

			log.Printf("== %s[%s] ==", cam.Name, camIp)
			log.Printf("- Data: %dB", len(cam.Data))

			// Check whether to print the data to stdout or to a file.
			if *imageOutputFilepath != "" {
				log.Println("Saving to ", *imageOutputFilepath)
				outputFile := *imageOutputFilepath

				// Seperate multiple images into distinct output files.
				// This can be determined by checking if the user requested a specific
				// camera to take a snapshot of.
				if *cameraIp == "" {
					dirPath := filepath.Dir(*imageOutputFilepath)
					basename := filepath.Base(*imageOutputFilepath)
					fileExt := filepath.Ext(basename)
					filename := strings.TrimSuffix(basename, fileExt)
					outputFile = fmt.Sprintf("%s/%s-%s%s", dirPath, filename, cam.Name, fileExt)
					log.Printf("Multiple camera snapshots detected, saving to %s", outputFile)
				}

				if err := os.WriteFile(outputFile, cam.Data, 0644); err != nil {
					return err
				}
			} else {
				log.Println("= base64 encoded data [start] =")
				dataB64 := base64.StdEncoding.EncodeToString(cam.Data)
				log.Println(dataB64)
				log.Println("= base64 encoded data [end] =")
			}
		}
	} else if *isCameraStream {
		wg := &sync.WaitGroup{}
		if err := streamCamera(wg); err != nil {
			return err
		}
		wg.Add(1)

		// Block until stream gets interrupted.
		wg.Wait()
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
	cameraIp = camCmd.PersistentFlags().String("ip", "", "(Optional) IP Address of a camera")
	resultLimit = camCmd.PersistentFlags().Uint64("limit", 10, "Pagination limit from HTTP GET requests")
	isCameraStream = camCmd.PersistentFlags().Bool("stream", false, "Toggles streaming from an available Camera")

	return camCmd
}
