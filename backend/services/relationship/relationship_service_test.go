package relationship_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/models"
	relationshipservice "github.com/subbu/family_tree/services/relationship"
)

func testDB(t *testing.T) postgresclient.Client {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://familytree:familytree@localhost:5432/familytree?sslmode=disable"
	}
	db, err := postgresclient.New(url)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	return db
}

func TestCreateSpousePreservesIndependentCouples(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	svc := relationshipservice.NewService(db)

	userID := uuid.New()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO users (id, google_sub, email, name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, userID, "test-sub-"+userID.String(), userID.String()+"@example.com", "Tester")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	createPerson := func(name string) uuid.UUID {
		var id uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO persons (given_name, created_by)
			VALUES ($1, $2) RETURNING id
		`, name, userID).Scan(&id)
		if err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		return id
	}

	linda := createPerson("LindaTest")
	ted := createPerson("TedTest")
	karla := createPerson("KarlaTest")
	david := createPerson("DavidTest")

	t.Cleanup(func() {
		_, _ = db.Pool().Exec(ctx, `DELETE FROM persons WHERE id = ANY($1)`, []uuid.UUID{linda, ted, karla, david})
		_, _ = db.Pool().Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err = svc.Create(ctx, relationshipservice.CreateInput{
		FromPersonID: linda,
		ToPersonID:   ted,
		Type:         models.RelationshipSpouse,
	}); err != nil {
		t.Fatalf("create linda-ted: %v", err)
	}

	if _, err = svc.Create(ctx, relationshipservice.CreateInput{
		FromPersonID: karla,
		ToPersonID:   david,
		Type:         models.RelationshipSpouse,
	}); err != nil {
		t.Fatalf("create karla-david: %v", err)
	}

	var count int
	err = db.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM relationships
		WHERE type = 'spouse'
		  AND (
		    (from_person_id = $1 AND to_person_id = $2)
		    OR (from_person_id = $2 AND to_person_id = $1)
		    OR (from_person_id = $3 AND to_person_id = $4)
		    OR (from_person_id = $4 AND to_person_id = $3)
		  )
	`, linda, ted, karla, david).Scan(&count)
	if err != nil {
		t.Fatalf("count spouses: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 spouse relationships, got %d", count)
	}
}

func TestCreateSpouseFindsReverseDirection(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	ctx := context.Background()
	svc := relationshipservice.NewService(db)

	userID := uuid.New()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO users (id, google_sub, email, name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, userID, "test-sub-"+userID.String(), userID.String()+"@example.com", "Tester")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	createPerson := func(name string) uuid.UUID {
		var id uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO persons (given_name, created_by)
			VALUES ($1, $2) RETURNING id
		`, name, userID).Scan(&id)
		if err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		return id
	}

	linda := createPerson("ReverseLinda")
	ted := createPerson("ReverseTed")

	t.Cleanup(func() {
		_, _ = db.Pool().Exec(ctx, `DELETE FROM persons WHERE id = ANY($1)`, []uuid.UUID{linda, ted})
		_, _ = db.Pool().Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Legacy non-canonical storage (larger UUID stored as from_person_id).
	fromID, toID := ted, linda
	if fromID.String() < toID.String() {
		fromID, toID = linda, ted
	}
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO relationships (from_person_id, to_person_id, type, metadata)
		VALUES ($1, $2, 'spouse', '{}')
	`, fromID, toID)
	if err != nil {
		t.Fatalf("insert reverse spouse row: %v", err)
	}

	rel, err := svc.Create(ctx, relationshipservice.CreateInput{
		FromPersonID: linda,
		ToPersonID:   ted,
		Type:         models.RelationshipSpouse,
	})
	if err != nil {
		t.Fatalf("create canonical linda-ted: %v", err)
	}
	if rel.FromPersonID != linda || rel.ToPersonID != ted {
		t.Fatalf("expected canonical direction %s-%s, got %s-%s", linda, ted, rel.FromPersonID, rel.ToPersonID)
	}

	var count int
	err = db.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM relationships
		WHERE type = 'spouse'
		  AND (
		    (from_person_id = $1 AND to_person_id = $2)
		    OR (from_person_id = $2 AND to_person_id = $1)
		  )
	`, linda, ted).Scan(&count)
	if err != nil {
		t.Fatalf("count spouses: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 spouse row, got %d", count)
	}
}