package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wybiral/hades/internal/server/api"
)

// App represents main server application.
type App struct {
	api *api.API
}

// NewApp returns new App from dbPath.
func NewApp(dbPath string) (*App, error) {
	api, err := api.NewAPI(dbPath)
	if err != nil {
		return nil, err
	}
	app := &App{
		api: api,
	}
	return app, nil
}

// ListenAndServe starts HTTP listener for App at addr.
func (a *App) ListenAndServe(addr string) error {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", a.indexGetHandler).Methods("GET")
	r.HandleFunc("/", a.indexPostHandler).Methods("POST")
	r.HandleFunc("/{key}", a.daemonGetHandler).Methods("GET")
	r.HandleFunc("/{key}", a.daemonDeleteHandler).Methods("DELETE")
	r.HandleFunc("/{key}/start", a.daemonStartHandler).Methods("PUT")
	r.HandleFunc("/{key}/stop", a.daemonStopHandler).Methods("PUT")
	r.HandleFunc("/{key}/pause", a.daemonPauseHandler).Methods("PUT")
	r.HandleFunc("/{key}/continue", a.daemonContinueHandler).Methods("PUT")
	return http.ListenAndServe(addr, r)
}

// indexGetHandler responds with JSON list of daemons.
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

// indexPostHandler adds new daemon from key, cmd, dir.
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

// daemonGetHandler responds with JSON object for one daemon.
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

// daemonDeleteHandler deletes a daemon.
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
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, struct{}{})
}

// daemonStartHandler starts one daemon.
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

// daemonStopHandler stops one daemon.
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

// daemonPauseHandler pauses one daemon.
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

// daemonContinueHandler continues one paused daemon.
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
