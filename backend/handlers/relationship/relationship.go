package relationship

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
	familyservice "github.com/subbu/family_tree/services/family"
	personservice "github.com/subbu/family_tree/services/person"
	relationshipservice "github.com/subbu/family_tree/services/relationship"
)

type Handler struct {
	relationships relationshipservice.Service
	families      familyservice.Service
	persons       personservice.Service
}

func NewHandler(
	relationships relationshipservice.Service,
	families familyservice.Service,
	persons personservice.Service,
) *Handler {
	return &Handler{relationships: relationships, families: families, persons: persons}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	relationships, err := h.relationships.ListForFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, relationships)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	var payload struct {
		FromPersonID string          `json:"from_person_id"`
		ToPersonID   string          `json:"to_person_id"`
		Type         string          `json:"type"`
		Metadata     json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fromID, err := uuid.Parse(payload.FromPersonID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid from_person_id")
		return
	}
	toID, err := uuid.Parse(payload.ToPersonID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid to_person_id")
		return
	}

	relType := models.RelationshipParent
	if payload.Type == "spouse" {
		relType = models.RelationshipSpouse
	}

	relationship, err := h.relationships.Create(r.Context(), relationshipservice.CreateInput{
		FromPersonID: fromID,
		ToPersonID:   toID,
		Type:         relType,
		Metadata:     payload.Metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, relationshipservice.ErrSelfLink),
			errors.Is(err, relationshipservice.ErrParentAndSpouse),
			errors.Is(err, relationshipservice.ErrExistingSpouse):
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already has a spouse") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch relType {
	case models.RelationshipSpouse:
		status := "married"
		if len(relationship.Metadata) > 0 {
			var meta struct {
				MaritalStatus string `json:"marital_status"`
			}
			_ = json.Unmarshal(relationship.Metadata, &meta)
			if meta.MaritalStatus != "" {
				status = meta.MaritalStatus
			}
		}
		if status != "divorced" {
			if err := h.persons.SyncSpouseFamilyLabels(r.Context(), familyID, relationship.FromPersonID, relationship.ToPersonID); err != nil {
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	case models.RelationshipParent:
		if err := h.syncParentLink(r.Context(), familyID, fromID, toID); err != nil {
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	response.JSON(w, http.StatusCreated, relationship)
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

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, relationshipID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(r.URL.Query().Get("family_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "family_id query param required")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	var payload struct {
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	relationship, err := h.relationships.Update(r.Context(), relationshipID, relationshipservice.UpdateInput{
		Metadata: payload.Metadata,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, relationshipservice.ErrExistingSpouse) || strings.Contains(err.Error(), "already has a spouse") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, relationship)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, relationshipID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(r.URL.Query().Get("family_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "family_id query param required")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	if err := h.relationships.Delete(r.Context(), relationshipID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}