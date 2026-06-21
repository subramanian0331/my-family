package person

import (
	"context"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/models"
	personservice "github.com/subbu/family_tree/services/person"
)

func (h *Handler) labelBulkPeople(
	ctx context.Context,
	userID, familyID uuid.UUID,
	input personservice.BulkCreateInput,
	refToID map[string]uuid.UUID,
) error {
	for _, personID := range refToID {
		marryIn, err := h.isMarryInToFamily(ctx, familyID, personID, input, refToID)
		if err != nil {
			return err
		}
		if err := h.persons.AddFamilyLabel(ctx, personID, familyID, marryIn); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) isMarryInToFamily(
	ctx context.Context,
	familyID, personID uuid.UUID,
	input personservice.BulkCreateInput,
	refToID map[string]uuid.UUID,
) (bool, error) {
	for _, rel := range input.Relationships {
		if rel.Type != models.RelationshipSpouse {
			continue
		}
		fromID, err := resolveBulkID(rel.FromEndpoint, refToID)
		if err != nil {
			return false, err
		}
		toID, err := resolveBulkID(rel.ToEndpoint, refToID)
		if err != nil {
			return false, err
		}
		if fromID != personID && toID != personID {
			continue
		}
		partnerID := toID
		if personID == toID {
			partnerID = fromID
		}
		partnerNative, err := h.persons.IsNativeInFamily(ctx, partnerID, familyID)
		if err != nil {
			return false, err
		}
		if partnerNative {
			return true, nil
		}
	}
	return false, nil
}