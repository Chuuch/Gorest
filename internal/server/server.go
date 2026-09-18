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

	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/auth/security"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	clientusecase "github.com/chuuch/gorest/internal/client/usecase"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	commentpostgres "github.com/chuuch/gorest/internal/comments/postgres"
	commentusecase "github.com/chuuch/gorest/internal/comments/usecase"
	"github.com/chuuch/gorest/internal/config"
	"github.com/chuuch/gorest/internal/database"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	filepostgres "github.com/chuuch/gorest/internal/files/postgres"
	fileusecase "github.com/chuuch/gorest/internal/files/usecase"
	"github.com/chuuch/gorest/internal/middleware"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	orgusecase "github.com/chuuch/gorest/internal/organization/usecase"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
	projectusecase "github.com/chuuch/gorest/internal/projects/usecase"
	"github.com/chuuch/gorest/internal/storage"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	taskusecase "github.com/chuuch/gorest/internal/tasks/usecase"
	timeentryhandler "github.com/chuuch/gorest/internal/timeentries/handler"
	timeentrypostgres "github.com/chuuch/gorest/internal/timeentries/postgres"
	timeentryusecase "github.com/chuuch/gorest/internal/timeentries/usecase"
	userhandler "github.com/chuuch/gorest/internal/user/handler"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
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

	objectStore, err := storage.NewS3Store(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("initialize object storage: %w", err)
	}

	if err := objectStore.EnsureBucket(context.Background(), cfg.CORS.AllowedOrigins); err != nil {
		return nil, fmt.Errorf("ensure storage bucket: %w", err)
	}

	// -------------------------------------------------------------
	// User domain
	// -------------------------------------------------------------
	userRepository := userpostgres.NewRepository(db)
	passwordHasher := password.NewBcryptHasher(cfg.Auth.BcryptCost)

	userService := userusecase.NewService(
		userRepository,
		passwordHasher,
	)

	userHandler := userhandler.NewHandler(userService)

	// -------------------------------------------------------------
	// Auth domain
	// -------------------------------------------------------------
	refreshTokenRepository := authpostgres.NewRepository(db)
	organizationRepository := orgpostgres.NewOrganizationRepository(db)
	membershipRepository := orgpostgres.NewMembershipRepository(db)

	tokenManager := security.NewJwtManager(
		cfg.Auth.AccessTokenSecret,
		cfg.Auth.Issuer,
		cfg.Auth.AccessTokenTTL,
	)

	authService := authusecase.NewService(
		userService,
		organizationRepository,
		membershipRepository,
		refreshTokenRepository,
		tokenManager,
		passwordHasher,
		db,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	authHandler := authhandler.NewHandler(
		authService,
		cfg.Auth.RefreshTokenTTL,
		cfg.Auth.CookieSecure,
	)

	// ------------------------------------------------------------
	// Organization Domain
	// -------------------------------------------------------------
	orgService := orgusecase.NewService(
		userService,
		membershipRepository,
		db,
	)

	orgHandler := orghandler.NewHandler(orgService)

	// -------------------------------------------------------------
	// Client domain
	// -------------------------------------------------------------
	clientRepository := clientpostgres.NewRepository(db)
	clientService := clientusecase.NewService(clientRepository)
	clientHandler := clienthandler.NewHandler(clientService)

	// -------------------------------------------------------------
	// Project domain
	// -------------------------------------------------------------
	projectRepository := projectpostgres.NewRepository(db)
	projectService := projectusecase.NewService(projectRepository, clientRepository)
	projectHandler := projecthandler.NewHandler(projectService)

	// -------------------------------------------------------------
	// Task domain
	// -------------------------------------------------------------
	taskRepository := taskpostgres.NewRepository(db)
	taskService := taskusecase.NewService(taskRepository, projectRepository)
	taskHandler := taskhandler.NewHandler(taskService)

	// -------------------------------------------------------------
	// Time entry domain
	// -------------------------------------------------------------
	timeEntryRepository := timeentrypostgres.NewRepository(db)
	timeEntryService := timeentryusecase.NewService(timeEntryRepository, taskRepository)
	timeEntryHandler := timeentryhandler.NewHandler(timeEntryService)

	// -------------------------------------------------------------
	// File domain
	// -------------------------------------------------------------
	fileRepository := filepostgres.NewRepository(db)
	fileSerivce := fileusecase.NewService(fileRepository, projectRepository, objectStore)
	fileHandler := filehandler.NewHandler(fileSerivce)

	// -------------------------------------------------------------
	// Comment  domain
	// -------------------------------------------------------------
	commentRepository := commentpostgres.NewRepository(db)
	commentService := commentusecase.NewService(commentRepository, taskRepository)
	commentHandler := commenthandler.NewHandler(commentService)

	// -------------------------------------------------------------
	// HTTP Server
	// -------------------------------------------------------------
	httpServer := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: middleware.CORS(cfg.CORS.AllowedOrigins)(
			newRouter(
				userHandler,
				authHandler,
				orgHandler,
				clientHandler,
				projectHandler,
				taskHandler,
				timeEntryHandler,
				fileHandler,
				commentHandler,
				tokenManager),
		),
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
