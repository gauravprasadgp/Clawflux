package services

import (
	"context"
	"strings"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type AppService struct {
	apps    domain.AppRepository
	tenants domain.TenantRepository
}

type CreateAppInput struct {
	Name   string           `json:"name"`
	Slug   string           `json:"slug"`
	Config domain.AppConfig `json:"config"`
}

type UpdateAppInput struct {
	Name         *string           `json:"name,omitempty"`
	DesiredState *string           `json:"desired_state,omitempty"`
	Config       *domain.AppConfig `json:"config,omitempty"`
}

func NewAppService(apps domain.AppRepository, tenants domain.TenantRepository) *AppService {
	return &AppService{apps: apps, tenants: tenants}
}

func (s *AppService) CreateApp(ctx context.Context, actor domain.Actor, in CreateAppInput) (*domain.App, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Slug) == "" {
		return nil, domain.ErrValidation
	}
	if _, err := s.tenants.GetMember(ctx, actor.TenantID, actor.UserID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	app := &domain.App{
		ID:           idgen.NewUUID(),
		TenantID:     actor.TenantID,
		Name:         in.Name,
		Slug:         normalizeSlug(in.Slug),
		DesiredState: "active",
		Config:       normalizeConfig(in.Config),
		CreatedBy:    actor.UserID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.apps.Create(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *AppService) ListApps(ctx context.Context, actor domain.Actor) ([]domain.App, error) {
	return s.apps.ListByTenant(ctx, actor.TenantID)
}

func (s *AppService) GetApp(ctx context.Context, actor domain.Actor, appID string) (*domain.App, error) {
	return s.apps.GetByID(ctx, actor.TenantID, appID)
}

func (s *AppService) UpdateApp(ctx context.Context, actor domain.Actor, appID string, in UpdateAppInput) (*domain.App, error) {
	app, err := s.apps.GetByID(ctx, actor.TenantID, appID)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		app.Name = strings.TrimSpace(*in.Name)
	}
	if in.DesiredState != nil {
		app.DesiredState = strings.TrimSpace(*in.DesiredState)
	}
	if in.Config != nil {
		app.Config = normalizeConfig(*in.Config)
	}
	app.UpdatedAt = time.Now().UTC()
	if err := s.apps.Update(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func normalizeSlug(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	in = strings.ReplaceAll(in, " ", "-")
	return in
}

func normalizeConfig(cfg domain.AppConfig) domain.AppConfig {
	isOpenClaw := cfg.OpenClaw != nil || strings.Contains(strings.ToLower(cfg.Image), "openclaw")
	if isOpenClaw {
		if cfg.OpenClaw == nil {
			cfg.OpenClaw = &domain.OpenClawConfig{Enabled: true}
		}
		if !cfg.OpenClaw.Enabled {
			cfg.OpenClaw.Enabled = true
		}
		if cfg.Image == "" {
			cfg.Image = "ghcr.io/openclaw/openclaw:latest"
		}
		if cfg.Port == 0 {
			cfg.Port = 18789
		}
		if cfg.OpenClaw.GatewayPort == 0 {
			cfg.OpenClaw.GatewayPort = 18789
		}
		if cfg.OpenClaw.GatewayBindAddress == "" {
			cfg.OpenClaw.GatewayBindAddress = "0.0.0.0"
		}
		if cfg.OpenClaw.WorkspaceStorage == "" {
			cfg.OpenClaw.WorkspaceStorage = "10Gi"
		}
		if cfg.OpenClaw.ProviderAPIKeys == nil {
			cfg.OpenClaw.ProviderAPIKeys = map[string]string{}
		}
		if cfg.OpenClaw.ExtraEnv == nil {
			cfg.OpenClaw.ExtraEnv = map[string]string{}
		}
	}
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.Replicas == 0 {
		cfg.Replicas = 1
	}
	if cfg.CPURequest == "" {
		cfg.CPURequest = "250m"
	}
	if cfg.MemoryRequest == "" {
		cfg.MemoryRequest = "256Mi"
	}
	if cfg.CPULimit == "" {
		cfg.CPULimit = "500m"
	}
	if cfg.MemoryLimit == "" {
		cfg.MemoryLimit = "512Mi"
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}

	return cfg
}
