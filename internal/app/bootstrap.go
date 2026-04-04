package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	httpapi "github.com/gauravprasad/clawcontrol/internal/api/http"
	"github.com/gauravprasad/clawcontrol/internal/auth/providers/medium"
	"github.com/gauravprasad/clawcontrol/internal/backends/kubernetes"
	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/observability"
	"github.com/gauravprasad/clawcontrol/internal/platform/database"
	"github.com/gauravprasad/clawcontrol/internal/queue/redis"
	"github.com/gauravprasad/clawcontrol/internal/repositories/memory"
	pgrepo "github.com/gauravprasad/clawcontrol/internal/repositories/postgres"
	"github.com/gauravprasad/clawcontrol/internal/services"
	workerpkg "github.com/gauravprasad/clawcontrol/internal/worker"
	workerhandlers "github.com/gauravprasad/clawcontrol/internal/worker/handlers"
)

type Runtime struct {
	Config            Config
	Logger            *slog.Logger
	DB                *sql.DB
	AuthService       *services.AuthService
	AppService        *services.AppService
	DeploymentService *services.DeploymentService
	Scheduler         domain.Scheduler
	Backend           domain.DeploymentBackend
	Queue             domain.JobQueue
	UserRepo          domain.UserRepository
	TenantRepo        domain.TenantRepository
	AppRepo           domain.AppRepository
	DeploymentRepo    domain.DeploymentRepository
	EventRepo         domain.EventRepository
}

func NewRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
	logger := observability.NewLogger("clawplane")
	queue := redis.NewClient(cfg.RedisAddr, cfg.RedisQueue)
	var (
		db             *sql.DB
		userRepo       domain.UserRepository
		tenantRepo     domain.TenantRepository
		appRepo        domain.AppRepository
		deploymentRepo domain.DeploymentRepository
		eventRepo      domain.EventRepository
	)

	switch cfg.RepositoryDriver {
	case "memory":
		state := memory.NewState()
		userRepo = memory.NewUserRepo(state)
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
		})
		if err != nil {
			return nil, err
		}
		base := pgrepo.NewBase(db)
		userRepo = pgrepo.NewUserRepo(base)
		tenantRepo = pgrepo.NewTenantRepo(base)
		appRepo = pgrepo.NewAppRepo(base)
		deploymentRepo = pgrepo.NewDeploymentRepo(base)
		eventRepo = pgrepo.NewEventRepo(base)
	default:
		return nil, fmt.Errorf("unsupported repository driver %q", cfg.RepositoryDriver)
	}

	authService := services.NewAuthService(userRepo, tenantRepo, []domain.AuthProvider{
		medium.New(cfg.MediumClientID),
	})
	appService := services.NewAppService(appRepo, tenantRepo)
	scheduler := services.NewSchedulerService(queue)
	deploymentService := services.NewDeploymentService(appRepo, deploymentRepo, eventRepo, scheduler)

	return &Runtime{
		Config:            cfg,
		Logger:            logger,
		DB:                db,
		AuthService:       authService,
		AppService:        appService,
		DeploymentService: deploymentService,
		Scheduler:         scheduler,
		Backend:           kubernetes.NewBackend(),
		Queue:             queue,
		UserRepo:          userRepo,
		TenantRepo:        tenantRepo,
		AppRepo:           appRepo,
		DeploymentRepo:    deploymentRepo,
		EventRepo:         eventRepo,
	}, nil
}

func (r *Runtime) HTTPHandler() http.Handler {
	return httpapi.NewRouter(r.Config.DevelopmentAuth, r.AuthService, r.AppService, r.DeploymentService)
}

func (r *Runtime) Worker() *workerpkg.Consumer {
	logger := observability.NewLogger("clawplane-worker")
	consumer := workerpkg.NewConsumer(logger, r.Queue)
	deploymentCreate := workerhandlers.NewDeploymentCreateHandler(r.AppRepo, r.DeploymentRepo, r.Backend, r.DeploymentService)
	consumer.Register(domain.JobTypeDeploymentCreate, deploymentCreate.Handle)
	return consumer
}

func (r *Runtime) Close() error {
	if r.DB != nil {
		return r.DB.Close()
	}
	return nil
}
