-- +goose Up
-- The contact importer reads Notifuse too (add-on transactional §6):
-- contacts and lists, and its templates (transactional ones as drafts).
ALTER TABLE imports DROP CONSTRAINT imports_source_check;
ALTER TABLE imports ADD CONSTRAINT imports_source_check
    CHECK (source IN ('brevo','mailchimp','mailerlite','listmonk','keila','notifuse','csv','xlsx'));

-- +goose Down
DELETE FROM imports WHERE source = 'notifuse';
ALTER TABLE imports DROP CONSTRAINT imports_source_check;
ALTER TABLE imports ADD CONSTRAINT imports_source_check
    CHECK (source IN ('brevo','mailchimp','mailerlite','listmonk','keila','csv','xlsx'));
