package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"github.com/mattn/go-sqlite3"
	"github.com/wybiral/hades/pkg/types"
	"os"
	"sync"
)

var (
	ErrNotFound       = errors.New("api: not found")
	ErrInitDatabase   = errors.New("api: error initializing db")
	ErrKeyGeneration  = errors.New("api: unable to generate random key")
	ErrKeyNotUnique   = errors.New("api: key not unique")
	ErrAlreadyStarted = errors.New("api: already started")
	ErrNotStarted     = errors.New("api: not started")
)

type Api struct {
	db          *sql.DB
	activeMutex *sync.RWMutex
	active      map[string]*activeDaemon
}

const schema = `
create table Daemon (
	key text not null primary key,
	cmd text not null,
	dir text not null,
	status string not null,
	disabled int not null
);
`

func NewApi(dbPath string) (*Api, error) {
	_, err := os.Stat(dbPath)
	create := false
	if err != nil {
		if os.IsNotExist(err) {
			// Keep track if db file doesn't exist
			create = true
		} else {
			return nil, ErrInitDatabase
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, ErrInitDatabase
	}
	if create {
		// Create tables if new db file
		_, err = db.Exec(schema)
		if err != nil {
			return nil, ErrInitDatabase
		}
	}
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
	db := api.db
	rows, err := db.Query(`
		select key
		from Daemon
		where disabled == 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
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
	db := api.db
	rows, err := db.Query(`
		select key, cmd, dir, status, disabled
		from Daemon
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	daemons := make([]*types.Daemon, 0)
	for rows.Next() {
		var key string
		var cmd string
		var dir string
		var status string
		var disabled int
		err = rows.Scan(&key, &cmd, &dir, &status, &disabled)
		if err != nil {
			return nil, err
		}
		daemons = append(daemons, &types.Daemon{
			Key:      key,
			Cmd:      cmd,
			Dir:      dir,
			Status:   status,
			Disabled: disabled == 1,
		})
	}
	return daemons, nil
}

// Return a single daemon (by key)
func (api *Api) GetDaemon(key string) (*types.Daemon, error) {
	var cmd string
	var dir string
	var status string
	var disabled int
	db := api.db
	row := db.QueryRow(`
		select cmd, dir, status, disabled
		from Daemon
		where key = ?
	`, key)
	err := row.Scan(&cmd, &dir, &status, &disabled)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &types.Daemon{
		Key:      key,
		Cmd:      cmd,
		Dir:      dir,
		Status:   status,
		Disabled: disabled == 1,
	}, nil
}

// Create a new daemon from a key, cmd strings
func (api *Api) CreateDaemon(key, cmd, dir string) (*types.Daemon, error) {
	if len(key) == 0 {
		return api.createDaemonKey(cmd, dir)
	}
	db := api.db
	_, err := db.Exec(`
		insert into Daemon (key, cmd, dir, status, disabled)
		values (?, ?, ?, "stopped", 1)
	`, key, cmd, dir)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if !ok {
			return nil, err
		}
		if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			return nil, ErrKeyNotUnique
		}
		return nil, err
	}
	return &types.Daemon{
		Key:      key,
		Cmd:      cmd,
		Dir:      dir,
		Status:   "stopped",
		Disabled: true,
	}, nil
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
	db := api.db
	_, err := db.Exec(`
		delete from Daemon
		where key = ?
	`, key)
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
	db := api.db
	res, err := db.Exec(`
		update Daemon
		set disabled = 0
		where key = ?
	`, key)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
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
