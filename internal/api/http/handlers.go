package http

import (
	"net/http"
	"strconv"
	
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
		if value != "ok" {
			code = http.StatusServiceUnavailable
			break
		}
	}
	writeJSON(w, code, status)
}
