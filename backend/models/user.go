package models

import (
	"time"

	"github.com/google/uuid"
)

type SiteRole string

const (
	SiteRoleUser  SiteRole = "user"
	SiteRoleAdmin SiteRole = "admin"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	GoogleSub  string    `json:"google_sub"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
	SiteRole   SiteRole  `json:"site_role"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}