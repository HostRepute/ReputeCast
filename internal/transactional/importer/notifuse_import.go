package importer

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// SavedTemplates keeps a template in the composer's saved templates
// (internal/templates).
type SavedTemplates interface {
	Create(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, in templates.EmailTemplateInput) (templates.EmailTemplateDetail, error)
}

// NotifuseStats are the templates a Notifuse contact import brought along.
type NotifuseStats struct {
	// Transactional templates arrived as drafts; Saved are marketing and
	// other templates added to the saved templates.
	Transactional, Saved int
	// Manual counts the drafts whose report asks for a manual fix.
	Manual int
	// Skipped are Notifuse's own system emails (double opt-in, unsubscribe…).
	Skipped int
}

// notifuseSystem are Notifuse's system emails: ReputeCast sends its own.
var notifuseSystem = map[string]bool{"opt_in": true, "unsubscribe": true, "bounce": true, "blocklist": true}

// ImportNotifuse brings a Notifuse workspace's templates along with its
// contacts (add-on §6): transactional templates as transactional drafts,
// the others as saved templates in the composer. by is who ran the import.
func (s *Service) ImportNotifuse(ctx context.Context, ws *store.Workspace, by *uuid.UUID, baseURL, apiKey string) (NotifuseStats, error) {
	var st NotifuseStats
	n, err := NewNotifuse(baseURL, apiKey, s.HTTP)
	if err != nil {
		return st, err
	}
	list, err := n.Templates(ctx)
	if err != nil {
		return st, err
	}
	p := &authz.Principal{Type: authz.ActorUser}
	if by != nil {
		p.UserID, p.ID = *by, *by
	}
	ctx = audit.WithPrincipal(ctx, p)
	taken, err := s.keysInUse(ctx, s.Q, ws.ID)
	if err != nil {
		return st, err
	}
	for _, t := range list {
		if notifuseSystem[t.Category] {
			st.Skipped++
			continue
		}
		src, err := n.Fetch(ctx, t.ID)
		if err != nil {
			return st, err
		}
		if src.Name == "" {
			src.Name = t.Name
		}
		if t.Category == "transactional" {
			c := Convert(src, Options{Dialect: DialectNotifuse, Transactional: true})
			it := item(c, src, taken)
			if _, err := s.Drafts.CreateTemplate(ctx, ws, p, createInput(it, SourceNotifuse, "")); err != nil {
				return st, fmt.Errorf("%s: %s", src.Name, problemText(err))
			}
			st.Transactional++
			if it.Status == StatusManual {
				st.Manual++
			}
			continue
		}
		// Marketing, welcome and other templates keep the contact's fields.
		c := Convert(src, Options{Dialect: DialectNotifuse})
		mjmlSrc := c.Body
		if c.Format != FormatMJML {
			mjmlSrc = "<mjml><mj-body><mj-section><mj-column><mj-raw>" + c.Body + "</mj-raw></mj-column></mj-section></mj-body></mjml>"
		}
		name := strings.TrimSpace(src.Name)
		for i := 2; ; i++ {
			_, err := s.Saved.Create(ctx, ws, by, templates.EmailTemplateInput{Kind: "template", Name: name, MJML: mjmlSrc, HTML: c.HTML})
			if err == nil {
				break
			}
			// A saved template with that name exists: "Welcome (2)".
			if !strings.Contains(problemText(err), "already exists") || i > 20 {
				return st, fmt.Errorf("%s: %s", src.Name, problemText(err))
			}
			name = fmt.Sprintf("%s (%d)", strings.TrimSpace(src.Name), i)
		}
		st.Saved++
	}
	return st, nil
}
