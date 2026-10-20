package earn

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ErrNoRoute means transactional mail has no route: member emails can't
// be sent until one is chosen in Sending providers › Routing.
var ErrNoRoute = errors.New("earn: no Transactional route")

// ErrSuppressed means the address is suppressed for all mail.
var ErrSuppressed = errors.New("earn: the recipient is suppressed")

// The eight member emails (brief 3.8), in the settings table's order.
const (
	EmailWelcome         = "welcome"
	EmailTrialEnding     = "trial_ending"
	EmailPaymentFailed   = "payment_failed"
	EmailRenewalReminder = "renewal_reminder"
	EmailCanceled        = "canceled"
	EmailGiftPurchased   = "gift_purchased"
	EmailGiftReceived    = "gift_received"
	EmailGiftEnding      = "gift_ending"
)

// MemberEmailOrder lists them in order.
var MemberEmailOrder = []string{EmailWelcome, EmailTrialEnding, EmailPaymentFailed, EmailRenewalReminder, EmailCanceled, EmailGiftPurchased, EmailGiftReceived, EmailGiftEnding}

// starter is a member email's template as the drafts start.
type starter struct {
	email, key, name, description string
	subject, preheader            string
	heading                       string
	body                          []string
	button, href                  string
	after                         string
	vars                          []TemplateVariable
}

var memberVar = []TemplateVariable{
	{Name: "first_name", Type: "text", Sample: "Sam", Default: "there"},
	{Name: "tier", Type: "text", Required: true, Sample: "Insider"},
}

func vars(extra ...TemplateVariable) []TemplateVariable {
	return append(append([]TemplateVariable{}, memberVar...), extra...)
}

var manageVar = TemplateVariable{Name: "manage_subscription_url", Type: "url", Required: true, Sensitive: true, Sample: "https://news.example/billing/sample-token"}

// starters are the member emails' drafts, created when a payment provider
// is first connected; each is sent only once it's published.
var starters = []starter{
	{email: EmailWelcome, key: "member-welcome", name: "Welcome to a paid tier", description: "When a trial or paid subscription starts",
		subject: "Welcome to {{ tier }}, {{ first_name }}", preheader: "Here is what your membership includes.",
		heading: "Welcome to {{ tier }}",
		body:    []string{"Hi {{ first_name }},", "Thank you for joining {{ tier }}. {{ plan_line }}", "{{ next_issue_line }}"},
		button:  "Manage your subscription", href: "{{ manage_subscription_url }}",
		after: "Stripe emails your receipts. You can switch between monthly and yearly, or cancel, any time from the link above.",
		vars: vars(TemplateVariable{Name: "plan_line", Type: "text", Sample: "You pay $80 a year, renewing on 7 Oct 2027."},
			TemplateVariable{Name: "next_issue_line", Type: "text", Sample: "Your first members-only issue arrives on Tuesday.", Default: ""}, manageVar)},
	{email: EmailTrialEnding, key: "member-trial-ending", name: "Trial ends soon", description: "3 days before a trial ends",
		subject: "Your {{ tier }} trial ends on {{ trial_end }}", preheader: "Nothing to do if you'd like to stay.",
		heading: "Your trial ends on {{ trial_end }}",
		body:    []string{"Hi {{ first_name }},", "Your free {{ tier }} trial ends on {{ trial_end }}. After that you'll pay {{ price }}, and nothing changes in what you get.", "To stop before you're charged, cancel from the link below. You'll still get the free issues."},
		button:  "Manage your subscription", href: "{{ manage_subscription_url }}",
		vars: vars(TemplateVariable{Name: "trial_end", Type: "text", Required: true, Sample: "18 Oct"}, TemplateVariable{Name: "price", Type: "text", Required: true, Sample: "$8 a month"}, manageVar)},
	{email: EmailPaymentFailed, key: "member-payment-failed", name: "Payment failed", description: "Stripe couldn't charge the card",
		subject: "We couldn't take your {{ tier }} payment", preheader: "Update your card to keep your membership.",
		heading: "Your payment didn't go through",
		body:    []string{"Hi {{ first_name }},", "We tried to charge {{ amount }} for {{ tier }}, but the payment didn't go through: {{ reason }}", "Stripe tries again on {{ next_try }}. You keep {{ tier }} meanwhile. Updating your card now avoids a gap."},
		button:  "Update your card", href: "{{ manage_subscription_url }}",
		vars: vars(TemplateVariable{Name: "amount", Type: "text", Required: true, Sample: "$8.00"}, TemplateVariable{Name: "reason", Type: "text", Sample: "Your card has insufficient funds.", Default: "the card was declined."},
			TemplateVariable{Name: "next_try", Type: "text", Sample: "16 Oct", Default: "in a few days"}, manageVar)},
	{email: EmailRenewalReminder, key: "member-renewal-reminder", name: "Renewal reminder", description: "7 days before a yearly renewal",
		subject: "Your {{ tier }} membership renews on {{ renews_at }}", preheader: "{{ amount }} for another year.",
		heading: "Renewing on {{ renews_at }}",
		body:    []string{"Hi {{ first_name }},", "Your yearly {{ tier }} membership renews on {{ renews_at }} for {{ amount }}. You don't need to do anything to keep it.", "To change your plan or cancel before then, use the link below."},
		button:  "Manage your subscription", href: "{{ manage_subscription_url }}",
		vars: vars(TemplateVariable{Name: "renews_at", Type: "text", Required: true, Sample: "22 Oct"}, TemplateVariable{Name: "amount", Type: "text", Required: true, Sample: "$80.00"}, manageVar)},
	{email: EmailCanceled, key: "member-canceled", name: "Membership canceled", description: "When a cancellation is confirmed",
		subject: "Your {{ tier }} membership ends on {{ ends_at }}", preheader: "You'll keep getting the free issues.",
		heading: "Your membership ends on {{ ends_at }}",
		body:    []string{"Hi {{ first_name }},", "Your {{ tier }} membership is canceled. You keep everything until {{ ends_at }}, and you'll still get the free issues after that.", "Changed your mind? You can come back any time."},
		button:  "Keep {{ tier }}", href: "{{ upgrade_url }}",
		vars: vars(TemplateVariable{Name: "ends_at", Type: "text", Required: true, Sample: "2 Nov"}, TemplateVariable{Name: "upgrade_url", Type: "url", Required: true, Sensitive: true, Sample: "https://news.example/upgrade?r=sample-token"})},
	{email: EmailGiftPurchased, key: "gift-purchased", name: "Gift receipt", description: "When a gift is bought",
		subject: "Your gift of {{ tier }} is on its way", preheader: "Thank you. Here is your receipt.",
		heading: "Thank you for your gift",
		body:    []string{"Hi {{ first_name }},", "You gave {{ recipient }} {{ gift }} of {{ tier }} for {{ amount }}. {{ delivery_line }}", "Stripe emails the payment's receipt separately."},
		vars: vars(TemplateVariable{Name: "recipient", Type: "text", Required: true, Sample: "Karim"}, TemplateVariable{Name: "gift", Type: "text", Required: true, Sample: "a year"},
			TemplateVariable{Name: "amount", Type: "text", Required: true, Sample: "$80.00"}, TemplateVariable{Name: "delivery_line", Type: "text", Sample: "It arrives on 24 Dec at 08:00.", Default: ""})},
	{email: EmailGiftReceived, key: "gift-received", name: "Your gift", description: "On the gift's delivery date",
		subject: "{{ giver }} gave you {{ gift }} of {{ tier }}", preheader: "No card needed to start.",
		heading: "A gift from {{ giver }}",
		body:    []string{"Hi {{ first_name }},", "{{ giver }} gave you {{ gift }} of {{ tier }}. {{ message }}", "Start it whenever you like: no card is needed, and nothing renews."},
		button:  "Start my gift", href: "{{ redeem_url }}",
		vars: vars(TemplateVariable{Name: "giver", Type: "text", Required: true, Sample: "Priya Natarajan"}, TemplateVariable{Name: "gift", Type: "text", Required: true, Sample: "a year"},
			TemplateVariable{Name: "message", Type: "text", Sample: "Enjoy the deep dives!", Default: ""}, TemplateVariable{Name: "redeem_url", Type: "url", Required: true, Sensitive: true, Sample: "https://news.example/gift/redeem/sample-code"})},
	{email: EmailGiftEnding, key: "gift-ending", name: "Gift ending", description: "7 days before a gift ends",
		subject: "Your {{ tier }} gift ends on {{ ends_at }}", preheader: "Keep it going if you like it.",
		heading: "Your gift ends on {{ ends_at }}",
		body:    []string{"Hi {{ first_name }},", "Your gift of {{ tier }} ends on {{ ends_at }}. After that you'll still get the free issues.", "Want to keep {{ tier }}? You can continue from the link below."},
		button:  "Keep {{ tier }}", href: "{{ upgrade_url }}",
		vars: vars(TemplateVariable{Name: "ends_at", Type: "text", Required: true, Sample: "11 Oct 2027"}, TemplateVariable{Name: "upgrade_url", Type: "url", Required: true, Sensitive: true, Sample: "https://news.example/upgrade?r=sample-token"})},
}

func starterOf(email string) (starter, bool) {
	for _, st := range starters {
		if st.email == email {
			return st, true
		}
	}
	return starter{}, false
}

// Starter is a member email's starter, as the drafts begin (the demo
// publishes them as they are).
type Starter struct {
	Email, Key, Name, Description string
	Subject, Preheader, MJML      string
	Variables                     []TemplateVariable
	Sample                        map[string]any
}

// Starters returns the eight member emails' starters for a publication and
// its postal address, in the settings table's order.
func Starters(product, address string) []Starter {
	out := make([]Starter, 0, len(starters))
	for _, st := range starters {
		out = append(out, Starter{Email: st.email, Key: st.key, Name: st.name, Description: st.description, Subject: st.subject, Preheader: st.preheader,
			MJML: st.mjml(product, address), Variables: st.vars, Sample: sampleData(st)})
	}
	return out
}

// StarterKey is a member email's template ID ("member-welcome").
func StarterKey(email string) string {
	st, _ := starterOf(email)
	return st.key
}

// mjml writes a starter's MJML: the publication's name, the message, its
// button and the transactional footer (Essential: no unsubscribe link).
func (st starter) mjml(product, address string) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString("<mjml>\n  <mj-head>\n    <mj-title>" + esc(st.subject) + "</mj-title>\n    <mj-preview>" + esc(st.preheader) + "</mj-preview>\n")
	b.WriteString("    <mj-attributes>\n      <mj-all font-family=\"Georgia, 'Times New Roman', serif\" />\n      <mj-text font-size=\"15px\" line-height=\"1.6\" color=\"#1F2937\" padding=\"0 0 14px\" />\n    </mj-attributes>\n  </mj-head>\n")
	b.WriteString("  <mj-body background-color=\"#F3F4F6\" width=\"600px\">\n")
	b.WriteString("    <mj-section background-color=\"#ffffff\" padding=\"20px 28px\" border-bottom=\"1px solid #EEF0F3\">\n      <mj-column>\n")
	b.WriteString("        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-weight=\"700\" font-size=\"15px\" color=\"#0B1220\" padding=\"0\">" + esc(product) + "</mj-text>\n")
	b.WriteString("      </mj-column>\n    </mj-section>\n")
	b.WriteString("    <mj-section background-color=\"#ffffff\" padding=\"26px 28px 8px\">\n      <mj-column>\n")
	b.WriteString("        <mj-text font-size=\"24px\" line-height=\"1.25\" color=\"#0B1220\" font-weight=\"700\">" + esc(st.heading) + "</mj-text>\n")
	for _, p := range st.body {
		b.WriteString("        <mj-text>" + esc(p) + "</mj-text>\n")
	}
	if st.button != "" {
		fmt.Fprintf(&b, "        <mj-button href=\"%s\" background-color=\"#4338CA\" color=\"#ffffff\" font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"14px\" font-weight=\"600\" border-radius=\"6px\" inner-padding=\"11px 18px\" align=\"left\" padding=\"8px 0 18px\">%s</mj-button>\n", st.href, esc(st.button))
	}
	if st.after != "" {
		b.WriteString("        <mj-text font-size=\"14px\" color=\"#4B5563\">" + esc(st.after) + "</mj-text>\n")
	}
	b.WriteString("      </mj-column>\n    </mj-section>\n")
	b.WriteString("    <mj-section background-color=\"#F3F4F6\" padding=\"16px 28px\">\n      <mj-column>\n")
	addr := esc(address)
	if strings.TrimSpace(address) == "" {
		addr = "Your company · Street and number · City, postcode"
	}
	b.WriteString("        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" color=\"#5B6472\" padding=\"0\">" + addr +
		"<br />You get this email because of your paid subscription to " + esc(product) + ". Membership emails can't be turned off.</mj-text>\n")
	b.WriteString("      </mj-column>\n    </mj-section>\n  </mj-body>\n</mjml>\n")
	return b.String()
}

// ensureMemberEmails creates the Memberships category and the member
// emails' drafts the workspace doesn't have yet. It returns how many it
// created.
func (s *Service) ensureMemberEmails(ctx context.Context, ws *store.Workspace, p *authz.Principal) (int, error) {
	have, err := s.Q.EarnMemberEmails(ctx, ws.ID)
	if err != nil {
		return 0, err
	}
	done := map[string]bool{}
	for _, h := range have {
		if h.TemplateID != nil {
			done[h.Email] = true
		}
	}
	if len(done) == len(starters) {
		return 0, nil
	}
	cat, err := s.Mailer.Category(ctx, ws)
	if err != nil {
		return 0, err
	}
	product := ws.Name
	address := templates.WorkspaceAddress(ws)
	n := 0
	for _, st := range starters {
		if done[st.email] {
			continue
		}
		id, err := s.Mailer.CreateDraft(ctx, ws, p, TemplateDraft{Key: st.key, Name: st.name, Description: st.description, Category: cat,
			Subject: st.subject, Preheader: st.preheader, MJML: st.mjml(product, address), Variables: st.vars, TestData: sampleData(st)})
		if err != nil {
			return n, fmt.Errorf("%s: %w", st.key, err)
		}
		tid := id
		if err := s.Q.EarnMemberEmailSet(ctx, store.EarnMemberEmailSetParams{WorkspaceID: ws.ID, Email: st.email, TemplateID: &tid}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func sampleData(st starter) map[string]any {
	out := map[string]any{}
	for _, v := range st.vars {
		out[v.Name] = v.Sample
	}
	return out
}

// shortTier is how readers name a tier: its name without the list it
// belongs to ("Weekly Digest Insider" on Weekly Digest is "Insider").
func shortTier(name string, lists []string) string { return paidaccess.ShortName(name, lists) }

// tierShortName loads a tier's short name.
func (s *Service) tierShortName(ctx context.Context, q *store.Queries, wsID uuid.UUID, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *id, WorkspaceID: wsID})
	if err != nil {
		return ""
	}
	names, _ := q.EarnListNames(ctx, t.ListIds)
	return shortTier(t.Name, names)
}

// BillingURL is a paid member's personal link to their billing portal.
func (s *Service) BillingURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID) string {
	tok := s.Links.Sign(linktoken.Claims{Kind: TokenBilling, WorkspaceID: ws.ID, ContactID: contact})
	return s.pageURL(ctx, ws, "billing/"+tok)
}

// UpgradeURL is a reader's personal upgrade link: the page knows them and
// credits the email it came from.
func (s *Service) UpgradeURL(ctx context.Context, ws *store.Workspace, contact, message uuid.UUID) string {
	tok := s.Links.Sign(linktoken.Claims{Kind: TokenUpgrade, WorkspaceID: ws.ID, ContactID: contact, MessageID: message})
	return s.pageURL(ctx, ws, "upgrade?r="+tok)
}

// memberEmailFor builds a member email of a subscription.
func (s *Service) memberEmailFor(email string, row store.PaidSubscription, ws *store.Workspace, tier string, at time.Time) memberEmail {
	data := map[string]any{"tier": tier}
	cur := row.Currency
	amount := row.AmountCents
	if p := publicPrice(amount, cur, row.Period); amount > 0 {
		data["price"] = p
	}
	if row.TrialEnd != nil {
		data["trial_end"] = s.dateText(ws, *row.TrialEnd)
	}
	if row.CurrentPeriodEnd != nil {
		data["renews_at"] = s.dateText(ws, *row.CurrentPeriodEnd)
	}
	end := firstTime(row.CancelAt, row.EndedAt, row.CurrentPeriodEnd)
	if end != nil {
		data["ends_at"] = s.dateText(ws, *end)
	}
	if email == EmailWelcome {
		switch {
		case row.Status == StatusTrialing && row.TrialEnd != nil && amount > 0:
			data["plan_line"] = fmt.Sprintf("Your trial is free until %s, then it's %s.", s.dateText(ws, *row.TrialEnd), publicPrice(amount, cur, row.Period))
		case row.CurrentPeriodEnd != nil && amount > 0:
			data["plan_line"] = fmt.Sprintf("You pay %s, renewing on %s.", publicPrice(amount, cur, row.Period), longDate(ws, *row.CurrentPeriodEnd))
		default:
			data["plan_line"] = ""
		}
	}
	return memberEmail{email: email, sub: row.ID, contact: row.ContactID, key: "earn:" + row.ID.String() + ":" + email, data: data}
}

// sendMemberEmail sends a member email after the event's changes are in,
// and puts it on the paid member's timeline: sent, or why not (no
// published version, no Transactional route, a suppressed address).
func (s *Service) sendMemberEmail(ctx context.Context, ws *store.Workspace, mode string, m memberEmail, pe *uuid.UUID) string {
	status, reason, version := "sent", "", 0
	var msgID *uuid.UUID
	var key string
	defer func() {
		data := map[string]any{"email": m.email, "status": status, "template": key}
		if reason != "" {
			data["reason"] = reason
		}
		if version > 0 {
			data["version"] = version
		}
		kind := "email_sent"
		if status != "sent" {
			kind = "email_not_sent"
		}
		sub := m.sub
		var subp *uuid.UUID
		if sub != uuid.Nil {
			subp = &sub
		}
		_ = s.timeline(ctx, s.Q, ws.ID, m.contact, subp, mode, kind, s.now(), nil, "", pe, "", nil, data)
		if subp != nil && (m.email == EmailWelcome || m.email == EmailCanceled) {
			_ = s.Q.EarnSubscriptionEmailed(ctx, store.EarnSubscriptionEmailedParams{ID: sub, Email: m.email})
		}
		if msgID != nil {
			_, _ = s.Pool.Exec(ctx, `UPDATE paid_events SET message_id = $2 WHERE id = (SELECT id FROM paid_events WHERE contact_id = $1 AND kind = 'email_sent' ORDER BY occurred_at DESC, id DESC LIMIT 1)`, m.contact, msgID)
		}
	}()
	if s.Mailer == nil {
		status, reason = "not_sent", "no_mailer"
		return status
	}
	tpl, err := s.memberTemplate(ctx, ws, m.email)
	if err != nil {
		status, reason = "not_sent", "no_template"
		return status
	}
	key, version = tpl.Key, tpl.LiveVersion
	c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: m.contact, WorkspaceID: ws.ID})
	if err != nil {
		status, reason = "not_sent", "no_contact"
		return status
	}
	data := map[string]any{}
	for k, v := range m.data {
		data[k] = v
	}
	if c.FirstName != "" {
		data["first_name"] = c.FirstName
	}
	data["manage_subscription_url"] = s.BillingURL(ctx, ws, c.ID)
	data["upgrade_url"] = s.UpgradeURL(ctx, ws, c.ID, uuid.Nil)
	if _, ok := data["next_issue_line"]; !ok && m.email == EmailWelcome {
		data["next_issue_line"] = ""
	}
	res, err := s.Mailer.Send(ctx, ws, MemberMessage{Template: tpl.Key, To: c.Email, ToName: strings.TrimSpace(c.FirstName + " " + c.LastName), Data: data, Key: m.key})
	switch {
	case errors.Is(err, ErrNotPublished):
		status, reason = "not_sent", "not_published"
	case errors.Is(err, ErrNoRoute):
		status, reason = "not_sent", "no_route"
	case errors.Is(err, ErrSuppressed):
		status, reason = "not_sent", "suppressed"
	case err != nil:
		status, reason = "not_sent", "failed"
		s.log().WarnContext(ctx, "member email", "email", m.email, "err", err)
	default:
		if res.MessageID != uuid.Nil {
			id := res.MessageID
			msgID = &id
		}
		if res.Provider != "" {
			reason = ""
		}
	}
	return status
}

// memberTemplate is the template a member email goes out with.
func (s *Service) memberTemplate(ctx context.Context, ws *store.Workspace, email string) (TemplateState, error) {
	rows, err := s.Q.EarnMemberEmails(ctx, ws.ID)
	if err != nil {
		return TemplateState{}, err
	}
	for _, r := range rows {
		if r.Email != email || r.TemplateID == nil {
			continue
		}
		states, err := s.Mailer.Templates(ctx, ws, []uuid.UUID{*r.TemplateID})
		if err != nil {
			return TemplateState{}, err
		}
		st, ok := states[*r.TemplateID]
		if !ok || st.Deleted {
			return TemplateState{}, errors.New("earn: template deleted")
		}
		return st, nil
	}
	return TemplateState{}, errors.New("earn: no template")
}
