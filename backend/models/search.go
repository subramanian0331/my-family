package models

import "github.com/google/uuid"

type PersonFamilyRef struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	MarriedIn bool      `json:"married_in,omitempty"`
}

type PersonSearchHit struct {
	Person         Person            `json:"person"`
	Families       []PersonFamilyRef `json:"families"`
	InTargetFamily bool              `json:"in_target_family"`
}