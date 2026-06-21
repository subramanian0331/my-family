package family

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/subbu/family_tree/models"
)

type EnsureNativeFamilyInput struct {
	PersonID        uuid.UUID
	UserID          uuid.UUID
	ContextFamilyID uuid.UUID
}

func DeriveFamilyName(givenName, patronymic, clanName string) string {
	if name := strings.TrimSpace(clanName); name != "" {
		return name
	}
	if name := strings.TrimSpace(patronymic); name != "" {
		return name
	}
	return strings.TrimSpace(givenName)
}

func (s *service) EnsureNativeFamilyForPerson(ctx context.Context, input EnsureNativeFamilyInput) (models.Family, error) {
	nativeID, err := s.nativeFamilyID(ctx, input.PersonID)
	if err != nil {
		return models.Family{}, err
	}
	if nativeID != uuid.Nil {
		return s.GetByID(ctx, nativeID)
	}

	var givenName, patronymic, clanName string
	err = s.db.Pool().QueryRow(ctx, `
		SELECT given_name, patronymic, clan_name
		FROM persons WHERE id = $1
	`, input.PersonID).Scan(&givenName, &patronymic, &clanName)
	if err != nil {
		return models.Family{}, err
	}

	name := DeriveFamilyName(givenName, patronymic, clanName)
	if name == "" {
		name = "Family"
	}

	if existing, ok, err := s.findByNameForUser(ctx, input.UserID, name); err != nil {
		return models.Family{}, err
	} else if ok {
		if err := s.addPersonFamilyLabel(ctx, input.PersonID, existing.ID, false); err != nil {
			return models.Family{}, err
		}
		return existing, nil
	}

	family, err := s.Create(ctx, CreateInput{
		Name:        name,
		Description: "Auto-created from family name",
		CreatedBy:   input.UserID,
	})
	if err != nil {
		return models.Family{}, err
	}

	if err := s.addPersonFamilyLabel(ctx, input.PersonID, family.ID, false); err != nil {
		return models.Family{}, err
	}

	return family, nil
}

func (s *service) nativeFamilyID(ctx context.Context, personID uuid.UUID) (uuid.UUID, error) {
	var familyID uuid.UUID
	err := s.db.Pool().QueryRow(ctx, `
		SELECT family_id
		FROM person_families
		WHERE person_id = $1 AND via_marriage = false
		ORDER BY created_at ASC
		LIMIT 1
	`, personID).Scan(&familyID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil
	}
	return familyID, err
}

func (s *service) findByNameForUser(ctx context.Context, userID uuid.UUID, name string) (models.Family, bool, error) {
	var family models.Family
	err := s.db.Pool().QueryRow(ctx, `
		SELECT f.id, f.name, f.slug, f.description, f.created_by, f.created_at, f.updated_at
		FROM families f
		JOIN family_members fm ON fm.family_id = f.id
		WHERE fm.user_id = $1 AND lower(f.name) = lower($2)
		ORDER BY f.created_at ASC
		LIMIT 1
	`, userID, name).Scan(
		&family.ID,
		&family.Name,
		&family.Slug,
		&family.Description,
		&family.CreatedBy,
		&family.CreatedAt,
		&family.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return models.Family{}, false, nil
	}
	if err != nil {
		return models.Family{}, false, err
	}
	return family, true, nil
}

func (s *service) addPersonFamilyLabel(ctx context.Context, personID, familyID uuid.UUID, viaMarriage bool) error {
	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO person_families (person_id, family_id, via_marriage)
		VALUES ($1, $2, $3)
		ON CONFLICT (person_id, family_id) DO UPDATE
		SET via_marriage = person_families.via_marriage AND EXCLUDED.via_marriage
	`, personID, familyID, viaMarriage)
	return err
}

