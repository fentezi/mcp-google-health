package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestAuth_validToken_NoToken(t *testing.T) {
	t.Parallel()

	a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))

	_, err := a.validToken(t.Context())

	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestAuth_validToken_ValidTokenSkipsRefresh(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "token.json")
	writeTokenFile(t, path, &oauth2.Token{AccessToken: "access", Expiry: time.Now().Add(time.Hour)})
	a := newTestAuth(t, path)
	stubTokenEndpoint(t, a, func(http.ResponseWriter, *http.Request) {
		t.Error("token endpoint must not be called for a valid token")
	})

	token, err := a.validToken(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "access", token.AccessToken)
}

func TestAuth_validToken_Refresh(t *testing.T) {
	t.Parallel()

	expired := &oauth2.Token{AccessToken: "stale", RefreshToken: "refresh", Expiry: time.Now().Add(-time.Hour)}

	tests := []struct {
		name           string
		tokenEndpoint  http.HandlerFunc
		wantErrIs      error
		wantErr        string
		wantAccess     string
		wantAuthorized bool
		wantTokenFile  bool
	}{
		{
			name:           "refreshes and persists",
			tokenEndpoint:  tokenResponse("fresh"),
			wantAccess:     "fresh",
			wantAuthorized: true,
			wantTokenFile:  true,
		},
		{
			name: "revoked refresh token drops the token",
			tokenEndpoint: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			},
			wantErrIs:      ErrNotAuthorized,
			wantAuthorized: false,
			wantTokenFile:  false,
		},
		{
			name: "transient failure keeps the token",
			tokenEndpoint: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			},
			wantErr:        "refresh token",
			wantAuthorized: true,
			wantTokenFile:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "token.json")
			writeTokenFile(t, path, expired)
			a := newTestAuth(t, path)
			stubTokenEndpoint(t, a, tt.tokenEndpoint)

			token, err := a.validToken(t.Context())

			switch {
			case tt.wantErrIs != nil:
				require.ErrorIs(t, err, tt.wantErrIs)
			case tt.wantErr != "":
				require.ErrorContains(t, err, tt.wantErr)
				require.NotErrorIs(t, err, ErrNotAuthorized)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.wantAccess, token.AccessToken)
				saved, err := tokenFromFile(path)
				require.NoError(t, err)
				assert.Equal(t, tt.wantAccess, saved.AccessToken)
			}
			assert.Equal(t, tt.wantAuthorized, a.Authorized())
			if tt.wantTokenFile {
				assert.FileExists(t, path)
			} else {
				assert.NoFileExists(t, path)
			}
		})
	}
}

func TestAuth_Client(t *testing.T) {
	t.Parallel()

	t.Run("sends bearer token", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "token.json")
		writeTokenFile(t, path, &oauth2.Token{AccessToken: "access", Expiry: time.Now().Add(time.Hour)})
		a := newTestAuth(t, path)

		var gotAuthorization string
		api := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotAuthorization = r.Header.Get("Authorization")
		}))
		t.Cleanup(api.Close)

		resp, err := a.Client(t.Context()).Get(api.URL)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, "Bearer access", gotAuthorization)
	})

	t.Run("fails before login", func(t *testing.T) {
		t.Parallel()

		a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))
		api := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("request must not reach the API without a token")
		}))
		t.Cleanup(api.Close)

		resp, err := a.Client(t.Context()).Get(api.URL)
		if err == nil {
			resp.Body.Close()
		}

		assert.ErrorIs(t, err, ErrNotAuthorized)
	})
}

func TestSaveToken(t *testing.T) {
	t.Parallel()

	t.Run("round trips through tokenFromFile", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "token.json")
		want := &oauth2.Token{
			AccessToken:  "access",
			TokenType:    "Bearer",
			RefreshToken: "refresh",
			Expiry:       time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
		}

		require.NoError(t, saveToken(path, want))
		got, err := tokenFromFile(path)

		require.NoError(t, err)
		assert.Equal(t, want.AccessToken, got.AccessToken)
		assert.Equal(t, want.TokenType, got.TokenType)
		assert.Equal(t, want.RefreshToken, got.RefreshToken)
		assert.True(t, want.Expiry.Equal(got.Expiry))
	})

	t.Run("creates missing directories", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "nested", "dir", "token.json")

		require.NoError(t, saveToken(path, &oauth2.Token{AccessToken: "access"}))

		assert.FileExists(t, path)
	})

	t.Run("is readable only by owner", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "token.json")

		require.NoError(t, saveToken(path, &oauth2.Token{AccessToken: "access"}))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("replaces existing token and leaves no temp files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "token.json")

		require.NoError(t, saveToken(path, &oauth2.Token{AccessToken: "old"}))
		require.NoError(t, saveToken(path, &oauth2.Token{AccessToken: "new"}))

		got, err := tokenFromFile(path)
		require.NoError(t, err)
		assert.Equal(t, "new", got.AccessToken)
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})
}

func TestTokenFromFile_Missing(t *testing.T) {
	t.Parallel()

	_, err := tokenFromFile(filepath.Join(t.TempDir(), "absent.json"))

	assert.ErrorIs(t, err, os.ErrNotExist)
}
