-- Allow divorced spouse relationships alongside active marriages.
-- Safe to run multiple times.

CREATE OR REPLACE FUNCTION enforce_one_spouse_per_person()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  new_status text;
BEGIN
  new_status := COALESCE(NEW.metadata->>'marital_status', 'married');
  IF new_status = 'divorced' THEN
    RETURN NEW;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM relationships r
    WHERE r.type = 'spouse'
      AND r.id IS DISTINCT FROM NEW.id
      AND COALESCE(r.metadata->>'marital_status', 'married') <> 'divorced'
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