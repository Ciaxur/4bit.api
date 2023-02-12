package node

import (
	"bytes"
	"crypto/md5"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"4bit.api/v0/database"
	"github.com/gorilla/mux"
)

func extractCertificateFingerprint(cert *x509.Certificate) string {
	fingerprintBytes := md5.Sum(cert.Raw)

	var fpBuffer bytes.Buffer
	for i, v := range fingerprintBytes {
		if i > 0 {
			fmt.Fprint(&fpBuffer, ":")
		}
		fmt.Fprintf(&fpBuffer, "%02X", v)
	}

	return fpBuffer.String()
}

// Requests the current node's entry.
func nodeGetHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the node's certificate signature.
	cert := r.TLS.PeerCertificates[0]
	fingerprint := extractCertificateFingerprint(cert)

	// Query the database to get the node with the matching fingerprint.
	node, err := database.GetNodeByFingerprint(fingerprint)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to find node with fingerprint '%s'", fingerprint),
			http.StatusNotFound,
		)
	}

	// Serialize response.
	serializedResponse, err := json.Marshal(node)
	if err != nil {
		http.Error(w, "failed to serialize response", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(serializedResponse)
}

// Requests new entry for the current node's.
func nodePostHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the node's certificate signature.
	cert := r.TLS.PeerCertificates[0]
	fingerprint := extractCertificateFingerprint(cert)

	// Query the database to check if node entry already exists.
	// Node already exists.
	if _, err := database.GetNodeByFingerprint(fingerprint); err == nil {
		http.Error(w, "node already exists", http.StatusConflict)
		return
	}

	// Create new entry for node.
	log.Printf("Creating new node entry with fingerprint %s", fingerprint)
	db := database.DbInstance
	node := database.Node{
		CertificateFingerprint: fingerprint,
	}
	node.Timestamp = time.Now().UTC()

	if _, err := db.Model(&node).Insert(); err != nil {
		log.Printf("Failed to create node entry for fingerprint '%s': %v", fingerprint, err)
		http.Error(w, "failed to create node entry", http.StatusInternalServerError)
		return
	}

	// Serialize the new entry.
	serializedNode, err := json.Marshal(node)
	if err != nil {
		log.Println("Internal Error: Failed to serialize node entry")
		http.Error(w, "failed to serialize response", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(serializedNode)
}

func CreateNodeRoute(r *mux.Router) {
	r.HandleFunc("", nodeGetHandler).Methods("GET")
	r.HandleFunc("", nodePostHandler).Methods("POST")
}
