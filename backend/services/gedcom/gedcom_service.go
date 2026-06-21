package gedcom

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

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

type gedcomIndividual struct {
	XRef       string
	GivenName  string
	Patronymic string
	ClanName   string
	Gender     string
	BirthDate  *time.Time
	DeathDate  *time.Time
	BirthPlace string
	DeathPlace string
	Notes      string
}

type gedcomFamily struct {
	XRef    string
	Husband string
	Wife    string
	Child   []string
}

func (s *service) ExportFamily(ctx context.Context, familyID uuid.UUID) ([]byte, error) {
	persons, err := s.listPersons(ctx, familyID)
	if err != nil {
		return nil, err
	}
	relationships, err := s.listRelationships(ctx, familyID)
	if err != nil {
		return nil, err
	}

	personXRef := make(map[uuid.UUID]string, len(persons))
	for i, person := range persons {
		personXRef[person.ID] = fmt.Sprintf("@I%d@", i+1)
	}

	var buf bytes.Buffer
	buf.WriteString("0 HEAD\n1 SOUR FamilyTree\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n")

	for i, person := range persons {
		xref := fmt.Sprintf("@I%d@", i+1)
		buf.WriteString(fmt.Sprintf("0 %s INDI\n", xref))
		buf.WriteString(fmt.Sprintf("1 NAME %s /%s/\n", person.GivenName, person.Patronymic))
		if person.ClanName != "" {
			buf.WriteString(fmt.Sprintf("2 SURN %s\n", person.ClanName))
		}
		if person.Gender != "" {
			sex := "U"
			switch strings.ToLower(person.Gender) {
			case "male", "m":
				sex = "M"
			case "female", "f":
				sex = "F"
			}
			buf.WriteString(fmt.Sprintf("1 SEX %s\n", sex))
		}
		if person.BirthDate != nil {
			buf.WriteString(fmt.Sprintf("1 BIRT\n2 DATE %s\n", person.BirthDate.Format("2 Jan 2006")))
			if person.BirthPlace != "" {
				buf.WriteString(fmt.Sprintf("2 PLAC %s\n", person.BirthPlace))
			}
		}
		if person.DeathDate != nil {
			buf.WriteString(fmt.Sprintf("1 DEAT\n2 DATE %s\n", person.DeathDate.Format("2 Jan 2006")))
			if person.DeathPlace != "" {
				buf.WriteString(fmt.Sprintf("2 PLAC %s\n", person.DeathPlace))
			}
		}
		if person.Notes != "" {
			buf.WriteString(fmt.Sprintf("1 NOTE %s\n", person.Notes))
		}
	}

	famIndex := 1
	spousePairs := make(map[string]bool)
	for _, rel := range relationships {
		if rel.Type != models.RelationshipSpouse {
			continue
		}
		key := spouseKey(rel.FromPersonID, rel.ToPersonID)
		if spousePairs[key] {
			continue
		}
		spousePairs[key] = true
		famXRef := fmt.Sprintf("@F%d@", famIndex)
		famIndex++
		buf.WriteString(fmt.Sprintf("0 %s FAM\n", famXRef))
		buf.WriteString(fmt.Sprintf("1 HUSB %s\n", personXRef[rel.FromPersonID]))
		buf.WriteString(fmt.Sprintf("1 WIFE %s\n", personXRef[rel.ToPersonID]))
	}

	for _, rel := range relationships {
		if rel.Type != models.RelationshipParent {
			continue
		}
		famXRef := fmt.Sprintf("@F%d@", famIndex)
		famIndex++
		buf.WriteString(fmt.Sprintf("0 %s FAM\n", famXRef))
		buf.WriteString(fmt.Sprintf("1 CHIL %s\n", personXRef[rel.FromPersonID]))
		buf.WriteString(fmt.Sprintf("1 HUSB %s\n", personXRef[rel.ToPersonID]))
	}

	buf.WriteString("0 TRLR\n")
	return buf.Bytes(), nil
}

func (s *service) PreviewImport(ctx context.Context, familyID uuid.UUID, reader io.Reader) (ImportPreview, error) {
	individuals, _, err := parseGEDCOM(reader)
	if err != nil {
		return ImportPreview{}, err
	}

	existing, err := s.listPersons(ctx, familyID)
	if err != nil {
		return ImportPreview{}, err
	}

	matches := 0
	for _, indi := range individuals {
		if matchExisting(indi, existing) {
			matches++
		}
	}

	return ImportPreview{
		NewPeople:       len(individuals) - matches,
		ExistingMatches: matches,
	}, nil
}

func (s *service) CommitImport(ctx context.Context, familyID, actorID uuid.UUID, reader io.Reader) error {
	individuals, families, err := parseGEDCOM(reader)
	if err != nil {
		return err
	}

	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	xrefToID := make(map[string]uuid.UUID, len(individuals))
	for _, indi := range individuals {
		var personID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO persons (
				given_name, patronymic, clan_name, gender,
				birth_date, death_date, birth_place, death_place, notes, created_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id
		`, indi.GivenName, indi.Patronymic, indi.ClanName, indi.Gender,
			indi.BirthDate, indi.DeathDate, indi.BirthPlace, indi.DeathPlace, indi.Notes, actorID,
		).Scan(&personID)
		if err != nil {
			return err
		}
		xrefToID[indi.XRef] = personID

		if _, err := tx.Exec(ctx, `
			INSERT INTO person_families (person_id, family_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, personID, familyID); err != nil {
			return err
		}
	}

	for _, fam := range families {
		if fam.Husband != "" && fam.Wife != "" {
			if err := insertRelationship(ctx, tx, xrefToID, fam.Husband, fam.Wife, models.RelationshipSpouse); err != nil {
				return err
			}
		}
		for _, child := range fam.Child {
			parent := fam.Husband
			if parent == "" {
				parent = fam.Wife
			}
			if parent == "" || child == "" {
				continue
			}
			if err := insertRelationship(ctx, tx, xrefToID, child, parent, models.RelationshipParent); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func insertRelationship(ctx context.Context, tx pgx.Tx, xrefToID map[string]uuid.UUID, fromXRef, toXRef string, relType models.RelationshipType) error {
	fromID, ok := xrefToID[fromXRef]
	if !ok {
		return nil
	}
	toID, ok := xrefToID[toXRef]
	if !ok {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO relationships (from_person_id, to_person_id, type)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, fromID, toID, relType)
	return err
}

func (s *service) listPersons(ctx context.Context, familyID uuid.UUID) ([]models.Person, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT p.id, p.given_name, p.patronymic, p.clan_name, p.gender,
		       p.birth_date, p.death_date, p.birth_place, p.death_place, p.notes,
		       p.created_by, p.created_at, p.updated_at
		FROM persons p
		JOIN person_families pf ON pf.person_id = p.id
		WHERE pf.family_id = $1
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		var person models.Person
		if err := rows.Scan(
			&person.ID, &person.GivenName, &person.Patronymic, &person.ClanName, &person.Gender,
			&person.BirthDate, &person.DeathDate, &person.BirthPlace, &person.DeathPlace, &person.Notes,
			&person.CreatedBy, &person.CreatedAt, &person.UpdatedAt,
		); err != nil {
			return nil, err
		}
		persons = append(persons, person)
	}
	return persons, rows.Err()
}

func (s *service) listRelationships(ctx context.Context, familyID uuid.UUID) ([]models.Relationship, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT r.id, r.from_person_id, r.to_person_id, r.type, r.metadata, r.created_at
		FROM relationships r
		WHERE r.from_person_id IN (SELECT person_id FROM person_families WHERE family_id = $1)
		  AND r.to_person_id IN (SELECT person_id FROM person_families WHERE family_id = $1)
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.Relationship
	for rows.Next() {
		var rel models.Relationship
		if err := rows.Scan(&rel.ID, &rel.FromPersonID, &rel.ToPersonID, &rel.Type, &rel.Metadata, &rel.CreatedAt); err != nil {
			return nil, err
		}
		relationships = append(relationships, rel)
	}
	return relationships, rows.Err()
}

func parseGEDCOM(reader io.Reader) (map[string]gedcomIndividual, map[string]gedcomFamily, error) {
	individuals := make(map[string]*gedcomIndividual)
	families := make(map[string]*gedcomFamily)

	scanner := bufio.NewScanner(reader)
	var currentIndi *gedcomIndividual
	var currentFam *gedcomFamily
	var currentTag string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		level := parts[0]
		tag := strings.ToUpper(parts[1])
		value := ""
		if len(parts) > 2 {
			value = strings.Join(parts[2:], " ")
		}

		switch level {
		case "0":
			if strings.HasPrefix(tag, "@") && len(parts) >= 3 {
				xref := tag
				recordType := strings.ToUpper(parts[2])
				if recordType == "INDI" {
					individuals[xref] = &gedcomIndividual{XRef: xref}
					currentIndi = individuals[xref]
					currentFam = nil
				} else if recordType == "FAM" {
					families[xref] = &gedcomFamily{XRef: xref}
					currentFam = families[xref]
					currentIndi = nil
				}
			} else {
				currentIndi = nil
				currentFam = nil
			}
			currentTag = tag
		case "1":
			currentTag = tag
			if currentIndi != nil {
				switch tag {
				case "NAME":
					given, patronymic := parseName(value)
					currentIndi.GivenName = given
					currentIndi.Patronymic = patronymic
				case "SEX":
					currentIndi.Gender = value
				case "NOTE":
					currentIndi.Notes = value
				}
			}
			if currentFam != nil {
				switch tag {
				case "HUSB":
					currentFam.Husband = value
				case "WIFE":
					currentFam.Wife = value
				case "CHIL":
					currentFam.Child = append(currentFam.Child, value)
				}
			}
		case "2":
			if currentIndi != nil {
				switch currentTag {
				case "BIRT":
					if tag == "DATE" {
						currentIndi.BirthDate = parseGEDCOMDate(value)
					}
					if tag == "PLAC" {
						currentIndi.BirthPlace = value
					}
				case "DEAT":
					if tag == "DATE" {
						currentIndi.DeathDate = parseGEDCOMDate(value)
					}
					if tag == "PLAC" {
						currentIndi.DeathPlace = value
					}
				case "NAME":
					if tag == "SURN" {
						currentIndi.ClanName = value
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	resultIndividuals := make(map[string]gedcomIndividual, len(individuals))
	for xref, indi := range individuals {
		resultIndividuals[xref] = *indi
	}
	resultFamilies := make(map[string]gedcomFamily, len(families))
	for xref, fam := range families {
		resultFamilies[xref] = *fam
	}
	return resultIndividuals, resultFamilies, nil
}

func parseName(value string) (string, string) {
	parts := strings.Split(value, "/")
	given := strings.TrimSpace(parts[0])
	patronymic := ""
	if len(parts) > 1 {
		patronymic = strings.TrimSpace(parts[1])
	}
	return given, patronymic
}

func parseGEDCOMDate(value string) *time.Time {
	formats := []string{"2 Jan 2006", "Jan 2006", "2006"}
	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return &t
		}
	}
	return nil
}

func matchExisting(indi gedcomIndividual, existing []models.Person) bool {
	for _, person := range existing {
		if strings.EqualFold(person.GivenName, indi.GivenName) &&
			strings.EqualFold(person.Patronymic, indi.Patronymic) {
			return true
		}
	}
	return false
}

func spouseKey(a, b uuid.UUID) string {
	if a.String() < b.String() {
		return a.String() + ":" + b.String()
	}
	return b.String() + ":" + a.String()
}