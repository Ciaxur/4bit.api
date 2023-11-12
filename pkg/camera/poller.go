package camera

import (
	"context"
	"fmt"
	"log"
	"time"

	"4bit.api/v0/database"
	"4bit.api/v0/internal/config"
)

const (
	// HTTP/1 Streaming endpoint format string.
	STREAM_ENDPOINT_FMT = "http://%s:%d/stream"
)

var (
	CameraPollerInstance *CameraPoller
	Verbose              bool
)

type CameraPoller struct {
	ctx *context.Context

	// List of all camera entries to poll.
	cameras         []database.CameraEntry
	PollWorkers     map[string]*CameraPollWorker
	PollingInterval time.Duration
	BufferSizeBytes uint64

	// Routine status.
	IsRunning           bool
	ShouldUpdateEntries bool
}

// Creates a new instance of camera poller.
func NewCameraPoller(ctx *context.Context) (*CameraPoller, error) {
	// Ensure camera poller singleton.
	if CameraPollerInstance != nil {
		log.Printf("Attempted to create a new CameraPoller while one already exists. Using existing one")
		return CameraPollerInstance, nil
	}

	Verbose = config.Verbose

	// Create the poller instance with default values.
	cameraPoller := CameraPoller{
		ctx:                 ctx,
		PollingInterval:     5 * time.Millisecond,
		BufferSizeBytes:     5 * (1024 * 1024), // 5MB
		IsRunning:           false,
		ShouldUpdateEntries: false,
		PollWorkers:         map[string]*CameraPollWorker{},
	}

	if err := cameraPoller.UpdateStatus(); err != nil {
		return nil, err
	}

	return &cameraPoller, nil
}

// UpdateStatus updates the status of active cameras from the database.
// It returns an error reflecting the state of failure.
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

// updateWorkerStatus is intended to run in a goroutine which constantly
// polls and updates the workers to reflect the current active state.
func (camPoller *CameraPoller) updateWorkerStatus() {
	// Construct a poll rate.
	tick := time.NewTicker(1 * time.Second)
	ctx := *camPoller.ctx
	workerCtxCancelMp := map[string]context.CancelFunc{}

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker status update terminating...")
			camPoller.IsRunning = false
			return
		case <-tick.C:
			log.Println("Updating CamerPoller status")

			// Grab the current state of all cameras.
			if camPoller.ShouldUpdateEntries {
				if err := camPoller.UpdateStatus(); err != nil {
					log.Printf("worker status update failed: %v\n", err)
				}
			}

			// Check the current state of workers, creating/terminating as needed
			// to reflect the current state.
			for _, cameraEntry := range camPoller.cameras {
				worker, ok := camPoller.PollWorkers[cameraEntry.IP]
				if !ok {
					log.Printf(
						"creating new worker to handle camera[ip=%s|name=%s]\n",
						cameraEntry.IP,
						cameraEntry.Name,
					)

					httpStreamEndpoint := fmt.Sprintf(STREAM_ENDPOINT_FMT, cameraEntry.IP, cameraEntry.Port)
					workerCtx, workerCancel := context.WithCancel(context.TODO())
					newWorker := NewCameraPollWorker(&workerCtx, CameraPollWorkerOptions{
						Endpoint: httpStreamEndpoint,
						Name:     cameraEntry.Name,
						RootCtx:  camPoller.ctx,
					})

					// Store the worker's context cancel func, used for tearing down workers.
					camPoller.PollWorkers[cameraEntry.IP] = newWorker
					workerCtxCancelMp[cameraEntry.IP] = workerCancel
				} else if !worker.IsRunning {
					log.Printf(
						"restarting worker[%s] for camera[ip=%s|name=%s]\n",
						worker.endpoint,
						cameraEntry.IP,
						cameraEntry.Name,
					)
					if err := worker.Start(); err != nil {
						log.Printf(
							"failed to start worker[%s] for camera[ip=%s|name=%s]: %v\n",
							worker.endpoint,
							cameraEntry.IP,
							cameraEntry.Name,
							err,
						)
					}
				}
			}

			for mpKey := range camPoller.PollWorkers {
				// Verify worker is not stale.
				found := false
				for _, camera := range camPoller.cameras {
					if mpKey == camera.IP {
						found = true
						break
					}
				}

				if !found {
					// Terminate worker.
					log.Printf("Stale worker[%s], terminating...\n", mpKey)
					workerCtxCancelMp[mpKey]()
					delete(camPoller.PollWorkers, mpKey)
					delete(workerCtxCancelMp, mpKey)
				}
			}
		}
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
	go camPoller.updateWorkerStatus()

	return nil
}
