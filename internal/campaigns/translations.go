package campaigns

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Translations are a campaign's language versions (internal/languages):
// the content endpoints take `language` and hand language versions to it.
type Translations interface {
	// SourceLanguage is the language the campaign is written in.
	SourceLanguage(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (string, error)
	// Content returns a language version's content (404 when it has none).
	Content(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, lang string) (TranslationContent, error)
	// Save stores a new version of a language version.
	Save(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, lang string, in ComposerSaveContent, minify bool) (ComposerSavedContent, error)
	// Choose picks the version a reader with this contact language gets.
	Choose(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, contactLanguage string) (lang, reason string, err error)
	// Ready lists the languages whose version is sent, the source first.
	Ready(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) ([]string, error)
}

// TranslationContent is a language version's content.
type TranslationContent struct {
	Language     string
	Subject      string
	Preheader    string
	MJML         string
	HTML         string
	PlainText    string
	ABTest       json.RawMessage
	Version      int
	VersionCount int
	ContentHash  string
	UpdatedAt    time.Time
	UpdatedBy    string
	AutosavedAt  *time.Time
}

// sourceLanguage returns the campaign's source language, "" without
// language versions support.
func (s *Service) sourceLanguage(ctx context.Context, ws *store.Workspace, id uuid.UUID) string {
	if s.Translations == nil {
		return ""
	}
	lang, err := s.Translations.SourceLanguage(ctx, ws, id)
	if err != nil {
		return ""
	}
	return lang
}

// translated reports whether lang names a language version rather than the
// source.
func (s *Service) translated(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) bool {
	return lang != "" && s.Translations != nil && lang != s.sourceLanguage(ctx, ws, id)
}
