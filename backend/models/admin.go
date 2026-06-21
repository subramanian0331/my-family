package models

import "github.com/google/uuid"

type AdminFamilyAccess struct {
	FamilyID   uuid.UUID  `json:"family_id"`
	FamilyName string     `json:"family_name"`
	Role       FamilyRole `json:"role"`
}

type AdminUserDetail struct {
	User     User                `json:"user"`
	Families []AdminFamilyAccess `json:"families"`
}

type AdminInviteDetail struct {
	Invite     Invite `json:"invite"`
	FamilyName string `json:"family_name"`
}

type AdminSettings struct {
	FrontendURL    string `json:"frontend_url"`
	GoogleEnabled  bool   `json:"google_enabled"`
	SiteAdminEmail string `json:"site_admin_email"`
	UserCount      int    `json:"user_count"`
	FamilyCount    int    `json:"family_count"`
	PendingInvites int    `json:"pending_invites"`
}