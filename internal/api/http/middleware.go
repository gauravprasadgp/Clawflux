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

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

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
		if email == "" {
			writeError(w, domain.ErrUnauthorized)
			return
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
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, req)
		requestID := rw.Header().Get("X-Request-ID")
		if requestID == "" {
			requestID = requestIDFromContext(req.Context())
		}
		r.logger.Info("http_request",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.Int("status", rw.status),
			slog.String("request_id", requestID),
			slog.String("remote_addr", req.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func (r *Router) withPanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Error("panic recovered",
					slog.String("path", req.URL.Path),
					slog.Any("panic", rec),
				)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

func (r *Router) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, req)
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
