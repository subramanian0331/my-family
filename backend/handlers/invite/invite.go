package invite

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	familyservice "github.com/subbu/family_tree/services/family"
	inviteservice "github.com/subbu/family_tree/services/invite"
)

type Handler struct {
	invites  inviteservice.Service
	families familyservice.Service
}

func NewHandler(invites inviteservice.Service, families familyservice.Service) *Handler {
	return &Handler{invites: invites, families: families}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireManage(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	var payload struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Email == "" {
		response.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	family, err := h.families.GetByID(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "family not found")
		return
	}

	result, err := h.invites.Create(r.Context(), inviteservice.CreateInput{
		FamilyID:    familyID,
		Email:       payload.Email,
		Role:        parseRole(payload.Role),
		CreatedBy:   user.ID,
		FamilyName:  family.Name,
		InviterName: user.Name,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{
		"invite":     result.Invite,
		"email_sent": result.EmailSent,
	})
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.invites.Accept(r.Context(), payload.Token, user); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	invites, err := h.invites.ListPendingForEmail(r.Context(), user.Email)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, invites)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request, inviteID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	invite, err := h.invites.GetByID(r.Context(), inviteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if _, ok := access.RequireManage(w, r, h.families, invite.FamilyID, user.ID); !ok {
		return
	}

	if err := h.invites.Revoke(r.Context(), inviteID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func parseRole(role string) models.FamilyRole {
	switch role {
	case "editor":
		return models.FamilyRoleEditor
	case "owner":
		return models.FamilyRoleOwner
	default:
		return models.FamilyRoleViewer
	}
}