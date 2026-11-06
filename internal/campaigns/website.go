package campaigns

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// DraftFromPost makes a campaign draft from a web-only post of the
// publication website ("Send as a campaign", design expansion phase 7):
// the post's title as the subject, its excerpt as the preheader, its
// blocks as the email. The audience is left for the person to pick.
//
// Like a new draft in the composer, it starts from a sender the person may
// send from: the workspace's default when it's within their scope.
func (s *Service) DraftFromPost(ctx context.Context, ws *store.Workspace, p *authz.Principal, name, subject, preheader, src string) (uuid.UUID, error) {
	sender := DefaultSender(ctx, s.Q, ws, p)
	html := ""
	if src != "" {
		html = mjml.Compile(mjml.PrepareCompile(src)).HTML
	}
	html = accessibility.FinalizeStored(ctx, s.Q, ws.ID, accessibility.OwnerCampaign, uuid.Nil, 0, html)
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	actor := p.UserID
	var autosaved *time.Time
	if src != "" {
		autosaved = &now
	}
	plain := contentcheck.PlainText(html)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InsertComposerCampaign(ctx, store.InsertComposerCampaignParams{
			ID: id, WorkspaceID: ws.ID, Name: name, Subject: subject, Preheader: preheader, SenderID: sender, Audience: []byte(`{}`),
			Mjml: src, Html: html, PlainText: plain, ContentHash: ContentHash(subject, preheader, src, html, nil),
			CreatedBy: &actor, Now: now, AutosavedAt: autosaved,
		}); err != nil {
			return err
		}
		if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: 1, Subject: subject, Preheader: preheader, Mjml: src, Html: html,
			PlainText: plain, SavedBy: &actor, SavedAt: now, Label: mVersionFromWebPost.Source(), Source: "create", SizeBytes: int32(len(html))}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "from": "website_post"}})
	})
	return id, err
}
