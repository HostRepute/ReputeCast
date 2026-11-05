package earn

import (
	"context"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// A paid member's billing link opens their Stripe portal, and their cancel
// link their cancel page and then the portal: whoever holds one can change
// the card, read the invoices and the address on them, or cancel. So the
// links stop working, and only an unexpired one opens the portal: one in
// a member email or a campaign after 30 days, one asked for after an
// hour. Pages reached through links that never expire (the preference
// center, which every "view online" page links to, the unsubscribe page,
// the upgrade page from an email) don't show them: the member asks there
// for a link, and it goes by email to the address on file
// (SendBillingLink).
const (
	// LinkValidAsked is how long a link asked for works: one emailed at
	// once, or shown on a page opened with a signed-in session or a link
	// that expires itself.
	LinkValidAsked = time.Hour
	// LinkValidInEmail is how long a link in a member email or a campaign
	// works.
	LinkValidInEmail = 30 * 24 * time.Hour
)

// BillingURL is a paid member's personal link to their billing portal,
// working for valid.
func (s *Service) BillingURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID, valid time.Duration) string {
	return s.pageURL(ctx, ws, "billing/"+s.memberToken(ws.ID, TokenBilling, contact, valid))
}

// CancelURL is a paid member's link to cancel (the cancel page, X17;
// until then the portal's cancel step), working for valid.
func (s *Service) CancelURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID, valid time.Duration) string {
	return s.pageURL(ctx, ws, "cancel/"+s.memberToken(ws.ID, TokenCancel, contact, valid))
}

// memberToken signs a billing or cancel link: its Link is the minute it
// stops working.
func (s *Service) memberToken(ws uuid.UUID, kind linktoken.Kind, contact uuid.UUID, valid time.Duration) string {
	return s.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: ws, ContactID: contact, Link: linktoken.UntilMinute(s.now().Add(valid))})
}

// linkExpired reports whether a billing or cancel link is past its time.
// Links signed before they expired carry no time: they have.
func (s *Service) linkExpired(c linktoken.Claims) bool {
	return !s.now().Before(linktoken.Until(c))
}

// errNoSystemMail means no System-stream provider can send the link.
var errNoSystemMail = errors.New("earn: System mail isn't available")

// SendBillingLink emails a paid member fresh links to their billing portal
// and their cancel page, working for LinkValidAsked, to the address on
// file, in a language. A contact without a subscription billed through
// Stripe gets nothing: complimentary access and gifts have nothing to pay.
func (s *Service) SendBillingLink(ctx context.Context, ws *store.Workspace, contact uuid.UUID, l i18n.Lang) error {
	var m mailer.Sender
	if s.Mail != nil {
		m = s.Mail()
	}
	if m == nil || s.Links == nil || !m.SystemAvailable(ctx, &ws.ID) {
		return errNoSystemMail
	}
	acc, err := s.accessOf(ctx, ws, contact, "")
	if err != nil || acc == nil || acc.Kind != "stripe" || acc.StripeCustomerID == "" {
		return err
	}
	c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	b := s.brand(ctx, ws)
	tier := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	billing := s.BillingURL(ctx, ws, contact, LinkValidAsked)
	cancel := ""
	if acc.CancelAt == nil {
		cancel = s.CancelURL(ctx, ws, contact, LinkValidAsked)
	}
	subject := s.t(l, "portal.email.subject", i18n.Args{"tier": firstNonEmpty(tier, "none")})
	greeting := s.t(l, "portal.email.greeting", i18n.Args{"name": firstNonEmpty(strings.TrimSpace(c.FirstName), "none")})
	lead, button, ignore := s.t(l, "portal.email.lead", nil), s.t(l, "portal.email.button", nil), s.t(l, "portal.email.ignore", nil)
	esc := html.EscapeString
	text := greeting + "\n\n" + lead + "\n\n" + button + ": " + billing + "\n"
	body := `<p style="margin:0 0 12px">` + esc(greeting) + `</p>` +
		`<p style="margin:0 0 20px">` + esc(lead) + `</p>` +
		`<p style="margin:0 0 16px"><a href="` + esc(billing) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(b.Accent) +
		`;color:` + esc(b.OnAccent) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(button) + `</a></p>`
	if cancel != "" {
		text += s.t(l, "portal.email.cancel", nil) + " " + cancel + "\n"
		link := func(inner string) string {
			return `<a href="` + esc(cancel) + `" style="color:#0B1220;text-decoration:underline">` + inner + `</a>`
		}
		body += `<p style="margin:0 0 16px">` + msgs.M("portal.email.cancel").HTML(l, nil, i18n.Tags{"link": link}) + `</p>`
	}
	text += "\n" + ignore + "\n\n" + b.Publisher + "\n"
	body += `<p style="margin:0;font-size:13px;color:#5B6472">` + esc(ignore) + `</p>`
	_, err = m.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: c.Email, ToName: strings.TrimSpace(c.FirstName + " " + c.LastName),
		Subject: subject, Text: text, HTML: body, Kind: "billing_link", Lang: l})
	return err
}
