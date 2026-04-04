package http

import (
	"net/http"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/services"
)

type Router struct {
	auth        *services.AuthService
	apps        *services.AppService
	deployments *services.DeploymentService
	devAuth     bool
}

func NewRouter(devAuth bool, auth *services.AuthService, apps *services.AppService, deployments *services.DeploymentService) http.Handler {
	r := &Router{
		auth:        auth,
		apps:        apps,
		deployments: deployments,
		devAuth:     devAuth,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", r.handleHealth)
	mux.HandleFunc("/v1/me", r.withActor(r.handleMe))
	mux.HandleFunc("/v1/auth/providers", r.handleProviders)
	mux.HandleFunc("/v1/auth/medium/login", r.handleMediumLogin)
	mux.HandleFunc("/v1/apps", r.withActor(r.handleApps))
	mux.HandleFunc("/v1/apps/", r.withActor(r.handleAppRoutes))
	mux.HandleFunc("/v1/deployments/", r.withActor(r.handleDeploymentRoutes))
	return mux
}

func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": []string{"medium"},
	})
}

func (r *Router) handleMediumLogin(w http.ResponseWriter, req *http.Request) {
	redirectURI := req.URL.Query().Get("redirect_uri")
	loginURL, err := r.auth.LoginURL(req.Context(), "medium", redirectURI)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": loginURL})
}

func (r *Router) handleApps(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.listApps(w, req)
	case http.MethodPost:
		r.createApp(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleAppRoutes(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/apps/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 {
		switch req.Method {
		case http.MethodGet:
			r.getApp(w, req, parts[0])
		case http.MethodPatch:
			r.updateApp(w, req, parts[0])
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 2 && parts[1] == "deployments" {
		switch req.Method {
		case http.MethodGet:
			r.listDeployments(w, req, parts[0])
		case http.MethodPost:
			r.createDeployment(w, req, parts[0])
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	http.NotFound(w, req)
}

func (r *Router) handleDeploymentRoutes(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/deployments/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && req.Method == http.MethodGet {
		r.getDeployment(w, req, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "events" && req.Method == http.MethodGet {
		r.listDeploymentEvents(w, req, parts[0])
		return
	}
	http.NotFound(w, req)
}
