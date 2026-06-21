package photo

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	storageclient "github.com/subbu/family_tree/client/storage"
	"github.com/subbu/family_tree/models"
)

type service struct {
	db      postgresclient.Client
	storage storageclient.Client
}

func NewService(db postgresclient.Client, storage storageclient.Client) Service {
	return &service{db: db, storage: storage}
}

func (s *service) Upload(ctx context.Context, input UploadInput) (models.Photo, error) {
	storagePath, err := s.storage.Save(ctx, input.PersonID, input.Filename, input.Reader)
	if err != nil {
		return models.Photo{}, err
	}

	var photo models.Photo
	err = s.db.Pool().QueryRow(ctx, `
		INSERT INTO photos (person_id, storage_path, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, person_id, storage_path, mime_type, size_bytes, uploaded_by, created_at
	`, input.PersonID, storagePath, input.MimeType, input.SizeBytes, input.UploadedBy).Scan(
		&photo.ID,
		&photo.PersonID,
		&photo.StoragePath,
		&photo.MimeType,
		&photo.SizeBytes,
		&photo.UploadedBy,
		&photo.CreatedAt,
	)
	if err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		return models.Photo{}, err
	}
	return photo, nil
}

func (s *service) GetByID(ctx context.Context, photoID uuid.UUID) (models.Photo, error) {
	return s.scanByID(ctx, photoID)
}

func (s *service) Delete(ctx context.Context, photoID uuid.UUID) error {
	photo, err := s.scanByID(ctx, photoID)
	if err != nil {
		return err
	}
	if err := s.storage.Delete(ctx, photo.StoragePath); err != nil {
		return err
	}
	_, err = s.db.Pool().Exec(ctx, `DELETE FROM photos WHERE id = $1`, photoID)
	return err
}

func (s *service) Open(ctx context.Context, photoID uuid.UUID) (io.ReadCloser, models.Photo, error) {
	photo, err := s.scanByID(ctx, photoID)
	if err != nil {
		return nil, models.Photo{}, err
	}
	file, err := s.storage.Open(ctx, photo.StoragePath)
	if err != nil {
		return nil, models.Photo{}, err
	}
	return file, photo, nil
}

func (s *service) scanByID(ctx context.Context, photoID uuid.UUID) (models.Photo, error) {
	var photo models.Photo
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id, person_id, storage_path, mime_type, size_bytes, uploaded_by, created_at
		FROM photos WHERE id = $1
	`, photoID).Scan(
		&photo.ID,
		&photo.PersonID,
		&photo.StoragePath,
		&photo.MimeType,
		&photo.SizeBytes,
		&photo.UploadedBy,
		&photo.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Photo{}, errors.New("photo not found")
	}
	return photo, err
}