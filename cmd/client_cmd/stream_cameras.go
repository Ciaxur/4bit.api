package clientcmd

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"4bit.api/v0/server/route/camera/interfaces"
)

// streamCamera is a helper function which handles opening a stream to a running
// server API.
// It returns an error instance reflecting the failure state.
func streamCamera(wg *sync.WaitGroup) error {
	// Root context to clean up on.
	ctx := *rootContext

	// Construct the stream request.
	streamReq := &interfaces.StreamCameraRequest{
		IP: *cameraIp,
	}

	// Open HTTP stream.
	resp, err := clientContext.NewStream("camera/subscribe", http.MethodGet, streamReq)
	if err != nil {
		return err
	}

	// Instantiate a reader instance for which to consume data from the response body.
	// We expect responses to be JSON and so use a json decoder.
	jsonDecoder := json.NewDecoder(resp.Body)

	// Start stream consumption.
	go func() {
		// Create a channel for which to signal termination
		sigTerm := make(chan struct{})

		for {
			select {
			case <-ctx.Done():
				log.Println("Closing stream...")
				wg.Done()
				resp.Body.Close()
				return

			case <-sigTerm:
				log.Println("Terminating stream...")
				wg.Done()
				resp.Body.Close()
				return

			default:
				// Consume the payload.
				streamResp := &interfaces.StreamCameraResponse{}
				if err := jsonDecoder.Decode(streamResp); err != nil {
					log.Printf("failed to decode response body: %v", err)
					close(sigTerm)
					continue
				}

				log.Printf("Recieved %d cameras:", len(streamResp.Cameras))
				for camIp, cam := range streamResp.Cameras {
					log.Printf("== %s[%s] ==", cam.Name, camIp)
					log.Printf("- Data: %dB", len(cam.Data))
				}
			}
		}
	}()

	return nil
}

// handleStreamCamerasCommand is a helper function for handling camera streams.
// It returns an error instance reflecting the failure state.
func handleStreamCamerasCommand() error {
	wg := &sync.WaitGroup{}
	if err := streamCamera(wg); err != nil {
		return err
	}
	wg.Add(1)

	// Block until stream gets interrupted.
	wg.Wait()

	return nil
}
