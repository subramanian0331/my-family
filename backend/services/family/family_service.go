package family

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/models"
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type service struct {
	db postgresclient.Client
}

func NewService(db postgresclient.Client) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, input CreateInput) (models.Family, error) {
	slug := uniqueSlug(input.Name)
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return models.Family{}, err
	}
	defer tx.Rollback(ctx)

	var family models.Family
	err = tx.QueryRow(ctx, `
		INSERT INTO families (name, slug, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, description, created_by, created_at, updated_at
	`, input.Name, slug, input.Description, input.CreatedBy).Scan(
		&family.ID,
		&family.Name,
		&family.Slug,
		&family.Description,
		&family.CreatedBy,
		&family.CreatedAt,
		&family.UpdatedAt,
	)
	if err != nil {
		return models.Family{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO family_members (family_id, user_id, role)
		VALUES ($1, $2, $3)
	`, family.ID, input.CreatedBy, models.FamilyRoleOwner)
	if err != nil {
		return models.Family{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Family{}, err
	}
	return family, nil
}

func (s *service) GetByID(ctx context.Context, familyID uuid.UUID) (models.Family, error) {
	row := s.db.Pool().QueryRow(ctx, `
		SELECT id, name, slug, description, created_by, created_at, updated_at
		FROM families WHERE id = $1
	`, familyID)
	return scanFamily(row)
}

func (s *service) Update(ctx context.Context, familyID uuid.UUID, input UpdateInput) (models.Family, error) {
	current, err := s.GetByID(ctx, familyID)
	if err != nil {
		return models.Family{}, err
	}

	name := current.Name
	description := current.Description
	if input.Name != nil {
		name = *input.Name
	}
	if input.Description != nil {
		description = *input.Description
	}

	row := s.db.Pool().QueryRow(ctx, `
		UPDATE families
		SET name = $2, description = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, description, created_by, created_at, updated_at
	`, familyID, name, description)
	return scanFamily(row)
}

func (s *service) Delete(ctx context.Context, familyID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `DELETE FROM families WHERE id = $1`, familyID)
	return err
}

func (s *service) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.FamilySummary, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT f.id, f.name, f.slug, f.description, f.created_by, f.created_at, f.updated_at, fm.role
		FROM families f
		JOIN family_members fm ON fm.family_id = f.id
		WHERE fm.user_id = $1
		ORDER BY f.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var families []models.FamilySummary
	for rows.Next() {
		var summary models.FamilySummary
		if err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.Slug,
			&summary.Description,
			&summary.CreatedBy,
			&summary.CreatedAt,
			&summary.UpdatedAt,
			&summary.Role,
		); err != nil {
			return nil, err
		}
		families = append(families, summary)
	}
	return families, rows.Err()
}

func (s *service) ListAll(ctx context.Context) ([]models.Family, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, name, slug, description, created_by, created_at, updated_at
		FROM families ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var families []models.Family
	for rows.Next() {
		family, err := scanFamily(rows)
		if err != nil {
			return nil, err
		}
		families = append(families, family)
	}
	return families, rows.Err()
}

func (s *service) ListMembers(ctx context.Context, familyID uuid.UUID) ([]MemberWithUser, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT u.id, u.email, u.name, u.avatar_url, fm.role
		FROM family_members fm
		JOIN users u ON u.id = fm.user_id
		WHERE fm.family_id = $1
		ORDER BY fm.role ASC, u.name ASC
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []MemberWithUser
	for rows.Next() {
		var member MemberWithUser
		if err := rows.Scan(&member.UserID, &member.Email, &member.Name, &member.AvatarURL, &member.Role); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *service) ListMembershipsForUser(ctx context.Context, userID uuid.UUID) ([]models.AdminFamilyAccess, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT f.id, f.name, fm.role
		FROM family_members fm
		JOIN families f ON f.id = fm.family_id
		WHERE fm.user_id = $1
		ORDER BY f.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []models.AdminFamilyAccess{}
	for rows.Next() {
		var access models.AdminFamilyAccess
		if err := rows.Scan(&access.FamilyID, &access.FamilyName, &access.Role); err != nil {
			return nil, err
		}
		memberships = append(memberships, access)
	}
	return memberships, rows.Err()
}

func (s *service) SetMemberRole(ctx context.Context, familyID, userID uuid.UUID, role models.FamilyRole) error {
	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO family_members (family_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (family_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, familyID, userID, role)
	return err
}

func (s *service) RemoveMember(ctx context.Context, familyID, userID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `
		DELETE FROM family_members WHERE family_id = $1 AND user_id = $2
	`, familyID, userID)
	return err
}

func (s *service) CountAll(ctx context.Context) (int, error) {
	var count int
	err := s.db.Pool().QueryRow(ctx, `SELECT count(*) FROM families`).Scan(&count)
	return count, err
}

func (s *service) UserRole(ctx context.Context, familyID, userID uuid.UUID) (models.FamilyRole, error) {
	var role models.FamilyRole
	err := s.db.Pool().QueryRow(ctx, `
		SELECT role FROM family_members WHERE family_id = $1 AND user_id = $2
	`, familyID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("not a family member")
	}
	return role, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFamily(row scannable) (models.Family, error) {
	var family models.Family
	err := row.Scan(
		&family.ID,
		&family.Name,
		&family.Slug,
		&family.Description,
		&family.CreatedBy,
		&family.CreatedAt,
		&family.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Family{}, errors.New("family not found")
	}
	return family, err
}

func uniqueSlug(name string) string {
	base := strings.Trim(slugSanitizer.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if base == "" {
		base = "family"
	}
	return base + "-" + uuid.NewString()[:8]
}