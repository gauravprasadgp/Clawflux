package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type APIKeyService struct {
	keys domain.APIKeyRepository
}

type APIKeyCreateResult struct {
	Key    *domain.APIKey `json:"key"`
	Secret string         `json:"secret"`
}

func NewAPIKeyService(keys domain.APIKeyRepository) *APIKeyService {
	return &APIKeyService{keys: keys}
}

func (s *APIKeyService) CreateKey(ctx context.Context, actor domain.Actor, name string) (*APIKeyCreateResult, error) {
	if strings.TrimSpace(name) == "" {
		return nil, domain.ErrValidation
	}
	raw, prefix, hash, err := newAPIKeyMaterial()
	if err != nil {
		return nil, err
	}
	key := &domain.APIKey{
		ID:        idgen.NewUUID(),
		TenantID:  actor.TenantID,
		UserID:    actor.UserID,
		Name:      strings.TrimSpace(name),
		KeyPrefix: prefix,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.keys.Create(ctx, key); err != nil {
		return nil, err
	}
	return &APIKeyCreateResult{Key: key, Secret: raw}, nil
}

func (s *APIKeyService) ListKeys(ctx context.Context, actor domain.Actor) ([]domain.APIKey, error) {
	return s.keys.ListByTenant(ctx, actor.TenantID)
}

func (s *APIKeyService) RevokeKey(ctx context.Context, actor domain.Actor, keyID string) error {
	return s.keys.Revoke(ctx, actor.TenantID, keyID, time.Now().UTC())
}

func (s *APIKeyService) Authenticate(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, domain.ErrUnauthorized
	}
	hash := hashAPIKey(rawKey)
	key, err := s.keys.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if key.RevokedAt != nil {
		return nil, domain.ErrUnauthorized
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now().UTC()) {
		return nil, domain.ErrUnauthorized
	}
	return key, nil
}

func newAPIKeyMaterial() (raw string, prefix string, hash string, err error) {
	buf := make([]byte, 18)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", err
	}
	secret := hex.EncodeToString(buf)
	raw = "cc_" + secret
	prefix = raw[:10]
	hash = hashAPIKey(raw)
	return raw, prefix, hash, nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
