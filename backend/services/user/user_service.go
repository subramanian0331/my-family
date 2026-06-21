package user

import (
	"context"
	"errors"

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

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row := s.db.Pool().QueryRow(ctx, `
		SELECT id, google_sub, email, name, avatar_url, site_role, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

func (s *service) GetByEmail(ctx context.Context, email string) (models.User, error) {
	row := s.db.Pool().QueryRow(ctx, `
		SELECT id, google_sub, email, name, avatar_url, site_role, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
	return scanUser(row)
}

func (s *service) UpsertGoogleUser(ctx context.Context, input UpsertGoogleUserInput) (models.User, error) {
	row := s.db.Pool().QueryRow(ctx, `
		INSERT INTO users (google_sub, email, name, avatar_url, site_role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (google_sub) DO UPDATE
		SET email = EXCLUDED.email,
		    name = EXCLUDED.name,
		    avatar_url = EXCLUDED.avatar_url,
		    site_role = EXCLUDED.site_role,
		    updated_at = now()
		RETURNING id, google_sub, email, name, avatar_url, site_role, created_at, updated_at
	`, input.GoogleSub, input.Email, input.Name, input.AvatarURL, input.SiteRole)
	return scanUser(row)
}

func (s *service) List(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, google_sub, email, name, avatar_url, site_role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(row scannable) (models.User, error) {
	var user models.User
	err := row.Scan(
		&user.ID,
		&user.GoogleSub,
		&user.Email,
		&user.Name,
		&user.AvatarURL,
		&user.SiteRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, errors.New("user not found")
	}
	return user, err
}