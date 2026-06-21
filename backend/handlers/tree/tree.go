package tree

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	familyservice "github.com/subbu/family_tree/services/family"
	personservice "github.com/subbu/family_tree/services/person"
	relationshipservice "github.com/subbu/family_tree/services/relationship"
)

type Response struct {
	Persons       []models.Person       `json:"persons"`
	Relationships []models.Relationship `json:"relationships"`
}

type Handler struct {
	persons       personservice.Service
	relationships relationshipservice.Service
	families      familyservice.Service
}

func NewHandler(
	persons personservice.Service,
	relationships relationshipservice.Service,
	families familyservice.Service,
) *Handler {
	return &Handler{persons: persons, relationships: relationships, families: families}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	persons, err := h.persons.ListByFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	memberIDs := make(map[uuid.UUID]bool, len(persons))
	for _, person := range persons {
		memberIDs[person.ID] = true
	}

	relationships, err := h.relationships.ListForFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	persons, memberIDs = includeSpousePartners(r.Context(), h.persons, persons, memberIDs, relationships)
	relationships = filterRelationshipsForMembers(relationships, memberIDs)

	response.JSON(w, http.StatusOK, Response{
		Persons:       persons,
		Relationships: relationships,
	})
}

func includeSpousePartners(
	ctx context.Context,
	personsSvc personservice.Service,
	persons []models.Person,
	memberIDs map[uuid.UUID]bool,
	relationships []models.Relationship,
) ([]models.Person, map[uuid.UUID]bool) {
	for _, rel := range relationships {
		if rel.Type != models.RelationshipSpouse {
			continue
		}
		inFamily := memberIDs[rel.FromPersonID] || memberIDs[rel.ToPersonID]
		if !inFamily {
			continue
		}
		for _, personID := range []uuid.UUID{rel.FromPersonID, rel.ToPersonID} {
			if memberIDs[personID] {
				continue
			}
			person, err := personsSvc.GetByID(ctx, personID)
			if err != nil {
				continue
			}
			persons = append(persons, person)
			memberIDs[personID] = true
		}
	}
	return persons, memberIDs
}

func filterRelationshipsForMembers(
	relationships []models.Relationship,
	memberIDs map[uuid.UUID]bool,
) []models.Relationship {
	filtered := make([]models.Relationship, 0, len(relationships))
	for _, rel := range relationships {
		if memberIDs[rel.FromPersonID] && memberIDs[rel.ToPersonID] {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}