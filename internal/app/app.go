package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/fentezi/mcp-google-health/internal/auth"
	"github.com/fentezi/mcp-google-health/internal/config"
	"github.com/fentezi/mcp-google-health/internal/mcpserver"
	"github.com/fentezi/mcp-google-health/internal/server"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/health/v4"
	"google.golang.org/api/option"
)

type App struct {
	cfg config.Config
	log *slog.Logger
}

func New(cfg config.Config, log *slog.Logger) *App {
	return &App{cfg: cfg, log: log}
}

// Run serves the login and MCP servers until ctx is canceled. MCP tools fail
// with auth.ErrNotAuthorized until the login flow completes.
func (a *App) Run(ctx context.Context) error {
	healthAuth, err := auth.New(a.cfg.OAuth, a.log)
	if err != nil {
		return err
	}

	if !healthAuth.Authorized() {
		a.log.Info("not authorized, open the login URL to grant access", "login_url", healthAuth.LoginURL())
	}

	healthClient, err := health.NewService(ctx, option.WithHTTPClient(healthAuth.Client(ctx)))
	if err != nil {
		return fmt.Errorf("create health client: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return a.serve(gctx, "login", server.New(a.cfg.Address(), healthAuth)) })
	g.Go(func() error { return a.serve(gctx, "mcp", a.newMCPServer(healthClient)) })
	return g.Wait()
}

func (a *App) newMCPServer(healthClient *health.Service) *http.Server {
	mcpHandler := mcpserver.Handler(mcpserver.New(healthClient))

	r := mux.NewRouter()
	r.Handle(a.cfg.MCP.BasePath, mcpserver.RequireBearerToken(a.cfg.MCP.AuthToken)(mcpHandler)).
		Methods(http.MethodGet, http.MethodPost, http.MethodDelete)

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.MCP.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
}

// serve runs srv until ctx is canceled, then shuts it down gracefully.
func (a *App) serve(ctx context.Context, name string, srv *http.Server) error {
	// Request contexts derive from ctx so SSE streams end on shutdown instead of blocking it.
	srv.BaseContext = func(net.Listener) context.Context { return ctx }

	serveErr := make(chan error, 1)
	go func() {
		a.log.Info("server listening", "server", name, "addr", srv.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("%s server: %w", name, err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("%s server shutdown: %w", name, err)
	}
	a.log.Info("server stopped", "server", name)
	return nil
}
