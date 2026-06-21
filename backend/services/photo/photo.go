package photo

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
)

type UploadInput struct {
	PersonID   uuid.UUID
	Filename   string
	MimeType   string
	SizeBytes  int64
	UploadedBy uuid.UUID
	Reader     io.Reader
}

type Service interface {
	Upload(ctx context.Context, input UploadInput) (models.Photo, error)
	GetByID(ctx context.Context, photoID uuid.UUID) (models.Photo, error)
	Delete(ctx context.Context, photoID uuid.UUID) error
	Open(ctx context.Context, photoID uuid.UUID) (io.ReadCloser, models.Photo, error)
}