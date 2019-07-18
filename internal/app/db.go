package app

import (
	"time"

	"github.com/boltdb/bolt"
)

func newDB(dbPath string) (*bolt.DB, error) {
	opts := &bolt.Options{Timeout: 1 * time.Second}
	return bolt.Open(dbPath, 0666, opts)
}
