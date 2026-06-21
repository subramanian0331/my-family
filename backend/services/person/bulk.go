package person

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/subbu/family_tree/models"
)

var (
	ErrBulkDuplicateRef = errors.New("duplicate person ref")
	ErrBulkMissingRef   = errors.New("person ref is required")
	ErrBulkUnknownRef   = errors.New("unknown person ref")
	ErrBulkInvalidRel   = errors.New("invalid relationship endpoint")
)

type BulkPersonEntry struct {
	Ref        string
	GivenName  string
	Patronymic string
	ClanName   string
	Gender     string
	Notes      string
}

type BulkRelEndpoint struct {
	Ref      *string
	PersonID *uuid.UUID
}

type BulkRelationshipEntry struct {
	FromEndpoint BulkRelEndpoint
	ToEndpoint   BulkRelEndpoint
	Type         models.RelationshipType
}

type BulkCreateInput struct {
	FamilyID      uuid.UUID
	CreatedBy     uuid.UUID
	People        []BulkPersonEntry
	Relationships []BulkRelationshipEntry
}

type BulkCreateResult struct {
	People  []models.Person
	RefToID map[string]uuid.UUID
}

func (s *service) BulkCreate(ctx context.Context, input BulkCreateInput) (BulkCreateResult, error) {
	if len(input.People) == 0 {
		return BulkCreateResult{}, errors.New("at least one person is required")
	}

	refSet := make(map[string]struct{}, len(input.People))
	for _, person := range input.People {
		if person.Ref == "" {
			return BulkCreateResult{}, ErrBulkMissingRef
		}
		if person.GivenName == "" {
			return BulkCreateResult{}, errors.New("given_name is required for each person")
		}
		if _, exists := refSet[person.Ref]; exists {
			return BulkCreateResult{}, fmt.Errorf("%w: %s", ErrBulkDuplicateRef, person.Ref)
		}
		refSet[person.Ref] = struct{}{}
	}

	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return BulkCreateResult{}, err
	}
	defer tx.Rollback(ctx)

	refToID := make(map[string]uuid.UUID, len(input.People))
	created := make([]models.Person, 0, len(input.People))

	for _, entry := range input.People {
		var person models.Person
		err := tx.QueryRow(ctx, `
			INSERT INTO persons (
				given_name, patronymic, clan_name, gender, notes, created_by
			) VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id, given_name, patronymic, clan_name, gender,
			          birth_date, death_date, birth_place, death_place, notes,
			          created_by, created_at, updated_at
		`, entry.GivenName, entry.Patronymic, entry.ClanName, entry.Gender, entry.Notes, input.CreatedBy,
		).Scan(
			&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
			&person.BirthDate, &person.DeathDate, &person.BirthPlace, &person.DeathPlace, &person.Notes,
			&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			return BulkCreateResult{}, err
		}

		refToID[entry.Ref] = person.ID
		created = append(created, person)
	}

	for _, rel := range input.Relationships {
		fromID, err := resolveBulkEndpoint(rel.FromEndpoint, refToID)
		if err != nil {
			return BulkCreateResult{}, err
		}
		toID, err := resolveBulkEndpoint(rel.ToEndpoint, refToID)
		if err != nil {
			return BulkCreateResult{}, err
		}
		if fromID == toID {
			return BulkCreateResult{}, errors.New("cannot link a person to themselves")
		}

		if rel.Type == models.RelationshipSpouse {
			fromID, toID = canonicalSpousePair(fromID, toID)
			if err := ensureNoOtherSpouseTx(ctx, tx, fromID, toID); err != nil {
				return BulkCreateResult{}, err
			}
			linked, err := hasParentLinkTx(ctx, tx, fromID, toID)
			if err != nil {
				return BulkCreateResult{}, err
			}
			if linked {
				return BulkCreateResult{}, errors.New("cannot be both parent and spouse of the same person")
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO relationships (from_person_id, to_person_id, type, metadata)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (from_person_id, to_person_id, type) DO UPDATE
			SET metadata = relationships.metadata
		`, fromID, toID, rel.Type, json.RawMessage(`{}`)); err != nil {
			return BulkCreateResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return BulkCreateResult{}, err
	}

	return BulkCreateResult{People: created, RefToID: refToID}, nil
}

func resolveBulkEndpoint(endpoint BulkRelEndpoint, refToID map[string]uuid.UUID) (uuid.UUID, error) {
	hasRef := endpoint.Ref != nil && *endpoint.Ref != ""
	hasID := endpoint.PersonID != nil && *endpoint.PersonID != uuid.Nil
	if hasRef == hasID {
		return uuid.Nil, ErrBulkInvalidRel
	}
	if hasRef {
		id, ok := refToID[*endpoint.Ref]
		if !ok {
			return uuid.Nil, fmt.Errorf("%w: %s", ErrBulkUnknownRef, *endpoint.Ref)
		}
		return id, nil
	}
	return *endpoint.PersonID, nil
}

func canonicalSpousePair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

func ensureNoOtherSpouseTx(ctx context.Context, tx pgx.Tx, fromID, toID uuid.UUID) error {
	for _, personID := range []uuid.UUID{fromID, toID} {
		var otherID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT CASE
			         WHEN r.from_person_id = $1 THEN r.to_person_id
			         ELSE r.from_person_id
			       END
			FROM relationships r
			WHERE r.type = 'spouse'
			  AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
			  AND (r.from_person_id = $1 OR r.to_person_id = $1)
			  AND NOT (
			    (r.from_person_id = $1 AND r.to_person_id = $2)
			    OR (r.from_person_id = $2 AND r.to_person_id = $1)
			  )
			LIMIT 1
		`, personID, toID).Scan(&otherID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		var name string
		_ = tx.QueryRow(ctx, `SELECT given_name FROM persons WHERE id = $1`, otherID).Scan(&name)
		if name == "" {
			name = "someone else"
		}
		return fmt.Errorf("already has a spouse: already married to %s", name)
	}
	return nil
}

func hasParentLinkTx(ctx context.Context, tx pgx.Tx, a, b uuid.UUID) (bool, error) {
	var linked bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM relationships
			WHERE type = 'parent'
			  AND (
			    (from_person_id = $1 AND to_person_id = $2)
			    OR (from_person_id = $2 AND to_person_id = $1)
			  )
		)
	`, a, b).Scan(&linked)
	return linked, err
}