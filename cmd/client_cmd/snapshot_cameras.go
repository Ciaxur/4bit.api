package clientcmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"4bit.api/v0/server/route/camera/interfaces"
)

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

// handleCameraSnapshotCommand is a helper function for handling grabbing a snapshot
// from available cameras.
// It returns an error instance reflecting the failure state.
func handleCameraSnapshotCommand() error {
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

	return nil
}
