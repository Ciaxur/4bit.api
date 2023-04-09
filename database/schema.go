package database

import "time"

type BaseEntry struct {
	Id        uint64
	Timestamp time.Time
}
