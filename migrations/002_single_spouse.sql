-- Remove spouse links where a parent link already exists between the same pair.
DELETE FROM relationships r
WHERE r.type = 'spouse'
  AND EXISTS (
    SELECT 1
    FROM relationships p
    WHERE p.type = 'parent'
      AND (
        (p.from_person_id = r.from_person_id AND p.to_person_id = r.to_person_id)
        OR (p.from_person_id = r.to_person_id AND p.to_person_id = r.from_person_id)
      )
  );

-- Drop duplicate spouse rows for the same unordered pair (keep oldest).
DELETE FROM relationships r
WHERE r.type = 'spouse'
  AND r.id NOT IN (
    SELECT DISTINCT ON (
      LEAST(from_person_id::text, to_person_id::text),
      GREATEST(from_person_id::text, to_person_id::text)
    ) id
    FROM relationships
    WHERE type = 'spouse'
    ORDER BY
      LEAST(from_person_id::text, to_person_id::text),
      GREATEST(from_person_id::text, to_person_id::text),
      created_at ASC
  );

-- Keep only the most recent spouse per person.
WITH spouse_links AS (
  SELECT id, created_at, from_person_id AS person_id
  FROM relationships
  WHERE type = 'spouse'
  UNION ALL
  SELECT id, created_at, to_person_id AS person_id
  FROM relationships
  WHERE type = 'spouse'
),
ranked AS (
  SELECT id,
    ROW_NUMBER() OVER (PARTITION BY person_id ORDER BY created_at DESC) AS rn
  FROM spouse_links
)
DELETE FROM relationships
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);