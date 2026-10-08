package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fentezi/mcp-google-health/internal/config"
)

var optionalVars = []string{
	"LOG_LEVEL",
	"SERVER_HOST",
	"SERVER_PORT",
	"MCP_PORT",
	"MCP_BASE_PATH",
	"OAUTH_REDIRECT_URL",
	"OAUTH_SCOPES",
	"OAUTH_TOKEN_FILE",
}

// setEnv isolates the test from the developer's environment and any .env file:
// required variables get valid values, optional ones are unset, then overrides apply.
func setEnv(t *testing.T, set map[string]string, unset ...string) {
	t.Helper()

	t.Chdir(t.TempDir())
	t.Setenv("MCP_AUTH_TOKEN", "token")
	t.Setenv("OAUTH_CLIENT_ID", "id")
	t.Setenv("OAUTH_CLIENT_SECRET", "secret")
	for key, value := range set {
		t.Setenv(key, value)
	}
	for _, key := range append(optionalVars, unset...) {
		if _, ok := set[key]; ok {
			continue
		}
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(t, nil)

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, &config.Config{
		LogLevel: "info",
		Server:   config.Server{Host: "localhost", Port: 8080},
		MCP:      config.MCP{Port: 8060, AuthToken: "token", BasePath: "/mcp/health"},
		OAuth: config.OAuth{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURL:  "http://localhost:8080/oauth/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/googlehealth"},
			TokenFile:    "token.json",
		},
	}, cfg)
}

func TestLoad_Overrides(t *testing.T) {
	setEnv(t, map[string]string{
		"SERVER_PORT":      "9000",
		"MCP_BASE_PATH":    "/mcp",
		"OAUTH_SCOPES":     "a,b",
		"OAUTH_TOKEN_FILE": "/data/token.json",
	})

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "/mcp", cfg.MCP.BasePath)
	assert.Equal(t, []string{"a", "b"}, cfg.OAuth.Scopes)
	assert.Equal(t, "/data/token.json", cfg.OAuth.TokenFile)
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		set     map[string]string
		unset   []string
		wantErr string
	}{
		{name: "missing MCP_AUTH_TOKEN", unset: []string{"MCP_AUTH_TOKEN"}, wantErr: "MCP_AUTH_TOKEN"},
		{name: "empty MCP_AUTH_TOKEN", set: map[string]string{"MCP_AUTH_TOKEN": ""}, wantErr: "MCP_AUTH_TOKEN"},
		{name: "missing OAUTH_CLIENT_ID", unset: []string{"OAUTH_CLIENT_ID"}, wantErr: "OAUTH_CLIENT_ID"},
		{name: "empty OAUTH_CLIENT_ID", set: map[string]string{"OAUTH_CLIENT_ID": ""}, wantErr: "OAUTH_CLIENT_ID"},
		{name: "missing OAUTH_CLIENT_SECRET", unset: []string{"OAUTH_CLIENT_SECRET"}, wantErr: "OAUTH_CLIENT_SECRET"},
		{name: "empty OAUTH_CLIENT_SECRET", set: map[string]string{"OAUTH_CLIENT_SECRET": ""}, wantErr: "OAUTH_CLIENT_SECRET"},
		{name: "non-numeric port", set: map[string]string{"SERVER_PORT": "http"}, wantErr: "Port"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.set, tt.unset...)

			_, err := config.Load()

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestConfig_Address(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Server: config.Server{Host: "0.0.0.0", Port: 8080}}

	assert.Equal(t, "0.0.0.0:8080", cfg.Address())
}
