package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/health/v4"
	"google.golang.org/api/option"

	"github.com/fentezi/mcp-google-health/internal/config"
)

func newTestApp() *App {
	return New(config.Config{
		Server: config.Server{Host: "127.0.0.1"},
		MCP:    config.MCP{Port: 8060, AuthToken: "secret", BasePath: "/mcp/health"},
	}, slog.New(slog.DiscardHandler))
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

func TestApp_serve_ShutdownEndsOpenStreams(t *testing.T) {
	t.Parallel()

	addr := freeAddr(t)
	streamOpen := make(chan struct{})
	stream := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(streamOpen)
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- newTestApp().serve(ctx, "test", &http.Server{Addr: addr, Handler: stream, ReadHeaderTimeout: time.Second})
	}()

	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "server did not start listening")

	go func() {
		resp, err := http.Get("http://" + addr)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-streamOpen

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not return after context cancellation")
	}
}

func TestApp_serve_ListenError(t *testing.T) {
	t.Parallel()

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = occupied.Close() })

	err = newTestApp().serve(t.Context(), "test", &http.Server{Addr: occupied.Addr().String(), ReadHeaderTimeout: time.Second})

	assert.ErrorContains(t, err, "test server")
}

func TestApp_newMCPServer(t *testing.T) {
	t.Parallel()

	healthClient, err := health.NewService(t.Context(), option.WithoutAuthentication())
	require.NoError(t, err)

	srv := newTestApp().newMCPServer(healthClient)

	assert.Equal(t, "127.0.0.1:8060", srv.Addr)

	tests := []struct {
		name          string
		method        string
		target        string
		authorization string
		wantCode      int
	}{
		{name: "without bearer token", method: http.MethodPost, target: "/mcp/health", wantCode: http.StatusUnauthorized},
		{name: "unsupported method", method: http.MethodPut, target: "/mcp/health", authorization: "Bearer secret", wantCode: http.StatusMethodNotAllowed},
		{name: "unknown path", method: http.MethodPost, target: "/other", authorization: "Bearer secret", wantCode: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.target, http.NoBody)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			srv.Handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}
