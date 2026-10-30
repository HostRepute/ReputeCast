-- +goose Up
-- The contacts a count found for each version of an email and each base
-- language of its audience, which the composer previews it as: reading
-- them back is a few rows, where finding them again read the whole
-- audience. {"versions": {"<hidden blocks>": "<contact>"}, "languages":
-- [{"language": "es", "id": "<contact>"}]}; counts made before keep '{}'
-- and are counted again when samples are asked for.
ALTER TABLE visibility_counts ADD COLUMN samples jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE visibility_counts DROP COLUMN samples;
