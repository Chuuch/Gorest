package server

import (
	activityhandler "github.com/chuuch/gorest/internal/activity/handler"
	activitypostgres "github.com/chuuch/gorest/internal/activity/postgres"
	activityusecase "github.com/chuuch/gorest/internal/activity/usecase"
	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	"github.com/chuuch/gorest/internal/auth/invites"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/auth/security"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
	clienthandler "github.com/chuuch/gorest/internal/clients/handler"
	clientpostgres "github.com/chuuch/gorest/internal/clients/postgres"
	clientusecase "github.com/chuuch/gorest/internal/clients/usecase"
	clientuserhandler "github.com/chuuch/gorest/internal/clients/users/handler"
	clientuserpostgres "github.com/chuuch/gorest/internal/clients/users/postgres"
	clientuserusecase "github.com/chuuch/gorest/internal/clients/users/usecase"
	invoicehandler "github.com/chuuch/gorest/internal/invoices/handler"
	invoicepostgres "github.com/chuuch/gorest/internal/invoices/postgres"
	invoiceusecase "github.com/chuuch/gorest/internal/invoices/usecase"
	navhandler "github.com/chuuch/gorest/internal/nav/handler"
	navpostgres "github.com/chuuch/gorest/internal/nav/postgres"
	navusecase "github.com/chuuch/gorest/internal/nav/usecase"
	notificationhandler "github.com/chuuch/gorest/internal/notifications/handler"
	notificationpostgres "github.com/chuuch/gorest/internal/notifications/postgres"
	notificationsusecase "github.com/chuuch/gorest/internal/notifications/usecase"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	orgusecase "github.com/chuuch/gorest/internal/organization/usecase"
	"github.com/chuuch/gorest/internal/platform/config"
	"github.com/chuuch/gorest/internal/platform/events"
	eventshandler "github.com/chuuch/gorest/internal/platform/events/handler"
	"github.com/chuuch/gorest/internal/platform/idempotency"
	"github.com/chuuch/gorest/internal/platform/mailer"
	"github.com/chuuch/gorest/internal/platform/storage"
	filehandler "github.com/chuuch/gorest/internal/projects/files/handler"
	filepostgres "github.com/chuuch/gorest/internal/projects/files/postgres"
	fileusecase "github.com/chuuch/gorest/internal/projects/files/usecase"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
	projectusecase "github.com/chuuch/gorest/internal/projects/usecase"
	reporthandler "github.com/chuuch/gorest/internal/reports/handler"
	reportspostgres "github.com/chuuch/gorest/internal/reports/postgres"
	reportsusecase "github.com/chuuch/gorest/internal/reports/usecase"
	commenthandler "github.com/chuuch/gorest/internal/tasks/comments/handler"
	commentpostgres "github.com/chuuch/gorest/internal/tasks/comments/postgres"
	commentusecase "github.com/chuuch/gorest/internal/tasks/comments/usecase"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	taskusecase "github.com/chuuch/gorest/internal/tasks/usecase"
	ticketcommenthandler "github.com/chuuch/gorest/internal/tickets/comments/handler"
	ticketcommentpostgres "github.com/chuuch/gorest/internal/tickets/comments/postgres"
	ticketcommentusecase "github.com/chuuch/gorest/internal/tickets/comments/usecase"
	ticketfilehandler "github.com/chuuch/gorest/internal/tickets/files/handler"
	ticketfilepostgres "github.com/chuuch/gorest/internal/tickets/files/postgres"
	ticketfileusecase "github.com/chuuch/gorest/internal/tickets/files/usecase"
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

type wiredApp struct {
	userHandler          *userhandler.Handler
	authHandler          *authhandler.Handler
	orgHandler           *orghandler.Handler
	clientHandler        *clienthandler.Handler
	projectHandler       *projecthandler.Handler
	taskHandler          *taskhandler.Handler
	timeEntryHandler     *timeentryhandler.Handler
	fileHandler          *filehandler.Handler
	commentHandler       *commenthandler.Handler
	clientUserHandler    *clientuserhandler.Handler
	ticketHandler        *tickethandler.Handler
	ticketFileHandler    *ticketfilehandler.Handler
	ticketCommentHandler *ticketcommenthandler.Handler
	activityHandler      *activityhandler.Handler
	reportHandler        *reporthandler.Handler
	notificationHandler  *notificationhandler.Handler
	invoiceHandler       *invoicehandler.Handler
	navHandler           *navhandler.Handler
	eventsHandler        *eventshandler.Handler
	tokenManager         security.TokenManager
	idem                 *idempotency.Middleware
}

func wireApp(
	cfg *config.Config,
	db *pgxpool.Pool,
	objectStore storage.ObjectStore,
	mailSender mailer.Mailer,
) *wiredApp {
	userRepository := userpostgres.NewRepository(db)
	passwordHasher := password.NewBcryptHasher(cfg.Auth.BcryptCost)
	userService := userusecase.NewService(userRepository, passwordHasher)
	userHandler := userhandler.NewHandler(userService)

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

	orgService := orgusecase.NewService(
		userService,
		membershipRepository,
		organizationRepository,
		inviteService,
		db,
	)
	orgHandler := orghandler.NewHandler(orgService)

	activityRepository := activitypostgres.NewRepository(db)
	eventHub := events.NewHub()
	eventsHandler := eventshandler.NewHandler(eventHub)
	activityService := activityusecase.EnableRealtime(
		activityusecase.NewService(activityRepository),
		eventHub,
	)
	activityHandler := activityhandler.NewHandler(activityService)

	reportRepository := reportspostgres.NewRepository(db)
	reportService := reportsusecase.NewService(reportRepository)
	reportHandler := reporthandler.NewHandler(reportService)

	notificationRepository := notificationpostgres.NewRepository(db)
	notificationService := notificationsusecase.EnableRealtime(
		notificationsusecase.NewService(notificationRepository, membershipRepository),
		eventHub,
	)
	notificationHandler := notificationhandler.NewHandler(notificationService)

	navRepository := navpostgres.NewRepository(db)
	navService := navusecase.NewService(navRepository)
	navHandler := navhandler.NewHandler(navService)

	clientRepository := clientpostgres.NewRepository(db)
	clientService := clientusecase.NewService(clientRepository)
	clientHandler := clienthandler.NewHandler(clientService)

	projectRepository := projectpostgres.NewRepository(db)
	projectService := projectusecase.NewService(projectRepository, clientRepository)
	projectHandler := projecthandler.NewHandler(projectService)

	ticketRepository := ticketpostgres.NewRepository(db)
	ticketService := ticketusecase.EnableNotifications(
		ticketusecase.NewService(ticketRepository, clientRepository),
		notificationService,
	)
	ticketHandler := tickethandler.NewHandler(ticketService).WithActivity(activityService)

	taskRepository := taskpostgres.NewRepository(db)
	taskService := taskusecase.EnableNotifications(
		taskusecase.NewService(taskRepository, projectRepository, ticketRepository, membershipRepository),
		notificationService,
	)
	taskHandler := taskhandler.NewHandler(taskService).WithActivity(activityService)

	timeEntryRepository := timeentrypostgres.NewRepository(db)
	timeEntryService := timeentryusecase.NewService(timeEntryRepository, taskRepository)
	timeEntryHandler := timeentryhandler.NewHandler(timeEntryService)

	fileRepository := filepostgres.NewRepository(db)
	fileService := fileusecase.NewService(fileRepository, projectRepository, objectStore)
	fileHandler := filehandler.NewHandler(fileService).WithActivity(activityService)

	commentRepository := commentpostgres.NewRepository(db)
	commentService := commentusecase.EnableNotifications(
		commentusecase.NewService(commentRepository, taskRepository),
		notificationService,
	)
	commentHandler := commenthandler.NewHandler(commentService).WithActivity(activityService)

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

	ticketFileRepository := ticketfilepostgres.NewRepository(db)
	ticketFileService := ticketfileusecase.NewService(ticketFileRepository, ticketRepository, objectStore)
	ticketFileHandler := ticketfilehandler.NewHandler(ticketFileService).WithActivity(activityService)

	ticketCommentRepository := ticketcommentpostgres.NewRepository(db)
	ticketCommentService := ticketcommentusecase.EnableNotifications(
		ticketcommentusecase.NewService(ticketCommentRepository, ticketRepository),
		notificationService,
	)
	ticketCommentHandler := ticketcommenthandler.NewHandler(ticketCommentService).WithActivity(activityService)

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

	return &wiredApp{
		userHandler:          userHandler,
		authHandler:          authHandler,
		orgHandler:           orgHandler,
		clientHandler:        clientHandler,
		projectHandler:       projectHandler,
		taskHandler:          taskHandler,
		timeEntryHandler:     timeEntryHandler,
		fileHandler:          fileHandler,
		commentHandler:       commentHandler,
		clientUserHandler:    clientUserHandler,
		ticketHandler:        ticketHandler,
		ticketFileHandler:    ticketFileHandler,
		ticketCommentHandler: ticketCommentHandler,
		activityHandler:      activityHandler,
		reportHandler:        reportHandler,
		notificationHandler:  notificationHandler,
		invoiceHandler:       invoiceHandler,
		navHandler:           navHandler,
		eventsHandler:        eventsHandler,
		tokenManager:         tokenManager,
		idem:                 idempotency.NewMiddleware(idempotency.NewPostgresStore(db)),
	}
}
