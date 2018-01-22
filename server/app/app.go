package app

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"github.com/google/shlex"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wybiral/hades/types"
	"os"
	"os/exec"
	"sync"
)

type App struct {
	mutex *sync.RWMutex
	db    *sql.DB
	// "stop signal" channels for running daemons
	running map[string]chan struct{}
}

func NewApp(dbPath string) (*App, error) {
	_, err := os.Stat(dbPath)
	create := false
	if err != nil {
		if os.IsNotExist(err) {
			// Keep track if db file doesn't exist
			create = true
		} else {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if create {
		// Create tables if new db file
		_, err = db.Exec(`
			create table Daemon (
				key text not null primary key,
				cmd text not null,
				running int not null
			);
		`)
		if err != nil {
			return nil, err
		}
	}
	app := &App{
		mutex:   &sync.RWMutex{},
		db:      db,
		running: make(map[string]chan struct{}),
	}
	// Start all running daemons
	running, err := app.getRunning()
	if err != nil {
		return nil, err
	}
	for _, key := range running {
		app.StartDaemon(key)
	}
	return app, nil
}

// Return array of keys for running daemons
func (app *App) getRunning() ([]string, error) {
	db := app.db
	rows, err := db.Query("select key from Daemon where running == 1")
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
func createKey() (string, error) {
	n := 8
	data := make([]byte, n)
	_n, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	if _n != n {
		return "", errors.New("createKey: not enough random")
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// Return array of all daemons
func (app *App) GetDaemons() ([]*types.Daemon, error) {
	db := app.db
	rows, err := db.Query("select key, cmd, running from Daemon")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	daemons := make([]*types.Daemon, 0)
	for rows.Next() {
		var key string
		var cmd string
		var running int
		err = rows.Scan(&key, &cmd, &running)
		if err != nil {
			return nil, err
		}
		daemons = append(daemons, &types.Daemon{
			Key:     key,
			Cmd:     cmd,
			Running: running == 1,
		})
	}
	return daemons, nil
}

// Return a single daemon (by key)
func (app *App) GetDaemon(key string) (*types.Daemon, error) {
	var cmd string
	var running int
	db := app.db
	row := db.QueryRow("select cmd, running from Daemon where key = ?", key)
	err := row.Scan(&cmd, &running)
	if err != nil {
		return nil, err
	}
	return &types.Daemon{Key: key, Cmd: cmd, Running: running != 0}, nil
}

// Create a new daemon from a cmd string, return key
func (app *App) CreateDaemon(cmd string) (string, error) {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	key, err := createKey()
	if err != nil {
		return "", err
	}
	db := app.db
	_, err = db.Exec("insert into Daemon (key, cmd, running) values (?, ?, 0)", key, cmd)
	if err != nil {
		return "", err
	}
	return key, nil
}

// Start daemon (by key)
func (app *App) StartDaemon(key string) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	_, exists := app.running[key]
	if exists {
		return errors.New("start: already running")
	}
	db := app.db
	_, err := db.Exec("update Daemon set running = 1 where key = ?", key)
	if err != nil {
		return err
	}
	stop := make(chan struct{}, 1)
	app.running[key] = stop
	go runLoop(app, key, stop)
	return nil
}

// Run daemon (by key) repeatedly until stop channel signal
func runLoop(app *App, key string, stop chan struct{}) {
	defer func() {
		app.mutex.Lock()
		defer app.mutex.Unlock()
		delete(app.running, key)
	}()
	var command string
	db := app.db
	row := db.QueryRow("select cmd from Daemon where key = ?", key)
	err := row.Scan(&command)
	if err != nil {
		return
	}
	parts, err := shlex.Split(command)
	if err != nil {
		return
	}
	for {
		cmd := exec.Command(parts[0], parts[1:]...)
		err := cmd.Start()
		if err != nil {
			continue
		}
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-stop:
			_ = cmd.Process.Kill()
			return
		case <-done:
			continue
		}
	}
}

// Stop a running daemon (by key)
func (app *App) StopDaemon(key string) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	stop, exists := app.running[key]
	if !exists {
		return errors.New("stop: not started")
	}
	defer func() {
		delete(app.running, key)
		db := app.db
		db.Exec("update Daemon set running = 0 where key = ?", key)
	}()
	select {
	case stop <- struct{}{}:
		return nil
	default:
		return nil
	}
}