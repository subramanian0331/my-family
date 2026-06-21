package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type Service interface {
	SearchPeopleInFamily(ctx context.Context, familyID uuid.UUID, query string) ([]models.Person, error)
	SearchPeopleForUser(ctx context.Context, userID uuid.UUID, query string, targetFamilyID *uuid.UUID) ([]models.PersonSearchHit, error)
}