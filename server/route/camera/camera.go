package camera

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"time"

	"4bit.api/v0/database"
	"4bit.api/v0/pkg/camera"
	"4bit.api/v0/server/route/camera/interfaces"
	"github.com/gorilla/mux"
)

// Adds a new unique Camera entry to track & poll.
// Request expected to be of type AddCameraRequest.
// On success, responds with new database entry.
func postAddCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Failed to read request body:%v\n", err)

		http.Error(
			w,
			"failed to read request body",
			http.StatusBadRequest,
		)
		return
	}

	req := interfaces.AddCameraRequest{}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("Failed to deserialize add camera request :%v\n", err)

		http.Error(
			w,
			"failed to deserialize body",
			http.StatusBadRequest,
		)
		return
	}

	// Validate required IP entry is valid.
	if ip := net.ParseIP(req.Camera.IP); ip == nil {
		log.Printf("Failed to create new camera entry. Invalid IP entry '%s'\n", req.Camera.IP)

		http.Error(
			w,
			"invalid ip entry",
			http.StatusBadRequest,
		)
		return
	}

	if req.Camera.Port == 0 {
		log.Printf("Failed to create new camera entry. Invalid Port entry '%d'\n", req.Camera.Port)

		http.Error(
			w,
			"invalid port entry",
			http.StatusBadRequest,
		)
		return
	}

	if req.Camera.Name == "" {
		log.Printf("Failed to create new camera entry. Invalid empty name entry '%s'\n", req.Camera.Name)

		http.Error(
			w,
			"invalid empty name entry",
			http.StatusBadRequest,
		)
		return
	}

	// Find whether this entry already exists.
	db := database.DbInstance
	if err := db.Model(&req.Camera).Where("camera_entry.ip = ?", req.Camera.IP).Select(); err == nil {
		log.Printf("Failed to add camera entry with IP '%s', because it already exists\n", req.Camera.IP)

		http.Error(
			w,
			fmt.Sprintf("camera entry with IP '%s' already exists", req.Camera.IP),
			http.StatusConflict,
		)
		return
	}

	// Create default camera adjustment.
	log.Printf("Adding default camera adjustment for '%s:%d\n'", req.Camera.IP, req.Camera.Port)
	camAdjust := database.CameraAdjsustment{
		BaseEntry: database.BaseEntry{
			Timestamp: time.Now(),
		},
		CropFrameHeight: 0.0,
		CropFrameWidth:  0.0,
		CropFrameX:      0,
		CropFrameY:      0,
	}
	if _, err := db.Model(&camAdjust).Insert(); err != nil {
		log.Printf("Failed to add new camera adjustment with ip '%s': %v\n", req.Camera.IP, err)

		http.Error(
			w,
			fmt.Sprintf("Failed to add new camera adjustment entry with ip '%s'\n", req.Camera.IP),
			http.StatusInternalServerError,
		)
		return
	}

	// Add new entry to database.
	log.Printf("Adding new camera entry with ip '%s:%d'\n", req.Camera.IP, req.Camera.Port)
	camEntry := req.Camera
	camEntry.CreatedAt = time.Now()
	camEntry.ModifiedAt = camEntry.CreatedAt
	camEntry.Adjustment = &camAdjust
	camEntry.AdjustmentId = camAdjust.Id
	if _, err := db.Model(&camEntry).Relation("Adjustment").Insert(); err != nil {
		log.Printf("Failed to add new camera entry with ip '%s': %v\n", camEntry.IP, err)

		http.Error(
			w,
			fmt.Sprintf("Failed to add new camera entry with ip '%s'\n", camEntry.IP),
			http.StatusInternalServerError,
		)
		return
	}

	// Serialize response.
	resBody, err := json.Marshal(camEntry)
	if err != nil {
		log.Printf("Failed to serialize new camera entry response: %v\n", err)

		http.Error(
			w,
			"failed to serialize response",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(resBody)

	// Update poller state.
	if camera.CameraPollerInstance != nil {
		camera.CameraPollerInstance.ShouldUpdateEntries = true
	}
}

// Removes a Camera entry from being tracked & polled.
// Request expected to be of type RemoveCameraRequest.
// On success, responds with emtpy message.
func postRemoveCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Failed to read request body:%v\n", err)

		http.Error(
			w,
			"failed to read request body",
			http.StatusBadRequest,
		)
		return
	}

	req := interfaces.RemoveCameraRequest{}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("Failed to deserialize remove camera request :%v\n", err)

		http.Error(
			w,
			"failed to deserialize body",
			http.StatusBadRequest,
		)
		return
	}

	// Validate required IP entry is valid.
	if ip := net.ParseIP(req.Camera.IP); ip == nil {
		log.Printf("Failed to remove camera entry. Invalid IP entry '%s'\n", req.Camera.IP)

		http.Error(
			w,
			"invalid ip entry",
			http.StatusBadRequest,
		)
		return
	}

	db := database.DbInstance

	// Grab the camera entry
	if err := db.Model(&req.Camera).Where("camera_entry.ip = ?", req.Camera.IP).Relation("Adjustment").Select(); err != nil {
		log.Printf("Failed to find camera entry with ip '%s': %v\n", req.Camera.IP, err)

		http.Error(
			w,
			fmt.Sprintf("Failed to find camera entry with ip '%s'", req.Camera.IP),
			http.StatusNotFound,
		)
		return
	}

	// Remove camera entry
	if _, err := db.Model(&req.Camera).Where("camera_entry.ip = ?", req.Camera.IP).Delete(); err != nil {
		log.Printf("Failed to remove camera entry with ip '%s': %v\n", req.Camera.IP, err)

		http.Error(
			w,
			fmt.Sprintf("Failed to remove camera entry with ip '%s'", req.Camera.IP),
			http.StatusBadRequest,
		)
		return
	}

	// Remove adjustment entry relation
	log.Printf("Removing associated camera adjustment id='%d'\n", req.Camera.AdjustmentId)
	if _, err := db.Model(req.Camera.Adjustment).WherePK().Delete(); err != nil {
		log.Printf("Failed to remove camera adjustment id='%d' for camera entry with ip '%s': %v\n", req.Camera.AdjustmentId, req.Camera.IP, err)
	}

	log.Printf("Successfuly removed camera entry with ip '%s'\n", req.Camera.IP)
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte("{}"))

	// Update poller state.
	if camera.CameraPollerInstance != nil {
		camera.CameraPollerInstance.ShouldUpdateEntries = true
	}
}

// Gets the current state of all listening cameras or the state of a given camera
// IP address.
// Expects a request of type SnapCameraRequest.
// On success, responds with SnapCameraResponse.
func getSnapCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Failed to read request body:%v\n", err)

		http.Error(
			w,
			"failed to read request body",
			http.StatusBadRequest,
		)
		return
	}

	req := interfaces.SnapCameraRequest{}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("Failed to deserialize add snap camera request :%v\n", err)

		http.Error(
			w,
			"failed to deserialize body",
			http.StatusBadRequest,
		)
		return
	}

	// Obtain the buffer of the specified camera or all cameras.
	resp := interfaces.SnapCameraResponse{
		Cameras: map[string]interfaces.CameraResponseBase{},
	}

	if ip := net.ParseIP(req.IP); ip == nil {
		// No specific camera snap request.
		// Obtain the image buffer.
		for ip, entry := range camera.CameraPollerInstance.PollWorkers {
			snapshot := entry.GetSnapshot()
			resp.Cameras[ip] = interfaces.CameraResponseBase{
				Name: entry.Name,
				Data: snapshot.ImageData,
			}
		}
	} else {
		// Verify the ip exists.
		if cam, ok := camera.CameraPollerInstance.PollWorkers[req.IP]; !ok {
			log.Printf("Failed snap camera request for '%s'. Camera not found.\n", req.IP)

			http.Error(
				w,
				"camera not found",
				http.StatusBadRequest,
			)
			return
		} else {
			snapshot := cam.GetSnapshot()
			resp.Cameras[req.IP] = interfaces.CameraResponseBase{
				Name: cam.Name,
				Data: snapshot.ImageData,
			}
		}
	}

	// Serialize response.
	resBody, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to serialize snap camera response: %v\n", err)

		http.Error(
			w,
			"failed to serialize response",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(resBody)
}

// Streaming endpoint for continuously listening to new camera data.
// Response stream with StreamCameraResponse.
func getSubscribeCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Keep the TCP connection open, with a json event stream.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a multipart writer to split multiple responses with boundaries.
	multipartWriter := multipart.NewWriter(w)
	defer multipartWriter.Close()
	w.Header().Set(
		"Content-Type",
		fmt.Sprintf("multipart/form-data; boundary=%s", multipartWriter.Boundary()),
	)
	w.WriteHeader(http.StatusOK)

	// Consume request body.
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[/subscribe] Failed to read request body:%v\n", err)

		http.Error(
			w,
			"failed to read request body",
			http.StatusBadRequest,
		)
		return
	}
	r.Body.Close()

	// Deserialize request body.
	streamReq := &interfaces.StreamCameraRequest{}
	if err := json.Unmarshal(reqBody, streamReq); err != nil {
		log.Printf("[/subscribe] Failed to deserialize request body:%v\n", err)

		http.Error(
			w,
			"failed to read request body",
			http.StatusBadRequest,
		)
		return
	}

	// Send initial camera states.
	sendState := func() error {
		resp := interfaces.StreamCameraResponse{
			Cameras: map[string]interfaces.CameraResponseBase{},
		}
		for ip, entry := range camera.CameraPollerInstance.PollWorkers {
			// Filter on specific camera IP. Otherwise, stream all cameras.
			if streamReq.IP != "" && ip != streamReq.IP {
				continue
			}

			snapshot := entry.GetSnapshot()
			resp.Cameras[ip] = interfaces.CameraResponseBase{
				Name: entry.Name,
				Data: snapshot.ImageData,
			}
		}
		resBody, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Failed to serialize camera response: %v", err)

			http.Error(
				w,
				"failed to serialize response",
				http.StatusInternalServerError,
			)
			return err
		}

		// Indicate that the partition is going to be of type json.
		partWriter, err := multipartWriter.CreatePart(map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {fmt.Sprintf("%d", len(resBody))},
		})
		if err != nil {
			log.Printf("[/subscribe] Failed create multi-part writer:%v", err)
			http.Error(
				w,
				"failed to create partition writer",
				http.StatusInternalServerError,
			)
			return err
		}

		if _, err := partWriter.Write(resBody); err != nil {
			log.Printf("Failed to write serialized body to buffer: %v", err)
		}

		w.(http.Flusher).Flush()
		return nil
	}
	sendState()

	// Listen for new data.
	ticker := time.NewTicker(10 * time.Millisecond)

	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client '%s' /subscribe connection closed", r.RemoteAddr)
			return

		case <-ticker.C:
			if err := sendState(); err != nil {
				log.Printf("Client '%s' /subscribe connection closed due to error: %v", r.RemoteAddr, err)
				return
			}
		}
	}
}

// Create routes & handlers.
func CreateCameraRoutes(r *mux.Router) {
	r.HandleFunc("/add", postAddCameraHandler).Methods("POST")
	r.HandleFunc("/remove", postRemoveCameraHandler).Methods("POST")
	r.HandleFunc("/snap", getSnapCameraHandler).Methods("GET")
	r.HandleFunc("/subscribe", getSubscribeCameraHandler).Methods("GET")
}
