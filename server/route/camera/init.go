package camera

import (
	"fmt"

	"github.com/gorilla/mux"
)

func CreateRoutes(r *mux.Router) error {
	CreateCameraRoutes(r)
	CreateCameraListRoute(r)

	// Create & start poller, since the poller is a dependency of those routes.
	camPoller, err := CreateCameraPoller()
	CameraPollerInstance = camPoller
	if err != nil {
		return fmt.Errorf("failed to create camera poller: %v", err)
	}

	if err := CameraPollerInstance.StartPolling(); err != nil {
		return fmt.Errorf("failed to start camera poller: %v", err)
	}

	return nil
}
