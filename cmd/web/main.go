// Command web serves the TimeTrak HTTP application.
//
// This entry point wires shared infrastructure (db pool, session store,
// templates, CSRF, logging, authz middleware) and mounts all domain handlers.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"timetrak/internal/auth"
	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/csrf"
	"timetrak/internal/shared/db"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/logging"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
	"timetrak/internal/tracking"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

type config struct {
	httpAddr      string
	databaseURL   string
	sessionSecret []byte
	cookieSecure  bool
	appEnv        string
	templatesDir  string
	staticDir     string
}

func loadConfig() (config, error) {
	c := config{
		httpAddr:     envOr("HTTP_ADDR", ":8080"),
		databaseURL:  os.Getenv("DATABASE_URL"),
		appEnv:       envOr("APP_ENV", "dev"),
		templatesDir: envOr("TEMPLATES_DIR", "web/templates"),
		staticDir:    envOr("STATIC_DIR", "web/static"),
	}
	if c.databaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return c, errors.New("SESSION_SECRET is required (>=32 bytes)")
	}
	if len(secret) < 32 {
		return c, errors.New("SESSION_SECRET must be at least 32 bytes")
	}
	c.sessionSecret = []byte(secret)
	c.cookieSecure = envOr("COOKIE_SECURE", "false") == "true"
	return c, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	logger := logging.New(cfg.appEnv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.databaseURL)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	store, err := session.NewStore(pool.Pool, cfg.sessionSecret, cfg.cookieSecure)
	if err != nil {
		logger.Error("session store", "err", err)
		os.Exit(1)
	}

	tplRoot := os.DirFS(cfg.templatesDir)
	tpls, err := templates.Load(tplRoot)
	if err != nil {
		logger.Error("templates load", "err", err)
		os.Exit(1)
	}

	authzSvc := authz.NewService(pool.Pool)
	sysClock := clock.System{}

	// Domain services.
	authSvc := auth.NewService(pool)
	limiter := auth.NewRateLimiter()
	wsSvc := workspace.NewService(pool, authzSvc, store)
	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	reportingSvc := reporting.NewService(pool, ratesSvc)
	trackingSvc := tracking.NewService(pool, sysClock)

	layoutBuilder := layout.New(pool, wsSvc)

	// Handlers.
	authHandler := auth.NewHandler(authSvc, store, tpls, limiter)
	wsHandler := workspace.NewHandler(wsSvc)
	clientsHandler := clients.NewHandler(clientsSvc, tpls, layoutBuilder)
	projectsHandler := projects.NewHandler(projectsSvc, clientsSvc, tpls, layoutBuilder)
	ratesHandler := rates.NewHandler(ratesSvc, clientsSvc, projectsSvc, tpls, layoutBuilder)
	trackingHandler := tracking.NewHandler(trackingSvc, projectsSvc, clientsSvc, reportingSvc, tpls, layoutBuilder)
	reportsHandler := reporting.NewHandler(reportingSvc, clientsSvc, projectsSvc, tpls, layoutBuilder)

	mux := http.NewServeMux()

	// Static assets (no auth).
	staticFS := http.FileServer(http.Dir(filepath.Clean(cfg.staticDir)))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

	// Healthcheck.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Root — redirect to dashboard or login depending on session state.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := session.FromContext(r.Context()); ok {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Public auth routes.
	authHandler.Register(mux)
	// Workspace switching requires auth but not an active workspace (we verify membership inside).
	wsHandler.Register(mux)

	// Protected domain routes: require session + active workspace membership.
	protect := func(next http.Handler) http.Handler {
		return authz.RequireAuth(authzSvc.RequireWorkspaceMember(next))
	}
	clientsHandler.Register(mux, protect)
	projectsHandler.Register(mux, protect)
	ratesHandler.Register(mux, protect)
	trackingHandler.Register(mux, protect)
	reportsHandler.Register(mux, protect)

	// Build middleware chain (outermost first):
	//   Recover → RequestID → Logging → SessionLoader → CSRF → routes
	handler := http.Handler(mux)
	handler = csrf.Middleware(cfg.sessionSecret, cfg.cookieSecure)(handler)
	handler = store.Loader(handler)
	handler = sharedhttp.Logging(logger)(handler)
	handler = sharedhttp.RequestID(handler)
	handler = sharedhttp.Recover(handler)

	srv := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("http server listening", "addr", cfg.httpAddr, "env", cfg.appEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
