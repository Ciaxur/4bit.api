package camera

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"4bit.api/v0/database"
)

var (
	CameraPollerInstance *CameraPoller
)

type CameraTCPSocket struct {
	TCPConnection net.Conn
	LastReadData  []byte
	LastUpdated   time.Time

	// Internal buffering.
	buffer *bytes.Buffer
	Name   string
}

type CameraPoller struct {
	// List of all camera entries to poll.
	cameras            []database.CameraEntry
	CameraConnectionMp map[string]*CameraTCPSocket
	PollingInterval    time.Duration
	KnownDelimiter     []byte
	BufferSizeBytes    uint64

	// Routine status.
	IsRunning           bool
	ShouldUpdateEntries bool
	LastUpdated         time.Time
	Mutex               sync.Mutex
}

// Creates a new instance of camera poller.
func CreateCameraPoller() (*CameraPoller, error) {
	// Ensure camera poller singleton.
	if CameraPollerInstance != nil {
		log.Printf("Attempted to create a new CameraPoller while one already exists. Using existing one")
		return CameraPollerInstance, nil
	}

	// Create the poller instance with default values.
	cameraPoller := CameraPoller{
		PollingInterval:     5 * time.Millisecond,
		KnownDelimiter:      []byte("\r\nDone\r\n"),
		BufferSizeBytes:     5 * (1024 * 1024), // 5MB
		IsRunning:           false,
		ShouldUpdateEntries: false,
		CameraConnectionMp:  map[string]*CameraTCPSocket{},
		LastUpdated:         time.Now(),
	}

	if err := cameraPoller.UpdateStatus(); err != nil {
		return nil, err
	}

	return &cameraPoller, nil
}

func (camPoller *CameraPoller) UpdateStatus() error {
	log.Println("Updating CamerPoller status")

	// Grab the current state of all cameras.
	db := database.DbInstance
	cameras := []database.CameraEntry{}
	if err := db.Model(&cameras).Select(); err != nil {
		return fmt.Errorf("failed to query all camera entries from database: %v", err)
	}
	camPoller.cameras = cameras

	return nil
}

// CameraPoller go routine.
func startPollerGoRoutine(camPoller *CameraPoller) {
	for camPoller.IsRunning {
		// Check if status is stale.
		if camPoller.ShouldUpdateEntries {
			if err := camPoller.UpdateStatus(); err != nil {
				log.Printf("Camera poller failed to update status: %v\n", err)
			}
			camPoller.ShouldUpdateEntries = false
		}

		// Poll the data from each camera.
		for _, cam := range camPoller.cameras {
			// Reuse open client socket.
			camCon, ok := camPoller.CameraConnectionMp[cam.IP]
			if !ok {
				s, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cam.IP, cam.Port))
				if err != nil {
					log.Printf("Failed to create tcp socket for camera '%s:%d': %v\n", cam.IP, cam.Port, err)
					continue
				}

				// Signal to the camera data consumption readiness.
				s.Write([]byte("I'm ready!"))

				// Keep track of open connection.
				camCon = &CameraTCPSocket{
					TCPConnection: s,
					Name:          cam.Name,
					buffer:        bytes.NewBuffer([]byte{}),
					LastUpdated:   time.Now(),
				}
				camPoller.CameraConnectionMp[cam.IP] = camCon
				log.Printf("Successfuly created a tcp socket for %s:%d\n", cam.IP, cam.Port)
			}

			// Consume camera data.
			dataBuffer := make([]byte, camPoller.BufferSizeBytes)
			bytesRead, err := camCon.TCPConnection.Read(dataBuffer)

			if err != nil || bytesRead == 0 {
				log.Printf("Read %d bytes. Failed to read data from camera '%s:%d': %v\n", bytesRead, cam.IP, cam.Port, err)

				// Clean up.
				camCon.TCPConnection.Close()
				delete(camPoller.CameraConnectionMp, cam.IP)
				continue
			}

			// Buffer data based on a known delimiter.
			bytesWritten, err := camCon.buffer.Write(dataBuffer[:bytesRead])
			if err != nil {
				log.Printf("Warning: Read %dbytes from tcp socket but written %dbytes to buffer", bytesRead, bytesWritten)
				log.Printf("Error: Failed to write data to buffer: %v", err)
				camCon.buffer.Reset()
				continue
			}

			delimiterIndexStart := bytes.Index(camCon.buffer.Bytes(), camPoller.KnownDelimiter)
			if delimiterIndexStart != -1 {
				// Extract the image up to the delimter.
				image := camCon.buffer.Bytes()[:delimiterIndexStart]
				subsequentImage := camCon.buffer.Bytes()[delimiterIndexStart+len(camPoller.KnownDelimiter):]

				// Reset and buffer the next image.
				camCon.buffer.Reset()
				camCon.buffer.Write(subsequentImage)

				camPoller.Mutex.Lock()

				// Store camera data for other clients to consume.
				camCon.LastReadData = image

				// Store last updated for this connection & overall poller.
				camPoller.LastUpdated = time.Now()

				// This syncs up consuming produced images by the device on demand.
				camCon.TCPConnection.Write([]byte("I'm ready!"))

				camPoller.Mutex.Unlock()
			}
		}

		time.Sleep(camPoller.PollingInterval)
	}
}

// Starts a single Camera poller goroutine.
func (camPoller *CameraPoller) StartPolling() error {
	// Verify poller is running only one poller a goroutine.
	if camPoller.IsRunning {
		return fmt.Errorf("poller is already running")
	}

	log.Println("Starting camera poller")
	camPoller.IsRunning = true
	go startPollerGoRoutine(camPoller)

	return nil
}
