package database

import (
	"4bit.api/v0/server/route/node/interfaces"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type Node struct {
	BaseEntry
	CertificateFingerprint string
}

type NodePowerState struct {
	BaseEntry
	interfaces.PowerState

	// Relationship.
	NodeId uint64
	Node   *Node `pg:"rel:has-one"`
}

type NodeBarometerState struct {
	BaseEntry
	interfaces.BarometerState

	// Relationship.
	NodeId uint64
	Node   *Node `pg:"rel:has-one"`
}

func CreateNodeSchema(db *pg.DB) error {
	models := []interface{}{
		(*Node)(nil),
		(*NodePowerState)(nil),
		(*NodeBarometerState)(nil),
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
