package http

import (
	"net/http"
	"strconv"

	"github.com/gauravprasad/clawcontrol/internal/services"
)

func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actor)
}

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

func (r *Router) createAPIKey(w http.ResponseWriter, req *http.Request) {
	actor, err := actorFromContext(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
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
