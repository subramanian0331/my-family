package relationship

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/models"
)

type service struct {
	db postgresclient.Client
}

func NewService(db postgresclient.Client) Service {
	return &service{db: db}
}

const activeSpouseSQL = `COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'`

func canonicalSpousePair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

func maritalStatus(metadata json.RawMessage) string {
	if len(metadata) == 0 {
		return "married"
	}
	var payload struct {
		MaritalStatus string `json:"marital_status"`
	}
	if err := json.Unmarshal(metadata, &payload); err != nil || payload.MaritalStatus == "" {
		return "married"
	}
	return payload.MaritalStatus
}

func marriedMetadata() json.RawMessage {
	return json.RawMessage(`{"marital_status":"married"}`)
}

func divorcedMetadata() json.RawMessage {
	return json.RawMessage(`{"marital_status":"divorced"}`)
}

func (s *service) Create(ctx context.Context, input CreateInput) (models.Relationship, error) {
	if input.FromPersonID == input.ToPersonID {
		return models.Relationship{}, ErrSelfLink
	}
	if input.Metadata == nil {
		input.Metadata = json.RawMessage(`{}`)
	}

	if input.Type == models.RelationshipSpouse {
		fromID, toID := canonicalSpousePair(input.FromPersonID, input.ToPersonID)
		input.FromPersonID = fromID
		input.ToPersonID = toID

		if len(input.Metadata) == 0 || string(input.Metadata) == "{}" || string(input.Metadata) == "null" {
			input.Metadata = marriedMetadata()
		}

		linked, err := s.hasParentLink(ctx, input.FromPersonID, input.ToPersonID)
		if err != nil {
			return models.Relationship{}, err
		}
		if linked {
			return models.Relationship{}, ErrParentAndSpouse
		}

		requestStatus := maritalStatus(input.Metadata)

		if existing, ok, err := s.findSpousePair(ctx, input.FromPersonID, input.ToPersonID); err != nil {
			return models.Relationship{}, err
		} else if ok {
			existingStatus := maritalStatus(existing.Metadata)
			if existingStatus == "divorced" && requestStatus == "married" {
				if err := s.ensureNoOtherSpouse(ctx, input.FromPersonID, input.ToPersonID); err != nil {
					return models.Relationship{}, err
				}
				return s.updateMetadata(ctx, existing.ID, marriedMetadata())
			}
			if existing.FromPersonID != input.FromPersonID || existing.ToPersonID != input.ToPersonID {
				return s.normalizeSpouseDirection(ctx, existing.ID, input.FromPersonID, input.ToPersonID)
			}
			return existing, nil
		}

		if requestStatus != "divorced" {
			if err := s.ensureNoOtherSpouse(ctx, input.FromPersonID, input.ToPersonID); err != nil {
				return models.Relationship{}, err
			}
		}
	}

	var relationship models.Relationship
	err := s.db.Pool().QueryRow(ctx, `
		INSERT INTO relationships (from_person_id, to_person_id, type, metadata)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (from_person_id, to_person_id, type) DO UPDATE
		SET metadata = relationships.metadata
		RETURNING id, from_person_id, to_person_id, type, metadata, created_at
	`, input.FromPersonID, input.ToPersonID, input.Type, input.Metadata).Scan(
		&relationship.ID,
		&relationship.FromPersonID,
		&relationship.ToPersonID,
		&relationship.Type,
		&relationship.Metadata,
		&relationship.CreatedAt,
	)
	return relationship, err
}

func (s *service) findSpousePair(ctx context.Context, a, b uuid.UUID) (models.Relationship, bool, error) {
	var relationship models.Relationship
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id, from_person_id, to_person_id, type, metadata, created_at
		FROM relationships
		WHERE type = 'spouse'
		  AND (
		    (from_person_id = $1 AND to_person_id = $2)
		    OR (from_person_id = $2 AND to_person_id = $1)
		  )
	`, a, b).Scan(
		&relationship.ID,
		&relationship.FromPersonID,
		&relationship.ToPersonID,
		&relationship.Type,
		&relationship.Metadata,
		&relationship.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Relationship{}, false, nil
	}
	if err != nil {
		return models.Relationship{}, false, err
	}
	return relationship, true, nil
}

func (s *service) normalizeSpouseDirection(ctx context.Context, relationshipID, fromID, toID uuid.UUID) (models.Relationship, error) {
	var relationship models.Relationship
	err := s.db.Pool().QueryRow(ctx, `
		UPDATE relationships
		SET from_person_id = $2, to_person_id = $3
		WHERE id = $1 AND type = 'spouse'
		RETURNING id, from_person_id, to_person_id, type, metadata, created_at
	`, relationshipID, fromID, toID).Scan(
		&relationship.ID,
		&relationship.FromPersonID,
		&relationship.ToPersonID,
		&relationship.Type,
		&relationship.Metadata,
		&relationship.CreatedAt,
	)
	return relationship, err
}

func (s *service) ensureNoOtherSpouse(ctx context.Context, personID, partnerID uuid.UUID) error {
	for _, id := range []uuid.UUID{personID, partnerID} {
		otherID, name, err := s.findOtherSpouse(ctx, id, partnerID)
		if err != nil {
			return err
		}
		if otherID != uuid.Nil {
			return fmt.Errorf("%w: already married to %s", ErrExistingSpouse, name)
		}
	}
	return nil
}

func (s *service) updateMetadata(ctx context.Context, relationshipID uuid.UUID, metadata json.RawMessage) (models.Relationship, error) {
	var relationship models.Relationship
	err := s.db.Pool().QueryRow(ctx, `
		UPDATE relationships
		SET metadata = $2
		WHERE id = $1
		RETURNING id, from_person_id, to_person_id, type, metadata, created_at
	`, relationshipID, metadata).Scan(
		&relationship.ID,
		&relationship.FromPersonID,
		&relationship.ToPersonID,
		&relationship.Type,
		&relationship.Metadata,
		&relationship.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Relationship{}, errors.New("relationship not found")
	}
	return relationship, err
}

func (s *service) Update(ctx context.Context, relationshipID uuid.UUID, input UpdateInput) (models.Relationship, error) {
	existing, err := s.getByID(ctx, relationshipID)
	if err != nil {
		return models.Relationship{}, err
	}

	if input.Metadata == nil {
		input.Metadata = json.RawMessage(`{}`)
	}

	if existing.Type == models.RelationshipSpouse && maritalStatus(input.Metadata) == "divorced" {
		// Marking divorced — no active-spouse conflict check needed.
	} else if existing.Type == models.RelationshipSpouse && maritalStatus(input.Metadata) == "married" {
		if err := s.ensureNoOtherSpouse(ctx, existing.FromPersonID, existing.ToPersonID); err != nil {
			return models.Relationship{}, err
		}
	}

	return s.updateMetadata(ctx, relationshipID, input.Metadata)
}

func (s *service) findOtherSpouse(ctx context.Context, personID, allowedPartnerID uuid.UUID) (uuid.UUID, string, error) {
	var otherID uuid.UUID
	err := s.db.Pool().QueryRow(ctx, `
		SELECT CASE
		         WHEN r.from_person_id = $1 THEN r.to_person_id
		         ELSE r.from_person_id
		       END
		FROM relationships r
		WHERE r.type = 'spouse'
		  AND `+activeSpouseSQL+`
		  AND (r.from_person_id = $1 OR r.to_person_id = $1)
		  AND NOT (
		    (r.from_person_id = $1 AND r.to_person_id = $2)
		    OR (r.from_person_id = $2 AND r.to_person_id = $1)
		  )
		LIMIT 1
	`, personID, allowedPartnerID).Scan(&otherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", nil
	}
	if err != nil {
		return uuid.Nil, "", err
	}

	var name string
	err = s.db.Pool().QueryRow(ctx, `SELECT given_name FROM persons WHERE id = $1`, otherID).Scan(&name)
	if err != nil {
		return otherID, "someone else", nil
	}
	return otherID, name, nil
}

func (s *service) hasParentLink(ctx context.Context, a, b uuid.UUID) (bool, error) {
	var linked bool
	err := s.db.Pool().QueryRow(ctx, `
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

func (s *service) Delete(ctx context.Context, relationshipID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `DELETE FROM relationships WHERE id = $1`, relationshipID)
	return err
}

func (s *service) ListForPerson(ctx context.Context, personID uuid.UUID) ([]models.Relationship, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, from_person_id, to_person_id, type, metadata, created_at
		FROM relationships
		WHERE from_person_id = $1 OR to_person_id = $1
	`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRelationships(rows)
}

func (s *service) ListForFamily(ctx context.Context, familyID uuid.UUID) ([]models.Relationship, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT r.id, r.from_person_id, r.to_person_id, r.type, r.metadata, r.created_at
		FROM relationships r
		WHERE r.from_person_id IN (
			SELECT person_id FROM person_families WHERE family_id = $1
		)
		OR r.to_person_id IN (
			SELECT person_id FROM person_families WHERE family_id = $1
		)
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRelationships(rows)
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanRelationships(rows rowScanner) ([]models.Relationship, error) {
	var relationships []models.Relationship
	for rows.Next() {
		var relationship models.Relationship
		if err := rows.Scan(
			&relationship.ID,
			&relationship.FromPersonID,
			&relationship.ToPersonID,
			&relationship.Type,
			&relationship.Metadata,
			&relationship.CreatedAt,
		); err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return relationships, nil
}

func (s *service) getByID(ctx context.Context, relationshipID uuid.UUID) (models.Relationship, error) {
	var relationship models.Relationship
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id, from_person_id, to_person_id, type, metadata, created_at
		FROM relationships WHERE id = $1
	`, relationshipID).Scan(
		&relationship.ID,
		&relationship.FromPersonID,
		&relationship.ToPersonID,
		&relationship.Type,
		&relationship.Metadata,
		&relationship.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Relationship{}, errors.New("relationship not found")
	}
	return relationship, err
}