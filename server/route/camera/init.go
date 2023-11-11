package camera

import (
	"context"
	"fmt"

	"4bit.api/v0/pkg/camera"
	"github.com/gorilla/mux"
)

func CreateRoutes(ctx *context.Context, r *mux.Router) error {
	CreateCameraRoutes(r)
	CreateCameraListRoute(r)

	// Create & start poller, since the poller is a dependency of those routes.
	camPoller, err := camera.NewCameraPoller(ctx)
	camera.CameraPollerInstance = camPoller
	if err != nil {
		return fmt.Errorf("failed to create camera poller: %v", err)
	}

	if err := camera.CameraPollerInstance.StartPolling(); err != nil {
		return fmt.Errorf("failed to start camera poller: %v", err)
	}

	return nil
}
