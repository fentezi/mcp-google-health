package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/fentezi/mcp-google-health/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// ErrNotAuthorized is returned by the client's token source when no usable token exists.
var ErrNotAuthorized = errors.New("auth: not authorized, open the login URL from the server logs")

const stateTTL = 10 * time.Minute

type Auth struct {
	oauthConfig  *oauth2.Config
	tokenFile    string
	loginKey     string
	loginURL     string
	callbackPath string
	log          *slog.Logger

	mu     sync.Mutex
	token  *oauth2.Token
	states map[string]time.Time
}

func New(cfg config.OAuth, log *slog.Logger) (*Auth, error) {
	redirect, err := url.Parse(cfg.RedirectURL)
	if err != nil || redirect.Scheme == "" || redirect.Host == "" {
		return nil, fmt.Errorf("invalid OAUTH_REDIRECT_URL %q", cfg.RedirectURL)
	}

	token, err := tokenFromFile(cfg.TokenFile)
	if errors.Is(err, os.ErrNotExist) {
		token = nil
	} else if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}

	loginKey := rand.Text()
	loginURL := url.URL{
		Scheme:   redirect.Scheme,
		Host:     redirect.Host,
		Path:     "/login",
		RawQuery: url.Values{"key": {loginKey}}.Encode(),
	}

	callbackPath := redirect.Path
	if callbackPath == "" {
		callbackPath = "/"
	}

	return &Auth{
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     endpoints.Google,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
		},
		tokenFile:    cfg.TokenFile,
		loginKey:     loginKey,
		loginURL:     loginURL.String(),
		callbackPath: callbackPath,
		log:          log,
		token:        token,
		states:       make(map[string]time.Time),
	}, nil
}

// LoginURL is the only way into the consent flow: /login rejects requests without its key.
func (a *Auth) LoginURL() string {
	return a.loginURL
}

func (a *Auth) CallbackPath() string {
	return a.callbackPath
}

// LoginHandler redirects the user to the Google consent page.
func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("key")), []byte(a.loginKey)) != 1 {
		a.log.Warn("login attempt with invalid key", "remote_addr", r.RemoteAddr)
		http.Error(w, "invalid login key", http.StatusForbidden)
		return
	}

	url := a.oauthConfig.AuthCodeURL(a.newState(), oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	a.log.Info("redirecting to google consent screen")
	http.Redirect(w, r, url, http.StatusFound)
}

// CallbackHandler must be served at CallbackPath.
func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if e := q.Get("error"); e != "" {
		a.log.Error("authorization denied", "error", e)
		http.Error(w, "authorization denied: "+e, http.StatusBadRequest)
		return
	}

	if !a.consumeState(q.Get("state")) {
		a.log.Warn("invalid or expired oauth state")
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}

	code := q.Get("code")
	if code == "" {
		a.log.Warn("oauth callback missing authorization code")
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := a.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		a.log.Error("could not exchange authorization code", "error", err)
		http.Error(w, "could not exchange authorization code", http.StatusBadGateway)
		return
	}

	if err := a.setToken(token); err != nil {
		a.log.Error("could not save token", "error", err)
		http.Error(w, "could not save token", http.StatusInternalServerError)
		return
	}

	a.log.Info("authorization completed, token saved")
	fmt.Fprintln(w, "Authorization successful, you can close this page.")
}

// Client returns an HTTP client that always uses the latest token, so a
// re-authorization takes effect without a restart.
func (a *Auth) Client(ctx context.Context) *http.Client {
	return oauth2.NewClient(ctx, tokenSource{ctx: ctx, auth: a})
}

func (a *Auth) Authorized() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.token != nil
}

func (a *Auth) setToken(token *oauth2.Token) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := saveToken(a.tokenFile, token); err != nil {
		return err
	}
	a.token = token
	return nil
}

func (a *Auth) newState() string {
	state := rand.Text()

	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for s, exp := range a.states {
		if now.After(exp) {
			delete(a.states, s)
		}
	}
	a.states[state] = now.Add(stateTTL)

	return state
}

func (a *Auth) consumeState(state string) bool {
	if state == "" {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	exp, ok := a.states[state]
	delete(a.states, state)

	return ok && time.Now().Before(exp)
}
