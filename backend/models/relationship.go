package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RelationshipType string

const (
	RelationshipParent RelationshipType = "parent"
	RelationshipSpouse RelationshipType = "spouse"
)

type Relationship struct {
	ID           uuid.UUID        `json:"id"`
	FromPersonID uuid.UUID        `json:"from_person_id"`
	ToPersonID   uuid.UUID        `json:"to_person_id"`
	Type         RelationshipType `json:"type"`
	Metadata     json.RawMessage  `json:"metadata"`
	CreatedAt    time.Time        `json:"created_at"`
}