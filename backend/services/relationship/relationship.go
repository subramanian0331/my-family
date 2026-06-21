package relationship

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type CreateInput struct {
	FromPersonID uuid.UUID
	ToPersonID   uuid.UUID
	Type         models.RelationshipType
	Metadata     json.RawMessage
}

var (
	ErrSelfLink        = errors.New("cannot link a person to themselves")
	ErrParentAndSpouse = errors.New("cannot be both parent and spouse of the same person")
	ErrExistingSpouse  = errors.New("already has a spouse")
)

type UpdateInput struct {
	Metadata json.RawMessage
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (models.Relationship, error)
	Update(ctx context.Context, relationshipID uuid.UUID, input UpdateInput) (models.Relationship, error)
	Delete(ctx context.Context, relationshipID uuid.UUID) error
	ListForPerson(ctx context.Context, personID uuid.UUID) ([]models.Relationship, error)
	ListForFamily(ctx context.Context, familyID uuid.UUID) ([]models.Relationship, error)
}