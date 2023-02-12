package database

import (
	"fmt"
)

// Query the database to get the node with the matching fingerprint.
func GetNodeByFingerprint(fingerprint string) (*Node, error) {
	node := Node{}
	if err := DbInstance.Model(&node).Where("node.certificate_fingerprint = ?", fingerprint).Select(); err != nil {
		return nil, fmt.Errorf("failed to find node with fingerprint '%s'", fingerprint)
	}
	return &node, nil
}
