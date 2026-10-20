package earn

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/paidaccess"
)

// The paid tokens (brief §3.14): {{ paid.tier }}, {{ paid.status }},
// {{ paid.price }}, {{ paid.renews_at }} and {{ paid.ends_at }} read the
// reader's paid access (paidaccess.Facts: the highest tier's subscription,
// grant or gift), and the two personal links are ReaderLinks. A free
// subscriber's paid values are empty: a token needs a default, or its block
// shows to paid access only. The composer's token menu lists them in its
// Paid group, with each value for the previewed contact.
//
// PaidVars fills a page of readers at once, keyed by contact, the shape of
// the per-reader data a send, a preview, a test email and view online merge
// into the Liquid variables.

// RootPaid is the Liquid variable the paid values live under.
const RootPaid = "paid"

// Token is a paid token of the composer's token menu.
type Token struct {
	Key   string
	Label string
}

// Tokens are the Paid group of the composer's token menu, in its order.
func Tokens() []Token {
	out := make([]Token, 0, len(tokenKeys))
	for _, k := range tokenKeys {
		out = append(out, Token{Key: k.key, Label: msgs.M(k.label).Text(PageLang, nil)})
	}
	return out
}

var tokenKeys = []struct{ key, label string }{
	{RootPaid + ".tier", "tokens.tier"},
	{RootPaid + ".status", "tokens.status"},
	{RootPaid + ".price", "tokens.price"},
	{RootPaid + ".renews_at", "tokens.renewsAt"},
	{RootPaid + ".ends_at", "tokens.endsAt"},
	{VarUpgradeURL, "tokens.upgradeURL"},
	{VarManageURL, "tokens.manageURL"},
}

var mStatusValue = msgs.M("tokens.statusValue")

// PaidVars are some readers' paid values, by contact: {"paid": {"tier":
// "Weekly Digest Insider", "status": "Active", "price": "$80/year",
// "renews_at": "7 Oct 2027", "ends_at": ""}}; every value is empty for a
// free subscriber. Test paid members never count.
func (s *Service) PaidVars(ctx context.Context, wsID uuid.UUID, contacts []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	out := make(map[uuid.UUID]map[string]any, len(contacts))
	if len(contacts) == 0 {
		return out, nil
	}
	facts, err := paidaccess.Facts(ctx, s.Q, wsID, contacts)
	if err != nil {
		return nil, err
	}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, id := range contacts {
		v := map[string]any{"tier": "", "status": "", "price": "", "renews_at": "", "ends_at": ""}
		if f, ok := facts[id]; ok {
			v["tier"] = f.Tier
			v["status"] = mStatusValue.Text(PageLang, i18n.Args{"status": f.Status})
			if f.AmountCents > 0 && f.Period != "" {
				v["price"] = priceText(f.AmountCents, f.Currency, f.Period)
			}
			if f.RenewsAt != nil {
				v["renews_at"] = longDate(&ws, *f.RenewsAt)
			}
			if f.EndsAt != nil {
				v["ends_at"] = longDate(&ws, *f.EndsAt)
			}
		}
		out[id] = map[string]any{RootPaid: v}
	}
	return out, nil
}
