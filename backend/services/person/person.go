package person

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type CreateInput struct {
	GivenName  string
	Patronymic string
	ClanName   string
	Gender     string
	BirthDate  *time.Time
	DeathDate  *time.Time
	BirthPlace string
	DeathPlace string
	Notes      string
	CreatedBy  uuid.UUID
	FamilyIDs  []uuid.UUID
}

type UpdateInput struct {
	GivenName  *string
	Patronymic *string
	ClanName   *string
	Gender     *string
	BirthDate  **time.Time
	DeathDate  **time.Time
	Deceased   *bool
	BirthPlace *string
	DeathPlace *string
	Notes      *string
	PhotoID    **uuid.UUID
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (models.Person, error)
	GetByID(ctx context.Context, personID uuid.UUID) (models.Person, error)
	Update(ctx context.Context, personID uuid.UUID, input UpdateInput) (models.Person, error)
	Delete(ctx context.Context, personID uuid.UUID) error
	ListByFamily(ctx context.Context, familyID uuid.UUID) ([]models.Person, error)
	AddFamilyLabel(ctx context.Context, personID, familyID uuid.UUID, viaMarriage bool) error
	SetFamilyMarriageLabel(ctx context.Context, personID, familyID uuid.UUID, viaMarriage bool) error
	HasFamilyLabel(ctx context.Context, personID, familyID uuid.UUID) (bool, error)
	IsNativeInFamily(ctx context.Context, personID, familyID uuid.UUID) (bool, error)
	ShouldBeMarryInToFamily(ctx context.Context, personID, familyID uuid.UUID) (bool, error)
	PrimaryNativeFamilyID(ctx context.Context, personID uuid.UUID) (uuid.UUID, error)
	RemoveFamilyLabel(ctx context.Context, personID, familyID uuid.UUID) error
	ListFamiliesForPerson(ctx context.Context, personID uuid.UUID) ([]models.PersonFamilyRef, error)
	UserCanAccess(ctx context.Context, userID, personID uuid.UUID) error
	SuggestPatronymic(ctx context.Context, personID uuid.UUID) (string, error)
	BulkCreate(ctx context.Context, input BulkCreateInput) (BulkCreateResult, error)
	SyncSpouseFamilyLabels(ctx context.Context, familyID, fromID, toID uuid.UUID) error
}