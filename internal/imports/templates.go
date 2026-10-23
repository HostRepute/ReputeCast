package imports

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// TemplateImporter brings a source's email templates along with its
// contacts (Notifuse, add-on transactional §6): transactional templates
// arrive as transactional drafts, marketing and other templates as saved
// templates in the composer. The transactional template importer
// implements it.
type TemplateImporter interface {
	ImportNotifuseTemplates(ctx context.Context, ws *store.Workspace, by *uuid.UUID, baseURL, apiKey string) (ImportTemplateStats, error)
}

// ImportTemplateStats are the templates an import brought along.
type ImportTemplateStats struct {
	Transactional int `json:"transactional" doc:"Transactional templates imported as drafts"`
	Saved         int `json:"saved" doc:"Marketing and other templates added to the saved templates"`
	// Manual counts the imported templates whose report asks for a manual fix.
	Manual int `json:"manual"`
	// Skipped are the source's system emails (double opt-in, unsubscribe…).
	Skipped int `json:"skipped"`
}

// importTemplates runs the template step at the end of a Notifuse import.
// A failure is logged; the contacts are imported either way.
func (s *Service) importTemplates(ctx context.Context, st *runState) {
	if st.imp.Source != "notifuse" || !st.opts.ImportTemplates || s.Templates == nil || len(st.imp.CredentialsEnc) == 0 {
		return
	}
	raw, err := s.Box.Open(st.imp.CredentialsEnc, credentialsAD(st.imp.ID))
	if err != nil {
		_ = s.logMsg(ctx, s.Q, st.imp.ID, "error", mLogTemplatesKey, nil)
		return
	}
	var c credentials
	_ = json.Unmarshal(raw, &c)
	base := s.BaseURLs["notifuse"]
	if base == "" {
		base = c.BaseURL
	}
	res, err := s.Templates.ImportNotifuseTemplates(ctx, &st.ws, st.actor, base, c.APIKey)
	if err != nil {
		_ = s.logMsg(ctx, s.Q, st.imp.ID, "error", mLogTemplatesError, i18n.Args{"reason": err.Error()})
		return
	}
	st.stats.Templates = &res
	// Counts as the English always wrote them (%d, no grouping).
	args := i18n.Args{"transactional": strconv.Itoa(res.Transactional), "saved": strconv.Itoa(res.Saved), "manual": strconv.Itoa(res.Manual)}
	m := mLogTemplates
	if res.Manual > 0 {
		m = mLogTemplatesFix
	}
	_ = s.logMsg(ctx, s.Q, st.imp.ID, "done", m, args)
}
