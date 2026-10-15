package imports

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

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
		_ = s.log(ctx, s.Q, st.imp.ID, "error", "Templates weren't imported: the stored API key can't be decrypted.")
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
		_ = s.log(ctx, s.Q, st.imp.ID, "error", "Templates weren't imported: "+err.Error())
		return
	}
	st.stats.Templates = &res
	msg := fmt.Sprintf("Imported %d transactional templates as drafts and %d saved templates", res.Transactional, res.Saved)
	if res.Manual > 0 {
		msg += fmt.Sprintf("; %d need a manual fix", res.Manual)
	}
	_ = s.log(ctx, s.Q, st.imp.ID, "done", msg)
}
