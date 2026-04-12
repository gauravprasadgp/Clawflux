package http

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/services"
)

type Router struct {
	logger      *slog.Logger
	auth        *services.AuthService
	apiKeys     *services.APIKeyService
	admin       *services.AdminService
	audit       *services.AuditService
	health      *services.HealthService
	apps        *services.AppService
	deployments *services.DeploymentService
	devAuth     bool
}

func NewRouter(logger *slog.Logger, devAuth bool, auth *services.AuthService, apiKeys *services.APIKeyService, admin *services.AdminService, audit *services.AuditService, health *services.HealthService, apps *services.AppService, deployments *services.DeploymentService) http.Handler {
	r := &Router{
		logger:      logger,
		auth:        auth,
		apiKeys:     apiKeys,
		admin:       admin,
		audit:       audit,
		health:      health,
		apps:        apps,
		deployments: deployments,
		devAuth:     devAuth,
	}

	mux := http.NewServeMux()
	// React frontend — served from frontend/dist if it exists
	if _, err := os.Stat("frontend/dist"); err == nil {
		fs := http.FileServer(http.Dir("frontend/dist"))
		mux.Handle("/ui/", http.StripPrefix("/ui/", fs))
		mux.HandleFunc("/ui", func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/ui/", http.StatusMovedPermanently)
		})
	}

	// Legacy HTML admin UI
	mux.HandleFunc("/admin", r.handleAdminUI)
	mux.HandleFunc("/admin/", r.handleAdminUI)
	mux.HandleFunc("/healthz", r.handleHealth)
	mux.HandleFunc("/readyz", r.handleReady)
	mux.HandleFunc("/swagger", r.handleSwaggerUI)
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/swagger", "/swagger/":
			r.handleSwaggerUI(w, req)
		default:
			r.handleSwaggerAssets(w, req)
		}
	})
	mux.HandleFunc("/v1/me", r.withMiddleware(r.withActor(r.handleMe)))
	mux.HandleFunc("/v1/auth/providers", r.handleProviders)
	mux.HandleFunc("/v1/auth/medium/login", r.handleMediumLogin)
	mux.HandleFunc("/v1/auth/medium/callback", r.withMiddleware(r.handleMediumCallback))
	mux.HandleFunc("/v1/api-keys", r.withMiddleware(r.withActor(r.handleAPIKeys)))
	mux.HandleFunc("/v1/api-keys/", r.withMiddleware(r.withActor(r.handleAPIKeyRoutes)))
	mux.HandleFunc("/v1/admin/instances", r.withMiddleware(r.withActor(r.handleAdminInstances)))
	mux.HandleFunc("/v1/admin/summary", r.withMiddleware(r.withActor(r.handleAdminSummary)))
	mux.HandleFunc("/v1/admin/audit-logs", r.withMiddleware(r.withActor(r.handleAuditLogs)))
	mux.HandleFunc("/v1/admin/users", r.withMiddleware(r.withActor(r.handleAdminUsers)))
	mux.HandleFunc("/v1/admin/openclaw/deploy", r.withMiddleware(r.withActor(r.handleAdminOpenClawDeploy)))
	mux.HandleFunc("/v1/apps", r.withMiddleware(r.withActor(r.handleApps)))
	mux.HandleFunc("/v1/apps/", r.withMiddleware(r.withActor(r.handleAppRoutes)))
	mux.HandleFunc("/v1/deployments/", r.withMiddleware(r.withActor(r.handleDeploymentRoutes)))

	// Chain global middleware: panic recovery → security headers → request logging
	return r.withPanicRecovery(r.withSecurityHeaders(r.withRequestLogging(mux)))
}

// handleHealth godoc
// @Summary Liveness check
// @Tags Health
// @Produce json
// @Success 200 {object} HealthStatusResponse
// @Router /healthz [get]
func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleProviders godoc
// @Summary List supported auth providers
// @Tags Auth
// @Produce json
// @Success 200 {object} AuthProvidersResponse
// @Router /v1/auth/providers [get]
func (r *Router) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": []string{"medium"},
	})
}

// handleMediumLogin godoc
// @Summary Get Medium OAuth login URL
// @Tags Auth
// @Produce json
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} LoginURLResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/auth/medium/login [get]
func (r *Router) handleMediumLogin(w http.ResponseWriter, req *http.Request) {
	redirectURI := req.URL.Query().Get("redirect_uri")
	loginURL, err := r.auth.LoginURL(req.Context(), "medium", redirectURI)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": loginURL})
}

// handleMediumCallback godoc
// @Summary Handle Medium OAuth callback
// @Tags Auth
// @Produce json
// @Param code query string false "OAuth callback code"
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} domain.Actor
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/auth/medium/callback [get]
func (r *Router) handleMediumCallback(w http.ResponseWriter, req *http.Request) {
	redirectURI := req.URL.Query().Get("redirect_uri")
	code := req.URL.Query().Get("code")
	actor, err := r.auth.HandleOAuthCallback(req.Context(), "medium", code, redirectURI)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actor)
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
	if len(parts) == 2 && parts[1] == "retry" && req.Method == http.MethodPost {
		r.retryDeployment(w, req, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" && req.Method == http.MethodPost {
		r.cancelDeployment(w, req, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "delete" && req.Method == http.MethodPost {
		r.deleteDeployment(w, req, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "events" && req.Method == http.MethodGet {
		r.listDeploymentEvents(w, req, parts[0])
		return
	}
	http.NotFound(w, req)
}

func (r *Router) handleAPIKeys(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.listAPIKeys(w, req)
	case http.MethodPost:
		r.createAPIKey(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleAPIKeyRoutes(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/api-keys/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && req.Method == http.MethodDelete {
		r.deleteAPIKey(w, req, parts[0])
		return
	}
	http.NotFound(w, req)
}
