package search

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/models"
)

type service struct {
	db postgresclient.Client
}

func NewService(db postgresclient.Client) Service {
	return &service{db: db}
}

func (s *service) SearchPeopleInFamily(ctx context.Context, familyID uuid.UUID, query string) ([]models.Person, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Person{}, nil
	}

	rows, err := s.db.Pool().Query(ctx, `
		SELECT p.id, p.given_name, p.patronymic, p.clan_name, p.gender,
		       p.birth_date, p.death_date, p.deceased, p.birth_place, p.death_place, p.notes,
		       p.created_by, p.created_at, p.updated_at,
		       (SELECT ph.id FROM photos ph WHERE ph.person_id = p.id ORDER BY ph.created_at DESC LIMIT 1),
		       EXISTS (
		         SELECT 1 FROM relationships r
		         WHERE r.type = 'spouse'
		           AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
		           AND (r.from_person_id = p.id OR r.to_person_id = p.id)
		       ) AS has_spouse,
		       COALESCE((
		         SELECT NULLIF(TRIM(CONCAT_WS(' ', sp.given_name, sp.patronymic)), '')
		         FROM relationships r
		         JOIN persons sp ON sp.id = CASE
		           WHEN r.from_person_id = p.id THEN r.to_person_id
		           ELSE r.from_person_id
		         END
		         WHERE r.type = 'spouse'
		           AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
		           AND (r.from_person_id = p.id OR r.to_person_id = p.id)
		         LIMIT 1
		       ), '') AS spouse_name
		FROM persons p
		JOIN person_families pf ON pf.person_id = p.id
		WHERE pf.family_id = $1
		  AND p.search_vector @@ plainto_tsquery('simple', $2)
		ORDER BY ts_rank(p.search_vector, plainto_tsquery('simple', $2)) DESC
		LIMIT 50
	`, familyID, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		var person models.Person
		if err := rows.Scan(
			&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
			&person.BirthDate, &person.DeathDate, &person.Deceased, &person.BirthPlace, &person.DeathPlace, &person.Notes,
			&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt, &person.PhotoID,
			&person.HasSpouse, &person.SpouseName,
		); err != nil {
			return nil, err
		}
		persons = append(persons, person)
	}
	return persons, rows.Err()
}

func (s *service) SearchPeopleForUser(
	ctx context.Context,
	userID uuid.UUID,
	query string,
	targetFamilyID *uuid.UUID,
) ([]models.PersonSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.PersonSearchHit{}, nil
	}

	targetID := uuid.Nil
	if targetFamilyID != nil {
		targetID = *targetFamilyID
	}

	rows, err := s.db.Pool().Query(ctx, `
		SELECT p.id, p.given_name, p.patronymic, p.clan_name, p.gender,
		       p.birth_date, p.death_date, p.deceased, p.birth_place, p.death_place, p.notes,
		       p.created_by, p.created_at, p.updated_at,
		       (SELECT ph.id FROM photos ph WHERE ph.person_id = p.id ORDER BY ph.created_at DESC LIMIT 1),
		       (
		         SELECT COALESCE(
		           json_agg(json_build_object('id', f.id, 'name', f.name) ORDER BY f.name),
		           '[]'::json
		         )
		         FROM person_families pf2
		         JOIN families f ON f.id = pf2.family_id
		         JOIN family_members fm2 ON fm2.family_id = pf2.family_id AND fm2.user_id = $1
		         WHERE pf2.person_id = p.id
		       ) AS families_json,
		       EXISTS (
		         SELECT 1 FROM person_families pft
		         WHERE pft.person_id = p.id AND pft.family_id = $3
		       ) AS in_target_family,
		       EXISTS (
		         SELECT 1 FROM relationships r
		         WHERE r.type = 'spouse'
		           AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
		           AND (r.from_person_id = p.id OR r.to_person_id = p.id)
		       ) AS has_spouse,
		       COALESCE((
		         SELECT NULLIF(TRIM(CONCAT_WS(' ', sp.given_name, sp.patronymic)), '')
		         FROM relationships r
		         JOIN persons sp ON sp.id = CASE
		           WHEN r.from_person_id = p.id THEN r.to_person_id
		           ELSE r.from_person_id
		         END
		         WHERE r.type = 'spouse'
		           AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
		           AND (r.from_person_id = p.id OR r.to_person_id = p.id)
		         LIMIT 1
		       ), '') AS spouse_name
		FROM persons p
		WHERE p.id IN (
			SELECT DISTINCT pf.person_id
			FROM person_families pf
			JOIN family_members fm ON fm.family_id = pf.family_id AND fm.user_id = $1
		)
		  AND p.search_vector @@ plainto_tsquery('simple', $2)
		ORDER BY ts_rank(p.search_vector, plainto_tsquery('simple', $2)) DESC
		LIMIT 50
	`, userID, query, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []models.PersonSearchHit
	for rows.Next() {
		var hit models.PersonSearchHit
		var familiesJSON []byte
		if err := rows.Scan(
			&hit.Person.ID, &hit.Person.GivenName, &hit.Person.Patronymic, &hit.Person.ClanName, &hit.Person.Gender,
			&hit.Person.BirthDate, &hit.Person.DeathDate, &hit.Person.Deceased, &hit.Person.BirthPlace, &hit.Person.DeathPlace, &hit.Person.Notes,
			&hit.Person.CreatedBy, &hit.Person.CreatedAt, &hit.Person.UpdatedAt, &hit.Person.PhotoID,
			&familiesJSON,
			&hit.InTargetFamily,
			&hit.Person.HasSpouse,
			&hit.Person.SpouseName,
		); err != nil {
			return nil, err
		}
		if len(familiesJSON) > 0 {
			if err := json.Unmarshal(familiesJSON, &hit.Families); err != nil {
				return nil, err
			}
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}