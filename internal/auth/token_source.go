package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

type tokenSource struct {
	ctx  context.Context
	auth *Auth
}

func (s tokenSource) Token() (*oauth2.Token, error) {
	return s.auth.validToken(s.ctx)
}

// validToken refreshes and persists the token when it expires. A rejected
// refresh token drops the stored token so the user can log in again.
func (a *Auth) validToken(ctx context.Context) (*oauth2.Token, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token == nil {
		return nil, ErrNotAuthorized
	}
	if a.token.Valid() {
		return a.token, nil
	}

	token, err := a.oauthConfig.TokenSource(ctx, a.token).Token()
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant" {
		a.log.Warn("refresh token rejected, re-authorization required", "login_url", a.loginURL)
		a.token = nil
		if err := os.Remove(a.tokenFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			a.log.Error("could not remove rejected token file", "error", err)
		}
		return nil, ErrNotAuthorized
	}
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	if err := saveToken(a.tokenFile, token); err != nil {
		return nil, fmt.Errorf("persist refreshed token: %w", err)
	}
	a.token = token
	return token, nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := new(oauth2.Token)
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	return token, nil
}

// saveToken writes the token atomically so a crash never leaves a truncated file.
func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}

	f, err := os.CreateTemp(dir, ".token-*.json")
	if err != nil {
		return fmt.Errorf("create temp token file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if err := json.NewEncoder(f).Encode(token); err != nil {
		f.Close()
		return fmt.Errorf("encode token: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp token file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace token file: %w", err)
	}
	return nil
}
