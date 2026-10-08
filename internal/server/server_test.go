package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fentezi/mcp-google-health/internal/auth"
	"github.com/fentezi/mcp-google-health/internal/config"
	"github.com/fentezi/mcp-google-health/internal/server"
)

func newTestAuth(t *testing.T) *auth.Auth {
	t.Helper()

	healthAuth, err := auth.New(config.OAuth{
		RedirectURL: "http://localhost:8080/oauth/callback",
		TokenFile:   filepath.Join(t.TempDir(), "token.json"),
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return healthAuth
}

func TestNew_Routes(t *testing.T) {
	t.Parallel()

	healthAuth := newTestAuth(t)
	srv := server.New("localhost:8080", healthAuth)

	tests := []struct {
		name     string
		method   string
		target   string
		wantCode int
	}{
		{name: "login without key", method: http.MethodGet, target: "/login", wantCode: http.StatusForbidden},
		{name: "login with key", method: http.MethodGet, target: healthAuth.LoginURL(), wantCode: http.StatusFound},
		{name: "login via POST", method: http.MethodPost, target: "/login", wantCode: http.StatusMethodNotAllowed},
		{name: "callback without state", method: http.MethodGet, target: "/oauth/callback", wantCode: http.StatusBadRequest},
		{name: "unknown path", method: http.MethodGet, target: "/other", wantCode: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, http.NoBody))

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestNew_Addr(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0.0.0.0:9000", server.New("0.0.0.0:9000", newTestAuth(t)).Addr)
}
