package access

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	familyservice "github.com/subbu/family_tree/services/family"
)

func CanView(role models.FamilyRole) bool {
	return role == models.FamilyRoleOwner || role == models.FamilyRoleEditor || role == models.FamilyRoleViewer
}

func CanEdit(role models.FamilyRole) bool {
	return role == models.FamilyRoleOwner || role == models.FamilyRoleEditor
}

func CanManage(role models.FamilyRole) bool {
	return role == models.FamilyRoleOwner
}

func FamilyRole(ctx context.Context, families familyservice.Service, familyID, userID uuid.UUID) (models.FamilyRole, error) {
	return families.UserRole(ctx, familyID, userID)
}

func RequireView(w http.ResponseWriter, r *http.Request, families familyservice.Service, familyID, userID uuid.UUID) (models.FamilyRole, bool) {
	role, err := FamilyRole(r.Context(), families, familyID, userID)
	if err != nil || !CanView(role) {
		response.Error(w, http.StatusForbidden, "forbidden")
		return "", false
	}
	return role, true
}

func RequireEdit(w http.ResponseWriter, r *http.Request, families familyservice.Service, familyID, userID uuid.UUID) (models.FamilyRole, bool) {
	role, err := FamilyRole(r.Context(), families, familyID, userID)
	if err != nil || !CanEdit(role) {
		response.Error(w, http.StatusForbidden, "forbidden")
		return "", false
	}
	return role, true
}

func RequireManage(w http.ResponseWriter, r *http.Request, families familyservice.Service, familyID, userID uuid.UUID) (models.FamilyRole, bool) {
	role, err := FamilyRole(r.Context(), families, familyID, userID)
	if err != nil || !CanManage(role) {
		response.Error(w, http.StatusForbidden, "forbidden")
		return "", false
	}
	return role, true
}

func RequireSiteAdmin(w http.ResponseWriter, user models.User) bool {
	if user.SiteRole != models.SiteRoleAdmin {
		response.Error(w, http.StatusForbidden, "admin only")
		return false
	}
	return true
}