package invite

import (
	"context"
	"errors"
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

func (s *service) Create(ctx context.Context, input CreateInput) (models.Invite, error) {
	token := uuid.NewString()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	var invite models.Invite
	err := s.db.Pool().QueryRow(ctx, `
		INSERT INTO invites (family_id, email, role, token, expires_at, created_by)
		VALUES ($1, lower($2), $3, $4, $5, $6)
		RETURNING id, family_id, email, role, token, expires_at, accepted_at, created_by, created_at
	`, input.FamilyID, input.Email, input.Role, token, expiresAt, input.CreatedBy).Scan(
		&invite.ID,
		&invite.FamilyID,
		&invite.Email,
		&invite.Role,
		&invite.Token,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.CreatedBy,
		&invite.CreatedAt,
	)
	return invite, err
}

func (s *service) GetByID(ctx context.Context, inviteID uuid.UUID) (models.Invite, error) {
	var invite models.Invite
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id, family_id, email, role, token, expires_at, accepted_at, created_by, created_at
		FROM invites WHERE id = $1
	`, inviteID).Scan(
		&invite.ID,
		&invite.FamilyID,
		&invite.Email,
		&invite.Role,
		&invite.Token,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.CreatedBy,
		&invite.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Invite{}, errors.New("invite not found")
	}
	return invite, err
}

func (s *service) Accept(ctx context.Context, token string, user models.User) error {
	var invite models.Invite
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id, family_id, email, role, token, expires_at, accepted_at, created_by, created_at
		FROM invites
		WHERE token = $1
	`, token).Scan(
		&invite.ID,
		&invite.FamilyID,
		&invite.Email,
		&invite.Role,
		&invite.Token,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.CreatedBy,
		&invite.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invite not found")
	}
	if err != nil {
		return err
	}
	if invite.AcceptedAt != nil {
		return errors.New("invite already accepted")
	}
	if time.Now().After(invite.ExpiresAt) {
		return errors.New("invite expired")
	}
	if !strings.EqualFold(invite.Email, user.Email) {
		return errors.New("google account email does not match invite")
	}

	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO family_members (family_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (family_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, invite.FamilyID, user.ID, invite.Role)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE invites SET accepted_at = now() WHERE id = $1
	`, invite.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *service) ListPendingForEmail(ctx context.Context, email string) ([]models.Invite, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, family_id, email, role, token, expires_at, accepted_at, created_by, created_at
		FROM invites
		WHERE lower(email) = lower($1) AND accepted_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []models.Invite
	for rows.Next() {
		var invite models.Invite
		if err := rows.Scan(
			&invite.ID,
			&invite.FamilyID,
			&invite.Email,
			&invite.Role,
			&invite.Token,
			&invite.ExpiresAt,
			&invite.AcceptedAt,
			&invite.CreatedBy,
			&invite.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *service) ListForFamily(ctx context.Context, familyID uuid.UUID) ([]models.Invite, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, family_id, email, role, token, expires_at, accepted_at, created_by, created_at
		FROM invites
		WHERE family_id = $1 AND accepted_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
	`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []models.Invite
	for rows.Next() {
		var invite models.Invite
		if err := rows.Scan(
			&invite.ID,
			&invite.FamilyID,
			&invite.Email,
			&invite.Role,
			&invite.Token,
			&invite.ExpiresAt,
			&invite.AcceptedAt,
			&invite.CreatedBy,
			&invite.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *service) Revoke(ctx context.Context, inviteID uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx, `DELETE FROM invites WHERE id = $1 AND accepted_at IS NULL`, inviteID)
	return err
}