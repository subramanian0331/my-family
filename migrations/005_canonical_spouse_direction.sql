-- Store spouse rows in canonical order (smaller UUID is from_person_id).
UPDATE relationships
SET
  from_person_id = CASE
    WHEN from_person_id::text < to_person_id::text THEN from_person_id
    ELSE to_person_id
  END,
  to_person_id = CASE
    WHEN from_person_id::text < to_person_id::text THEN to_person_id
    ELSE from_person_id
  END
WHERE type = 'spouse'
  AND from_person_id::text > to_person_id::text;