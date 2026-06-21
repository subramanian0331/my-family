CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE site_role AS ENUM ('user', 'admin');
CREATE TYPE family_role AS ENUM ('owner', 'editor', 'viewer');
CREATE TYPE relationship_type AS ENUM ('parent', 'spouse');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    site_role site_role NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE family_members (
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role family_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (family_id, user_id)
);

CREATE TABLE invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role family_role NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invites_email ON invites (email);
CREATE INDEX idx_invites_token ON invites (token);

CREATE TABLE persons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    given_name TEXT NOT NULL,
    patronymic TEXT NOT NULL DEFAULT '',
    clan_name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL DEFAULT '',
    birth_date DATE,
    death_date DATE,
    birth_place TEXT NOT NULL DEFAULT '',
    death_place TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE person_families (
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (person_id, family_id)
);

CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    to_person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    type relationship_type NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT relationships_no_self CHECK (from_person_id <> to_person_id),
    UNIQUE (from_person_id, to_person_id, type)
);

CREATE INDEX idx_relationships_from ON relationships (from_person_id);
CREATE INDEX idx_relationships_to ON relationships (to_person_id);

CREATE TABLE photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    storage_path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_photos_person ON photos (person_id);

ALTER TABLE persons
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector(
            'simple',
            coalesce(given_name, '') || ' ' ||
            coalesce(patronymic, '') || ' ' ||
            coalesce(clan_name, '') || ' ' ||
            coalesce(notes, '')
        )
    ) STORED;

CREATE INDEX idx_persons_search ON persons USING GIN (search_vector);