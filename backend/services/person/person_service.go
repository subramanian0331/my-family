package person

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	familyservice "github.com/subbu/family_tree/services/family"
	"github.com/subbu/family_tree/models"
)

type service struct {
	db postgresclient.Client
}

func NewService(db postgresclient.Client) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, input CreateInput) (models.Person, error) {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return models.Person{}, err
	}
	defer tx.Rollback(ctx)

	var person models.Person
	err = tx.QueryRow(ctx, `
		INSERT INTO persons (
			given_name, patronymic, clan_name, gender,
			birth_date, death_date, birth_place, death_place, notes, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, given_name, patronymic, clan_name, gender,
		          birth_date, death_date, birth_place, death_place, notes,
		          created_by, created_at, updated_at
	`, input.GivenName, input.Patronymic, input.ClanName, input.Gender,
		input.BirthDate, input.DeathDate, input.BirthPlace, input.DeathPlace, input.Notes, input.CreatedBy,
	).Scan(
		&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
		&person.BirthDate, &person.DeathDate, &person.BirthPlace, &person.DeathPlace, &person.Notes,
		&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt,
	)
	if err != nil {
		return models.Person{}, err
	}

	for _, familyID := range input.FamilyIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO person_families (person_id, family_id) VALUES ($1, $2)
		`, person.ID, familyID); err != nil {
			return models.Person{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Person{}, err
	}
	return person, nil
}

func (s *service) GetByID(ctx context.Context, personID uuid.UUID) (models.Person, error) {
	row := s.db.Pool().QueryRow(ctx, `
		SELECT p.id, p.given_name, p.patronymic, p.clan_name, p.gender,
		       p.birth_date, p.death_date, p.deceased, p.birth_place, p.death_place, p.notes,
		       p.created_by, p.created_at, p.updated_at,
		       (SELECT ph.id FROM photos ph WHERE ph.person_id = p.id ORDER BY ph.created_at DESC LIMIT 1)
		FROM persons p
		WHERE p.id = $1
	`, personID)
	return scanPerson(row)
}

func (s *service) Update(ctx context.Context, personID uuid.UUID, input UpdateInput) (models.Person, error) {
	current, err := s.GetByID(ctx, personID)
	if err != nil {
		return models.Person{}, err
	}

	applyUpdate(&current, input)

	row := s.db.Pool().QueryRow(ctx, `
		UPDATE persons
		SET given_name = $2, patronymic = $3, clan_name = $4, gender = $5,
		    birth_date = $6, death_date = $7, deceased = $8, birth_place = $9, death_place = $10,
		    notes = $11, updated_at = now()
		WHERE id = $1
		RETURNING id, given_name, patronymic, clan_name, gender,
		          birth_date, death_date, deceased, birth_place, death_place, notes,
		          created_by, created_at, updated_at,
		          (SELECT ph.id FROM photos ph WHERE ph.person_id = persons.id ORDER BY ph.created_at DESC LIMIT 1)
	`, personID, current.GivenName, current.Patronymic, current.ClanName, current.Gender,
		current.BirthDate, current.DeathDate, current.Deceased, current.BirthPlace, current.DeathPlace, current.Notes,
	)
	return scanPerson(row)
}

func (s *service) Delete(ctx context.Context, personID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `DELETE FROM persons WHERE id = $1`, personID)
	return err
}

const activeSpouseFilterSQL = `COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'`

const personSpouseSelectSQL = `
	EXISTS (
		SELECT 1 FROM relationships r
		WHERE r.type = 'spouse'
		  AND ` + activeSpouseFilterSQL + `
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
		  AND ` + activeSpouseFilterSQL + `
		  AND (r.from_person_id = p.id OR r.to_person_id = p.id)
		LIMIT 1
	), '') AS spouse_name`

func (s *service) ListByFamily(ctx context.Context, familyID uuid.UUID) ([]models.Person, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT p.id, p.given_name, p.patronymic, p.clan_name, p.gender,
		       p.birth_date, p.death_date, p.deceased, p.birth_place, p.death_place, p.notes,
		       p.created_by, p.created_at, p.updated_at,
		       (SELECT ph.id FROM photos ph WHERE ph.person_id = p.id ORDER BY ph.created_at DESC LIMIT 1),
		       pf.via_marriage,
		       `+personSpouseSelectSQL+`
		FROM persons p
		JOIN person_families pf ON pf.person_id = p.id
		WHERE pf.family_id = $1
		ORDER BY p.given_name ASC
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		person, err := scanFamilyPerson(rows)
		if err != nil {
			return nil, err
		}
		persons = append(persons, person)
	}
	return persons, rows.Err()
}

func (s *service) AddFamilyLabel(ctx context.Context, personID, familyID uuid.UUID, viaMarriage bool) error {
	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO person_families (person_id, family_id, via_marriage) VALUES ($1, $2, $3)
		ON CONFLICT (person_id, family_id) DO UPDATE
		SET via_marriage = person_families.via_marriage OR EXCLUDED.via_marriage
	`, personID, familyID, viaMarriage)
	return err
}

func (s *service) SetFamilyMarriageLabel(ctx context.Context, personID, familyID uuid.UUID, viaMarriage bool) error {
	tag, err := s.db.Pool().Exec(ctx, `
		UPDATE person_families
		SET via_marriage = $3
		WHERE person_id = $1 AND family_id = $2
	`, personID, familyID, viaMarriage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("person is not in this family")
	}
	return nil
}

func (s *service) HasFamilyLabel(ctx context.Context, personID, familyID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.Pool().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM person_families WHERE person_id = $1 AND family_id = $2
		)
	`, personID, familyID).Scan(&exists)
	return exists, err
}

func (s *service) ShouldBeMarryInToFamily(ctx context.Context, personID, familyID uuid.UUID) (bool, error) {
	familyName, err := s.familyName(ctx, familyID)
	if err != nil {
		return false, err
	}
	return s.shouldBeMarryInToFamily(ctx, personID, familyName)
}

func (s *service) familyName(ctx context.Context, familyID uuid.UUID) (string, error) {
	var name string
	err := s.db.Pool().QueryRow(ctx, `SELECT name FROM families WHERE id = $1`, familyID).Scan(&name)
	return name, err
}

func (s *service) shouldBeMarryInToFamily(ctx context.Context, personID uuid.UUID, familyName string) (bool, error) {
	var givenName, patronymic, clanName string
	err := s.db.Pool().QueryRow(ctx, `
		SELECT given_name, patronymic, clan_name FROM persons WHERE id = $1
	`, personID).Scan(&givenName, &patronymic, &clanName)
	if err != nil {
		return false, err
	}
	derived := familyservice.DeriveFamilyName(givenName, patronymic, clanName)
	return !strings.EqualFold(strings.TrimSpace(derived), strings.TrimSpace(familyName)), nil
}

func (s *service) PrimaryNativeFamilyID(ctx context.Context, personID uuid.UUID) (uuid.UUID, error) {
	var familyID uuid.UUID
	err := s.db.Pool().QueryRow(ctx, `
		SELECT family_id
		FROM person_families
		WHERE person_id = $1 AND via_marriage = false
		ORDER BY created_at ASC
		LIMIT 1
	`, personID).Scan(&familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	return familyID, err
}

func (s *service) IsNativeInFamily(ctx context.Context, personID, familyID uuid.UUID) (bool, error) {
	var native bool
	err := s.db.Pool().QueryRow(ctx, `
		SELECT NOT via_marriage
		FROM person_families
		WHERE person_id = $1 AND family_id = $2
	`, personID, familyID).Scan(&native)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return native, err
}

func (s *service) RemoveFamilyLabel(ctx context.Context, personID, familyID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `
		DELETE FROM person_families WHERE person_id = $1 AND family_id = $2
	`, personID, familyID)
	return err
}

func (s *service) ListFamiliesForPerson(ctx context.Context, personID uuid.UUID) ([]models.PersonFamilyRef, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT f.id, f.name, pf.via_marriage
		FROM families f
		JOIN person_families pf ON pf.family_id = f.id
		WHERE pf.person_id = $1
		ORDER BY f.name ASC
	`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var families []models.PersonFamilyRef
	for rows.Next() {
		var ref models.PersonFamilyRef
		if err := rows.Scan(&ref.ID, &ref.Name, &ref.MarriedIn); err != nil {
			return nil, err
		}
		families = append(families, ref)
	}
	return families, rows.Err()
}

func (s *service) UserCanAccess(ctx context.Context, userID, personID uuid.UUID) error {
	var allowed bool
	err := s.db.Pool().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM person_families pf
			JOIN family_members fm ON fm.family_id = pf.family_id
			WHERE pf.person_id = $1 AND fm.user_id = $2
		)
	`, personID, userID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("forbidden")
	}
	return nil
}

func (s *service) SuggestPatronymic(ctx context.Context, personID uuid.UUID) (string, error) {
	var givenName string
	err := s.db.Pool().QueryRow(ctx, `
		SELECT parent.given_name
		FROM relationships r
		JOIN persons parent ON parent.id = r.to_person_id
		WHERE r.from_person_id = $1 AND r.type = 'parent'
		ORDER BY parent.created_at ASC
		LIMIT 1
	`, personID).Scan(&givenName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return givenName, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanPerson(row scannable) (models.Person, error) {
	var person models.Person
	err := row.Scan(
		&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
		&person.BirthDate, &person.DeathDate, &person.Deceased, &person.BirthPlace, &person.DeathPlace, &person.Notes,
		&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt, &person.PhotoID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Person{}, errors.New("person not found")
	}
	return person, err
}

func scanFamilyPerson(row scannable) (models.Person, error) {
	var person models.Person
	var viaMarriage bool
	err := row.Scan(
		&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
		&person.BirthDate, &person.DeathDate, &person.Deceased, &person.BirthPlace, &person.DeathPlace, &person.Notes,
		&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt, &person.PhotoID,
		&viaMarriage,
		&person.HasSpouse,
		&person.SpouseName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Person{}, errors.New("person not found")
		}
		return models.Person{}, err
	}
	person.MarriedIn = viaMarriage
	return person, nil
}

func applyUpdate(person *models.Person, input UpdateInput) {
	if input.GivenName != nil {
		person.GivenName = *input.GivenName
	}
	if input.Patronymic != nil {
		person.Patronymic = *input.Patronymic
	}
	if input.ClanName != nil {
		person.ClanName = *input.ClanName
	}
	if input.Gender != nil {
		person.Gender = *input.Gender
	}
	if input.BirthDate != nil {
		person.BirthDate = *input.BirthDate
	}
	if input.DeathDate != nil {
		person.DeathDate = *input.DeathDate
	}
	if input.Deceased != nil {
		person.Deceased = *input.Deceased
	}
	if input.BirthPlace != nil {
		person.BirthPlace = *input.BirthPlace
	}
	if input.DeathPlace != nil {
		person.DeathPlace = *input.DeathPlace
	}
	if input.Notes != nil {
		person.Notes = *input.Notes
	}
	if input.PhotoID != nil {
		person.PhotoID = *input.PhotoID
	}
}