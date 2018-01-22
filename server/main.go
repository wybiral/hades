package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/shlex"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wybiral/hades/types"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

type App struct {
	mutex   *sync.RWMutex
	DB      *sql.DB
	Running map[string]chan struct{}
}

func NewApp(dbPath string) (*App, error) {
	_, err := os.Stat(dbPath)
	create := false
	if err != nil {
		if os.IsNotExist(err) {
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
		DB:      db,
		Running: make(map[string]chan struct{}),
	}
	running, err := app.getRunning()
	if err != nil {
		return nil, err
	}
	for _, key := range running {
		app.StartDaemon(key)
	}
	return app, nil
}

func (app *App) getRunning() ([]string, error) {
	db := app.DB
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

func (app *App) GetDaemons() ([]*types.Daemon, error) {
	db := app.DB
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

func (app *App) GetDaemon(key string) (*types.Daemon, error) {
	var cmd string
	var running int
	db := app.DB
	row := db.QueryRow("select cmd, running from Daemon where key = ?", key)
	err := row.Scan(&cmd, &running)
	if err != nil {
		return nil, err
	}
	return &types.Daemon{Key: key, Cmd: cmd, Running: running != 0}, nil
}

func (app *App) CreateDaemon(cmd string) (string, error) {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	key, err := createKey()
	if err != nil {
		return "", err
	}
	db := app.DB
	_, err = db.Exec("insert into Daemon (key, cmd, running) values (?, ?, 0)", key, cmd)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (app *App) StartDaemon(key string) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	_, exists := app.Running[key]
	if exists {
		return errors.New("start: already running")
	}
	db := app.DB
	_, err := db.Exec("update Daemon set running = 1 where key = ?", key)
	if err != nil {
		return err
	}
	stop := make(chan struct{}, 1)
	app.Running[key] = stop
	go runLoop(app, key, stop)
	return nil
}

func runLoop(app *App, key string, stop chan struct{}) {
	defer func() {
		app.mutex.Lock()
		defer app.mutex.Unlock()
		delete(app.Running, key)
	}()
	var command string
	db := app.DB
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
			return
		}
	}
}

func (app *App) StopDaemon(key string) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	stop, exists := app.Running[key]
	if !exists {
		return errors.New("stop: not started")
	}
	defer func() {
		delete(app.Running, key)
		db := app.DB
		db.Exec("update Daemon set running = 0 where key = ?", key)
	}()
	select {
	case stop <- struct{}{}:
		return nil
	default:
		return nil
	}
}

func main() {
	app, err := NewApp("app.db")
	if err != nil {
		log.Println(err)
		return
	}
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexHandler(app, w, r)
	}).Methods("GET")
	r.HandleFunc("/{key}", func(w http.ResponseWriter, r *http.Request) {
		daemonGetHandler(app, w, r)
	}).Methods("GET")
	r.HandleFunc("/{key}/start", func(w http.ResponseWriter, r *http.Request) {
		daemonStartHandler(app, w, r)
	})
	r.HandleFunc("/{key}/stop", func(w http.ResponseWriter, r *http.Request) {
		daemonStopHandler(app, w, r)
	})
	r.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		addPostHandler(app, w, r)
	}).Methods("POST")
	addr := "127.0.0.1:8080"
	log.Println("Serving at", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Println(err)
	}
}

func indexHandler(app *App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	daemons, err := app.GetDaemons()
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemons)
	if err != nil {
		return
	}
}

func daemonGetHandler(app *App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	daemon, err := app.GetDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemon)
	if err != nil {
		return
	}
}

func daemonStartHandler(app *App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := app.StartDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode("OK")
	if err != nil {
		return
	}
}

func daemonStopHandler(app *App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := app.StopDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode("OK")
	if err != nil {
		return
	}
}

func addPostHandler(app *App, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cmd := r.PostForm.Get("cmd")
	if len(cmd) == 0 {
		return
	}
	key, err := app.CreateDaemon(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	daemon, err := app.GetDaemon(key)
	if err != nil {
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemon)
	if err != nil {
		return
	}
}
