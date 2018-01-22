package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/shlex"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

type App struct {
	mutex   *sync.RWMutex
	Daemons map[string]*Daemon `json:"daemons"`
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

func (app *App) NewDaemon(cmd string) (*Daemon, error) {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	var key string
	var err error
	for {
		key, err = createKey()
		if err != nil {
			return nil, err
		}
		_, exists := app.Daemons[key]
		if !exists {
			break
		}
	}
	daemon := &Daemon{
		Key:     key,
		Cmd:     cmd,
		mutex:   &sync.RWMutex{},
		Running: false,
	}
	app.Daemons[key] = daemon
	return daemon, nil
}

func (app *App) GetDaemon(key string) (*Daemon, error) {
	daemon, exists := app.Daemons[key]
	if !exists {
		return nil, errors.New("GetDaemon: key not found")
	}
	return daemon, nil
}

type Daemon struct {
	Key     string        `json:"key"`
	Cmd     string        `json:"cmd"`
	mutex   *sync.RWMutex // locks Running/stop
	Running bool          `json:"running"`
	stop    chan struct{}
}

func (d *Daemon) Start() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.Running {
		return
	}
	d.Running = true
	d.stop = make(chan struct{}, 1)
	go d.runLoop()
}

func (d *Daemon) runLoop() {
	defer func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		d.Running = false
	}()
	parts, err := shlex.Split(d.Cmd)
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
		case <-d.stop:
			_ = cmd.Process.Kill()
			return
		case <-done:
			return
		}
	}
}

func (d *Daemon) Stop() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	if d.Running {
		select {
		case d.stop <- struct{}{}:
			return
		default:
			return
		}
	}
}

func main() {
	app := &App{
		mutex:   &sync.RWMutex{},
		Daemons: make(map[string]*Daemon),
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
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Println(err)
	}
}

func indexHandler(app *App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(app)
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
	daemon, err := app.GetDaemon(key)
	if err != nil {
		return
	}
	daemon.Start()
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
	daemon, err := app.GetDaemon(key)
	if err != nil {
		return
	}
	daemon.Stop()
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
	daemon, err := app.NewDaemon(cmd)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemon)
	if err != nil {
		return
	}
}
