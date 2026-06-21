-- Enforce one spouse per person and stable spouse pairs.
-- Safe to run multiple times.

CREATE OR REPLACE FUNCTION enforce_one_spouse_per_person()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM relationships r
    WHERE r.type = 'spouse'
      AND r.id IS DISTINCT FROM NEW.id
      AND (
        r.from_person_id IN (NEW.from_person_id, NEW.to_person_id)
        OR r.to_person_id IN (NEW.from_person_id, NEW.to_person_id)
      )
  ) THEN
    RAISE EXCEPTION 'person already has a spouse';
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_enforce_one_spouse_per_person ON relationships;
CREATE TRIGGER trg_enforce_one_spouse_per_person
  BEFORE INSERT ON relationships
  FOR EACH ROW
  WHEN (NEW.type = 'spouse')
  EXECUTE FUNCTION enforce_one_spouse_per_person();

-- One stored row per unordered couple (canonical direction: smaller UUID is from).
CREATE UNIQUE INDEX IF NOT EXISTS ux_spouse_unordered_pair
  ON relationships (
    LEAST(from_person_id::text, to_person_id::text),
    GREATEST(from_person_id::text, to_person_id::text)
  )
  WHERE type = 'spouse';