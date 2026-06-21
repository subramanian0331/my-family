package models

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID         uuid.UUID  `json:"id"`
	FamilyID   uuid.UUID  `json:"family_id"`
	Email      string     `json:"email"`
	Role       FamilyRole `json:"role"`
	Token      string     `json:"token"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}