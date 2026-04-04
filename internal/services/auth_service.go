package services

import (
	"context"
	"fmt"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type AuthService struct {
	users      domain.UserRepository
	tenants    domain.TenantRepository
	identities domain.AuthIdentityRepository
	apiKeys    *APIKeyService
	providers  map[string]domain.AuthProvider
}

func NewAuthService(users domain.UserRepository, tenants domain.TenantRepository, identities domain.AuthIdentityRepository, apiKeys *APIKeyService, providers []domain.AuthProvider) *AuthService {
	index := make(map[string]domain.AuthProvider, len(providers))
	for _, provider := range providers {
		index[provider.Name()] = provider
	}
	return &AuthService{
		users:      users,
		tenants:    tenants,
		identities: identities,
		apiKeys:    apiKeys,
		providers:  index,
	}
}

func (s *AuthService) EnsureActor(ctx context.Context, email string, displayName string) (*domain.Actor, error) {
	if email == "" {
		return nil, domain.ErrUnauthorized
	}
	user, err := s.users.UpsertByEmail(ctx, email, displayName)
	if err != nil {
		return nil, err
	}

	tenantID := user.DefaultTenantID
	role := domain.RoleOwner
	if tenantID == "" {
		tenant, err := s.tenants.CreatePersonalTenant(ctx, user)
		if err != nil {
			return nil, err
		}
		tenantID = tenant.ID
	} else {
		member, err := s.tenants.GetMember(ctx, tenantID, user.ID)
		if err == nil {
			role = member.Role
		}
	}

	return &domain.Actor{
		UserID:   user.ID,
		TenantID: tenantID,
		Email:    user.Email,
		Role:     role,
	}, nil
}

func (s *AuthService) AuthenticateAPIKey(ctx context.Context, rawKey string) (*domain.Actor, error) {
	if s.apiKeys == nil {
		return nil, domain.ErrUnauthorized
	}
	key, err := s.apiKeys.Authenticate(ctx, rawKey)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, key.UserID)
	if err != nil {
		return nil, err
	}
	member, err := s.tenants.GetMember(ctx, key.TenantID, user.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Actor{
		UserID:   user.ID,
		TenantID: key.TenantID,
		Email:    user.Email,
		Role:     member.Role,
		APIKeyID: key.ID,
	}, nil
}

func (s *AuthService) LoginURL(ctx context.Context, providerName string, redirectURI string) (string, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return "", fmt.Errorf("%w: provider %s", domain.ErrNotFound, providerName)
	}
	return provider.BeginAuth(ctx, "state-placeholder", redirectURI)
}

func (s *AuthService) HandleOAuthCallback(ctx context.Context, providerName string, code string, redirectURI string) (*domain.Actor, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("%w: provider %s", domain.ErrNotFound, providerName)
	}
	identity, err := provider.HandleCallback(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	actor, err := s.EnsureActor(ctx, identity.Email, identity.DisplayName)
	if err != nil {
		return nil, err
	}
	if s.identities != nil {
		_ = s.identities.Upsert(ctx, &domain.AuthIdentity{
			ID:             idgen.NewUUID(),
			UserID:         actor.UserID,
			Provider:       identity.Provider,
			ProviderUserID: identity.ProviderUserID,
			Metadata:       identity.Metadata,
		})
	}
	return actor, nil
}
