package routes

import (
	"github.com/wybiral/hades/server/app"
	"net/http"
)

type AppHandler func(*app.App, http.ResponseWriter, *http.Request)

// Create middleware to inject App argument
func AppMiddleware(a *app.App) func(handler AppHandler) http.HandlerFunc {
	return func(handler AppHandler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			handler(a, w, r)
		}
	}
}
