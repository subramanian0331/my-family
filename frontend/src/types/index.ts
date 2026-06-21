export type SiteRole = "user" | "admin";
export type FamilyRole = "owner" | "editor" | "viewer";
export type RelationshipType = "parent" | "spouse";

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
  site_role: SiteRole;
}

export interface FamilySummary {
  id: string;
  name: string;
  slug: string;
  description: string;
  role: FamilyRole;
}

export interface FamilyDetail {
  family: {
    id: string;
    name: string;
    description: string;
  };
  role: FamilyRole;
}

export interface Person {
  id: string;
  given_name: string;
  patronymic: string;
  clan_name: string;
  gender: string;
  birth_date?: string;
  death_date?: string;
  deceased?: boolean;
  birth_place: string;
  death_place: string;
  notes: string;
  photo_id?: string;
  married_in?: boolean;
  has_spouse?: boolean;
  spouse_name?: string;
}

export type MaritalStatus = "married" | "divorced";

export interface RelationshipMetadata {
  marital_status?: MaritalStatus;
}

export interface Relationship {
  id: string;
  from_person_id: string;
  to_person_id: string;
  type: RelationshipType;
  metadata?: RelationshipMetadata | string;
}

export interface TreeData {
  persons: Person[];
  relationships: Relationship[];
}

export interface Invite {
  id: string;
  family_id: string;
  email: string;
  role: FamilyRole;
  token: string;
}

export interface FamilyMember {
  user_id: string;
  email: string;
  name: string;
  avatar_url: string;
  role: FamilyRole;
}

export interface PersonFamilyRef {
  id: string;
  name: string;
  married_in?: boolean;
}

export interface PersonSearchHit {
  person: Person;
  families: PersonFamilyRef[];
  in_target_family: boolean;
}

export interface BulkCreatePerson {
  ref: string;
  given_name: string;
  patronymic?: string;
  clan_name?: string;
  gender?: string;
  notes?: string;
}

export interface BulkCreateEndpoint {
  ref?: string;
  person_id?: string;
}

export interface BulkCreateRelationship {
  from: BulkCreateEndpoint;
  to: BulkCreateEndpoint;
  type: "parent" | "spouse";
}

export interface BulkCreatePayload {
  people: BulkCreatePerson[];
  relationships: BulkCreateRelationship[];
}

export interface AdminFamilyAccess {
  family_id: string;
  family_name: string;
  role: FamilyRole;
}

export interface AdminUserDetail {
  user: User & { created_at?: string };
  families: AdminFamilyAccess[];
}

export interface AdminInviteDetail {
  invite: Invite & { expires_at?: string; created_at?: string };
  family_name: string;
}

export interface AdminSettings {
  frontend_url: string;
  google_enabled: boolean;
  site_admin_email: string;
  user_count: number;
  family_count: number;
  pending_invites: number;
}

export interface AdminFamily {
  id: string;
  name: string;
  slug: string;
  description: string;
}