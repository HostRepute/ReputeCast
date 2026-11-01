// Package dependents finds what uses a custom field or a segment before it
// is deleted (review 2026-11 AU-11): segments, forms, scoring rules,
// automations, campaigns not sent yet (with their SMS, push and WhatsApp
// content), digests, saved views, transactional templates and Liquid in
// their content (AU-18). A form's confirmation email is built from its
// branding and the contact's first name only, so it names no custom field. It reads each kind once and matches in memory, so a page
// listing every field asks a handful of queries.
package dependents

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Kinds of users.
const (
	KindSegment    = "segment"
	KindForm       = "form"
	KindScoring    = "scoring"
	KindAutomation = "automation"
	KindCampaign   = "campaign"
	KindDigest     = "digest"
	KindView       = "view"
	// KindTransactional is a transactional template, its draft or live
	// version (review 2026-11 AU-18).
	KindTransactional = "transactional"
)

// Use is one thing that would stop working without the field or segment.
type Use struct {
	Kind string `json:"kind" enum:"segment,form,scoring,automation,campaign,digest,view,transactional"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Querier runs a query (a pool or a transaction).
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// item is a candidate user with the text its references live in.
type item struct {
	kind, id, name string
	text           string
	// sql is a SQL segment's query, where a field shows as a bare word.
	sql string
}

// Index holds a workspace's candidates.
type Index struct{ items []item }

// Load reads the workspace's segments, forms, scoring rules, automations,
// unsent campaigns, digests, saved views and transactional templates. A row's deleted_at, where the
// table has one, leaves it out.
func Load(ctx context.Context, q Querier, ws uuid.UUID) (Index, error) {
	var idx Index
	add := func(kind, sql string) error {
		rows, err := q.Query(ctx, sql, ws)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var it item
			var id uuid.UUID
			if err := rows.Scan(&id, &it.name, &it.text, &it.sql); err != nil {
				return err
			}
			it.kind, it.id = kind, id.String()
			idx.items = append(idx.items, it)
		}
		return rows.Err()
	}
	// to_jsonb(row) reads every column, deleted_at included where there is
	// one, without naming columns some tables lack.
	for _, k := range []struct{ kind, sql string }{
		{KindSegment, `SELECT s.id, s.name, s.definition::text, CASE WHEN s.mode = 'sql' THEN s.sql ELSE '' END FROM segments s
			WHERE s.workspace_id = $1 AND to_jsonb(s)->>'deleted_at' IS NULL`},
		{KindForm, `SELECT f.id, f.name, f.fields::text || ' ' || f.content::text, '' FROM forms f WHERE f.workspace_id = $1 AND to_jsonb(f)->>'deleted_at' IS NULL`},
		{KindScoring, `SELECT $1::uuid, '', v.config::text, '' FROM scoring_configs c JOIN scoring_versions v ON v.workspace_id = c.workspace_id AND v.version = c.version
			WHERE c.workspace_id = $1`},
		{KindAutomation, `SELECT DISTINCT ON (a.id) a.id, a.name, v.graph::text, '' FROM automations a JOIN automation_versions v ON v.automation_id = a.id
			WHERE a.workspace_id = $1 AND a.deleted_at IS NULL ORDER BY a.id, v.version DESC`},
		// A text's body, a push's message and a WhatsApp campaign's
		// variables ({"source": "field", "field": "plan"}) are in their
		// channel's table (AU-18).
		{KindCampaign, `SELECT c.id, c.name, (to_jsonb(c) - 'html')::text || ' ' || c.html
				|| ' ' || coalesce((SELECT s.body FROM sms_campaigns s WHERE s.campaign_id = c.id), '')
				|| ' ' || coalesce((SELECT to_jsonb(p)::text FROM push_campaigns p WHERE p.campaign_id = c.id), '')
				|| ' ' || coalesce((SELECT w.variables::text FROM whatsapp_campaigns w WHERE w.campaign_id = c.id), ''), ''
			FROM campaigns c
			WHERE c.workspace_id = $1 AND c.deleted_at IS NULL AND c.status IN ('draft', 'approval', 'approved', 'scheduled')`},
		{KindDigest, `SELECT d.id, d.name, to_jsonb(d)::text, '' FROM digests d WHERE d.workspace_id = $1 AND to_jsonb(d)->>'deleted_at' IS NULL`},
		{KindView, `SELECT v.id, v.name, v.filters::text || ' ' || array_to_string(v.columns, ' '), '' FROM saved_views v WHERE v.workspace_id = $1`},
		{KindTransactional, `SELECT t.id, t.name, concat_ws(' ', t.subject, t.preheader, t.mjml, t.html, t.plain_text, t.variables::text,
				(SELECT concat_ws(' ', v.subject, v.preheader, v.mjml, v.html, v.plain_text) FROM transactional_template_versions v
				 WHERE v.template_id = t.id AND v.version = t.live_version)), ''
			FROM transactional_templates t WHERE t.workspace_id = $1 AND t.deleted_at IS NULL AND t.archived_at IS NULL`},
	} {
		if err := add(k.kind, k.sql); err != nil {
			return idx, err
		}
	}
	return idx, nil
}

// Field lists what uses the custom field key: a segment or automation
// condition on it ("custom:plan"), a form or scoring rule that sets or
// reads it ("field": "plan" or "key": "plan"), Liquid ({{ contact.plan }}),
// a saved view's column, or a SQL segment naming it.
func (idx Index) Field(key string) []Use {
	q := regexp.QuoteMeta(key)
	inJSON := regexp.MustCompile(`"custom:` + q + `"|"(?:field|key)":\s*"` + q + `"|\bcontact\.` + q + `\b|\bfield:` + q + `\b`)
	inSQL := regexp.MustCompile(`(?i)\b` + q + `\b`)
	out := []Use{}
	for _, it := range idx.items {
		if inJSON.MatchString(it.text) || (it.sql != "" && inSQL.MatchString(it.sql)) {
			out = append(out, it.use())
		}
	}
	return out
}

// Segment lists what uses the segment id: campaigns and digests sending to
// it, automations triggered by or checking it, saved views filtering on
// it, and segments built on it.
func (idx Index) Segment(id uuid.UUID) []Use {
	s := id.String()
	out := []Use{}
	for _, it := range idx.items {
		if it.id != s && strings.Contains(it.text, s) {
			out = append(out, it.use())
		}
	}
	return out
}

func (it item) use() Use {
	return Use{Kind: it.kind, ID: it.id, Name: it.name}
}
