-- +goose Up
-- Settings › General › Sidebar sections: the sections a workspace hides from
-- the sidebar, the icon rail, the phone tab bar and the More sheet. Hiding
-- one changes nothing else: its screens open by link and keep working.
ALTER TABLE workspaces ADD COLUMN hidden_sections text[] NOT NULL DEFAULT '{}'
  CONSTRAINT workspaces_hidden_sections_known CHECK (hidden_sections <@ ARRAY['transactional', 'grow', 'earn']::text[]);

-- +goose Down
ALTER TABLE workspaces DROP COLUMN hidden_sections;
