package person

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	personservice "github.com/subbu/family_tree/services/person"
)

func (h *Handler) BulkCreate(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	var payload bulkCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input, err := payload.toServiceInput(familyID, user.ID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	for _, rel := range input.Relationships {
		for _, endpoint := range []personservice.BulkRelEndpoint{rel.FromEndpoint, rel.ToEndpoint} {
			if endpoint.PersonID == nil {
				continue
			}
			if err := h.persons.UserCanAccess(r.Context(), user.ID, *endpoint.PersonID); err != nil {
				response.Error(w, http.StatusForbidden, "forbidden")
				return
			}
		}
	}

	result, err := h.persons.BulkCreate(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, personservice.ErrBulkDuplicateRef),
			errors.Is(err, personservice.ErrBulkMissingRef),
			errors.Is(err, personservice.ErrBulkUnknownRef),
			errors.Is(err, personservice.ErrBulkInvalidRel):
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		default:
			if strings.Contains(err.Error(), "already has a spouse") ||
				err.Error() == "cannot link a person to themselves" ||
				err.Error() == "cannot be both parent and spouse of the same person" ||
				err.Error() == "at least one person is required" ||
				err.Error() == "given_name is required for each person" {
				response.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := h.labelBulkPeople(r.Context(), user.ID, familyID, input, result.RefToID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.applyBulkRelationshipHooks(r.Context(), familyID, input, result.RefToID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"people": result.People,
		"count":  len(result.People),
	})
}

func (h *Handler) applyBulkRelationshipHooks(
	ctx context.Context,
	familyID uuid.UUID,
	input personservice.BulkCreateInput,
	refToID map[string]uuid.UUID,
) error {
	for _, rel := range input.Relationships {
		fromID, err := resolveBulkID(rel.FromEndpoint, refToID)
		if err != nil {
			return err
		}
		toID, err := resolveBulkID(rel.ToEndpoint, refToID)
		if err != nil {
			return err
		}

		switch rel.Type {
		case models.RelationshipSpouse:
			if err := h.persons.SyncSpouseFamilyLabels(ctx, familyID, fromID, toID); err != nil {
				return err
			}
		case models.RelationshipParent:
			if err := h.syncParentLink(ctx, familyID, fromID, toID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) syncParentLink(ctx context.Context, familyID, childID, parentID uuid.UUID) error {
	parentInFamily, err := h.persons.HasFamilyLabel(ctx, parentID, familyID)
	if err != nil {
		return err
	}
	if parentInFamily {
		return h.persons.AddFamilyLabel(ctx, childID, familyID, false)
	}
	return nil
}

func resolveBulkID(endpoint personservice.BulkRelEndpoint, refToID map[string]uuid.UUID) (uuid.UUID, error) {
	if endpoint.PersonID != nil {
		return *endpoint.PersonID, nil
	}
	if endpoint.Ref == nil {
		return uuid.Nil, personservice.ErrBulkInvalidRel
	}
	id, ok := refToID[*endpoint.Ref]
	if !ok {
		return uuid.Nil, personservice.ErrBulkUnknownRef
	}
	return id, nil
}

type bulkCreatePayload struct {
	People        []bulkPersonPayload       `json:"people"`
	Relationships []bulkRelationshipPayload `json:"relationships"`
}

type bulkPersonPayload struct {
	Ref        string `json:"ref"`
	GivenName  string `json:"given_name"`
	Patronymic string `json:"patronymic"`
	ClanName   string `json:"clan_name"`
	Gender     string `json:"gender"`
	Notes      string `json:"notes"`
}

type bulkRelationshipPayload struct {
	From bulkEndpointPayload `json:"from"`
	To   bulkEndpointPayload `json:"to"`
	Type string              `json:"type"`
}

type bulkEndpointPayload struct {
	Ref      string `json:"ref"`
	PersonID string `json:"person_id"`
}

func (p bulkCreatePayload) toServiceInput(familyID, userID uuid.UUID) (personservice.BulkCreateInput, error) {
	people := make([]personservice.BulkPersonEntry, 0, len(p.People))
	for _, person := range p.People {
		people = append(people, personservice.BulkPersonEntry{
			Ref:        person.Ref,
			GivenName:  person.GivenName,
			Patronymic: person.Patronymic,
			ClanName:   person.ClanName,
			Gender:     person.Gender,
			Notes:      person.Notes,
		})
	}

	rels := make([]personservice.BulkRelationshipEntry, 0, len(p.Relationships))
	for _, rel := range p.Relationships {
		from, err := rel.From.toEndpoint()
		if err != nil {
			return personservice.BulkCreateInput{}, err
		}
		to, err := rel.To.toEndpoint()
		if err != nil {
			return personservice.BulkCreateInput{}, err
		}
		relType := models.RelationshipParent
		if rel.Type == "spouse" {
			relType = models.RelationshipSpouse
		}
		rels = append(rels, personservice.BulkRelationshipEntry{
			FromEndpoint: from,
			ToEndpoint:   to,
			Type:         relType,
		})
	}

	return personservice.BulkCreateInput{
		FamilyID:      familyID,
		CreatedBy:     userID,
		People:        people,
		Relationships: rels,
	}, nil
}

func (e bulkEndpointPayload) toEndpoint() (personservice.BulkRelEndpoint, error) {
	hasRef := e.Ref != ""
	hasID := e.PersonID != ""
	if hasRef == hasID {
		return personservice.BulkRelEndpoint{}, personservice.ErrBulkInvalidRel
	}
	if hasRef {
		return personservice.BulkRelEndpoint{Ref: &e.Ref}, nil
	}
	id, err := uuid.Parse(e.PersonID)
	if err != nil {
		return personservice.BulkRelEndpoint{}, errors.New("invalid person_id")
	}
	return personservice.BulkRelEndpoint{PersonID: &id}, nil
}