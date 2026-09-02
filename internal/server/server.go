package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chuuch/gorest/internal/auth"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/config"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/user"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	httpServer *http.Server
	db         *pgxpool.Pool
}

func New(cfg *config.Config) (*Server, error) {
	db, err := database.NewPostgresPool(context.Background(), cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	// -------------------------------------------------------------
	// User domain
	// -------------------------------------------------------------
	userRepository := userpostgres.NewRepository(db)
	passwordHasher := password.NewBcryptHasher(cfg.Auth.BcryptCost)
	userService := user.NewService(userRepository, passwordHasher)
	userHandler := user.NewHandler(userService)

	// -------------------------------------------------------------
	// Auth domain
	// -------------------------------------------------------------
	refreshTokenRepository := authpostgres.NewRepository(db)

	tokenManager := auth.NewJwtManager(
		cfg.Auth.AccessTokenSecret,
		cfg.Auth.Issuer,
		cfg.Auth.AccessTokenTTL,
	)

	authService := auth.NewService(
		userService,
		refreshTokenRepository,
		tokenManager,
		passwordHasher,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	authHandler := auth.NewHandler(authService)

	// ------------------------------------------------------------
	// HTTP Server
	// ------------------------------------------------------------
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      newRouter(userHandler, authHandler),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		db:         db,
	}, nil
}

func (s *Server) Start() error {
	slog.Info("starting HTTP server", "address", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("HTTP server: %w", err)
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error(
			"HTTP server shutdown failed",
			"error", err,
		)

		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	s.db.Close()
	slog.Info("server shutdown complete")

	return nil
}

func (s *Server) Run(cfg config.Config) error {
	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- s.Start()
	}()

	select {
	case err := <-serverErr:
		return err

	case <-shutdownCtx.Done():
		slog.Info(
			"shutdown signal received",
			"signal", shutdownCtx.Err(),
		)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			cfg.Server.ShutdownTimeout,
		)
		defer cancel()

		return s.Shutdown(ctx)
	}
}
