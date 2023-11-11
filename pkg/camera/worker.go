// The camera package contains workers for which concurrently handle
// a given endpoint to poll from.
package camera

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"sync"
	"time"
)

type CameraPollSnapshot struct {
	ImageData   []byte
	LastUpdated time.Time
}

type CameraPollWorker struct {
	ctx          *context.Context
	rootCtx      *context.Context
	endpoint     string
	lastReadData []byte
	lastUpdated  time.Time
	mutex        *sync.Mutex

	IsRunning bool
	Name      string
}

type CameraPollWorkerOptions struct {
	Endpoint string
	Name     string
	RootCtx  *context.Context
}

// NewCameraPollWorker creates a new CameraPollWorker instance given the options
// and context.
func NewCameraPollWorker(ctx *context.Context, opts CameraPollWorkerOptions) *CameraPollWorker {
	return &CameraPollWorker{
		ctx:          ctx,
		rootCtx:      opts.RootCtx,
		endpoint:     opts.Endpoint,
		lastReadData: []byte{},
		lastUpdated:  time.Now(),
		IsRunning:    false,
		Name:         opts.Name,
		mutex:        &sync.Mutex{},
	}
}

// GetSnapshot returns a copy of the last image taken.
func (worker *CameraPollWorker) GetSnapshot() *CameraPollSnapshot {
	// Grab a lock to deconflict with a race condition.
	worker.mutex.Lock()
	defer worker.mutex.Unlock()

	// Copy the internal data and return it.
	data := make([]byte, len(worker.lastReadData))
	copy(data, worker.lastReadData)

	return &CameraPollSnapshot{
		ImageData:   data,
		LastUpdated: worker.lastUpdated,
	}
}

// cleanup unregisters the worker.
func (worker *CameraPollWorker) cleanup() {
	worker.IsRunning = false
}

// poll is intended to run in a goroutine which starts polling data from
// the constructed endpoint.
func (worker *CameraPollWorker) poll() {
	client := http.Client{
		// Set a timeout for 1min for which to reconnect, assuming stale.
		Timeout: 1 * time.Minute,
	}
	ctx := *worker.ctx
	rootCtx := *worker.rootCtx

	// Establish a downstream connection.
	resp, err := client.Get(worker.endpoint)
	if err != nil {
		log.Printf("failed to establish connection: %v\n", err)
		worker.cleanup()
		return
	}
	defer resp.Body.Close()

	// Deadline timer.
	deadlineDuration := 1 * time.Second
	deadlineCtx, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(deadlineDuration, func() {
		cancel()
	})

	// Start the loop!
pollLoop:
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker context closed, terminating worker[%s]\n", worker.endpoint)
			break pollLoop

		case <-rootCtx.Done():
			log.Printf("root context closed, terminating worker[%s]\n", worker.endpoint)
			break pollLoop

		case <-deadlineCtx.Done():
			log.Printf("deadline exceeded, terminating worker[%s]\n", worker.endpoint)
			break pollLoop

		default:
			// Consume and decode image.
			img, imgFmt, err := image.Decode(resp.Body)
			if err != nil {
				continue
			}
			if Verbose {
				log.Printf("worker[%s] decoded image format: %s\n", worker.endpoint, imgFmt)
			}

			// Encode image into jpeg
			buf := new(bytes.Buffer)
			if err := jpeg.Encode(buf, img, nil); err != nil {
				log.Printf("worker[%s] failed jpeg encoding: %v\n", worker.endpoint, err)
				continue
			}
			if Verbose {
				log.Printf("worker[%s] encoded jpeg image into %dB buffer\n", worker.endpoint, buf.Len())
			}

			// Store the decoded image.
			worker.mutex.Lock()
			worker.lastReadData = buf.Bytes()
			worker.lastUpdated = time.Now()
			worker.mutex.Unlock()

			// Deadline met, reset.
			timer.Reset(deadlineDuration)
		}
	}

	// Unregister worker.
	worker.cleanup()
}

// Start spins up worker.
// It returns an error reflecting the failure state.
func (worker *CameraPollWorker) Start() error {
	if worker.IsRunning {
		return fmt.Errorf("worker[%s] already running", worker.endpoint)
	}

	// Start the goroutine.
	worker.IsRunning = true
	go worker.poll()

	return nil
}
