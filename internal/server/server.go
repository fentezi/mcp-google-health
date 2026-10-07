package server

import (
	"net/http"
	"time"

	"github.com/fentezi/mcp-google-health/internal/auth"
	"github.com/gorilla/mux"
)

// New builds (but does not start) the OAuth login HTTP server, so the caller
// can control both ListenAndServe and Shutdown.
func New(addr string, auth *auth.Auth) *http.Server {
	r := mux.NewRouter()

	r.HandleFunc("/login", auth.LoginHandler).Methods("GET")
	r.HandleFunc(auth.CallbackPath(), auth.CallbackHandler).Methods("GET")

	return &http.Server{
		Addr:         addr,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}
