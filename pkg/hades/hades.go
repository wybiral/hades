package hades

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync"

	"github.com/boltdb/bolt"
)

var (
	// ErrNotFound returned when daemon doesn't exist in DB.
	ErrNotFound = errors.New("hades: not found")
	// ErrInitDatabase returned from errors initializing DB.
	ErrInitDatabase = errors.New("hades: error initializing db")
	// ErrAlreadyStarted returned when starting daemon already started.
	ErrAlreadyStarted = errors.New("hades: already started")
	// ErrNotStarted returned when stopping daemon not started.
	ErrNotStarted = errors.New("hades: not started")
)

// bolt.DB bucket for daemons
var daemonBucket = []byte("daemons")

// Hades represents main daemon manager.
type Hades struct {
	db          *bolt.DB
	activeMutex sync.RWMutex
	active      map[uint64]*activeDaemon
}

// NewHades returns new Hades instance from dbPath.
func NewHades(db *bolt.DB) (*Hades, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(daemonBucket)
		return err
	})
	if err != nil {
		return nil, ErrInitDatabase
	}
	h := &Hades{
		db:          db,
		activeMutex: sync.RWMutex{},
		active:      make(map[uint64]*activeDaemon),
	}
	// Start all active daemons
	active, err := h.getActive()
	if err != nil {
		return nil, err
	}
	for _, id := range active {
		err := h.Start(id)
		if err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Return array of ids for running daemons
func (h *Hades) getActive() ([]uint64, error) {
	ids := make([]uint64, 0)
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := &Daemon{}
			err := json.Unmarshal(v, d)
			if err != nil {
				return err
			}
			if !d.Disabled {
				ids = append(ids, d.ID)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// Daemons returns array of all daemons.
func (h *Hades) Daemons() ([]*Daemon, error) {
	daemons := make([]*Daemon, 0)
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := &Daemon{}
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

// Get returns a single daemon by id.
func (h *Hades) Get(id uint64) (*Daemon, error) {
	d := &Daemon{}
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get(itob(id))
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

// Add adds a new daemon to Hades from cmd, dir.
func (h *Hades) Add(cmd, dir string) (*Daemon, error) {
	d := &Daemon{
		Cmd:      cmd,
		Dir:      dir,
		Status:   "stopped",
		Disabled: true,
	}
	err := h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		d.ID = id
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put(itob(id), enc)
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Remove removes a daemon from Hades.
func (h *Hades) Remove(id uint64) error {
	h.activeMutex.Lock()
	defer h.activeMutex.Unlock()
	_, exists := h.active[id]
	if exists {
		return ErrAlreadyStarted
	}
	err := h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		return b.Delete(itob(id))
	})
	if err != nil {
		return err
	}
	return nil
}

// Start starts a daemon.
func (h *Hades) Start(id uint64) error {
	h.activeMutex.Lock()
	defer h.activeMutex.Unlock()
	_, exists := h.active[id]
	if exists {
		return ErrAlreadyStarted
	}
	err := h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get(itob(id))
		d := &Daemon{}
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		d.Disabled = false
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put(itob(id), enc)
	})
	if err != nil {
		return err
	}
	h.active[id] = newActiveDaemon(h, id)
	return nil
}

// getActiveDaemon returns an active daemon (if exists).
func (h *Hades) getActiveDaemon(id uint64) (*activeDaemon, error) {
	h.activeMutex.RLock()
	defer h.activeMutex.RUnlock()
	ad, exists := h.active[id]
	if !exists {
		return nil, ErrNotStarted
	}
	return ad, nil
}

// Stop sends "KILL" signal to running daemon.
func (h *Hades) Stop(id uint64) error {
	ad, err := h.getActiveDaemon(id)
	if err != nil {
		return err
	}
	ad.sigkill()
	return nil
}

// Pause sends a "STOP" signal to running daemon to pause it.
func (h *Hades) Pause(id uint64) error {
	ad, err := h.getActiveDaemon(id)
	if err != nil {
		return err
	}
	ad.sigstop()
	return nil
}

// Resume sends a "CONT" signal to a paused daemon to resume it.
func (h *Hades) Resume(id uint64) error {
	ad, err := h.getActiveDaemon(id)
	if err != nil {
		return err
	}
	ad.sigcont()
	return nil
}

// convert uint64 to big engian bytes (for IDs)
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
