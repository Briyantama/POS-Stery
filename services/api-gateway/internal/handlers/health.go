package handlers

import (
	"context"
	"net/http"
	"time"
)

// RedisPinger can ping Redis to check liveness.
type RedisPinger interface {
	Ping(ctx context.Context) error
}

// Health handles GET /healthz — always returns 200.
func Health(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready handles GET /readyz — returns 503 if Redis is unreachable.
func Ready(rdb RedisPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := rdb.Ping(ctx); err != nil {
			renderJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"reason": "redis unreachable",
			})
			return
		}
		renderJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
