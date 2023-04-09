package database

import (
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type CameraEntry struct {
	Id         uint64
	Name       string
	CreatedAt  time.Time
	ModifiedAt time.Time
	IP         string
	Port       uint16

	// Adjustment Relationship
	AdjustmentId uint64
	Adjustment   *CameraAdjsustment `pg:"rel:has-one"`
}

type CameraAdjsustment struct {
	BaseEntry

	// Frame Crop
	CropFrameHeight float64
	CropFrameWidth  float64
	CropFrameX      uint64
	CropFrameY      uint64
}

func CreateCameraSchema(db *pg.DB) error {
	models := []interface{}{
		(*CameraEntry)(nil),
		(*CameraAdjsustment)(nil),
	}

	// Attempt to create the table schemas
	for _, model := range models {
		if err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		}); err != nil {
			return fmt.Errorf("failed to create tables: %v", err)
		}
	}

	return nil
}
