package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/fentezi/mcp-google-health/internal/auth"
)

// New builds (but does not start) the OAuth login HTTP server, so the caller
// can control both ListenAndServe and Shutdown.
func New(addr string, a *auth.Auth) *http.Server {
	r := mux.NewRouter()

	r.HandleFunc("/login", a.LoginHandler).Methods("GET")
	r.HandleFunc(a.CallbackPath(), a.CallbackHandler).Methods("GET")

	return &http.Server{
		Addr:         addr,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}
