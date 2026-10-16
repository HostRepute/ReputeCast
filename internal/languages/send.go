package languages

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// CampaignDigest fingerprints what a campaign's language versions add to
// its content: each version's content, status and hold, and the
// workspace's languages. It is empty for a campaign without versions, so
// fingerprints of single-language campaigns don't change.
func (s *Service) CampaignDigest(ctx context.Context, q *store.Queries, wsID, campaignID uuid.UUID) string {
	if q == nil {
		q = s.Q
	}
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerCampaign, OwnerID: campaignID})
	if err != nil || len(rows) == 0 {
		return ""
	}
	st, err := s.Settings(ctx, q, wsID)
	if err != nil {
		return ""
	}
	src := ""
	if row, err := q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: OwnerCampaign, OwnerID: campaignID}); err == nil {
		src = row.Language
	}
	parts := []string{src, st.Default, st.Fallback}
	parts = append(parts, st.Languages...)
	for _, r := range rows {
		parts = append(parts, r.Language, r.ContentHash, r.Status, fmt.Sprint(r.Held))
	}
	return HashText(strings.Join(parts, "\x00"))
}

// NewSettings builds workspace settings from their parts (the sending
// engine keeps them in a send's snapshot).
func NewSettings(def, fallback string, langs []string) Settings {
	st := Settings{Default: def, Fallback: fallback, Languages: append([]string{}, langs...), offered: map[string]bool{}}
	for _, l := range langs {
		st.offered[l] = true
	}
	if len(st.Languages) == 0 && def != "" {
		st.Languages = []string{def}
		st.offered[def] = true
	}
	return st
}

// SendSet is what a send needs to give each reader their version: the
// source's language, the workspace's languages and the email's versions.
type SendSet struct {
	Source   string
	Default  string
	Fallback string
	Offered  []string
	Versions []SendVersion
}

// SendVersion is one language version at send start. Content is only set
// for versions that can be sent (translated, not held, offered).
type SendVersion struct {
	Language       string
	Status         string
	Held           bool
	Subject        string
	Preheader      string
	HTML           string
	Text           string
	ABTest         json.RawMessage
	TranslatedFrom int
	Version        int
}

// Settings returns the set's workspace settings.
func (s SendSet) Settings() Settings { return NewSettings(s.Default, s.Fallback, s.Offered) }

// CampaignSendSet returns a campaign's languages for a send.
func (s *Service) CampaignSendSet(ctx context.Context, wsID, campaignID uuid.UUID) (SendSet, error) {
	return s.sendSet(ctx, wsID, OwnerCampaign, campaignID, 0)
}

// AutomationSendSet returns the languages of an automation email version.
func (s *Service) AutomationSendSet(ctx context.Context, wsID, emailID uuid.UUID, version int) (SendSet, error) {
	return s.sendSet(ctx, wsID, OwnerAutomationEmail, emailID, int32(version))
}

func (s *Service) sendSet(ctx context.Context, wsID uuid.UUID, kind string, id uuid.UUID, version int32) (SendSet, error) {
	st, err := s.Settings(ctx, s.Q, wsID)
	if err != nil {
		return SendSet{}, err
	}
	out := SendSet{Source: st.Default, Default: st.Default, Fallback: st.Fallback, Offered: st.Languages, Versions: []SendVersion{}}
	if row, err := s.Q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: kind, OwnerID: id, OwnerVersion: version}); err == nil {
		out.Source = row.Language
	}
	rows, err := s.Q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: kind, OwnerID: id, OwnerVersion: version})
	if err != nil {
		return SendSet{}, err
	}
	for _, r := range rows {
		if r.Language == out.Source || !st.Offers(r.Language) {
			continue
		}
		v := SendVersion{Language: r.Language, Status: r.Status, Held: r.Held, TranslatedFrom: int(r.TranslatedFrom), Version: int(r.Version)}
		if r.Status == storedTranslated && !r.Held {
			v.Subject, v.Preheader, v.HTML, v.Text, v.ABTest = r.Subject, r.Preheader, r.Html, r.PlainText, r.AbTest
		}
		out.Versions = append(out.Versions, v)
	}
	return out, nil
}

// Chooser returns the set's chooser.
func (s SendSet) Chooser() Chooser {
	states := make([]VersionState, 0, len(s.Versions))
	for _, v := range s.Versions {
		states = append(states, VersionState{Language: v.Language, Status: v.Status, Held: v.Held})
	}
	return NewChooser(s.Source, s.Settings(), states)
}
