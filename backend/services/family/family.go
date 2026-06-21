package family

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type CreateInput struct {
	Name        string
	Description string
	CreatedBy   uuid.UUID
}

type UpdateInput struct {
	Name        *string
	Description *string
}

type MemberWithUser struct {
	UserID    uuid.UUID        `json:"user_id"`
	Email     string           `json:"email"`
	Name      string           `json:"name"`
	AvatarURL string           `json:"avatar_url"`
	Role      models.FamilyRole `json:"role"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (models.Family, error)
	GetByID(ctx context.Context, familyID uuid.UUID) (models.Family, error)
	Update(ctx context.Context, familyID uuid.UUID, input UpdateInput) (models.Family, error)
	Delete(ctx context.Context, familyID uuid.UUID) error
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.FamilySummary, error)
	ListAll(ctx context.Context) ([]models.Family, error)
	ListMembers(ctx context.Context, familyID uuid.UUID) ([]MemberWithUser, error)
	UserRole(ctx context.Context, familyID, userID uuid.UUID) (models.FamilyRole, error)
	ListMembershipsForUser(ctx context.Context, userID uuid.UUID) ([]models.AdminFamilyAccess, error)
	SetMemberRole(ctx context.Context, familyID, userID uuid.UUID, role models.FamilyRole) error
	RemoveMember(ctx context.Context, familyID, userID uuid.UUID) error
	CountAll(ctx context.Context) (int, error)
	EnsureNativeFamilyForPerson(ctx context.Context, input EnsureNativeFamilyInput) (models.Family, error)
}