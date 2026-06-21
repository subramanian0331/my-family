package models

import (
	"time"

	"github.com/google/uuid"
)

type Person struct {
	ID         uuid.UUID  `json:"id"`
	GivenName  string     `json:"given_name"`
	Patronymic string     `json:"patronymic"`
	ClanName   string     `json:"clan_name"`
	Gender     string     `json:"gender"`
	BirthDate  *time.Time `json:"birth_date,omitempty"`
	DeathDate  *time.Time `json:"death_date,omitempty"`
	Deceased   bool       `json:"deceased,omitempty"`
	BirthPlace string     `json:"birth_place"`
	DeathPlace string     `json:"death_place"`
	Notes      string     `json:"notes"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	PhotoID    *uuid.UUID `json:"photo_id,omitempty"`
	MarriedIn  bool       `json:"married_in,omitempty"`
	HasSpouse  bool       `json:"has_spouse,omitempty"`
	SpouseName string     `json:"spouse_name,omitempty"`
}

type PersonFamily struct {
	PersonID  uuid.UUID `json:"person_id"`
	FamilyID  uuid.UUID `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
}