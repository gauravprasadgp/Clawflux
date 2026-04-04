package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type actorContextKey struct{}

func (r *Router) withActor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
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
		ctx := context.WithValue(req.Context(), actorContextKey{}, *actor)
		next(w, req.WithContext(ctx))
	}
}

func actorFromContext(ctx context.Context) (domain.Actor, error) {
	actor, ok := ctx.Value(actorContextKey{}).(domain.Actor)
	if !ok {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	return actor, nil
}
