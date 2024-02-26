//go:build arm64

// BUG: Building gotk library on arm64 indefinitely blocks during compilation.
// Disable for now.
package clientcmd

import (
	"fmt"
	"runtime"
)

// handleStreamCamerasCommand is disabled so return an error.
func handleStreamCamerasCommand() error {
	return fmt.Errorf("Camera stream disabled for: %s", runtime.GOARCH)
}
