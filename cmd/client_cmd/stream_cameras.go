package clientcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"sync"

	"4bit.api/v0/server/route/camera/interfaces"
	"github.com/gotk3/gotk3/gtk"
	"github.com/nfnt/resize"
)

// Gtk window constants.
const (
	WINDOW_HEIGHT = 800
	WINDOW_WIDTH  = 800
)

// streamCamera is a helper function which handles opening a stream to a running
// server API into a gtk image instance. The window channel signals that the window has
// been terminated.
// Upon clean up of this thread, the gtk window is invoked to close.
// It returns an error instance reflecting the failure state.
func streamCamera(wg *sync.WaitGroup, win *gtk.Window, winTermSig chan struct{}) error {
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

	// Helper function for clean up.
	cleanup := func() {
		wg.Done()
		resp.Body.Close()
		win.Close()
	}

	// Create a GTK Box for images to live in.
	vbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	if err != nil {
		return fmt.Errorf("failed creating a gtk box: %v", err)
	}
	win.Add(vbox)
	win.ShowAll()

	// Create a mapping between each camera and a corresponding gtk image instance
	// to bind images to.
	gtkImgMap := map[string]*gtk.Image{}

	// Helper function for grabbing a new gtk image instances bound to a camera IP.
	getGtkImage := func(camIp string) (*gtk.Image, error) {
		if imgView, ok := gtkImgMap[camIp]; ok {
			return imgView, nil
		}

		// Create a Gtk image for which to apply streamed images onto.
		imageView, err := gtk.ImageNew()
		if err != nil {
			return nil, fmt.Errorf("failed to create new gtk image for %s: %v", camIp, err)
		}
		vbox.Add(imageView)
		imageView.Show()

		// Add to map for tracking.
		gtkImgMap[camIp] = imageView

		// Adjust spacing based on the number of images.
		vbox.SetSpacing(len(gtkImgMap))

		return imageView, nil
	}

	// Start stream consumption.
	go func() {
		// Create a channel for which to signal termination
		sigTerm := make(chan struct{})

		for {
			select {
			case <-ctx.Done():
				log.Println("Root context closed, terminating stream...")
				cleanup()
				return

			case <-sigTerm:
				log.Println("Stream broke, terminating stream...")
				cleanup()
				return

			case <-winTermSig:
				log.Println("GTK Window closed, terminating stream...")
				cleanup()
				return

			default:
				// Consume the payload.
				streamResp := &interfaces.StreamCameraResponse{}
				if err := jsonDecoder.Decode(streamResp); err != nil {
					log.Printf("failed to decode response body: %v", err)
					close(sigTerm)
					continue
				}

				numCams := len(streamResp.Cameras)
				log.Printf("Received %d cameras:", numCams)
				for camIp, cam := range streamResp.Cameras {
					log.Printf("== %s[%s] ==", cam.Name, camIp)
					log.Printf("- Data: %dB", len(cam.Data))

					// Decode jpeg image.
					imageBuffReader := bytes.NewReader(cam.Data)
					img, _, err := image.Decode(imageBuffReader)
					if err != nil {
						log.Printf("failed decoding image: %v\n", err)
						continue
					}

					// Resize image to fit the window.
					resize.Resize(
						uint(WINDOW_WIDTH),
						uint(WINDOW_HEIGHT),
						img,
						resize.Lanczos3,
					)

					// Apply image to the gtk image buffer.
					jpegBuf := new(bytes.Buffer)
					if err := jpeg.Encode(jpegBuf, img, nil); err != nil {
						log.Printf("failed jpeg encoding: %v\n", err)
						continue
					}

					// Create a gtk image from buffer.
					i, err := gtkImageFromBuffer(jpegBuf)
					if err != nil {
						log.Printf("failed to create gtk image from buffer: %v", err)
						continue
					}

					// Grab the a gtk image to place jpeg in.
					gtkImage, err := getGtkImage(camIp)
					if err != nil {
						log.Printf("Failed to get a GTK Image instance: %v", err)
						continue
					}
					gtkImage.SetFromPixbuf(i.GetPixbuf())
				}

			}
		}
	}()

	return nil
}

// gtkImageFromBuffer is a helper function for concerting a given buffer
// into a GTK Image.
// It returns a gtk image pointer and an error reflecting the failure state.
func gtkImageFromBuffer(buf *bytes.Buffer) (*gtk.Image, error) {
	// Save to tempfile for which to consume.
	fd, err := os.CreateTemp("/tmp", "tmpjpgstream-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer fd.Close()
	defer os.Remove(fd.Name())

	// Write the buffer to that file.
	if _, err := fd.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed write to temp file: %v", err)
	}

	// Create a GTK Image from that tempfile.
	gtkImage, err := gtk.ImageNewFromFile(fd.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to create gtk image from file: %v", err)
	}

	return gtkImage, nil
}

// handleStreamCamerasCommand is a helper function for handling camera streams.
// It returns an error instance reflecting the failure state.
func handleStreamCamerasCommand() error {
	// Create a GTK window to stream images to.
	gtk.Init(nil)
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	winTermSig := make(chan struct{})
	win.SetTitle("4bit: Stream Viewer")
	win.Connect("destroy", func() {
		gtk.MainQuit()
		close(winTermSig)
	})
	win.SetDefaultSize(WINDOW_WIDTH, WINDOW_HEIGHT)

	// Start the stream.
	wg := &sync.WaitGroup{}
	if err := streamCamera(wg, win, winTermSig); err != nil {
		return err
	}
	wg.Add(1)

	// Start the GTK main loop.
	gtk.Main()

	// Block until stream gets interrupted.
	wg.Wait()

	return nil
}
