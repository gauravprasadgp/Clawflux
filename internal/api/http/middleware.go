package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type actorContextKey struct{}
type requestIDContextKey struct{}

func (r *Router) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return r.withRequestID(next)
}

func (r *Router) withActor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if apiKey := strings.TrimSpace(req.Header.Get("X-API-Key")); apiKey != "" {
			actor, err := r.auth.AuthenticateAPIKey(req.Context(), apiKey)
			if err != nil {
				writeError(w, err)
				return
			}
			if strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Platform-Admin")), "true") {
				actor.IsPlatformAdmin = true
			}
			ctx := context.WithValue(req.Context(), actorContextKey{}, *actor)
			next(w, req.WithContext(ctx))
			return
		}

		email := strings.TrimSpace(req.Header.Get("X-User-Email"))
		if email == "" && r.devAuth {
			email = "developer@local"
		}
		displayName := strings.TrimSpace(req.Header.Get("X-User-Name"))
		actor, err := r.auth.EnsureActor(req.Context(), email, displayName)
		if err != nil {
			writeError(w, err)
			return
		}
		if strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Platform-Admin")), "true") {
			actor.IsPlatformAdmin = true
		}
		ctx := context.WithValue(req.Context(), actorContextKey{}, *actor)
		next(w, req.WithContext(ctx))
	}
}

func (r *Router) withRequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestID := strings.TrimSpace(req.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = idgen.New("req")
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(req.Context(), requestIDContextKey{}, requestID)
		next(w, req.WithContext(ctx))
	}
}

func (r *Router) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			requestID = requestIDFromContext(req.Context())
		}
		r.logger.Info("http_request",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.String("request_id", requestID),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func actorFromContext(ctx context.Context) (domain.Actor, error) {
	actor, ok := ctx.Value(actorContextKey{}).(domain.Actor)
	if !ok {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	return actor, nil
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
