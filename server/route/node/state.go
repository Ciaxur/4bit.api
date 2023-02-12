package node

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"4bit.api/v0/database"
	"4bit.api/v0/server/route/node/interfaces"
	"github.com/gorilla/mux"
)

// GET request handler for retrieving the current node's information.
// The current node is determined by the request certificate.
func nodeStateGetHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request.
	bodyBuffer, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to parse the request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	stateRequest := interfaces.StateGetRequest{}
	if err := json.Unmarshal(bodyBuffer, &stateRequest); err != nil {
		log.Println("Internal Error: Failed to de-serialize State request body")
		http.Error(w, "failed to de-serialize request body", http.StatusInternalServerError)
		return
	}

	// TODO: Cache these.
	// Verify client node already exists in the DB.
	clientCert := r.TLS.PeerCertificates[0]
	fingerprint := extractCertificateFingerprint(clientCert)
	node, err := database.GetNodeByFingerprint(fingerprint)
	if err != nil {
		http.Error(w, "node does not exist. create a node entry first", http.StatusUnauthorized)
		return
	}

	// Set defaults.
	if stateRequest.Limit == nil {
		*stateRequest.Limit = 5
	}

	// Handle request based on type.
	db := database.DbInstance
	var responseBuffer []byte
	switch stateRequest.Type {
	case interfaces.BAROMETER:
		barometerStates := []database.NodeBarometerState{}
		if err := db.Model(&barometerStates).Relation("Node").Where("node_id = ?", node.Id).Limit(int(*stateRequest.Limit)).Select(); err != nil {
			log.Printf("Failed to requeste barometer data client '%s': %v", node.CertificateFingerprint, err)
			http.Error(w, "failed to request barometer entries", http.StatusInternalServerError)
			return
		}

		// Serialize response.
		buffer, err := json.Marshal(barometerStates)
		if err != nil {
			log.Println("Internal Error: Failed to serialize barometer response")
			http.Error(w, "failed to serialize response", http.StatusInternalServerError)
			return
		}
		responseBuffer = buffer

	case interfaces.POWER:
		powerStates := []database.NodePowerState{}
		if err := db.Model(&powerStates).Relation("Node").Where("node_id = ?", node.Id).Limit(int(*stateRequest.Limit)).Select(); err != nil {
			log.Printf("Failed to requeste power data client '%s': %v", node.CertificateFingerprint, err)
			http.Error(w, "failed to request power entries", http.StatusInternalServerError)
			return
		}

		// Serialize response.
		buffer, err := json.Marshal(powerStates)
		if err != nil {
			log.Println("Internal Error: Failed to serialize power response")
			http.Error(w, "failed to serialize response", http.StatusInternalServerError)
			return
		}
		responseBuffer = buffer

	default:
		http.Error(w, "unknown request type", http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(responseBuffer)
}

// POST request handler for creating an entry for the current node.
// The current node is determined by the request certificate.
func nodeStatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request.
	bodyBuffer, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to parse the request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	stateRequest := interfaces.StatePostRequest{}
	if err := json.Unmarshal(bodyBuffer, &stateRequest); err != nil {
		log.Println("Internal Error: Failed to de-serialize State request body")
		http.Error(w, "failed to de-serialize request body", http.StatusInternalServerError)
		return
	}

	// Early return on invalid request.
	if stateRequest.BarometerState == nil && stateRequest.Power == nil {
		http.Error(w, "empty states in request not allowed", http.StatusBadRequest)
		return
	}

	// TODO: Cache these.
	// Verify client node already exists in the DB.
	clientCert := r.TLS.PeerCertificates[0]
	fingerprint := extractCertificateFingerprint(clientCert)
	node, err := database.GetNodeByFingerprint(fingerprint)
	if err != nil {
		http.Error(w, "node does not exist. create a node entry first", http.StatusUnauthorized)
		return
	}

	// Handle request based on the given states.
	// TODO: Bulk apply to the database.
	db := database.DbInstance
	if stateRequest.BarometerState != nil {
		// Create a node state entry associated with the client node.
		nodeBarStateEntry := database.NodeBarometerState{
			BarometerState: *stateRequest.BarometerState,
			NodeId:         node.Id,
		}
		nodeBarStateEntry.Timestamp = time.Now().UTC()

		// Create new entry.
		if _, err := db.Model(&nodeBarStateEntry).Insert(); err != nil {
			log.Printf("New Barometer entry failed for node '%s': %v", node.CertificateFingerprint, err)
			http.Error(
				w,
				fmt.Sprintf("failed to create new barometer entry: %v", err),
				http.StatusBadRequest,
			)
			return
		}
		log.Printf("New barometer entry[%d] created for node '%s'", nodeBarStateEntry.Id, node.CertificateFingerprint)
	}

	if stateRequest.Power != nil {
		// Create a node state entry associated with the client node.
		nodePowerStateEntry := database.NodePowerState{
			PowerState: *stateRequest.Power,
			NodeId:     node.Id,
		}
		nodePowerStateEntry.Timestamp = time.Now().UTC()

		// Create new entry.
		if _, err := db.Model(&nodePowerStateEntry).Insert(); err != nil {
			log.Printf("New Power entry failed for node '%s': %v", node.CertificateFingerprint, err)
			http.Error(
				w,
				fmt.Sprintf("failed to create new power entry: %v", err),
				http.StatusBadRequest,
			)
			return
		}
		log.Printf("New power entry[%d] created for node '%s'", nodePowerStateEntry.Id, node.CertificateFingerprint)
	}
	w.WriteHeader(http.StatusOK)
}

func CreateStateRoute(r *mux.Router) {
	r.HandleFunc("/state", nodeStateGetHandler).Methods("GET")
	r.HandleFunc("/state", nodeStatePostHandler).Methods("POST")
}
