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

	activityhandler "github.com/chuuch/gorest/internal/activity/handler"
	activitypostgres "github.com/chuuch/gorest/internal/activity/postgres"
	activityusecase "github.com/chuuch/gorest/internal/activity/usecase"
	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/auth/security"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	clientusecase "github.com/chuuch/gorest/internal/client/usecase"
	clientuserhandler "github.com/chuuch/gorest/internal/clientusers/handler"
	clientuserpostgres "github.com/chuuch/gorest/internal/clientusers/postgres"
	clientuserusecase "github.com/chuuch/gorest/internal/clientusers/usecase"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	commentpostgres "github.com/chuuch/gorest/internal/comments/postgres"
	commentusecase "github.com/chuuch/gorest/internal/comments/usecase"
	"github.com/chuuch/gorest/internal/config"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/events"
	eventshandler "github.com/chuuch/gorest/internal/events/handler"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	filepostgres "github.com/chuuch/gorest/internal/files/postgres"
	fileusecase "github.com/chuuch/gorest/internal/files/usecase"
	"github.com/chuuch/gorest/internal/invites"
	invoicehandler "github.com/chuuch/gorest/internal/invoices/handler"
	invoicepostgres "github.com/chuuch/gorest/internal/invoices/postgres"
	invoiceusecase "github.com/chuuch/gorest/internal/invoices/usecase"
	"github.com/chuuch/gorest/internal/mailer"
	"github.com/chuuch/gorest/internal/middleware"
	notificationhandler "github.com/chuuch/gorest/internal/notifications/handler"
	notificationpostgres "github.com/chuuch/gorest/internal/notifications/postgres"
	notificationsusecase "github.com/chuuch/gorest/internal/notifications/usecase"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	orgusecase "github.com/chuuch/gorest/internal/organization/usecase"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
	projectusecase "github.com/chuuch/gorest/internal/projects/usecase"
	reporthandler "github.com/chuuch/gorest/internal/reports/handler"
	reportspostgres "github.com/chuuch/gorest/internal/reports/postgres"
	reportsusecase "github.com/chuuch/gorest/internal/reports/usecase"
	"github.com/chuuch/gorest/internal/storage"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	taskusecase "github.com/chuuch/gorest/internal/tasks/usecase"
	ticketcommenthandler "github.com/chuuch/gorest/internal/ticketcomments/handler"
	ticketcommentpostgres "github.com/chuuch/gorest/internal/ticketcomments/postgres"
	ticketcommentusecase "github.com/chuuch/gorest/internal/ticketcomments/usecase"
	ticketfilehandler "github.com/chuuch/gorest/internal/ticketfiles/handler"
	ticketfilepostgres "github.com/chuuch/gorest/internal/ticketfiles/postgres"
	ticketfileusecase "github.com/chuuch/gorest/internal/ticketfiles/usecase"
	tickethandler "github.com/chuuch/gorest/internal/tickets/handler"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	ticketusecase "github.com/chuuch/gorest/internal/tickets/usecase"
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
	mailer     mailer.Mailer
}

func New(cfg *config.Config) (*Server, error) {
	// -----------------------------
	// Open DB connection pool
	// -----------------------------
	db, err := database.NewPostgresPool(context.Background(), cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	// --------------------------------
	// Initialize S3 storage & bucket
	// ---------------------------------
	objectStore, err := storage.NewS3Store(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("initialize object storage: %w", err)
	}

	if err := objectStore.EnsureBucket(context.Background(), cfg.CORS.AllowedOrigins); err != nil {
		return nil, fmt.Errorf("ensure storage bucket: %w", err)
	}

	//-------------------------------------
	// Initialize mailer client
	// ------------------------------------
	mailSender, err := mailer.New(cfg.Mailer)
	if err != nil {
		return nil, fmt.Errorf("initialize mailer: %w", err)
	}
	slog.Info("mailer ready", "driver", cfg.Mailer.Driver)

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

	inviteService := invites.NewService(
		invites.NewRepository(db),
		userService,
		tokenManager,
		mailSender,
		refreshTokenRepository,
		db,
		cfg.Auth.InviteTTL,
		cfg.Auth.ResetTTL,
		cfg.App.PublicURL,
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
		inviteService,
	)

	// ------------------------------------------------------------
	// Organization Domain
	// -------------------------------------------------------------
	orgService := orgusecase.NewService(
		userService,
		membershipRepository,
		organizationRepository,
		inviteService,
		db,
	)

	orgHandler := orghandler.NewHandler(orgService)

	// -------------------------------------------------------------
	// Activity domain
	// -------------------------------------------------------------
	activityRepository := activitypostgres.NewRepository(db)

	eventHub := events.NewHub()
	eventsHandler := eventshandler.NewHandler(eventHub)

	activityService := activityusecase.EnableRealtime(
		activityusecase.NewService(activityRepository),
		eventHub,
	)
	activityHandler := activityhandler.NewHandler(activityService)

	// -------------------------------------------------------------
	// Reports domain
	// -------------------------------------------------------------
	reportRepository := reportspostgres.NewRepository(db)
	reportService := reportsusecase.NewService(reportRepository)
	reportHandler := reporthandler.NewHandler(reportService)

	// -------------------------------------------------------------
	// Notifications domain
	// -------------------------------------------------------------
	notificationRepository := notificationpostgres.NewRepository(db)
	notificationService := notificationsusecase.EnableRealtime(
		notificationsusecase.NewService(
			notificationRepository,
			membershipRepository,
		),
		eventHub,
	)
	notificationHandler := notificationhandler.NewHandler(notificationService)

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
	// Ticket domain
	// -------------------------------------------------------------
	ticketRepository := ticketpostgres.NewRepository(db)
	ticketService := ticketusecase.EnableNotifications(
		ticketusecase.NewService(ticketRepository, clientRepository),
		notificationService,
	)
	ticketHandler := tickethandler.NewHandler(ticketService).WithActivity(activityService)

	// -------------------------------------------------------------
	// Task domain
	// -------------------------------------------------------------
	taskRepository := taskpostgres.NewRepository(db)
	taskService := taskusecase.EnableNotifications(
		taskusecase.NewService(taskRepository, projectRepository, ticketRepository, membershipRepository),
		notificationService,
	)
	taskHandler := taskhandler.NewHandler(taskService).WithActivity(activityService)

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
	fileHandler := filehandler.NewHandler(fileSerivce).WithActivity(activityService)

	// -------------------------------------------------------------
	// Comment  domain
	// -------------------------------------------------------------
	commentRepository := commentpostgres.NewRepository(db)
	commentService := commentusecase.EnableNotifications(
		commentusecase.NewService(commentRepository, taskRepository),
		notificationService,
	)
	commentHandler := commenthandler.NewHandler(commentService).WithActivity(activityService)

	// -------------------------------------------------------------
	// Client User domain
	// -------------------------------------------------------------
	clientUserRepository := clientuserpostgres.NewRepository(db)
	clientUserService := clientuserusecase.NewService(
		userService,
		clientRepository,
		clientUserRepository,
		membershipRepository,
		organizationRepository,
		refreshTokenRepository,
		tokenManager,
		passwordHasher,
		inviteService,
		db,
		cfg.Auth.RefreshTokenTTL,
	)
	clientUserHandler := clientuserhandler.NewHandler(
		clientUserService,
		cfg.Auth.RefreshTokenTTL,
		cfg.Auth.CookieSecure,
	)

	// -------------------------------------------------------------
	// Ticket file domain
	// -------------------------------------------------------------
	ticketFileRepository := ticketfilepostgres.NewRepository(db)
	ticketFileService := ticketfileusecase.NewService(ticketFileRepository, ticketRepository, objectStore)
	ticketFileHandler := ticketfilehandler.NewHandler(ticketFileService).WithActivity(activityService)

	// -------------------------------------------------------------
	// Ticket comment domain
	// -------------------------------------------------------------
	ticketCommentRepository := ticketcommentpostgres.NewRepository(db)
	ticketCommentService := ticketcommentusecase.EnableNotifications(
		ticketcommentusecase.NewService(ticketCommentRepository, ticketRepository),
		notificationService,
	)
	ticketCommentHandler := ticketcommenthandler.NewHandler(ticketCommentService).WithActivity(activityService)

	// -------------------------------------------------------------
	// Invoices domain
	// -------------------------------------------------------------
	invoiceRepository := invoicepostgres.NewRepository(db)
	invoiceService := invoiceusecase.NewService(
		invoiceRepository,
		clientRepository,
		organizationRepository,
		clientUserRepository,
		userRepository,
		mailSender,
		db,
		cfg.App.PublicURL,
	)
	invoiceHandler := invoicehandler.NewHandler(invoiceService)

	// -------------------------------------------------------------
	// HTTP Server
	// -------------------------------------------------------------
	router := newRouter(
		userHandler,
		authHandler,
		orgHandler,
		clientHandler,
		projectHandler,
		taskHandler,
		timeEntryHandler,
		fileHandler,
		commentHandler,
		clientUserHandler,
		ticketHandler,
		ticketFileHandler,
		ticketCommentHandler,
		activityHandler,
		reportHandler,
		notificationHandler,
		invoiceHandler,
		eventsHandler,
		tokenManager,
	)

	httpServer := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: middleware.CORS(cfg.CORS.AllowedOrigins)(
			middleware.RequestLog(
				middleware.Metrics(router),
			),
		),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		db:         db,
		mailer:     mailSender,
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
