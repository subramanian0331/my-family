package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type Service interface {
	LoginURL(state string) string
	HandleCallback(ctx context.Context, code string) (token string, user models.User, err error)
	ValidateToken(ctx context.Context, token string) (models.User, error)
	UserIDFromToken(ctx context.Context, token string) (uuid.UUID, error)
}