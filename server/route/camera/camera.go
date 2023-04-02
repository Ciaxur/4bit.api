package camera

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"4bit.api/v0/database"
	"4bit.api/v0/server/route/camera/interfaces"
	"github.com/gorilla/mux"
)

// Adds a new unique Camera entry to track & poll.
// Request expected to be of type AddCameraRequest.
// On success, responds with new database entry.
func postAddCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := ioutil.ReadAll(r.Body)
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

	// Add new entry to database.
	log.Printf("Adding new camera entry with ip '%s:%d'\n", req.Camera.IP, req.Camera.Port)
	camEntry := req.Camera
	camEntry.CreatedAt = time.Now()
	camEntry.ModifiedAt = camEntry.CreatedAt
	if _, err := db.Model(&camEntry).Insert(); err != nil {
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
	if CameraPollerInstance != nil {
		CameraPollerInstance.ShouldUpdateEntries = true
	}
}

// Removes a Camera entry from being tracked & polled.
// Request expected to be of type RemoveCameraRequest.
// On success, responds with emtpy message.
func postRemoveCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := ioutil.ReadAll(r.Body)
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
	if _, err := db.Model(&req.Camera).Where("camera_entry.ip = ?", req.Camera.IP).Delete(); err != nil {
		log.Printf("Failed to remove camera entry with ip '%s': %v\n", req.Camera.IP, err)

		http.Error(
			w,
			fmt.Sprintf("Failed to remove camera entry with ip '%s'", req.Camera.IP),
			http.StatusBadRequest,
		)
		return
	}

	log.Printf("Successfuly removed camera entry with ip '%s'\n", req.Camera.IP)
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte("{}"))

	// Update poller state.
	if CameraPollerInstance != nil {
		CameraPollerInstance.ShouldUpdateEntries = true
	}
}

// Gets the current state of all listening cameras or the state of a given camera
// IP address.
// Expects a request of type SnapCameraRequest.
// On success, responds with SnapCameraResponse.
func getSnapCameraHandler(w http.ResponseWriter, r *http.Request) {
	// Deserialize expected request.
	bodyBytes, err := ioutil.ReadAll(r.Body)
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
		for ip, entry := range CameraPollerInstance.CameraConnectionMp {
			resp.Cameras[ip] = interfaces.CameraResponseBase{
				Name: entry.Name,
				Data: entry.LastReadData,
			}
		}
	} else {
		// Verify the ip exists.
		if cam, ok := CameraPollerInstance.CameraConnectionMp[req.IP]; !ok {
			log.Printf("Failed snap camera request for '%s'. Camera not found.\n", req.IP)

			http.Error(
				w,
				"camera not found",
				http.StatusBadRequest,
			)
			return
		} else {
			resp.Cameras[req.IP] = interfaces.CameraResponseBase{
				Name: cam.Name,
				Data: cam.LastReadData,
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Send initial camera states.
	sendState := func() {
		resp := interfaces.StreamCameraResponse{
			Cameras: map[string]interfaces.CameraResponseBase{},
		}
		for ip, entry := range CameraPollerInstance.CameraConnectionMp {
			resp.Cameras[ip] = interfaces.CameraResponseBase{
				Name: entry.Name,
				Data: entry.LastReadData,
			}
		}
		resBody, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Failed to serialize camera response: %v\n", err)

			http.Error(
				w,
				"failed to serialize response",
				http.StatusInternalServerError,
			)
			return
		}

		if _, err := w.Write(resBody); err != nil {
			log.Printf("Failed to write serialized body to buffer: %v\n", err)
		}
		w.(http.Flusher).Flush()
	}
	sendState()

	// Listen for new data.
	ticker := time.NewTicker(500 * time.Millisecond)
	lastChecked := CameraPollerInstance.LastUpdated

	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client '%s' connection closed\n", r.RemoteAddr)
			return

		case <-ticker.C:
			// Check if any entires where updated.
			if lastChecked != CameraPollerInstance.LastUpdated {
				lastChecked = CameraPollerInstance.LastUpdated
				sendState()
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
