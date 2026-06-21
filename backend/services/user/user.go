package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type UpsertGoogleUserInput struct {
	GoogleSub string
	Email     string
	Name      string
	AvatarURL string
	SiteRole  models.SiteRole
}

type Service interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	UpsertGoogleUser(ctx context.Context, input UpsertGoogleUserInput) (models.User, error)
	List(ctx context.Context) ([]models.User, error)
	UpdateSiteRole(ctx context.Context, userID uuid.UUID, role models.SiteRole) (models.User, error)
	CountBySiteRole(ctx context.Context, role models.SiteRole) (int, error)
}