package models

import (
	"time"

	"github.com/google/uuid"
)

type FamilyRole string

const (
	FamilyRoleOwner  FamilyRole = "owner"
	FamilyRoleEditor FamilyRole = "editor"
	FamilyRoleViewer FamilyRole = "viewer"
)

type Family struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FamilyMember struct {
	FamilyID  uuid.UUID  `json:"family_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Role      FamilyRole `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
}

type FamilySummary struct {
	Family
	Role FamilyRole `json:"role"`
}