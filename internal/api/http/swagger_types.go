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

type AdminBackendListResponse struct {
	Items []domain.BackendCapabilities `json:"items"`
}

type AdminPreflightCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type AdminPreflightResponse struct {
	Status             string                     `json:"status"`
	Backend            string                     `json:"backend"`
	RepositoryDriver   string                     `json:"repository_driver"`
	DevelopmentAuth    bool                       `json:"development_auth"`
	DefaultIngressHost string                     `json:"default_ingress_host"`
	Capabilities       domain.BackendCapabilities `json:"capabilities"`
	Readiness          map[string]string          `json:"readiness"`
	Checks             []AdminPreflightCheck      `json:"checks"`
	Recommendations    []string                   `json:"recommendations"`
}

type APIKeyCreateResultResponse struct {
	Key    *domain.APIKey `json:"key"`
	Secret string         `json:"secret"`
}
