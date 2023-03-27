package camera

import (
	"bytes"
	"fmt"
	"io"
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
		PollingInterval:     100 * time.Millisecond,
		KnownDelimiter:      []byte("\r\nDone\r\n"),
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

				// Keep track of open connection.
				camCon = &CameraTCPSocket{
					TCPConnection: s,
					Name:          cam.Name,
					buffer:        bytes.NewBuffer(make([]byte, 8192)),
					LastUpdated:   time.Now(),
				}
				camPoller.CameraConnectionMp[cam.IP] = camCon

				// Signal to the camera data consumption readiness.
				s.Write([]byte("I'm ready!"))

				log.Printf("Successfuly created a tcp socket for %s:%d\n", cam.IP, cam.Port)
			}

			// Consume camera data.
			// dataBuffer, err := bufio.NewReader(camCon.TCPConnection)
			dataBuffer := make([]byte, 4096)
			n, err := io.ReadFull(camCon.TCPConnection, dataBuffer)

			if err != nil || n == 0 {
				log.Printf("Read %d bytes. Failed to read data from camera '%s:%d': %v\n", n, cam.IP, cam.Port, err)

				// Clean up.
				camCon.TCPConnection.Close()
				delete(camPoller.CameraConnectionMp, cam.IP)
				continue
			}

			// Buffer data based on a known delimiter.
			camCon.buffer.Write(dataBuffer)
			if bytes.Contains(camCon.buffer.Bytes(), camPoller.KnownDelimiter) {
				// Obtain the data.
				buffer := bytes.Split(camCon.buffer.Bytes(), camPoller.KnownDelimiter)
				camCon.buffer.Reset()

				// Fill in the buffer with the remaining data.
				for _, b := range buffer[1:] {
					camCon.buffer.Write(b)
				}

				// Store camera data for other clients to consume.
				// Strip off the '\rDone\n\r' from the string.
				camCon.LastReadData = buffer[0]

				// Store last updated for this connection & overall poller.
				camPoller.LastUpdated = time.Now()
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
