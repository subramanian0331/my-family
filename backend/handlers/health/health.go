package health

import (
	"context"
	"net/http"
	"time"

	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/handlers/response"
)

type Handler struct {
	db postgresclient.Client
}

func NewHandler(db postgresclient.Client) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"db":     err.Error(),
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}