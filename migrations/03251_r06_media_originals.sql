-- +goose Up
-- Uploaded images are stored resized and compressed (review 2026-10, CO-12):
-- at most 1,200 px wide, turned upright, metadata removed. What was
-- uploaded is kept as numbers only, so the library can say what changed;
-- NULL means the image is stored exactly as it was uploaded. sha256 stays
-- the uploaded file's, so uploading the same file twice still finds it.
ALTER TABLE media_assets
    ADD COLUMN original_size_bytes   integer CHECK (original_size_bytes > 0),
    ADD COLUMN original_width        integer CHECK (original_width > 0),
    ADD COLUMN original_height       integer CHECK (original_height > 0),
    ADD COLUMN original_content_type text CHECK (original_content_type IN ('image/png','image/jpeg','image/gif'));

-- +goose Down
ALTER TABLE media_assets
    DROP COLUMN original_size_bytes,
    DROP COLUMN original_width,
    DROP COLUMN original_height,
    DROP COLUMN original_content_type;
