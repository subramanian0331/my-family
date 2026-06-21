package person

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	persons       personservice.Service
	families      familyservice.Service
	relationships relationshipservice.Service
}

func NewHandler(
	persons personservice.Service,
	families familyservice.Service,
	relationships relationshipservice.Service,
) *Handler {
	return &Handler{persons: persons, families: families, relationships: relationships}
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

	persons, err := h.persons.ListByFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, persons)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	person, err := h.persons.GetByID(r.Context(), personID)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	familyID, err := uuid.Parse(r.URL.Query().Get("family_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "family_id query param required")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	response.JSON(w, http.StatusOK, person)
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

	var payload personPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.GivenName == "" {
		response.Error(w, http.StatusBadRequest, "given_name is required")
		return
	}

	person, err := h.persons.Create(r.Context(), personservice.CreateInput{
		GivenName:  payload.GivenName,
		Patronymic: payload.Patronymic,
		ClanName:   payload.ClanName,
		Gender:     payload.Gender,
		BirthDate:  parseDate(payload.BirthDate),
		DeathDate:  parseDate(payload.DeathDate),
		BirthPlace: payload.BirthPlace,
		DeathPlace: payload.DeathPlace,
		Notes:      payload.Notes,
		CreatedBy:  user.ID,
		FamilyIDs:  []uuid.UUID{familyID},
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, person)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
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

	var payload personPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := personservice.UpdateInput{
		GivenName:  &payload.GivenName,
		Patronymic: &payload.Patronymic,
		ClanName:   &payload.ClanName,
		Gender:     &payload.Gender,
		BirthPlace: &payload.BirthPlace,
		DeathPlace: &payload.DeathPlace,
		Notes:      &payload.Notes,
		Deceased:   &payload.Deceased,
	}
	if payload.BirthDate != nil {
		d := parseDate(payload.BirthDate)
		input.BirthDate = &d
	}
	if !payload.Deceased {
		var cleared *time.Time
		input.DeathDate = &cleared
	} else if payload.DeathDate != nil {
		d := parseDate(payload.DeathDate)
		input.DeathDate = &d
	}

	person, err := h.persons.Update(r.Context(), personID, input)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, person)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
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

	if err := h.persons.Delete(r.Context(), personID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AddToFamily(w http.ResponseWriter, r *http.Request, familyID, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}
	if err := h.persons.UserCanAccess(r.Context(), user.ID, personID); err != nil {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	if _, err := h.families.EnsureNativeFamilyForPerson(r.Context(), familyservice.EnsureNativeFamilyInput{
		PersonID:        personID,
		UserID:          user.ID,
		ContextFamilyID: familyID,
	}); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.persons.AddFamilyLabel(r.Context(), personID, familyID, false); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	person, err := h.persons.GetByID(r.Context(), personID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, person)
}

func (h *Handler) SetFamilyMarriageLabel(w http.ResponseWriter, r *http.Request, familyID, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}
	if err := h.persons.UserCanAccess(r.Context(), user.ID, personID); err != nil {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var payload struct {
		MarriedIn bool `json:"married_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.persons.SetFamilyMarriageLabel(r.Context(), personID, familyID, payload.MarriedIn); err != nil {
		if strings.Contains(err.Error(), "not in this family") {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	families, err := h.persons.ListFamiliesForPerson(r.Context(), personID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, families)
}

func (h *Handler) RemoveFromFamily(w http.ResponseWriter, r *http.Request, familyID, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}
	if err := h.persons.UserCanAccess(r.Context(), user.ID, personID); err != nil {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.persons.RemoveFamilyLabel(r.Context(), personID, familyID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) ListFamilies(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.persons.UserCanAccess(r.Context(), user.ID, personID); err != nil {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	families, err := h.persons.ListFamiliesForPerson(r.Context(), personID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if families == nil {
		families = []models.PersonFamilyRef{}
	}
	response.JSON(w, http.StatusOK, families)
}

func (h *Handler) SuggestPatronymic(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = user

	suggestion, err := h.persons.SuggestPatronymic(r.Context(), personID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"patronymic": suggestion})
}

type personPayload struct {
	GivenName  string  `json:"given_name"`
	Patronymic string  `json:"patronymic"`
	ClanName   string  `json:"clan_name"`
	Gender     string  `json:"gender"`
	BirthDate  *string `json:"birth_date"`
	DeathDate  *string `json:"death_date"`
	Deceased   bool    `json:"deceased"`
	BirthPlace string  `json:"birth_place"`
	DeathPlace string  `json:"death_place"`
	Notes      string  `json:"notes"`
}

func parseDate(value *string) *time.Time {
	if value == nil || *value == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil
	}
	return &t
}