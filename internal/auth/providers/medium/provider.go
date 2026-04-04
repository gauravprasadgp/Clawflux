package medium

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Provider struct {
	clientID string
}

func New(clientID string) *Provider {
	return &Provider{clientID: clientID}
}

func (p *Provider) Name() string {
	return "medium"
}

func (p *Provider) BeginAuth(_ context.Context, state string, redirectURI string) (string, error) {
	u, _ := url.Parse("https://medium.com/m/oauth/authorize")
	q := u.Query()
	q.Set("client_id", p.clientID)
	q.Set("state", state)
	q.Set("redirect_uri", redirectURI)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *Provider) HandleCallback(_ context.Context, code string, _ string) (*domain.ExternalIdentity, error) {
	if code == "" {
		return nil, domain.ErrUnauthorized
	}
	return &domain.ExternalIdentity{
		Provider:       "medium",
		ProviderUserID: code,
		Email:          fmt.Sprintf("%s@medium.local", code),
		DisplayName:    "Medium User",
	}, nil
}
