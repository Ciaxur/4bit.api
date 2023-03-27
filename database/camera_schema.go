package database

import (
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

	// TODO: mTLS & inherit that from Node
}

func CreateCameraSchema(db *pg.DB) error {
	models := []interface{}{
		(*CameraEntry)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})

		if err != nil {
			return err
		}
	}

	return nil
}
