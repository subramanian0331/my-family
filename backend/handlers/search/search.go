package search

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	familyservice "github.com/subbu/family_tree/services/family"
	searchservice "github.com/subbu/family_tree/services/search"
)

type Handler struct {
	search   searchservice.Service
	families familyservice.Service
}

func NewHandler(search searchservice.Service, families familyservice.Service) *Handler {
	return &Handler{search: search, families: families}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	query := r.URL.Query().Get("q")
	results, err := h.search.SearchPeopleInFamily(r.Context(), familyID, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *Handler) SearchGlobal(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query().Get("q")
	var targetFamilyID *uuid.UUID
	if raw := r.URL.Query().Get("family_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid family_id")
			return
		}
		if _, ok := access.RequireView(w, r, h.families, id, user.ID); !ok {
			return
		}
		targetFamilyID = &id
	}

	results, err := h.search.SearchPeopleForUser(r.Context(), user.ID, query, targetFamilyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, results)
}