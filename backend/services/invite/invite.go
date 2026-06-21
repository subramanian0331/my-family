package invite

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type CreateInput struct {
	FamilyID  uuid.UUID
	Email     string
	Role      models.FamilyRole
	CreatedBy uuid.UUID
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (models.Invite, error)
	GetByID(ctx context.Context, inviteID uuid.UUID) (models.Invite, error)
	Accept(ctx context.Context, token string, user models.User) error
	ListPendingForEmail(ctx context.Context, email string) ([]models.Invite, error)
	ListForFamily(ctx context.Context, familyID uuid.UUID) ([]models.Invite, error)
	Revoke(ctx context.Context, inviteID uuid.UUID) error
}