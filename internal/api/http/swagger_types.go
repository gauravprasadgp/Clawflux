package http

import (
	"github.com/gauravprasad/clawcontrol/internal/domain"
)

// type ErrorResponse struct {
// 	Error string `json:"error"`
// }

type HealthStatusResponse struct {
	Status string `json:"status"`
}

type AuthProvidersResponse struct {
	Providers []string `json:"providers"`
}

type LoginURLResponse struct {
	URL string `json:"url"`
}

type APIKeyCreateRequest struct {
	Name string `json:"name"`
}

type ReadinessStatusResponse map[string]string

type AppListResponse struct {
	Items []domain.App `json:"items"`
}

type DeploymentListResponse struct {
	Items []domain.Deployment `json:"items"`
}

type DeploymentEventListResponse struct {
	Items []domain.DeploymentEvent `json:"items"`
}

type APIKeyListResponse struct {
	Items []domain.APIKey `json:"items"`
}

type AuditLogListResponse struct {
	Items []domain.AuditLog `json:"items"`
}

type AdminInstanceListResponse struct {
	Items []domain.AdminInstance `json:"items"`
}

type APIKeyCreateResultResponse struct {
	Key    *domain.APIKey `json:"key"`
	Secret string         `json:"secret"`
}
