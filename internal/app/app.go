package app

import (
	"github.com/gorilla/mux"
	"github.com/wybiral/hades/internal/api"
	"log"
	"net/http"
)

type App struct {
	api *api.Api
}

func NewApp(dbPath string) (*App, error) {
	api, err := api.NewApi(dbPath)
	if err != nil {
		return nil, err
	}
	app := &App{
		api: api,
	}
	return app, nil
}

func (a *App) ListenAndServe(addr string) error {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", a.indexGetHandler).Methods("GET")
	r.HandleFunc("/", a.indexPostHandler).Methods("POST")
	r.HandleFunc("/{key}", a.daemonGetHandler).Methods("GET")
	r.HandleFunc("/{key}", a.daemonDeleteHandler).Methods("DELETE")
	r.HandleFunc("/{key}/start", a.daemonStartHandler)
	r.HandleFunc("/{key}/stop", a.daemonStopHandler)
	r.HandleFunc("/{key}/pause", a.daemonPauseHandler)
	r.HandleFunc("/{key}/continue", a.daemonContinueHandler)
	return http.ListenAndServe(addr, r)
}

// Respond with JSON list of daemons
func (a *App) indexGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	daemons, err := a.api.GetDaemons()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemons)
}

// Add new daemon from cmd string
func (a *App) indexPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()
	key := r.PostForm.Get("key")
	cmd := r.PostForm.Get("cmd")
	dir := r.PostForm.Get("dir")
	if len(cmd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "cmd required")
		return
	}
	daemon, err := a.api.CreateDaemon(key, cmd, dir)
	if err == api.ErrKeyNotUnique {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "key already exists")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Respond with JSON object for one daemon
func (a *App) daemonGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	daemon, err := a.api.GetDaemon(key)
	if err == api.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		jsonError(w, "not found")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Delete a daemon
func (a *App) daemonDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.api.DeleteDaemon(key)
	if err == api.ErrAlreadyStarted {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "stop daemon before deleting")
		return
	} else if err == api.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		jsonError(w, "not found")
		return
	} else if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, struct{}{})
}

// Start one daemon
func (a *App) daemonStartHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.api.StartDaemon(key)
	if err == api.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		jsonError(w, "not found")
		return
	} else if err == api.ErrAlreadyStarted {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "already started")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	daemon, err := a.api.GetDaemon(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Stop one daemon
func (a *App) daemonStopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.api.StopDaemon(key)
	if err == api.ErrNotStarted {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "not started")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	daemon, err := a.api.GetDaemon(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Pause one daemon
func (a *App) daemonPauseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.api.PauseDaemon(key)
	if err == api.ErrNotStarted {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "not started")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	daemon, err := a.api.GetDaemon(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Continue one daemon
func (a *App) daemonContinueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.api.ContinueDaemon(key)
	if err == api.ErrNotStarted {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "not started")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	daemon, err := a.api.GetDaemon(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}
