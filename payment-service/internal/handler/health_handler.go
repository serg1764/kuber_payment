package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool       *pgxpool.Pool
	appVersion string
}

func NewHealthHandler(pool *pgxpool.Pool, appVersion string) *HealthHandler {
	return &HealthHandler{
		pool:       pool,
		appVersion: appVersion,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":      "ok",
		"version":     h.appVersion,
		"information": "что-то доработали",
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ready",
		"version": h.appVersion,
	})
}
