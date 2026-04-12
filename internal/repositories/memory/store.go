package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type State struct {
	mu            sync.RWMutex
	users         map[string]*domain.User
	usersByEmail  map[string]string
	tenants       map[string]*domain.Tenant
	members       map[string]domain.TenantMember
	authIDs       map[string]domain.AuthIdentity
	apiKeys       map[string]*domain.APIKey
	auditLogs     []domain.AuditLog
	apps          map[string]*domain.App
	deployments   map[string]*domain.Deployment
	events        map[string][]domain.DeploymentEvent
	deployVersion map[string]int
}

func NewState() *State {
	return &State{
		users:         map[string]*domain.User{},
		usersByEmail:  map[string]string{},
		tenants:       map[string]*domain.Tenant{},
		members:       map[string]domain.TenantMember{},
		authIDs:       map[string]domain.AuthIdentity{},
		apiKeys:       map[string]*domain.APIKey{},
		auditLogs:     []domain.AuditLog{},
		apps:          map[string]*domain.App{},
		deployments:   map[string]*domain.Deployment{},
		events:        map[string][]domain.DeploymentEvent{},
		deployVersion: map[string]int{},
	}
}

type UserRepo struct{ state *State }
type TenantRepo struct{ state *State }
type AuthIdentityRepo struct{ state *State }
type APIKeyRepo struct{ state *State }
type AdminRepo struct{ state *State }
type AuditRepo struct{ state *State }
type AppRepo struct{ state *State }
type DeploymentRepo struct{ state *State }
type EventRepo struct{ state *State }

func NewUserRepo(state *State) *UserRepo                 { return &UserRepo{state: state} }
func NewTenantRepo(state *State) *TenantRepo             { return &TenantRepo{state: state} }
func NewAuthIdentityRepo(state *State) *AuthIdentityRepo { return &AuthIdentityRepo{state: state} }
func NewAPIKeyRepo(state *State) *APIKeyRepo             { return &APIKeyRepo{state: state} }
func NewAdminRepo(state *State) *AdminRepo               { return &AdminRepo{state: state} }
func NewAuditRepo(state *State) *AuditRepo               { return &AuditRepo{state: state} }
func NewAppRepo(state *State) *AppRepo                   { return &AppRepo{state: state} }
func NewDeploymentRepo(state *State) *DeploymentRepo     { return &DeploymentRepo{state: state} }
func NewEventRepo(state *State) *EventRepo               { return &EventRepo{state: state} }

func memberKey(tenantID, userID string) string {
	return tenantID + ":" + userID
}

func (r *UserRepo) UpsertByEmail(_ context.Context, email string, displayName string) (*domain.User, error) {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	key := strings.ToLower(strings.TrimSpace(email))
	if userID, ok := r.state.usersByEmail[key]; ok {
		user := *r.state.users[userID]
		if displayName != "" {
			user.DisplayName = displayName
		}
		user.UpdatedAt = time.Now().UTC()
		r.state.users[userID] = &user
		return copyUser(&user), nil
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:          idgen.NewUUID(),
		Email:       key,
		DisplayName: displayName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.state.users[user.ID] = user
	r.state.usersByEmail[key] = user.ID
	return copyUser(user), nil
}

func (r *UserRepo) GetByID(_ context.Context, userID string) (*domain.User, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	user, ok := r.state.users[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return copyUser(user), nil
}

func (r *TenantRepo) CreatePersonalTenant(_ context.Context, user *domain.User) (*domain.Tenant, error) {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	now := time.Now().UTC()
	tenant := &domain.Tenant{
		ID:        idgen.NewUUID(),
		Slug:      sanitizeSlug(user.Email),
		Name:      fallbackString(user.DisplayName, user.Email),
		Plan:      "free",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.state.tenants[tenant.ID] = tenant
	r.state.members[memberKey(tenant.ID, user.ID)] = domain.TenantMember{
		TenantID:  tenant.ID,
		UserID:    user.ID,
		Role:      domain.RoleOwner,
		CreatedAt: now,
	}

	updatedUser := *user
	updatedUser.DefaultTenantID = tenant.ID
	updatedUser.UpdatedAt = now
	r.state.users[user.ID] = &updatedUser

	return copyTenant(tenant), nil
}

func (r *TenantRepo) GetByID(_ context.Context, tenantID string) (*domain.Tenant, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	tenant, ok := r.state.tenants[tenantID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return copyTenant(tenant), nil
}

func (r *TenantRepo) AddMember(_ context.Context, member domain.TenantMember) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	r.state.members[memberKey(member.TenantID, member.UserID)] = member
	return nil
}

func (r *TenantRepo) GetMember(_ context.Context, tenantID, userID string) (*domain.TenantMember, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	member, ok := r.state.members[memberKey(tenantID, userID)]
	if !ok {
		return nil, domain.ErrForbidden
	}
	m := member
	return &m, nil
}

func (r *AuthIdentityRepo) Upsert(_ context.Context, identity *domain.AuthIdentity) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	key := identity.Provider + ":" + identity.ProviderUserID
	cp := *identity
	r.state.authIDs[key] = cp
	return nil
}

func (r *APIKeyRepo) Create(_ context.Context, key *domain.APIKey) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	cp := *key
	r.state.apiKeys[key.ID] = &cp
	return nil
}

func (r *APIKeyRepo) ListByTenant(_ context.Context, tenantID string) ([]domain.APIKey, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	out := make([]domain.APIKey, 0)
	for _, key := range r.state.apiKeys {
		if key.TenantID == tenantID {
			out = append(out, *key)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *APIKeyRepo) Revoke(_ context.Context, tenantID, keyID string, revokedAt time.Time) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	key, ok := r.state.apiKeys[keyID]
	if !ok || key.TenantID != tenantID {
		return domain.ErrNotFound
	}
	key.RevokedAt = &revokedAt
	return nil
}

func (r *APIKeyRepo) GetByHash(_ context.Context, keyHash string) (*domain.APIKey, error) {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	for _, key := range r.state.apiKeys {
		if key.KeyHash == keyHash {
			now := time.Now().UTC()
			key.LastUsedAt = &now
			cp := *key
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *AdminRepo) Summary(_ context.Context) (*domain.AdminSummary, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	summary := &domain.AdminSummary{
		Users:       len(r.state.users),
		Tenants:     len(r.state.tenants),
		Apps:        len(r.state.apps),
		Deployments: len(r.state.deployments),
	}
	for _, dep := range r.state.deployments {
		if dep.Status == domain.DeploymentStatusFailed {
			summary.FailedDeployments++
		}
	}
	return summary, nil
}

func (r *AdminRepo) ListAllInstances(_ context.Context) ([]domain.AdminInstance, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()

	// build user email lookup via tenant owner
	ownerEmail := map[string]string{} // tenantID -> email
	for _, m := range r.state.members {
		if m.Role == domain.RoleOwner {
			if u, ok := r.state.users[m.UserID]; ok {
				ownerEmail[m.TenantID] = u.Email
			}
		}
	}

	// build latest deployment per app
	latestDep := map[string]*domain.Deployment{} // appID -> latest
	for _, dep := range r.state.deployments {
		existing, ok := latestDep[dep.AppID]
		if !ok || dep.Version > existing.Version {
			cp := *dep
			latestDep[dep.AppID] = &cp
		}
	}

	out := make([]domain.AdminInstance, 0, len(r.state.apps))
	for _, app := range r.state.apps {
		cp := *app
		inst := domain.AdminInstance{
			App:       cp,
			UserEmail: ownerEmail[app.TenantID],
		}
		if dep, ok := latestDep[app.ID]; ok {
			inst.Deployment = dep
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].App.CreatedAt.After(out[j].App.CreatedAt)
	})
	return out, nil
}

func (r *AuditRepo) Create(_ context.Context, log *domain.AuditLog) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	r.state.auditLogs = append(r.state.auditLogs, *log)
	return nil
}

func (r *AuditRepo) ListByTenant(_ context.Context, tenantID string, limit int) ([]domain.AuditLog, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]domain.AuditLog, 0, limit)
	for i := len(r.state.auditLogs) - 1; i >= 0 && len(out) < limit; i-- {
		log := r.state.auditLogs[i]
		if log.TenantID == tenantID {
			out = append(out, log)
		}
	}
	return out, nil
}

func (r *AppRepo) Create(_ context.Context, app *domain.App) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	for _, existing := range r.state.apps {
		if existing.TenantID == app.TenantID && existing.Slug == app.Slug {
			return domain.ErrConflict
		}
	}
	cp := *app
	r.state.apps[app.ID] = &cp
	return nil
}

func (r *AppRepo) Update(_ context.Context, app *domain.App) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if _, ok := r.state.apps[app.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *app
	r.state.apps[app.ID] = &cp
	return nil
}

func (r *AppRepo) GetByID(_ context.Context, tenantID, appID string) (*domain.App, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	app, ok := r.state.apps[appID]
	if !ok || app.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	cp := *app
	return &cp, nil
}

func (r *AppRepo) ListByTenant(_ context.Context, tenantID string) ([]domain.App, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	out := make([]domain.App, 0)
	for _, app := range r.state.apps {
		if app.TenantID == tenantID {
			out = append(out, *app)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *DeploymentRepo) Create(_ context.Context, deployment *domain.Deployment) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	cp := *deployment
	r.state.deployments[deployment.ID] = &cp
	return nil
}

func (r *DeploymentRepo) Update(_ context.Context, deployment *domain.Deployment) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if _, ok := r.state.deployments[deployment.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *deployment
	r.state.deployments[deployment.ID] = &cp
	return nil
}

func (r *DeploymentRepo) GetByID(_ context.Context, tenantID, deploymentID string) (*domain.Deployment, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	deployment, ok := r.state.deployments[deploymentID]
	if !ok || deployment.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	cp := *deployment
	return &cp, nil
}

func (r *DeploymentRepo) ListByApp(_ context.Context, tenantID, appID string) ([]domain.Deployment, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	out := make([]domain.Deployment, 0)
	for _, deployment := range r.state.deployments {
		if deployment.TenantID == tenantID && deployment.AppID == appID {
			out = append(out, *deployment)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func (r *DeploymentRepo) NextVersion(_ context.Context, tenantID, appID string) (int, error) {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	key := tenantID + ":" + appID
	r.state.deployVersion[key]++
	return r.state.deployVersion[key], nil
}

func (r *EventRepo) Create(_ context.Context, event *domain.DeploymentEvent) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	r.state.events[event.DeploymentID] = append(r.state.events[event.DeploymentID], *event)
	return nil
}

func (r *EventRepo) ListByDeployment(_ context.Context, tenantID, deploymentID string) ([]domain.DeploymentEvent, error) {
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	events := r.state.events[deploymentID]
	out := make([]domain.DeploymentEvent, 0, len(events))
	for _, event := range events {
		if event.TenantID == tenantID {
			out = append(out, event)
		}
	}
	return out, nil
}

func copyUser(user *domain.User) *domain.User {
	cp := *user
	return &cp
}

func copyTenant(tenant *domain.Tenant) *domain.Tenant {
	cp := *tenant
	return &cp
}

func sanitizeSlug(in string) string {
	in = strings.ToLower(in)
	in = strings.ReplaceAll(in, "@", "-")
	in = strings.ReplaceAll(in, ".", "-")
	in = strings.ReplaceAll(in, "_", "-")
	if in == "" {
		return "tenant"
	}
	return in
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
