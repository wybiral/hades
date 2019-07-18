package app

import (
	"crypto/rand"

	"github.com/boltdb/bolt"
	"github.com/gorilla/sessions"
)

func newSessions(a *App) (sessions.Store, error) {
	hashKey, err := readKey(a.DB, "hash_key")
	if err != nil {
		return nil, err
	}
	blockKey, err := readKey(a.DB, "block_key")
	if err != nil {
		return nil, err
	}
	s := sessions.NewCookieStore(hashKey, blockKey)
	s.Options.HttpOnly = true
	return s, nil
}

// readKey reads sessions keys from database by name, generating them if they
// don't exist already.
func readKey(db *bolt.DB, name string) ([]byte, error) {
	key := make([]byte, 32)
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("settings"))
		if err != nil {
			return err
		}
		// return key if exists
		k := b.Get([]byte(name))
		if k != nil {
			copy(key, k)
			return nil
		}
		// if key not found, generate one
		_, err = rand.Read(key)
		if err != nil {
			return err
		}
		return b.Put([]byte(name), key)
	})
	if err != nil {
		return nil, err
	}
	return key, nil
}
