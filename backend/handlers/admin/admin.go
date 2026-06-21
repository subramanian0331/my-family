package admin

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/config"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	familyservice "github.com/subbu/family_tree/services/family"
	inviteservice "github.com/subbu/family_tree/services/invite"
	userservice "github.com/subbu/family_tree/services/user"
)

type Handler struct {
	users    userservice.Service
	families familyservice.Service
	invites  inviteservice.Service
	cfg      config.Config
}

func NewHandler(
	users userservice.Service,
	families familyservice.Service,
	invites inviteservice.Service,
	cfg config.Config,
) *Handler {
	return &Handler{users: users, families: families, invites: invites, cfg: cfg}
}

func (h *Handler) ListFamilies(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, user) {
		return
	}

	families, err := h.families.ListAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, families)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, user) {
		return
	}

	users, err := h.users.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	details := make([]models.AdminUserDetail, 0, len(users))
	for _, u := range users {
		families, err := h.families.ListMembershipsForUser(r.Context(), u.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		details = append(details, models.AdminUserDetail{
			User:     u,
			Families: families,
		})
	}
	response.JSON(w, http.StatusOK, details)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, user) {
		return
	}

	invites, err := h.invites.ListAllPending(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, invites)
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, user) {
		return
	}

	users, err := h.users.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	familyCount, err := h.families.CountAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	pendingInvites, err := h.invites.CountPending(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, models.AdminSettings{
		FrontendURL:    h.cfg.FrontendURL,
		GoogleEnabled:  h.cfg.GoogleClientID != "" && h.cfg.GoogleClientSecret != "",
		SiteAdminEmail: h.cfg.SiteAdminEmail,
		UserCount:      len(users),
		FamilyCount:    familyCount,
		PendingInvites: pendingInvites,
	})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	actor, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, actor) {
		return
	}

	var payload struct {
		SiteRole string `json:"site_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := models.SiteRole(payload.SiteRole)
	if role != models.SiteRoleUser && role != models.SiteRoleAdmin {
		response.Error(w, http.StatusBadRequest, "invalid site_role")
		return
	}

	target, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}

	if target.SiteRole == models.SiteRoleAdmin && role == models.SiteRoleUser {
		adminCount, err := h.users.CountBySiteRole(r.Context(), models.SiteRoleAdmin)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if adminCount <= 1 {
			response.Error(w, http.StatusBadRequest, "cannot remove the last site admin")
			return
		}
	}

	updated, err := h.users.UpdateSiteRole(r.Context(), userID, role)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) SetUserFamilyAccess(w http.ResponseWriter, r *http.Request, userID, familyID uuid.UUID) {
	actor, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, actor) {
		return
	}

	if _, err := h.users.GetByID(r.Context(), userID); err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if _, err := h.families.GetByID(r.Context(), familyID); err != nil {
		response.Error(w, http.StatusNotFound, "family not found")
		return
	}

	var payload struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := parseFamilyRole(payload.Role)
	if role == "" {
		response.Error(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.families.SetMemberRole(r.Context(), familyID, userID, role); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) RemoveUserFamilyAccess(w http.ResponseWriter, r *http.Request, userID, familyID uuid.UUID) {
	actor, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, actor) {
		return
	}

	if err := h.families.RemoveMember(r.Context(), familyID, userID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, actor) {
		return
	}

	var payload struct {
		FamilyID string `json:"family_id"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Email == "" {
		response.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	familyID, err := uuid.Parse(payload.FamilyID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid family_id")
		return
	}
	if _, err := h.families.GetByID(r.Context(), familyID); err != nil {
		response.Error(w, http.StatusNotFound, "family not found")
		return
	}

	role := parseFamilyRole(payload.Role)
	if role == "" {
		role = models.FamilyRoleViewer
	}

	invite, err := h.invites.Create(r.Context(), inviteservice.CreateInput{
		FamilyID:  familyID,
		Email:     payload.Email,
		Role:      role,
		CreatedBy: actor.ID,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, invite)
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request, inviteID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !access.RequireSiteAdmin(w, user) {
		return
	}

	if err := h.invites.Revoke(r.Context(), inviteID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parseFamilyRole(raw string) models.FamilyRole {
	switch models.FamilyRole(raw) {
	case models.FamilyRoleOwner:
		return models.FamilyRoleOwner
	case models.FamilyRoleEditor:
		return models.FamilyRoleEditor
	case models.FamilyRoleViewer:
		return models.FamilyRoleViewer
	default:
		return ""
	}
}