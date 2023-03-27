package camera

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"4bit.api/v0/database"
	"4bit.api/v0/server/route/camera/interfaces"
	"github.com/gorilla/mux"
)

// GET endpoint request for retrieving all available cameras.
// Expects the Request to be of type ListCamerasRequest.
// Returns a ListCamerasResponse.
func getCameraListHandler(w http.ResponseWriter, r *http.Request) {
	// Grab request body.
	bodyBuffer, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to parse the request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req := interfaces.ListCamerasRequest{}
	if err := json.Unmarshal(bodyBuffer, &req); err != nil {
		log.Printf("Failed to de-serialize camera list request: %v\n", err)

		http.Error(
			w,
			"failed to de-serialize request body",
			http.StatusBadRequest,
		)
		return
	}

	// Set default limit.
	if req.Limit == 0 {
		req.Limit = 10
	}

	// Query all cameras.
	db := database.DbInstance
	cameras := []database.CameraEntry{}
	if err := db.Model(&cameras).Limit(int(req.Limit)).Select(); err != nil {
		log.Printf("Failed to query cameras for camera list request: %v\n", err)
		http.Error(
			w,
			"failed to query all cameras",
			http.StatusNotFound,
		)
		return
	}

	// Respond with a list of all cameras.
	resp := interfaces.ListCameraResponse{}
	resp.Cameras = cameras

	respBody, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to serialize response message for camera list request: %v\n", err)
		http.Error(
			w,
			"failed to serialize response body",
			http.StatusInternalServerError,
		)
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(respBody)
}

// Creates request routes & handlers.
func CreateCameraListRoute(r *mux.Router) {
	r.HandleFunc("/list", getCameraListHandler).Methods("GET")
}
