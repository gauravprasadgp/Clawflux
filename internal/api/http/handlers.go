package http

import (
	"net/http"

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
