package admin

import (
	"net/http"

	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	familyservice "github.com/subbu/family_tree/services/family"
	userservice "github.com/subbu/family_tree/services/user"
)

type Handler struct {
	users    userservice.Service
	families familyservice.Service
}

func NewHandler(users userservice.Service, families familyservice.Service) *Handler {
	return &Handler{users: users, families: families}
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
	response.JSON(w, http.StatusOK, users)
}