package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpapi "github.com/gauravprasad/clawcontrol/internal/api/http"
	"github.com/gauravprasad/clawcontrol/internal/auth/providers/medium"
	"github.com/gauravprasad/clawcontrol/internal/backends/kubernetes"
	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/observability"
	"github.com/gauravprasad/clawcontrol/internal/platform/database"
	memoryqueue "github.com/gauravprasad/clawcontrol/internal/queue/memory"
	"github.com/gauravprasad/clawcontrol/internal/queue/redis"
	"github.com/gauravprasad/clawcontrol/internal/repositories/memory"
	pgrepo "github.com/gauravprasad/clawcontrol/internal/repositories/postgres"
	"github.com/gauravprasad/clawcontrol/internal/services"
	workerpkg "github.com/gauravprasad/clawcontrol/internal/worker"
	workerhandlers "github.com/gauravprasad/clawcontrol/internal/worker/handlers"
)

type Runtime struct {
	Config             Config
	Logger             *slog.Logger
	DB                 *sql.DB
	AuthService        *services.AuthService
	APIKeyService      *services.APIKeyService
	AdminService       *services.AdminService
	AuditService       *services.AuditService
	HealthService      *services.HealthService
	AppService         *services.AppService
	DeploymentService  *services.DeploymentService
	ReliabilityService *services.ReliabilityService
	Scheduler          domain.Scheduler
	Backend            domain.DeploymentBackend
	Queue              domain.JobQueue
	UserRepo           domain.UserRepository
	AuthIdentityRepo   domain.AuthIdentityRepository
	APIKeyRepo         domain.APIKeyRepository
	AdminRepo          domain.AdminRepository
	AuditRepo          domain.AuditRepository
	TenantRepo         domain.TenantRepository
	AppRepo            domain.AppRepository
	DeploymentRepo     domain.DeploymentRepository
	EventRepo          domain.EventRepository
}

func NewRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
	logger := observability.NewLoggerWithLevel("clawflux", cfg.LogLevel)

	// Warn loudly if running with dev auth in a non-dev context.
	if cfg.DevelopmentAuth {
		logger.Warn("DEVELOPMENT_AUTH is enabled — all requests are trusted without credentials; do NOT use in production")
	}

	var queue domain.JobQueue
	var queueHealth domain.HealthChecker
	if cfg.RedisAddr == "memory" {
		memQueue := memoryqueue.NewClient()
		queue = memQueue
		queueHealth = memQueue
	} else {
		redisQueue := redis.NewClientWithPassword(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisQueue)
		queue = redisQueue
		queueHealth = redisQueue
	}

	var (
		db               *sql.DB
		userRepo         domain.UserRepository
		authIdentityRepo domain.AuthIdentityRepository
		apiKeyRepo       domain.APIKeyRepository
		adminRepo        domain.AdminRepository
		auditRepo        domain.AuditRepository
		tenantRepo       domain.TenantRepository
		appRepo          domain.AppRepository
		deploymentRepo   domain.DeploymentRepository
		eventRepo        domain.EventRepository
	)

	switch cfg.RepositoryDriver {
	case "memory":
		state := memory.NewState()
		userRepo = memory.NewUserRepo(state)
		authIdentityRepo = memory.NewAuthIdentityRepo(state)
		apiKeyRepo = memory.NewAPIKeyRepo(state)
		adminRepo = memory.NewAdminRepo(state)
		auditRepo = memory.NewAuditRepo(state)
		tenantRepo = memory.NewTenantRepo(state)
		appRepo = memory.NewAppRepo(state)
		deploymentRepo = memory.NewDeploymentRepo(state)
		eventRepo = memory.NewEventRepo(state)

	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required when REPOSITORY_DRIVER=postgres")
		}
		var err error
		db, err = database.Open(ctx, database.Config{
			URL:             cfg.DatabaseURL,
			MaxOpenConns:    cfg.DBMaxOpenConns,
			MaxIdleConns:    cfg.DBMaxIdleConns,
			ConnMaxLifetime: cfg.DBConnMaxLifetime,
			ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
		})
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		base := pgrepo.NewBase(db)
		userRepo = pgrepo.NewUserRepo(base)
		authIdentityRepo = pgrepo.NewAuthIdentityRepo(base)
		apiKeyRepo = pgrepo.NewAPIKeyRepo(base)
		adminRepo = pgrepo.NewAdminRepo(base)
		auditRepo = pgrepo.NewAuditRepo(base)
		tenantRepo = pgrepo.NewTenantRepo(base)
		appRepo = pgrepo.NewAppRepo(base)
		deploymentRepo = pgrepo.NewDeploymentRepo(base)
		eventRepo = pgrepo.NewEventRepo(base)

	default:
		return nil, fmt.Errorf("unsupported repository driver %q (choose postgres or memory)", cfg.RepositoryDriver)
	}

	apiKeyService := services.NewAPIKeyService(apiKeyRepo)
	authService := services.NewAuthService(userRepo, tenantRepo, authIdentityRepo, apiKeyService, []domain.AuthProvider{
		medium.New(cfg.MediumClientID),
	})
	appService := services.NewAppService(appRepo, tenantRepo)
	scheduler := services.NewSchedulerService(queue)
	deploymentService := services.NewDeploymentService(appRepo, deploymentRepo, eventRepo, scheduler)
	reliabilityService := services.NewReliabilityService(queue, deploymentRepo, scheduler)
	adminService := services.NewAdminService(adminRepo, cfg.RepositoryDriver)
	auditService := services.NewAuditService(auditRepo)
	healthService := services.NewHealthService(db, queueHealth)
	backend, err := kubernetes.NewBackend()
	if err != nil {
		return nil, fmt.Errorf("init kubernetes backend: %w", err)
	}
	deploymentService.SetBackend(backend)

	return &Runtime{
		Config:             cfg,
		Logger:             logger,
		DB:                 db,
		AuthService:        authService,
		APIKeyService:      apiKeyService,
		AdminService:       adminService,
		AuditService:       auditService,
		HealthService:      healthService,
		AppService:         appService,
		DeploymentService:  deploymentService,
		ReliabilityService: reliabilityService,
		Scheduler:          scheduler,
		Backend:            backend,
		Queue:              queue,
		UserRepo:           userRepo,
		AuthIdentityRepo:   authIdentityRepo,
		APIKeyRepo:         apiKeyRepo,
		AdminRepo:          adminRepo,
		AuditRepo:          auditRepo,
		TenantRepo:         tenantRepo,
		AppRepo:            appRepo,
		DeploymentRepo:     deploymentRepo,
		EventRepo:          eventRepo,
	}, nil
}

func (r *Runtime) HTTPHandler() http.Handler {
	return httpapi.NewRouter(
		r.Logger,
		r.Config.DevelopmentAuth,
		r.Config.RepositoryDriver,
		r.Backend.Name(),
		r.Config.AllowedIngressHost,
		r.AuthService,
		r.APIKeyService,
		r.AdminService,
		r.AuditService,
		r.HealthService,
		r.AppService,
		r.DeploymentService,
		r.ReliabilityService,
	)
}

func (r *Runtime) Worker() *workerpkg.Consumer {
	logger := observability.NewLoggerWithLevel("clawflux-worker", r.Config.LogLevel)
	consumer := workerpkg.NewConsumer(logger, r.Queue, r.Config.JobMaxAttempts, r.Config.JobRetryBackoff)
	deploymentCreate := workerhandlers.NewDeploymentCreateHandler(r.AppRepo, r.DeploymentRepo, r.Backend, r.DeploymentService, r.Scheduler)
	deploymentDelete := workerhandlers.NewDeploymentDeleteHandler(r.AppRepo, r.DeploymentRepo, r.Backend, r.DeploymentService)
	deploymentSync := workerhandlers.NewDeploymentSyncHandler(r.AppRepo, r.DeploymentRepo, r.Backend, r.DeploymentService, r.Scheduler)
	consumer.Register(domain.JobTypeDeploymentCreate, deploymentCreate.Handle)
	consumer.Register(domain.JobTypeDeploymentDelete, deploymentDelete.Handle)
	consumer.Register(domain.JobTypeDeploymentSync, deploymentSync.Handle)
	return consumer
}

func (r *Runtime) RunReconciler(ctx context.Context) {
	interval := r.Config.ReconcileInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.ReliabilityService.PromoteDue(ctx)
			result, err := r.ReliabilityService.ReconcileOnce(ctx, 100)
			if err != nil {
				r.Logger.Warn("deployment reconciliation failed", "error", err)
				continue
			}
			if result.Scheduled > 0 {
				r.Logger.Info("deployment reconciliation scheduled work", "scanned", result.Scanned, "scheduled", result.Scheduled)
			}
		}
	}
}

func (r *Runtime) Close() error {
	if r.DB != nil {
		r.Logger.Info("closing database connection pool")
		return r.DB.Close()
	}
	return nil
}
