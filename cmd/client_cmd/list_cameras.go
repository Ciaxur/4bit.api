package clientcmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"4bit.api/v0/server/route/camera/interfaces"
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

// handleListCameraCommand is a helper function for handling listing cameras.
// It returns an error instance reflecting the failure state.
func handleListCameraCommand() error {
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

	return nil
}
