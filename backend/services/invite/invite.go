package invite

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type CreateInput struct {
	FamilyID    uuid.UUID
	Email       string
	Role        models.FamilyRole
	CreatedBy   uuid.UUID
	FamilyName  string
	InviterName string
}

type CreateResult struct {
	Invite    models.Invite
	EmailSent bool
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (CreateResult, error)
	GetByID(ctx context.Context, inviteID uuid.UUID) (models.Invite, error)
	Accept(ctx context.Context, token string, user models.User) error
	ListPendingForEmail(ctx context.Context, email string) ([]models.Invite, error)
	ListForFamily(ctx context.Context, familyID uuid.UUID) ([]models.Invite, error)
	ListAllPending(ctx context.Context) ([]models.AdminInviteDetail, error)
	CountPending(ctx context.Context) (int, error)
	Revoke(ctx context.Context, inviteID uuid.UUID) error
}