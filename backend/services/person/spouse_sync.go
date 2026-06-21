package person

import (
	"context"

	"github.com/google/uuid"
)

func (s *service) SyncSpouseFamilyLabels(ctx context.Context, familyID, fromID, toID uuid.UUID) error {
	fromNative, err := s.PrimaryNativeFamilyID(ctx, fromID)
	if err != nil {
		return err
	}
	toNative, err := s.PrimaryNativeFamilyID(ctx, toID)
	if err != nil {
		return err
	}

	fromInTarget, err := s.HasFamilyLabel(ctx, fromID, familyID)
	if err != nil {
		return err
	}
	toInTarget, err := s.HasFamilyLabel(ctx, toID, familyID)
	if err != nil {
		return err
	}

	switch {
	case fromInTarget && !toInTarget:
		if err := s.AddFamilyLabel(ctx, toID, familyID, true); err != nil {
			return err
		}
	case !fromInTarget && toInTarget:
		if err := s.AddFamilyLabel(ctx, fromID, familyID, true); err != nil {
			return err
		}
	case !fromInTarget && !toInTarget:
		if err := s.addMarryInToFamily(ctx, fromID, familyID, fromNative); err != nil {
			return err
		}
		if err := s.addMarryInToFamily(ctx, toID, familyID, toNative); err != nil {
			return err
		}
	}

	if toNative != uuid.Nil && toNative != familyID {
		has, err := s.HasFamilyLabel(ctx, fromID, toNative)
		if err != nil {
			return err
		}
		if !has {
			if err := s.AddFamilyLabel(ctx, fromID, toNative, true); err != nil {
				return err
			}
		}
	}
	if fromNative != uuid.Nil && fromNative != familyID {
		has, err := s.HasFamilyLabel(ctx, toID, fromNative)
		if err != nil {
			return err
		}
		if !has {
			if err := s.AddFamilyLabel(ctx, toID, fromNative, true); err != nil {
				return err
			}
		}
	}

	return s.reconcileSpouseMarryIn(ctx, familyID, fromID, toID)
}

func (s *service) addMarryInToFamily(ctx context.Context, personID, familyID, nativeFamilyID uuid.UUID) error {
	if nativeFamilyID != uuid.Nil && nativeFamilyID != familyID {
		return s.AddFamilyLabel(ctx, personID, familyID, true)
	}
	return s.AddFamilyLabel(ctx, personID, familyID, false)
}

func (s *service) reconcileSpouseMarryIn(ctx context.Context, familyID, fromID, toID uuid.UUID) error {
	for _, personID := range []uuid.UUID{fromID, toID} {
		inTarget, err := s.HasFamilyLabel(ctx, personID, familyID)
		if err != nil || !inTarget {
			continue
		}
		partnerID := toID
		if personID == toID {
			partnerID = fromID
		}
		partnerNative, err := s.IsNativeInFamily(ctx, partnerID, familyID)
		if err != nil || !partnerNative {
			continue
		}
		marryIn, err := s.ShouldBeMarryInToFamily(ctx, personID, familyID)
		if err != nil || !marryIn {
			continue
		}
		if err := s.AddFamilyLabel(ctx, personID, familyID, true); err != nil {
			return err
		}
	}
	return nil
}