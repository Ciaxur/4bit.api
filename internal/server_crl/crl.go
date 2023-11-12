package server_crl

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"os"
	"sync"

	"4bit.api/v0/utils/filewatcher"
)

var (
	CachedCaCrl   *pkix.CertificateList
	clr_load_m    *sync.Mutex
	crl_filepath_ string
)

// Initialize pre-req. variables and cache the initial CRL.
func Init(crl_filepath string) error {
	clr_load_m = &sync.Mutex{}
	crl_filepath_ = crl_filepath
	if err := LoadCACrl(crl_filepath_); err != nil {
		return fmt.Errorf("failed to initiate crl: %v", err)
	}

	// Start a go routine which handles updating the CRL when the file changes.
	go ListenForCACrlChanges()
	return nil
}

// Function indented to be run in a go routine, which updates the CA's cached
// CRL when the file is modified.
func ListenForCACrlChanges() {
	// Start listening for CRL changes.
	fw := filewatcher.NewFileWatcher(crl_filepath_)
	defer fw.Close()

	for fw.IsRunning {
		<-fw.ChangeTriggerChan
		if err := LoadCACrl(crl_filepath_); err != nil {
			log.Printf("failed to reload CA CRL: %v", err)
		}
		log.Println("CA's CRL file successfully updated.")
	}
}

// Update & cache the CA's Certificate Revocation List (CRL).
// This function could be called concurrently and thus accounts for concurrent
// calls.
func LoadCACrl(crl_filepath string) error {
	// Ensure we don't collide with concurrent function calls.
	clr_load_m.Lock()
	defer clr_load_m.Unlock()

	rawCrl, err := os.ReadFile(crl_filepath)
	if err != nil {
		return fmt.Errorf("failed to read CA's CRL file from %s: %v", crl_filepath, err)
	}

	CachedCaCrl, err = x509.ParseCRL(rawCrl)
	if err != nil {
		return fmt.Errorf("failed to parse crl: %v", err)
	}

	return nil
}
