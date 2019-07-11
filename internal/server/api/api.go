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
	ErrNotFound       = errors.New("api: not found")
	ErrInitDatabase   = errors.New("api: error initializing db")
	ErrKeyGeneration  = errors.New("api: unable to generate random key")
	ErrKeyNotUnique   = errors.New("api: key not unique")
	ErrAlreadyStarted = errors.New("api: already started")
	ErrNotStarted     = errors.New("api: not started")
)

var daemonBucket = []byte("daemons")

type Api struct {
	db          *bolt.DB
	activeMutex *sync.RWMutex
	active      map[string]*activeDaemon
}

func NewApi(dbPath string) (*Api, error) {
	opts := &bolt.Options{Timeout: 1 * time.Second}
	db, err := bolt.Open(dbPath, 0666, opts)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(daemonBucket)
		return err
	})
	api := &Api{
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
func (api *Api) getActive() ([]string, error) {
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
	_n, err := rand.Read(data)
	if err != nil {
		return "", ErrKeyGeneration
	}
	if _n != n {
		return "", ErrKeyGeneration
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// Return array of all daemons
func (api *Api) GetDaemons() ([]*types.Daemon, error) {
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

// Return a single daemon (by key)
func (api *Api) GetDaemon(key string) (*types.Daemon, error) {
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

// Create a new daemon from a key, cmd strings
func (api *Api) CreateDaemon(key, cmd, dir string) (*types.Daemon, error) {
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

// When CreateDaemon is called with empty key, this will generate one
// XXX: Find a better key generation scheme
func (api *Api) createDaemonKey(cmd, dir string) (*types.Daemon, error) {
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
			fails += 1
			if fails > 4 {
				// Too many fails, increase key size
				fails = 0
				n += 1
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		return daemon, nil
	}
}

func (api *Api) DeleteDaemon(key string) error {
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

// Start daemon (by key)
func (api *Api) StartDaemon(key string) error {
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

func (api *Api) getActiveDaemon(key string) (*activeDaemon, error) {
	api.activeMutex.RLock()
	defer api.activeMutex.RUnlock()
	ad, exists := api.active[key]
	if !exists {
		return nil, ErrNotStarted
	}
	return ad, nil
}

func (api *Api) StopDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigkill()
	return nil
}

func (api *Api) PauseDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigstop()
	return nil
}

func (api *Api) ContinueDaemon(key string) error {
	ad, err := api.getActiveDaemon(key)
	if err != nil {
		return err
	}
	ad.sigcont()
	return nil
}
