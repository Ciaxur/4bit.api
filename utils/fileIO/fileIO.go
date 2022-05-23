package fileio

import (
	"os"
)

// Helper function that checks if a file exists.
func FileExists(filename string) bool {
	if fstat, err := os.Stat(filename); err == nil && !fstat.IsDir() {
		return true
	}
	return false
}
