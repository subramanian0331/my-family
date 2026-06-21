package models

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	ID          uuid.UUID `json:"id"`
	PersonID    uuid.UUID `json:"person_id"`
	StoragePath string    `json:"storage_path"`
	MimeType    string    `json:"mime_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}