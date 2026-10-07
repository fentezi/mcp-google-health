package mcpserver

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/health/v4"
)

type PingInput struct{}

type PingOutput struct {
	Message string `json:"message" jsonschema:"a friendly pong reply"`
}

// New builds the MCP server and registers its tools against the given,
// already-authorized Google Health client.
func New(healthClient *health.Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "googlehealth-mcp", Version: "v0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Health check tool that replies with pong.",
	}, ping)

	registerDataTypeTools(server, healthClient)

	return server
}

func ping(_ context.Context, _ *mcp.CallToolRequest, _ PingInput) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{Message: "pong"}, nil
}

// Handler serves the given MCP server over the Streamable HTTP transport.
// Localhost protection is off: RequireBearerToken guards it, and Docker proxies send non-localhost Host headers.
func Handler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{DisableLocalhostProtection: true})
}

// RequireBearerToken rejects requests whose Authorization header doesn't carry
// the expected bearer token, before they reach the wrapped handler.
func RequireBearerToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
