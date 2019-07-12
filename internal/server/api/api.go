package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/wybiral/hades/pkg/types"
)

var (
	// ErrNotFound returned when daemon doesn't exist in DB.
	ErrNotFound = errors.New("api: not found")
	// ErrInitDatabase returned from errors initializing DB.
	ErrInitDatabase = errors.New("api: error initializing db")
	// ErrKeyNotUnique returned when daemon key isn't unique.
	ErrKeyNotUnique = errors.New("api: key not unique")
	// ErrAlreadyStarted returned when starting daemon already started.
	ErrAlreadyStarted = errors.New("api: already started")
	// ErrNotStarted returned when stopping daemon not started.
	ErrNotStarted = errors.New("api: not started")
)

var daemonBucket = []byte("daemons")

// API represents server-side API.
type API struct {
	db          *bolt.DB
	activeMutex *sync.RWMutex
	active      map[string]*activeDaemon
}

// NewAPI returns new server Api from dbPath.
func NewAPI(dbPath string) (*API, error) {
	opts := &bolt.Options{Timeout: 1 * time.Second}
	db, err := bolt.Open(dbPath, 0666, opts)
	if err != nil {
		return nil, ErrInitDatabase
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(daemonBucket)
		return err
	})
	if err != nil {
		return nil, ErrInitDatabase
	}
	api := &API{
		db:          db,
		activeMutex: &sync.RWMutex{},
		active:      make(map[string]*activeDaemon),
	}
	// Start all active daemons
	active, err := api.getActive()
	if err != nil {
		return nil, err
	}
	for _, key := range active {
		err := api.StartDaemon(key)
		if err != nil {
			return nil, err
		}
	}
	return api, nil
}

// Return array of keys for running daemons
func (api *API) getActive() ([]string, error) {
	keys := make([]string, 0)
	err := api.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := &types.Daemon{}
			err := json.Unmarshal(v, d)
			if err != nil {
				return err
			}
			if !d.Disabled {
				keys = append(keys, d.Key)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Create a new daemon key
func generateKey(n int) (string, error) {
	data := make([]byte, n)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// GetDaemons returns array of all daemons.
func (api *API) GetDaemons() ([]*types.Daemon, error) {
	daemons := make([]*types.Daemon, 0)
	err := api.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := &types.Daemon{}
			err := json.Unmarshal(v, d)
			if err != nil {
				return err
			}
			daemons = append(daemons, d)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return daemons, nil
}

// GetDaemon returns a single daemon by key.
func (api *API) GetDaemon(key string) (*types.Daemon, error) {
	d := &types.Daemon{}
	err := api.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get([]byte(key))
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// CreateDaemon creates a new daemon from key, cmd, dir.
func (api *API) CreateDaemon(key, cmd, dir string) (*types.Daemon, error) {
	if len(key) == 0 {
		return api.createDaemonKey(cmd, dir)
	}
	d := &types.Daemon{
		Key:      key,
		Cmd:      cmd,
		Dir:      dir,
		Status:   "stopped",
		Disabled: true,
	}
	err := api.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), enc)
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// When CreateDaemon is called with empty key, this will generate one by
// generating random base64 strings, checking for collisions, and growing
// gradually if too many collisions are found.
func (api *API) createDaemonKey(cmd, dir string) (*types.Daemon, error) {
	n := 3
	fails := 0
	for {
		key, err := generateKey(n)
		if err != nil {
			return nil, err
		}
		daemon, err := api.CreateDaemon(key, cmd, dir)
		if err == ErrKeyNotUnique {
			// Key isn't unique, try again
			fails++
			if fails > 4 {
				// Too many fails, increase key size
				fails = 0
				n++
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		return daemon, nil
	}
}

// DeleteDaemon deletes a daemon.
func (api *API) DeleteDaemon(key string) error {
	api.activeMutex.Lock()
	defer api.activeMutex.Unlock()
	_, exists := api.active[key]
	if exists {
		return ErrAlreadyStarted
	}
	err := api.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		return b.Delete([]byte(key))
	})
	if err != nil {
		return err
	}
	return nil
}

// StartDaemon starts a daemon.
func (api *API) StartDaemon(key string) error {
	api.activeMutex.Lock()
	defer api.activeMutex.Unlock()
	_, exists := api.active[key]
	if exists {
		return ErrAlreadyStarted
	}
	err := api.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get([]byte(key))
		d := &types.Daemon{}
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		d.Disabled = false
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), enc)
	})
	if err != nil {
		return err
	}
	api.active[key] = newActiveDaemon(api, key)
	return nil
}

// getActiveDaemon returns an active daemon (if exists).
func (api *API) getActiveDaemon(key string) (*activeDaemon, error) {
	api.activeMutex.RLock()
	defer api.activeMutex.RUnlock()
	ad, exists := api.active[key]
	if !exists {
		return nil, ErrNotStarted
	}
	return ad, nil
}

// StopDaemon sends "KILL" signal to running daemon.
func (api *API) StopDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigkill()
	return nil
}

// PauseDaemon sends a "STOP" signal to running daemon to pause it.
func (api *API) PauseDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigstop()
	return nil
}

// ContinueDaemon sends a "CONT" signal to a paused daemon to unpause it.
func (api *API) ContinueDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigcont()
	return nil
}
