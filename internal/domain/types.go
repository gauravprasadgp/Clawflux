package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation failed")
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type DeploymentStatus string

const (
	DeploymentStatusRequested    DeploymentStatus = "requested"
	DeploymentStatusQueued       DeploymentStatus = "queued"
	DeploymentStatusProvisioning DeploymentStatus = "provisioning"
	DeploymentStatusRunning      DeploymentStatus = "running"
	DeploymentStatusFailed       DeploymentStatus = "failed"
	DeploymentStatusCancelled    DeploymentStatus = "cancelled"
	DeploymentStatusDeleting     DeploymentStatus = "deleting"
	DeploymentStatusDeleted      DeploymentStatus = "deleted"
)

type Actor struct {
	UserID          string `json:"user_id"`
	TenantID        string `json:"tenant_id"`
	Email           string `json:"email"`
	Role            Role   `json:"role"`
	APIKeyID        string `json:"api_key_id,omitempty"`
	IsPlatformAdmin bool   `json:"is_platform_admin"`
}

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	DisplayName     string    `json:"display_name"`
	DefaultTenantID string    `json:"default_tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Tenant struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantMember struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type App struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	DesiredState        string    `json:"desired_state"`
	CurrentDeploymentID string    `json:"current_deployment_id,omitempty"`
	Config              AppConfig `json:"config"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AppConfig struct {
	Image              string            `json:"image"`
	Port               int               `json:"port"`
	Env                map[string]string `json:"env"`
	Replicas           int32             `json:"replicas"`
	CPURequest         string            `json:"cpu_request"`
	MemoryRequest      string            `json:"memory_request"`
	CPULimit           string            `json:"cpu_limit"`
	MemoryLimit        string            `json:"memory_limit"`
	Public             bool              `json:"public"`
	Domain             string            `json:"domain,omitempty"`
	ServiceAccountName string            `json:"service_account_name,omitempty"`
	OpenClaw           *OpenClawConfig   `json:"openclaw,omitempty"`
}

type OpenClawConfig struct {
	Enabled            bool              `json:"enabled"`
	GatewayBindAddress string            `json:"gateway_bind_address,omitempty"`
	GatewayPort        int               `json:"gateway_port,omitempty"`
	GatewayToken       string            `json:"gateway_token,omitempty"`
	WorkspaceStorage   string            `json:"workspace_storage,omitempty"`
	ProviderAPIKeys    map[string]string `json:"provider_api_keys,omitempty"`
	AgentsMarkdown     string            `json:"agents_markdown,omitempty"`
	SettingsJSON       string            `json:"settings_json,omitempty"`
	ExtraEnv           map[string]string `json:"extra_env,omitempty"`
	ExistingSecretName string            `json:"existing_secret_name,omitempty"`
}

type Deployment struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	AppID          string           `json:"app_id"`
	Version        int              `json:"version"`
	ImageRef       string           `json:"image_ref"`
	ConfigSnapshot AppConfig        `json:"config_snapshot"`
	Status         DeploymentStatus `json:"status"`
	StatusReason   string           `json:"status_reason,omitempty"`
	Backend        string           `json:"backend"`
	BackendRef     BackendRef       `json:"backend_ref"`
	RequestedBy    string           `json:"requested_by"`
	StartedAt      *time.Time       `json:"started_at,omitempty"`
	FinishedAt     *time.Time       `json:"finished_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type BackendRef struct {
	Namespace   string `json:"namespace"`
	Deployment  string `json:"deployment"`
	Service     string `json:"service"`
	IngressName string `json:"ingress_name,omitempty"`
}

type BackendCapability string

const (
	BackendCapabilityDeploy            BackendCapability = "deploy"
	BackendCapabilityDelete            BackendCapability = "delete"
	BackendCapabilityStatus            BackendCapability = "status"
	BackendCapabilitySync              BackendCapability = "sync"
	BackendCapabilityDomains           BackendCapability = "domains"
	BackendCapabilitySecrets           BackendCapability = "secrets"
	BackendCapabilityPersistentStorage BackendCapability = "persistent_storage"
	BackendCapabilityReplicas          BackendCapability = "replicas"
	BackendCapabilityDryRun            BackendCapability = "dry_run"
)

type BackendCapabilities struct {
	Name         string              `json:"name"`
	DisplayName  string              `json:"display_name"`
	Capabilities []BackendCapability `json:"capabilities"`
	Notes        []string            `json:"notes,omitempty"`
}

type BackendPlanRequest struct {
	App         App
	NextVersion int
}

type DeploymentPlan struct {
	Backend      string              `json:"backend"`
	Version      int                 `json:"version"`
	ImageRef     string              `json:"image_ref"`
	BackendRef   BackendRef          `json:"backend_ref"`
	Capabilities BackendCapabilities `json:"capabilities"`
	Resources    []PlannedResource   `json:"resources"`
	Environment  []PlannedEnvVar     `json:"environment"`
	Secrets      []PlannedSecret     `json:"secrets"`
	Volumes      []PlannedVolume     `json:"volumes"`
	Exposure     PlannedExposure     `json:"exposure"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type PlannedResource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Action    string `json:"action"`
	Note      string `json:"note,omitempty"`
}

type PlannedEnvVar struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type PlannedSecret struct {
	Name    string   `json:"name"`
	Keys    []string `json:"keys"`
	Managed bool     `json:"managed"`
}

type PlannedVolume struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	MountPath string `json:"mount_path"`
	Size      string `json:"size,omitempty"`
}

type PlannedExposure struct {
	Public      bool   `json:"public"`
	Host        string `json:"host,omitempty"`
	ServiceName string `json:"service_name"`
	Port        int    `json:"port"`
	IngressName string `json:"ingress_name,omitempty"`
}

type DeploymentEvent struct {
	ID           string    `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	TenantID     string    `json:"tenant_id"`
	Type         string    `json:"type"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AuthIdentity struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	Provider       string            `json:"provider"`
	ProviderUserID string            `json:"provider_user_id"`
	AccessToken    string            `json:"-"`
	RefreshToken   string            `json:"-"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type ExternalIdentity struct {
	Provider       string            `json:"provider"`
	ProviderUserID string            `json:"provider_user_id"`
	Email          string            `json:"email"`
	DisplayName    string            `json:"display_name"`
	Metadata       map[string]string `json:"metadata"`
}

type AdminSummary struct {
	Users             int    `json:"users"`
	Tenants           int    `json:"tenants"`
	Apps              int    `json:"apps"`
	Deployments       int    `json:"deployments"`
	FailedDeployments int    `json:"failed_deployments"`
	RepositoryDriver  string `json:"repository_driver"`
}

type AdminInstance struct {
	App        App         `json:"app"`
	Deployment *Deployment `json:"deployment,omitempty"`
	UserEmail  string      `json:"user_email"`
}

type AuditLog struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ActorUserID   string    `json:"actor_user_id,omitempty"`
	ActorAPIKeyID string    `json:"actor_api_key_id,omitempty"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuthProvider interface {
	Name() string
	BeginAuth(ctx context.Context, state string, redirectURI string) (string, error)
	HandleCallback(ctx context.Context, code string, redirectURI string) (*ExternalIdentity, error)
}

type BackendDeployRequest struct {
	App        App
	Deployment Deployment
}

type BackendStatus struct {
	Status DeploymentStatus
	Reason string
	Ref    BackendRef
}

type DeploymentBackend interface {
	Name() string
	Capabilities() BackendCapabilities
	Plan(ctx context.Context, req BackendPlanRequest) (*DeploymentPlan, error)
	Submit(ctx context.Context, req BackendDeployRequest) (*BackendStatus, error)
	Delete(ctx context.Context, ref BackendRef) error
	GetStatus(ctx context.Context, ref BackendRef) (*BackendStatus, error)
}

type JobType string

const (
	JobTypeDeploymentCreate JobType = "deployment.create"
	JobTypeDeploymentDelete JobType = "deployment.delete"
	JobTypeDeploymentSync   JobType = "deployment.sync"
)

type Job struct {
	ID           string    `json:"id"`
	Type         JobType   `json:"type"`
	TenantID     string    `json:"tenant_id"`
	AppID        string    `json:"app_id,omitempty"`
	DeploymentID string    `json:"deployment_id,omitempty"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"created_at"`
}

type JobQueue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context) (Job, error)
}

type HealthChecker interface {
	Check(ctx context.Context) error
}

type Scheduler interface {
	ScheduleDeployment(ctx context.Context, deployment *Deployment) error
	ScheduleDelete(ctx context.Context, deployment *Deployment) error
	ScheduleSync(ctx context.Context, deployment *Deployment) error
}

type UserRepository interface {
	UpsertByEmail(ctx context.Context, email string, displayName string) (*User, error)
	GetByID(ctx context.Context, userID string) (*User, error)
}

type AuthIdentityRepository interface {
	Upsert(ctx context.Context, identity *AuthIdentity) error
}

type TenantRepository interface {
	CreatePersonalTenant(ctx context.Context, user *User) (*Tenant, error)
	GetByID(ctx context.Context, tenantID string) (*Tenant, error)
	AddMember(ctx context.Context, member TenantMember) error
	GetMember(ctx context.Context, tenantID, userID string) (*TenantMember, error)
}

type AppRepository interface {
	Create(ctx context.Context, app *App) error
	Update(ctx context.Context, app *App) error
	GetByID(ctx context.Context, tenantID, appID string) (*App, error)
	ListByTenant(ctx context.Context, tenantID string) ([]App, error)
}

type DeploymentRepository interface {
	Create(ctx context.Context, deployment *Deployment) error
	Update(ctx context.Context, deployment *Deployment) error
	GetByID(ctx context.Context, tenantID, deploymentID string) (*Deployment, error)
	ListByApp(ctx context.Context, tenantID, appID string) ([]Deployment, error)
	NextVersion(ctx context.Context, tenantID, appID string) (int, error)
}

type EventRepository interface {
	Create(ctx context.Context, event *DeploymentEvent) error
	ListByDeployment(ctx context.Context, tenantID, deploymentID string) ([]DeploymentEvent, error)
}

type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	ListByTenant(ctx context.Context, tenantID string) ([]APIKey, error)
	Revoke(ctx context.Context, tenantID, keyID string, revokedAt time.Time) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
}

type AdminRepository interface {
	Summary(ctx context.Context) (*AdminSummary, error)
	ListAllInstances(ctx context.Context) ([]AdminInstance, error)
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	ListByTenant(ctx context.Context, tenantID string, limit int) ([]AuditLog, error)
}
