ALTER TABLE person_families
    ADD COLUMN IF NOT EXISTS via_marriage BOOLEAN NOT NULL DEFAULT false;

-- Children and other relatives should not inherit a family label from a marry-in parent.
DELETE FROM person_families pf
USING persons p, families f
WHERE pf.person_id = p.id
  AND pf.family_id = f.id
  AND f.name = 'kodikalakodi'
  AND p.given_name IN ('Linda', 'Ted');

-- Kayo joined kodikalakodi through marriage to Subramanian.
UPDATE person_families pf
SET via_marriage = true
FROM persons p, families f
WHERE pf.person_id = p.id
  AND pf.family_id = f.id
  AND f.name = 'kodikalakodi'
  AND p.given_name = 'Kayo';

-- Subramanian joined Valenti through marriage to Kayo.
UPDATE person_families pf
SET via_marriage = true
FROM persons p, families f
WHERE pf.person_id = p.id
  AND pf.family_id = f.id
  AND f.name = 'Valenti'
  AND p.given_name = 'Subramanian';