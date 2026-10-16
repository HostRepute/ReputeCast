package accessibility

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Owner kinds of language versions (content languages).
const (
	OwnerCampaign        = "campaign"
	OwnerAutomationEmail = "automation_email"
	OwnerTransactional   = "transactional_template"
)

// WorkspaceLanguage is the language a workspace writes new emails in: its
// default content language, English when it never chose one.
func WorkspaceLanguage(ctx context.Context, q store.Querier, wsID uuid.UUID) string {
	rows, err := q.LanguagesList(ctx, wsID)
	if err != nil {
		return "en"
	}
	for _, r := range rows {
		if r.IsDefault && r.RemovedAt == nil {
			return r.Code
		}
	}
	return "en"
}

// EmailLanguage is the language an email's own content (its source) is
// written in: the source language content languages recorded, otherwise
// the workspace's default.
func EmailLanguage(ctx context.Context, q store.Querier, wsID uuid.UUID, kind string, id uuid.UUID, version int32) string {
	if id != uuid.Nil {
		if row, err := q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: kind, OwnerID: id, OwnerVersion: version}); err == nil && row.Language != "" {
			return row.Language
		}
	}
	return WorkspaceLanguage(ctx, q, wsID)
}

// FinalizeStored adds lang, dir and the layout tables' role to an email's
// HTML as it is stored, in the email's source language. Every save of
// compiled HTML (campaigns, automation emails, transactional templates)
// goes through it; language versions are finalized in their own language.
func FinalizeStored(ctx context.Context, q store.Querier, wsID uuid.UUID, kind string, id uuid.UUID, version int32, html string) string {
	if html == "" {
		return html
	}
	out, _ := Finalize(html, EmailLanguage(ctx, q, wsID, kind, id, version))
	return out
}
