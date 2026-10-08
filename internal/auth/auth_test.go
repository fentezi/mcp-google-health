package auth

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fentezi/mcp-google-health/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const testRedirectURL = "http://localhost:8080/oauth/callback"

func newTestAuth(t *testing.T, tokenFile string) *Auth {
	t.Helper()

	a, err := New(config.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  testRedirectURL,
		Scopes:       []string{"scope"},
		TokenFile:    tokenFile,
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	return a
}

// stubTokenEndpoint points the OAuth token exchange and refresh at handler.
func stubTokenEndpoint(t *testing.T, a *Auth, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	a.oauthConfig.Endpoint = oauth2.Endpoint{
		AuthURL:   srv.URL + "/auth",
		TokenURL:  srv.URL + "/token",
		AuthStyle: oauth2.AuthStyleInParams,
	}
}

func tokenResponse(accessToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","token_type":"Bearer","expires_in":3600,"refresh_token":"refresh"}`))
	}
}

func writeTokenFile(t *testing.T, path string, token *oauth2.Token) {
	t.Helper()
	require.NoError(t, saveToken(path, token))
}

func TestNew_InvalidRedirectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redirectURL string
	}{
		{name: "empty", redirectURL: ""},
		{name: "path only", redirectURL: "/oauth/callback"},
		{name: "no scheme", redirectURL: "localhost"},
		{name: "unparsable", redirectURL: "http://[::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(config.OAuth{RedirectURL: tt.redirectURL}, slog.New(slog.DiscardHandler))

			assert.ErrorContains(t, err, "invalid OAUTH_REDIRECT_URL")
		})
	}
}

func TestNew_CallbackPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redirectURL string
		want        string
	}{
		{name: "explicit path", redirectURL: "http://localhost:8080/oauth/callback", want: "/oauth/callback"},
		{name: "no path", redirectURL: "http://localhost:8080", want: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := New(config.OAuth{
				RedirectURL: tt.redirectURL,
				TokenFile:   filepath.Join(t.TempDir(), "token.json"),
			}, slog.New(slog.DiscardHandler))
			require.NoError(t, err)

			assert.Equal(t, tt.want, a.CallbackPath())
		})
	}
}

func TestNew_LoginURL(t *testing.T) {
	t.Parallel()

	a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))

	u, err := url.Parse(a.LoginURL())
	require.NoError(t, err)
	assert.Equal(t, "http", u.Scheme)
	assert.Equal(t, "localhost:8080", u.Host)
	assert.Equal(t, "/login", u.Path)
	assert.NotEmpty(t, u.Query().Get("key"))
}

func TestNew_LoginKeyIsRandom(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := newTestAuth(t, filepath.Join(dir, "token.json"))
	second := newTestAuth(t, filepath.Join(dir, "token.json"))

	assert.NotEqual(t, first.LoginURL(), second.LoginURL())
}

func TestNew_TokenFile(t *testing.T) {
	t.Parallel()

	t.Run("missing file starts unauthorized", func(t *testing.T) {
		t.Parallel()

		a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))

		assert.False(t, a.Authorized())
	})

	t.Run("existing file starts authorized", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "token.json")
		writeTokenFile(t, path, &oauth2.Token{AccessToken: "access"})

		a := newTestAuth(t, path)

		assert.True(t, a.Authorized())
	})

	t.Run("corrupt file fails", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "token.json")
		require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

		_, err := New(config.OAuth{RedirectURL: testRedirectURL, TokenFile: path}, slog.New(slog.DiscardHandler))

		assert.ErrorContains(t, err, "read token file")
	})
}

func TestAuth_LoginHandler_RejectsInvalidKey(t *testing.T) {
	t.Parallel()

	a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))

	tests := []struct {
		name   string
		target string
	}{
		{name: "missing key", target: "/login"},
		{name: "empty key", target: "/login?key="},
		{name: "wrong key", target: "/login?key=wrong"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			a.LoginHandler(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

func TestAuth_LoginHandler_RedirectsToConsent(t *testing.T) {
	t.Parallel()

	a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))

	rec := httptest.NewRecorder()
	a.LoginHandler(rec, httptest.NewRequest(http.MethodGet, a.LoginURL(), nil))

	require.Equal(t, http.StatusFound, rec.Code)
	location, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "accounts.google.com", location.Host)

	q := location.Query()
	assert.Equal(t, "client-id", q.Get("client_id"))
	assert.Equal(t, testRedirectURL, q.Get("redirect_uri"))
	assert.Equal(t, "offline", q.Get("access_type"))
	assert.Equal(t, "consent", q.Get("prompt"))
	assert.True(t, a.consumeState(q.Get("state")), "state in the redirect must be accepted by the callback")
}

func TestAuth_CallbackHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tokenEndpoint http.HandlerFunc
		breakTokenDir bool
		query         func(state string) url.Values
		wantCode      int
		wantAuthorize bool
	}{
		{
			name:     "authorization denied",
			query:    func(state string) url.Values { return url.Values{"error": {"access_denied"}, "state": {state}} },
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing state",
			query:    func(string) url.Values { return url.Values{"code": {"code"}} },
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "unknown state",
			query:    func(string) url.Values { return url.Values{"code": {"code"}, "state": {"forged"}} },
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing code",
			query:    func(state string) url.Values { return url.Values{"state": {state}} },
			wantCode: http.StatusBadRequest,
		},
		{
			name: "exchange fails",
			tokenEndpoint: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			query:    func(state string) url.Values { return url.Values{"code": {"code"}, "state": {state}} },
			wantCode: http.StatusBadGateway,
		},
		{
			name:          "token cannot be saved",
			tokenEndpoint: tokenResponse("access"),
			breakTokenDir: true,
			query:         func(state string) url.Values { return url.Values{"code": {"code"}, "state": {state}} },
			wantCode:      http.StatusInternalServerError,
		},
		{
			name:          "success",
			tokenEndpoint: tokenResponse("access"),
			query:         func(state string) url.Values { return url.Values{"code": {"code"}, "state": {state}} },
			wantCode:      http.StatusOK,
			wantAuthorize: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))
			if tt.tokenEndpoint != nil {
				stubTokenEndpoint(t, a, tt.tokenEndpoint)
			}
			if tt.breakTokenDir {
				notADir := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(notADir, nil, 0o600))
				a.tokenFile = filepath.Join(notADir, "token.json")
			}

			rec := httptest.NewRecorder()
			target := a.CallbackPath() + "?" + tt.query(a.newState()).Encode()
			a.CallbackHandler(rec, httptest.NewRequest(http.MethodGet, target, nil))

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantAuthorize, a.Authorized())
		})
	}
}

func TestAuth_CallbackHandler_PersistsToken(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "token.json")
	a := newTestAuth(t, path)
	stubTokenEndpoint(t, a, tokenResponse("fresh-access"))

	rec := httptest.NewRecorder()
	query := url.Values{"code": {"code"}, "state": {a.newState()}}
	a.CallbackHandler(rec, httptest.NewRequest(http.MethodGet, a.CallbackPath()+"?"+query.Encode(), nil))
	require.Equal(t, http.StatusOK, rec.Code)

	saved, err := tokenFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "fresh-access", saved.AccessToken)
	assert.Equal(t, "refresh", saved.RefreshToken)
}

func TestAuth_CallbackHandler_StateIsSingleUse(t *testing.T) {
	t.Parallel()

	a := newTestAuth(t, filepath.Join(t.TempDir(), "token.json"))
	stubTokenEndpoint(t, a, tokenResponse("access"))
	target := a.CallbackPath() + "?" + url.Values{"code": {"code"}, "state": {a.newState()}}.Encode()

	first := httptest.NewRecorder()
	a.CallbackHandler(first, httptest.NewRequest(http.MethodGet, target, nil))
	replay := httptest.NewRecorder()
	a.CallbackHandler(replay, httptest.NewRequest(http.MethodGet, target, nil))

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusBadRequest, replay.Code)
}

func TestAuth_State_Expiry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		wait time.Duration
		want bool
	}{
		{name: "just before TTL", wait: stateTTL - time.Nanosecond, want: true},
		{name: "at TTL", wait: stateTTL, want: false},
		{name: "after TTL", wait: stateTTL + time.Minute, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				a := &Auth{states: make(map[string]time.Time)}
				state := a.newState()

				time.Sleep(tt.wait)

				assert.Equal(t, tt.want, a.consumeState(state))
			})
		})
	}
}
