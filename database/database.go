package database

import (
	"fmt"
	"log"

	"github.com/go-pg/pg/v10"
)

// Singleton database connection
var (
	DbInstance *pg.DB
)

// Creates a new connection with a given postgres sql server, setting up the schemas for the db.
func NewConnection(options *pg.Options) (*pg.DB, error) {
	// Return already established connection if present.
	if DbInstance != nil {
		return DbInstance, nil
	}

	DbInstance = pg.Connect(options)

	// Create schemas.
	log.Println("Attempting to create Node schema")
	if err := CreateNodeSchema(DbInstance); err != nil {
		return nil, fmt.Errorf("failed to create node schema: %v", err)
	}

	log.Println("Attempting to create Camera schema")
	if err := CreateCameraSchema(DbInstance); err != nil {
		return nil, fmt.Errorf("failed to create camera schema: %v", err)
	}

	return DbInstance, nil
}
