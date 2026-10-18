package earn

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// The paid tokens a campaign fills in per reader at send time (brief §3.7):
// {{ upgrade_url }} is the upgrade page, which knows the reader and credits
// the email it came from; {{ manage_subscription_url }} is the reader's
// billing link (Stripe's portal). Like the unsubscribe link, they aren't
// redirected through click tracking: the upgrade page records the click
// itself (UpgradeClicked).

// ReaderLink is a link a campaign fills in per reader.
type ReaderLink struct {
	Var string
	URL func(messageID, contactID uuid.UUID) string
}

// The paid link tokens.
const (
	VarUpgradeURL = "upgrade_url"
	VarManageURL  = "manage_subscription_url"
)

// ReaderLinks are the per-reader Earn links of a workspace's campaigns.
func (s *Service) ReaderLinks(ctx context.Context, wsID uuid.UUID) ([]ReaderLink, error) {
	if s.Links == nil {
		return nil, nil
	}
	w, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	upgrade, billing := s.pageURL(ctx, &w, "upgrade"), s.pageURL(ctx, &w, "billing/")
	return []ReaderLink{
		{Var: VarUpgradeURL, URL: func(message, contact uuid.UUID) string {
			if contact == uuid.Nil {
				return upgrade
			}
			return upgrade + "?r=" + s.Links.Sign(linktoken.Claims{Kind: TokenUpgrade, WorkspaceID: wsID, ContactID: contact, MessageID: message})
		}},
		{Var: VarManageURL, URL: func(_, contact uuid.UUID) string {
			if contact == uuid.Nil {
				return ""
			}
			return billing + s.Links.Sign(linktoken.Claims{Kind: TokenBilling, WorkspaceID: wsID, ContactID: contact})
		}},
	}, nil
}

// UpgradeClicked records, on the reader's timeline, that the upgrade page
// was opened from an email's personal link ("Clicked the upgrade link ·
// Weekly Digest #141 · credited to this email"), once per email.
func (s *Service) UpgradeClicked(ctx context.Context, ws *store.Workspace, token string) {
	reader, claims, ok := s.readerOf(ctx, ws, token)
	if !ok || claims.MessageID == uuid.Nil {
		return
	}
	var seen bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM paid_events WHERE contact_id = $1 AND kind = 'clicked_upgrade' AND message_id = $2)`,
		reader.ID, claims.MessageID).Scan(&seen); err != nil || seen {
		return
	}
	at := s.emailAttribution(ctx, claims.MessageID)
	if at == nil {
		return
	}
	data, _ := json.Marshal(map[string]any{"campaign": at["label"], "credited": true})
	if err := s.Q.EarnEventInsert(ctx, store.EarnEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: reader.ID, Mode: "live", Kind: "clicked_upgrade",
		OccurredAt: s.now(), MessageID: &claims.MessageID, Data: data}); err != nil {
		s.log().WarnContext(ctx, "record upgrade click", "err", err)
	}
}
