package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/services"
)

// handleMe godoc
// @Summary Return the current actor
// @Tags Auth
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.Actor
// @Failure 401 {object} ErrorResponse
// @Router /v1/me [get]
func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actor)
}

// createApp godoc
// @Summary Create an app
// @Tags Apps
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Param input body services.CreateAppInput true "App definition"
// @Success 201 {object} domain.App
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /v1/apps [post]
func (r *Router) createApp(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var input services.CreateAppInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(w, err)
		return
	}
	app, err := r.apps.CreateApp(req.Context(), actor, input)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "app.create", "app", app.ID, "app created")
	writeJSON(w, http.StatusCreated, app)
}

// listApps godoc
// @Summary List apps for the current tenant
// @Tags Apps
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} AppListResponse
// @Failure 401 {object} ErrorResponse
// @Router /v1/apps [get]
func (r *Router) listApps(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	apps, err := r.apps.ListApps(req.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": apps})
}

// getApp godoc
// @Summary Get an app
// @Tags Apps
// @Produce json
// @Param appID path string true "App ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.App
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/apps/{appID} [get]
func (r *Router) getApp(w http.ResponseWriter, req *http.Request, appID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	app, err := r.apps.GetApp(req.Context(), actor, appID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

// updateApp godoc
// @Summary Update an app
// @Tags Apps
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Param input body services.UpdateAppInput true "Partial app update"
// @Success 200 {object} domain.App
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/apps/{appID} [patch]
func (r *Router) updateApp(w http.ResponseWriter, req *http.Request, appID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var input services.UpdateAppInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(w, err)
		return
	}
	app, err := r.apps.UpdateApp(req.Context(), actor, appID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "app.update", "app", app.ID, "app updated")
	writeJSON(w, http.StatusOK, app)
}

// createDeployment godoc
// @Summary Create a deployment for an app
// @Tags Deployments
// @Produce json
// @Param appID path string true "App ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 201 {object} domain.Deployment
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/apps/{appID}/deployments [post]
func (r *Router) createDeployment(w http.ResponseWriter, req *http.Request, appID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployment, err := r.deployments.CreateDeployment(req.Context(), actor, appID)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "deployment.create", "deployment", deployment.ID, "deployment created")
	writeJSON(w, http.StatusCreated, deployment)
}

// listDeployments godoc
// @Summary List deployments for an app
// @Tags Deployments
// @Produce json
// @Param appID path string true "App ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} DeploymentListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/apps/{appID}/deployments [get]
func (r *Router) listDeployments(w http.ResponseWriter, req *http.Request, appID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployments, err := r.deployments.ListDeployments(req.Context(), actor, appID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": deployments})
}

// getDeployment godoc
// @Summary Get a deployment
// @Tags Deployments
// @Produce json
// @Param deploymentID path string true "Deployment ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.Deployment
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/deployments/{deploymentID} [get]
func (r *Router) getDeployment(w http.ResponseWriter, req *http.Request, deploymentID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployment, err := r.deployments.GetDeployment(req.Context(), actor, deploymentID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deployment)
}

// listDeploymentEvents godoc
// @Summary List deployment events
// @Tags Deployments
// @Produce json
// @Param deploymentID path string true "Deployment ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} DeploymentEventListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/deployments/{deploymentID}/events [get]
func (r *Router) listDeploymentEvents(w http.ResponseWriter, req *http.Request, deploymentID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	events, err := r.deployments.ListEvents(req.Context(), actor, deploymentID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

// retryDeployment godoc
// @Summary Retry a failed or cancelled deployment
// @Tags Deployments
// @Produce json
// @Param deploymentID path string true "Deployment ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.Deployment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/deployments/{deploymentID}/retry [post]
func (r *Router) retryDeployment(w http.ResponseWriter, req *http.Request, deploymentID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployment, err := r.deployments.RetryDeployment(req.Context(), actor, deploymentID)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "deployment.retry", "deployment", deployment.ID, "deployment retry requested")
	writeJSON(w, http.StatusOK, deployment)
}

// cancelDeployment godoc
// @Summary Cancel a queued or provisioning deployment
// @Tags Deployments
// @Produce json
// @Param deploymentID path string true "Deployment ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.Deployment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/deployments/{deploymentID}/cancel [post]
func (r *Router) cancelDeployment(w http.ResponseWriter, req *http.Request, deploymentID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployment, err := r.deployments.CancelDeployment(req.Context(), actor, deploymentID)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "deployment.cancel", "deployment", deployment.ID, "deployment cancelled")
	writeJSON(w, http.StatusOK, deployment)
}

// deleteDeployment godoc
// @Summary Queue deployment deletion
// @Tags Deployments
// @Produce json
// @Param deploymentID path string true "Deployment ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.Deployment
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/deployments/{deploymentID}/delete [post]
func (r *Router) deleteDeployment(w http.ResponseWriter, req *http.Request, deploymentID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	deployment, err := r.deployments.DeleteDeployment(req.Context(), actor, deploymentID)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "deployment.delete", "deployment", deployment.ID, "deployment deletion requested")
	writeJSON(w, http.StatusOK, deployment)
}

// createAPIKey godoc
// @Summary Create a tenant API key
// @Tags API Keys
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Param input body APIKeyCreateRequest true "API key create request"
// @Success 201 {object} APIKeyCreateResultResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /v1/api-keys [post]
func (r *Router) createAPIKey(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var input APIKeyCreateRequest
	if err := decodeJSON(req, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := r.apiKeys.CreateKey(req.Context(), actor, input.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "api_key.create", "api_key", result.Key.ID, "api key created")
	writeJSON(w, http.StatusCreated, result)
}

// listAPIKeys godoc
// @Summary List tenant API keys
// @Tags API Keys
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} APIKeyListResponse
// @Failure 401 {object} ErrorResponse
// @Router /v1/api-keys [get]
func (r *Router) listAPIKeys(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	keys, err := r.apiKeys.ListKeys(req.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": keys})
}

// deleteAPIKey godoc
// @Summary Revoke an API key
// @Tags API Keys
// @Produce json
// @Param keyID path string true "API key ID"
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/api-keys/{keyID} [delete]
func (r *Router) deleteAPIKey(w http.ResponseWriter, req *http.Request, keyID string) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if err := r.apiKeys.RevokeKey(req.Context(), actor, keyID); err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "api_key.revoke", "api_key", keyID, "api key revoked")
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminInstances godoc
// @Summary List all deployed OpenClaw instances across all tenants
// @Tags Admin
// @Produce json
// @Param X-User-Email header string true "Admin email"
// @Param X-Platform-Admin header string true "Set to true"
// @Success 200 {object} AdminInstanceListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/admin/instances [get]
func (r *Router) handleAdminInstances(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	instances, err := r.admin.ListAllInstances(req.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": instances})
}

// handleAdminSummary godoc
// @Summary Get platform summary
// @Tags Admin
// @Produce json
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} domain.AdminSummary
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/admin/summary [get]
func (r *Router) handleAdminSummary(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	summary, err := r.admin.Summary(req.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleAdminPreflight godoc
// @Summary Inspect runtime readiness and setup guidance
// @Tags Admin
// @Produce json
// @Param X-User-Email header string true "Admin email"
// @Param X-Platform-Admin header string true "Set to true"
// @Success 200 {object} AdminPreflightResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/admin/preflight [get]
func (r *Router) handleAdminPreflight(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if !actor.IsPlatformAdmin {
		writeError(w, domain.ErrForbidden)
		return
	}

	readiness := r.health.Readiness(req.Context())
	response := AdminPreflightResponse{
		Status:             "ready",
		Backend:            r.backend,
		RepositoryDriver:   r.repository,
		DevelopmentAuth:    r.devAuth,
		DefaultIngressHost: r.ingressHost,
		Readiness:          readiness,
	}

	add := func(id, label, status, severity, message string) {
		response.Checks = append(response.Checks, AdminPreflightCheck{
			ID:       id,
			Label:    label,
			Status:   status,
			Severity: severity,
			Message:  message,
		})
		switch status {
		case "fail":
			response.Status = "blocked"
		case "warn":
			if response.Status == "ready" {
				response.Status = "warning"
			}
		}
	}

	add("api", "API process", "pass", "info", "The HTTP API is responding.")

	if r.repository == "memory" {
		add("persistence", "Persistence", "warn", "warning", "Using the in-memory repository. Great for demos, but data resets on restart.")
		response.Recommendations = append(response.Recommendations, "Set REPOSITORY_DRIVER=postgres and DATABASE_URL before inviting real users.")
	} else if unhealthyReadinessValue(readiness["database"]) {
		add("persistence", "Persistence", "fail", "critical", readiness["database"])
		response.Recommendations = append(response.Recommendations, "Check DATABASE_URL, run migrations, and confirm the API can reach Postgres.")
	} else {
		add("persistence", "Persistence", "pass", "info", "Postgres is reachable.")
	}

	if unhealthyReadinessValue(readiness["redis"]) {
		add("queue", "Job queue", "fail", "critical", readiness["redis"])
		response.Recommendations = append(response.Recommendations, "Start Redis or update REDIS_ADDR so deployment jobs can be queued.")
	} else {
		add("queue", "Job queue", "pass", "info", "Redis is reachable for async deployment jobs.")
	}

	if r.backend == "" {
		add("backend", "Deployment backend", "fail", "critical", "No deployment backend is registered.")
		response.Recommendations = append(response.Recommendations, "Configure a deployment backend before creating OpenClaw instances.")
	} else {
		add("backend", "Deployment backend", "pass", "info", "Using the "+r.backend+" deployment backend.")
	}

	if r.devAuth {
		add("auth", "Auth mode", "warn", "warning", "Development auth is enabled and trusts request headers.")
		response.Recommendations = append(response.Recommendations, "Disable DEVELOPMENT_AUTH and use real identity or API-key auth outside local development.")
	} else {
		add("auth", "Auth mode", "pass", "info", "Development header auth is disabled.")
	}

	if strings.TrimSpace(r.ingressHost) == "" {
		add("ingress", "Ingress default", "warn", "warning", "No default ingress host is configured.")
		response.Recommendations = append(response.Recommendations, "Set DEFAULT_INGRESS_HOST to make generated domains predictable.")
	} else {
		add("ingress", "Ingress default", "pass", "info", "Default ingress host is "+r.ingressHost+".")
	}

	writeJSON(w, http.StatusOK, response)
}

// handleAuditLogs godoc
// @Summary List tenant audit logs
// @Tags Audit
// @Produce json
// @Param limit query int false "Maximum number of logs to return" minimum(1) maximum(200)
// @Param X-API-Key header string false "Tenant API key"
// @Param X-User-Email header string false "Development/local user email"
// @Param X-User-Name header string false "Display name"
// @Param X-Platform-Admin header string false "Set to true for platform admin requests"
// @Success 200 {object} AuditLogListResponse
// @Failure 401 {object} ErrorResponse
// @Router /v1/admin/audit-logs [get]
func (r *Router) handleAuditLogs(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	limit := 50
	if raw := req.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	items, err := r.audit.ListTenantAuditLogs(req.Context(), actor, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handleReady godoc
// @Summary Readiness check
// @Tags Health
// @Produce json
// @Success 200 {object} ReadinessStatusResponse
// @Failure 503 {object} ReadinessStatusResponse
// @Router /readyz [get]
func (r *Router) handleReady(w http.ResponseWriter, req *http.Request) {
	status := r.health.Readiness(req.Context())
	code := http.StatusOK
	for _, value := range status {
		if unhealthyReadinessValue(value) {
			code = http.StatusServiceUnavailable
			break
		}
	}
	writeJSON(w, code, status)
}

func unhealthyReadinessValue(value string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "error:")
}

type adminCreateUserInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type adminDeployOpenClawInput struct {
	UserEmail          string            `json:"user_email"`
	UserName           string            `json:"user_name"`
	AppName            string            `json:"app_name"`
	AppSlug            string            `json:"app_slug"`
	Image              string            `json:"image"`
	Replicas           int32             `json:"replicas"`
	Public             bool              `json:"public"`
	Domain             string            `json:"domain"`
	GatewayBindAddress string            `json:"gateway_bind_address"`
	GatewayPort        int               `json:"gateway_port"`
	GatewayToken       string            `json:"gateway_token"`
	WorkspaceStorage   string            `json:"workspace_storage"`
	ProviderAPIKeys    map[string]string `json:"provider_api_keys"`
	AgentsMarkdown     string            `json:"agents_markdown"`
	SettingsJSON       string            `json:"settings_json"`
	ExtraEnv           map[string]string `json:"extra_env"`
	ExistingSecretName string            `json:"existing_secret_name"`
}

type adminDeployOpenClawResponse struct {
	User            *domain.Actor      `json:"user"`
	App             *domain.App        `json:"app"`
	Deployment      *domain.Deployment `json:"deployment"`
	UsedExistingApp bool               `json:"used_existing_app"`
}

func (r *Router) handleAdminUsers(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		r.createAdminUser(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// createAdminUser godoc
// @Summary Provision a user as platform admin
// @Tags Admin
// @Accept json
// @Produce json
// @Param X-User-Email header string true "Admin email"
// @Param X-Platform-Admin header string true "Set to true"
// @Param input body adminCreateUserInput true "User provision payload"
// @Success 201 {object} domain.Actor
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/admin/users [post]
func (r *Router) createAdminUser(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if !actor.IsPlatformAdmin {
		writeError(w, domain.ErrForbidden)
		return
	}

	var input adminCreateUserInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Email == "" {
		writeError(w, domain.ErrValidation)
		return
	}

	provisioned, err := r.auth.EnsureActor(req.Context(), input.Email, input.DisplayName)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = r.audit.Record(req.Context(), actor, "admin.user.provision", "user", provisioned.UserID, "admin provisioned user")
	writeJSON(w, http.StatusCreated, provisioned)
}

func (r *Router) handleAdminOpenClawDeploy(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		r.adminDeployOpenClaw(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// adminDeployOpenClaw godoc
// @Summary Deploy OpenClaw for a target user
// @Tags Admin
// @Accept json
// @Produce json
// @Param X-User-Email header string true "Admin email"
// @Param X-Platform-Admin header string true "Set to true"
// @Param input body adminDeployOpenClawInput true "OpenClaw deployment payload"
// @Success 201 {object} adminDeployOpenClawResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/admin/openclaw/deploy [post]
func (r *Router) adminDeployOpenClaw(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if !actor.IsPlatformAdmin {
		writeError(w, domain.ErrForbidden)
		return
	}

	var input adminDeployOpenClawInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(w, err)
		return
	}

	input.UserEmail = strings.TrimSpace(input.UserEmail)
	input.UserName = strings.TrimSpace(input.UserName)
	input.AppName = strings.TrimSpace(input.AppName)
	input.AppSlug = normalizeAdminSlug(input.AppSlug)
	input.Image = strings.TrimSpace(input.Image)
	input.Domain = strings.TrimSpace(input.Domain)
	input.GatewayBindAddress = strings.TrimSpace(input.GatewayBindAddress)
	input.GatewayToken = strings.TrimSpace(input.GatewayToken)
	input.WorkspaceStorage = strings.TrimSpace(input.WorkspaceStorage)
	input.ExistingSecretName = strings.TrimSpace(input.ExistingSecretName)
	if input.ProviderAPIKeys == nil {
		input.ProviderAPIKeys = map[string]string{}
	}
	for key, value := range input.ProviderAPIKeys {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		delete(input.ProviderAPIKeys, key)
		if trimmedKey != "" {
			input.ProviderAPIKeys[trimmedKey] = trimmedValue
		}
	}
	if input.ExtraEnv == nil {
		input.ExtraEnv = map[string]string{}
	}
	for key, value := range input.ExtraEnv {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		delete(input.ExtraEnv, key)
		if trimmedKey != "" && trimmedValue != "" {
			input.ExtraEnv[trimmedKey] = trimmedValue
		}
	}
	if input.UserEmail == "" {
		writeError(w, domain.ErrValidation)
		return
	}
	if input.AppName == "" {
		input.AppName = "openclaw"
	}
	if input.AppSlug == "" {
		input.AppSlug = normalizeAdminSlug(input.AppName)
	}
	if input.AppSlug == "" {
		input.AppSlug = "openclaw"
	}
	if input.Image == "" {
		input.Image = "ghcr.io/openclaw/openclaw:latest"
	}
	if input.GatewayBindAddress == "" {
		input.GatewayBindAddress = "0.0.0.0"
	}
	if input.GatewayPort <= 0 {
		input.GatewayPort = 18789
	}
	if input.WorkspaceStorage == "" {
		input.WorkspaceStorage = "10Gi"
	}
	if input.ExistingSecretName == "" {
		for key, value := range input.ProviderAPIKeys {
			if strings.TrimSpace(value) == "" {
				delete(input.ProviderAPIKeys, key)
			}
		}
	}
	if input.Replicas <= 0 {
		input.Replicas = 1
	}

	targetActor, err := r.auth.EnsureActor(req.Context(), input.UserEmail, input.UserName)
	if err != nil {
		writeError(w, err)
		return
	}

	appInput := services.CreateAppInput{
		Name: input.AppName,
		Slug: input.AppSlug,
		Config: domain.AppConfig{
			Image:    input.Image,
			Port:     input.GatewayPort,
			Env:      map[string]string{},
			Replicas: input.Replicas,
			Public:   input.Public,
			Domain:   input.Domain,
			OpenClaw: &domain.OpenClawConfig{
				Enabled:            true,
				GatewayBindAddress: input.GatewayBindAddress,
				GatewayPort:        input.GatewayPort,
				GatewayToken:       input.GatewayToken,
				WorkspaceStorage:   input.WorkspaceStorage,
				ProviderAPIKeys:    input.ProviderAPIKeys,
				AgentsMarkdown:     input.AgentsMarkdown,
				SettingsJSON:       input.SettingsJSON,
				ExtraEnv:           input.ExtraEnv,
				ExistingSecretName: input.ExistingSecretName,
			},
		},
	}

	app, err := r.apps.CreateApp(req.Context(), *targetActor, appInput)
	usedExisting := false
	if err != nil {
		if !errors.Is(err, domain.ErrConflict) {
			writeError(w, err)
			return
		}
		apps, listErr := r.apps.ListApps(req.Context(), *targetActor)
		if listErr != nil {
			writeError(w, listErr)
			return
		}
		for i := range apps {
			if apps[i].Slug == appInput.Slug {
				candidate := apps[i]
				app = &candidate
				usedExisting = true
				break
			}
		}
		if app == nil {
			writeError(w, err)
			return
		}
	}

	deployment, err := r.deployments.CreateDeployment(req.Context(), *targetActor, app.ID)
	if err != nil {
		writeError(w, err)
		return
	}

	_ = r.audit.Record(req.Context(), actor, "admin.openclaw.deploy", "deployment", deployment.ID, "admin deployed OpenClaw")
	writeJSON(w, http.StatusCreated, adminDeployOpenClawResponse{
		User:            targetActor,
		App:             app,
		Deployment:      deployment,
		UsedExistingApp: usedExisting,
	})
}

func normalizeAdminSlug(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	in = strings.ReplaceAll(in, " ", "-")
	in = strings.ReplaceAll(in, "_", "-")
	in = strings.ReplaceAll(in, ".", "-")
	in = strings.Trim(in, "-")
	return in
}
