// Package app manages main hades application server.
package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/wybiral/hades/pkg/hades"
	"golang.org/x/crypto/bcrypt"
)

// App represents main application.
type App struct {
	Host      string
	Port      int
	DB        *bolt.DB
	Sessions  sessions.Store
	Hades     *hades.Hades
	Templates *template.Template
	Listener  net.Listener
	Router    *mux.Router
}

// NewApp returns a new instance of App.
func NewApp(host string, port int) (*App, error) {
	a := &App{
		Host: host,
		Port: port,
	}
	// setup DB
	db, err := newDB("hades.db")
	if err != nil {
		return nil, err
	}
	a.DB = db
	// setup Sessions
	s, err := newSessions(a)
	if err != nil {
		return nil, err
	}
	a.Sessions = s
	// setup Hades
	h, err := hades.NewHades(db)
	if err != nil {
		return nil, err
	}
	a.Hades = h
	// setup Listener
	ln, err := newListener(a)
	if err != nil {
		return nil, err
	}
	a.Listener = ln
	// setup Templates
	t, err := newTemplates("../../templates")
	a.Templates = t
	// setup Router
	r := mux.NewRouter().StrictSlash(true)
	// static file handler
	sbox := packr.NewBox("../../static")
	fsHandler := http.StripPrefix("/static/", http.FileServer(sbox))
	r.PathPrefix("/static/").Handler(fsHandler).Methods("GET")
	// application routes
	r.HandleFunc("/", a.getIndexHandler).Methods("GET")
	r.HandleFunc("/error", a.getErrorHandler).Methods("GET")
	r.HandleFunc("/login", a.getLoginHandler).Methods("GET")
	r.HandleFunc("/login", a.postLoginHandler).Methods("POST")
	r.HandleFunc("/logout", a.getLogoutHandler).Methods("POST")
	r.HandleFunc("/add", a.getAddHandler).Methods("GET")
	r.HandleFunc("/add", a.postAddHandler).Methods("POST")
	r.HandleFunc("/{id}/action", a.postActionHandler).Methods("POST")
	a.Router = r
	return a, nil
}

// Run imports the library and starts server.
func (a *App) Run() error {
	return http.Serve(a.Listener, a.Router)
}

// index page handler
func (a *App) getIndexHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	token, err := a.getUserToken(s)
	if err != nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	daemons, err := a.Hades.Daemons()
	if err != nil {
		return
	}
	flashes := a.getFlashes(s)
	s.Save(r, w)
	a.Templates.ExecuteTemplate(w, "index.html", struct {
		Token   string
		Errors  []string
		Daemons []*hades.Daemon
	}{
		Token:   token,
		Errors:  flashes,
		Daemons: daemons,
	})
}

// error page handler
func (a *App) getErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	a.Templates.ExecuteTemplate(w, "error.html", nil)
}

// login page handler
func (a *App) getLoginHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	_, err := a.getUserToken(s)
	if err == nil {
		// no error getting token means they're logged in
		http.Redirect(w, r, "/", 302)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	flashes := a.getFlashes(s)
	s.Save(r, w)
	a.Templates.ExecuteTemplate(w, "login.html", struct {
		Errors []string
	}{
		Errors: flashes,
	})
}

// login post handler
func (a *App) postLoginHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	err := r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	var hash []byte
	err = a.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("settings"))
		h := b.Get([]byte("password-hash"))
		if h == nil {
			return fmt.Errorf("missing password hash")
		}
		hash = make([]byte, len(h))
		copy(hash, h)
		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	password := r.PostForm.Get("password")
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		s.AddFlash("invalid login")
		s.Save(r, w)
		http.Redirect(w, r, "/login", 302)
		return
	}
	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	s.Values["user"] = token
	s.Save(r, w)
	http.Redirect(w, r, "/", 302)
}

// logout post handler
func (a *App) getLogoutHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	token, err := a.getUserToken(s)
	if err != nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	err = r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	formtoken := r.PostForm.Get("token")
	if formtoken != token {
		http.Redirect(w, r, "/error", 302)
		return
	}
	s.Options.MaxAge = -1
	s.Save(r, w)
	http.Redirect(w, r, "/login", 302)
}

// add page handler
func (a *App) getAddHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	token, err := a.getUserToken(s)
	if err != nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	a.Templates.ExecuteTemplate(w, "add.html", struct {
		Token string
	}{
		Token: token,
	})
}

// add post handler
func (a *App) postAddHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	token, err := a.getUserToken(s)
	if err != nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	err = r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	formtoken := r.PostForm.Get("token")
	if formtoken != token {
		http.Redirect(w, r, "/error", 302)
		return
	}
	cmd := r.PostForm.Get("cmd")
	dir := r.PostForm.Get("dir")
	_, err = a.Hades.Add(cmd, dir)
	if err != nil {
		s.AddFlash("error adding daemon")
		s.Save(r, w)
	}
	http.Redirect(w, r, "/", 302)
}

// action post handler
func (a *App) postActionHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := a.Sessions.Get(r, "session")
	token, err := a.getUserToken(s)
	if err != nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	err = r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	formtoken := r.PostForm.Get("token")
	action := r.PostForm.Get("action")
	if formtoken != token {
		http.Redirect(w, r, "/error", 302)
		return
	}
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/error", 302)
		return
	}
	switch action {
	case "start":
		err = a.Hades.Start(id)
	case "resume":
		err = a.Hades.Resume(id)
	case "pause":
		err = a.Hades.Pause(id)
	case "stop":
		err = a.Hades.Stop(id)
	case "remove":
		err = a.Hades.Remove(id)
	}
	if err == hades.ErrAlreadyStarted {
		s.AddFlash("daemon already running")
		s.Save(r, w)
	} else if err == hades.ErrNotStarted {
		s.AddFlash("daemon not running")
		s.Save(r, w)
	} else if err != nil {
		s.AddFlash("action failed")
		s.Save(r, w)
	}
	http.Redirect(w, r, "/", 302)
}

// getFlashes returns all flash messages attached to session
// (does not clear them)
func (a *App) getFlashes(s *sessions.Session) []string {
	flashes := make([]string, 0)
	for _, x := range s.Flashes() {
		flashes = append(flashes, x.(string))
	}
	return flashes
}

// getUserToken returns the current session token (or error if not logged in)
func (a *App) getUserToken(s *sessions.Session) (string, error) {
	rawToken, ok := s.Values["user"]
	if !ok {
		return "", fmt.Errorf("no token found")
	}
	token, ok := rawToken.(string)
	if !ok {
		return "", fmt.Errorf("invalid token")
	}
	return token, nil
}
