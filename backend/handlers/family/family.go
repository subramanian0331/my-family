package family

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	familyservice "github.com/subbu/family_tree/services/family"
	inviteservice "github.com/subbu/family_tree/services/invite"
)

type Handler struct {
	families familyservice.Service
	invites  inviteservice.Service
}

func NewHandler(families familyservice.Service, invites inviteservice.Service) *Handler {
	return &Handler{families: families, invites: invites}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	families, err := h.families.ListForUser(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, families)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	family, err := h.families.Create(r.Context(), familyservice.CreateInput{
		Name:        payload.Name,
		Description: payload.Description,
		CreatedBy:   user.ID,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, family)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, ok := access.RequireView(w, r, h.families, familyID, user.ID)
	if !ok {
		return
	}

	family, err := h.families.GetByID(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"family": family,
		"role":   role,
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireManage(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	var payload struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	family, err := h.families.Update(r.Context(), familyID, familyservice.UpdateInput{
		Name:        payload.Name,
		Description: payload.Description,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, family)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireManage(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	if err := h.families.Delete(r.Context(), familyID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	members, err := h.families.ListMembers(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, members)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireManage(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	invites, err := h.invites.ListForFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, invites)
}