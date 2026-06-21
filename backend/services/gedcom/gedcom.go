package gedcom

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type ImportPreview struct {
	NewPeople       int `json:"new_people"`
	ExistingMatches int `json:"existing_matches"`
}

type Service interface {
	ExportFamily(ctx context.Context, familyID uuid.UUID) ([]byte, error)
	PreviewImport(ctx context.Context, familyID uuid.UUID, reader io.Reader) (ImportPreview, error)
	CommitImport(ctx context.Context, familyID, actorID uuid.UUID, reader io.Reader) error
}