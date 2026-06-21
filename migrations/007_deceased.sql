-- Mark whether a person is deceased (death date may still be unknown).
ALTER TABLE persons
    ADD COLUMN IF NOT EXISTS deceased BOOLEAN NOT NULL DEFAULT false;

UPDATE persons SET deceased = true WHERE death_date IS NOT NULL;