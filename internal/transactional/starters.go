package transactional

import (
	"context"
	"fmt"
	"html"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// TransactionalStarter is a ready-made template: password reset, sign-in
// link, email verification, welcome, receipt or invoice, trial ending, team
// invitation and usage alert. Each comes with its variables declared and
// test data.
type TransactionalStarter struct {
	Key         string                  `json:"key" example:"password-reset"`
	Name        string                  `json:"name" example:"Password reset"`
	Description string                  `json:"description"`
	Kind        string                  `json:"kind" enum:"essential,optional" doc:"The kind of category it belongs in"`
	Attachment  bool                    `json:"attachment" doc:"Usually sent with an attachment (a PDF invoice)"`
	Subject     string                  `json:"subject"`
	Preheader   string                  `json:"preheader"`
	Variables   []TransactionalVariable `json:"variables"`
	TestData    []TransactionalTestData `json:"testData"`
	// body is the main section: a heading, paragraphs and a button.
	heading string
	body    []string
	button  string
	href    string
	after   string
	// bodyAfter are paragraphs after the button, in the body's style; why
	// is the Essential footer's reason ("You get this email because you
	// bought from …"), the account's by default.
	bodyAfter []string
	why       string
}

// TransactionalStarterList is the starter gallery.
type TransactionalStarterList struct {
	Items []TransactionalStarter `json:"items"`
}

var starterList = []TransactionalStarter{
	{
		Key: "password-reset", Name: "Password reset", Description: "A reset link that expires", Kind: KindEssential,
		Subject: "Reset your {{product}} password", Preheader: "This link works for {{ expires_in }}.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Sam", Default: "there"},
			{Name: "reset_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://app.example.com/reset/sample-token"},
			{Name: "expires_in", Type: TypeText, Required: true, Sample: "30 minutes"},
		},
		TestData: []TransactionalTestData{
			{Name: "New user", Data: map[string]any{"first_name": "Sam", "reset_url": "https://app.example.com/reset/sample-token", "expires_in": "30 minutes"}},
			{Name: "No first name", Data: map[string]any{"reset_url": "https://app.example.com/reset/sample-token", "expires_in": "30 minutes"}},
		},
		heading: "Reset your password",
		body:    []string{"Hi {{ first_name }},", "We got a request to reset the password for your {{product}} account. The link below works for {{ expires_in }}."},
		button:  "Reset password", href: "{{ reset_url }}",
		after: "If you didn't ask for this, you can ignore this email. Your password stays the same.",
	},
	{
		Key: "magic-link", Name: "Sign-in link", Description: "A one-time link instead of a password", Kind: KindEssential,
		Subject: "Your sign-in link for {{product}}", Preheader: "It works once, for {{ expires_in }}.",
		Variables: []TransactionalVariable{
			{Name: "sign_in_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://app.example.com/login/sample-token"},
			{Name: "expires_in", Type: TypeText, Required: true, Sample: "15 minutes"},
		},
		TestData: []TransactionalTestData{
			{Name: "Sign-in", Data: map[string]any{"sign_in_url": "https://app.example.com/login/sample-token", "expires_in": "15 minutes"}},
		},
		heading: "Sign in to {{product}}",
		body:    []string{"Use this link to sign in. It works once, for {{ expires_in }}."},
		button:  "Sign in", href: "{{ sign_in_url }}",
		after: "If you didn't try to sign in, you can ignore this email.",
	},
	{
		Key: "email-verification", Name: "Email verification", Description: "Confirm a new address", Kind: KindEssential,
		Subject: "Confirm your email address", Preheader: "One click and you're set.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Sam", Default: "there"},
			{Name: "verify_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://app.example.com/verify/sample-token"},
		},
		TestData: []TransactionalTestData{
			{Name: "New user", Data: map[string]any{"first_name": "Sam", "verify_url": "https://app.example.com/verify/sample-token"}},
		},
		heading: "Confirm your email address",
		body:    []string{"Hi {{ first_name }},", "Confirm that this is your address so we can reach you about your {{product}} account."},
		button:  "Confirm my address", href: "{{ verify_url }}",
		after: "If you didn't create an account, you can ignore this email.",
	},
	{
		Key: "welcome", Name: "Welcome", Description: "The first email after sign-up", Kind: KindEssential,
		Subject: "Welcome to {{product}}, {{ first_name }}", Preheader: "Your account is ready.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Priya", Default: "there"},
			{Name: "dashboard_url", Type: TypeURL, Required: true, Sample: "https://app.example.com/"},
		},
		TestData: []TransactionalTestData{
			{Name: "New user", Data: map[string]any{"first_name": "Priya", "dashboard_url": "https://app.example.com/"}},
			{Name: "No first name", Data: map[string]any{"dashboard_url": "https://app.example.com/"}},
		},
		heading: "Welcome to {{product}}",
		body:    []string{"Hi {{ first_name }},", "Your account is ready. Everything you set up is waiting on your dashboard."},
		button:  "Open your dashboard", href: "{{ dashboard_url }}",
		after: "Reply to this email if you need a hand; a person reads every reply.",
	},
	{
		Key: "invoice-paid", Name: "Receipt or invoice", Description: "A receipt, with the PDF attached", Kind: KindEssential, Attachment: true,
		Subject: "Invoice {{ invoice_number }} paid · {{ amount }}", Preheader: "Thanks for your payment.",
		Variables: []TransactionalVariable{
			{Name: "invoice_number", Type: TypeText, Required: true, Sample: "INV-2026-0418"},
			{Name: "amount", Type: TypeText, Required: true, Sample: "$49.00"},
			{Name: "paid_on", Type: TypeDate, Required: true, Sample: "2026-10-15"},
			{Name: "plan_name", Type: TypeText, Sample: "Pro", Default: "your plan"},
			{Name: "invoice_url", Type: TypeURL, Required: true, Sample: "https://app.example.com/billing/INV-2026-0418"},
		},
		TestData: []TransactionalTestData{
			{Name: "Monthly plan", Data: map[string]any{"invoice_number": "INV-2026-0418", "amount": "$49.00", "paid_on": "2026-10-15", "plan_name": "Pro", "invoice_url": "https://app.example.com/billing/INV-2026-0418"}},
		},
		heading: "Payment received",
		body:    []string{"We received {{ amount }} for {{ plan_name }} on {{ paid_on }}. Invoice {{ invoice_number }} is attached as a PDF."},
		button:  "View invoice", href: "{{ invoice_url }}",
		after: "Questions about this invoice? Reply to this email.",
	},
	{
		Key: "trial-ending", Name: "Trial ending", Description: "A reminder before the trial ends", Kind: KindOptional,
		Subject: "Your {{product}} trial ends in {{ days_left }} days", Preheader: "Pick a plan to keep your work.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Sam", Default: "there"},
			{Name: "days_left", Type: TypeNumber, Required: true, Sample: "3"},
			{Name: "upgrade_url", Type: TypeURL, Required: true, Sample: "https://app.example.com/billing/plans"},
		},
		TestData: []TransactionalTestData{
			{Name: "Trial user", Data: map[string]any{"first_name": "Sam", "days_left": 3, "upgrade_url": "https://app.example.com/billing/plans"}},
			{Name: "No first name", Data: map[string]any{"days_left": 3, "upgrade_url": "https://app.example.com/billing/plans"}},
		},
		heading: "Your trial ends in {{ days_left }} days",
		body:    []string{"Hi {{ first_name }},", "Pick a plan before your trial ends to keep your projects and settings."},
		button:  "Choose a plan", href: "{{ upgrade_url }}",
		after: "Not ready to decide? Your data stays for 30 days after the trial.",
	},
	{
		Key: "team-invitation", Name: "Team invitation", Description: "Invite someone to a team", Kind: KindEssential,
		Subject: "{{ inviter_name }} invited you to {{ team_name }}", Preheader: "Accept the invitation to join.",
		Variables: []TransactionalVariable{
			{Name: "inviter_name", Type: TypeText, Required: true, Sample: "Marcus Chen"},
			{Name: "team_name", Type: TypeText, Required: true, Sample: "Acme Engineering"},
			{Name: "invite_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://app.example.com/invite/sample-token"},
			{Name: "expires_in", Type: TypeText, Sample: "7 days", Default: "7 days"},
		},
		TestData: []TransactionalTestData{
			{Name: "Invitation", Data: map[string]any{"inviter_name": "Marcus Chen", "team_name": "Acme Engineering", "invite_url": "https://app.example.com/invite/sample-token", "expires_in": "7 days"}},
		},
		heading: "Join {{ team_name }}",
		body:    []string{"{{ inviter_name }} invited you to {{ team_name }} on {{product}}. The invitation works for {{ expires_in }}."},
		button:  "Accept the invitation", href: "{{ invite_url }}",
		after: "If you weren't expecting this, you can ignore it.",
	},
	{
		Key: "usage-alert", Name: "Usage alert", Description: "When an account nears a limit", Kind: KindEssential,
		Subject: "You've used {{ used_percent }}% of your {{ metric }}", Preheader: "Here is where you stand.",
		Variables: []TransactionalVariable{
			{Name: "metric", Type: TypeText, Required: true, Sample: "API requests"},
			{Name: "used_percent", Type: TypeNumber, Required: true, Sample: "92"},
			{Name: "limit", Type: TypeText, Required: true, Sample: "1,000,000 a month"},
			{Name: "usage_url", Type: TypeURL, Required: true, Sample: "https://app.example.com/usage"},
		},
		TestData: []TransactionalTestData{
			{Name: "Near the limit", Data: map[string]any{"metric": "API requests", "used_percent": 92, "limit": "1,000,000 a month", "usage_url": "https://app.example.com/usage"}},
		},
		heading: "{{ used_percent }}% of your {{ metric }} used",
		body:    []string{"Your plan includes {{ limit }}. You've used {{ used_percent }}% so far."},
		button:  "See your usage", href: "{{ usage_url }}",
		after: "You get this email when an account passes 80% of a limit.",
	},
}

func starterByKey(key string) (TransactionalStarter, bool) {
	for _, s := range starterList {
		if s.Key == key {
			return s, true
		}
	}
	return TransactionalStarter{}, false
}

// in is the starter in a language: its words from the catalog
// (starter.<key>.*), the English above where the catalog has none. A new
// template's content is the workspace's own from then on, so it is written
// in the workspace's language (CreateTemplate); the gallery is in the
// member's.
func (st TransactionalStarter) in(l i18n.Lang) TransactionalStarter {
	if l.Locale == i18n.English || l.Locale == "" {
		return st
	}
	word := func(field, english string) string {
		m, err := texts.Lookup("starter." + st.Key + "." + field)
		if err != nil {
			return english
		}
		return m.Text(l, nil)
	}
	st.Name, st.Description = word("name", st.Name), word("description", st.Description)
	st.Subject, st.Preheader = word("subject", st.Subject), word("preheader", st.Preheader)
	st.heading, st.button, st.after, st.why = word("heading", st.heading), word("button", st.button), word("after", st.after), word("why", st.why)
	body := make([]string, len(st.body))
	for i, p := range st.body {
		body[i] = word("body."+strconv.Itoa(i), p)
	}
	after := make([]string, len(st.bodyAfter))
	for i, p := range st.bodyAfter {
		after[i] = word("bodyAfter."+strconv.Itoa(i), p)
	}
	st.body, st.bodyAfter = body, after
	vars := slices.Clone(st.Variables)
	samples := map[string][2]string{}
	for i, v := range vars {
		vars[i].Sample, vars[i].Default = word("sample."+v.Name, v.Sample), word("default."+v.Name, v.Default)
		samples[v.Name] = [2]string{v.Sample, vars[i].Sample}
	}
	tests := make([]TransactionalTestData, len(st.TestData))
	for i, td := range st.TestData {
		data := maps.Clone(td.Data)
		for k, v := range data {
			s, ok := v.(string)
			if !ok {
				continue
			}
			if m, err := texts.Lookup(fmt.Sprintf("starter.%s.test.%d.%s", st.Key, i, k)); err == nil {
				data[k] = m.Text(l, nil)
			} else if sm, ok := samples[k]; ok && sm[0] == s {
				data[k] = sm[1]
			}
		}
		tests[i] = TransactionalTestData{Name: word("test."+strconv.Itoa(i), td.Name), Data: data}
	}
	st.Variables, st.TestData = vars, tests
	return st
}

// Starters lists the starter templates in the member's language, with the
// workspace's product name filled in.
func (s *Service) Starters(ctx context.Context, ws *store.Workspace) (TransactionalStarterList, error) {
	product := s.product(ctx, ws)
	out := TransactionalStarterList{Items: make([]TransactionalStarter, 0, len(starterList))}
	for _, st := range starterList {
		st = st.in(lang(ctx))
		st.Subject = fillProduct(st.Subject, product)
		out.Items = append(out.Items, st)
	}
	return out, nil
}

// product is the product's name in starters and pages.
func (s *Service) product(ctx context.Context, ws *store.Workspace) string {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return ws.Name
	}
	senders, _ := s.transactionalSenders(ctx, s.Q, ws.ID)
	return productName(st, senders, ws)
}

func fillProduct(s, product string) string { return strings.ReplaceAll(s, "{{product}}", product) }

// mjmlFor writes the starter's MJML: a header with the product name, the
// message, its button, and the footer. Essential categories get the
// transactional footer (company and address, no unsubscribe link);
// Optional ones the category's unsubscribe link. The footer's words are in
// l, the starter's language (in).
func (st TransactionalStarter) mjmlFor(l i18n.Lang, product, address string) string {
	foot := func(key string) string { return texts.M(key).Text(l, i18n.Args{"product": html.EscapeString(product)}) }
	esc := func(s string) string { return fillProduct(html.EscapeString(s), product) }
	var b strings.Builder
	b.WriteString("<mjml>\n  <mj-head>\n    <mj-title>" + esc(st.Subject) + "</mj-title>\n    <mj-preview>" + esc(st.Preheader) + "</mj-preview>\n")
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
		fmt.Fprintf(&b, "        <mj-button href=\"%s\" background-color=\"#0F766E\" color=\"#ffffff\" font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"14px\" font-weight=\"600\" border-radius=\"6px\" inner-padding=\"11px 18px\" align=\"left\" padding=\"8px 0 18px\">%s</mj-button>\n", st.href, esc(st.button))
	}
	for _, p := range st.bodyAfter {
		b.WriteString("        <mj-text>" + esc(p) + "</mj-text>\n")
	}
	if st.after != "" {
		b.WriteString("        <mj-text font-size=\"14px\" color=\"#4B5563\">" + esc(st.after) + "</mj-text>\n")
	}
	b.WriteString("      </mj-column>\n    </mj-section>\n")
	b.WriteString("    <mj-section background-color=\"#F3F4F6\" padding=\"16px 28px\">\n      <mj-column>\n")
	addr := esc(address)
	if strings.TrimSpace(address) == "" {
		addr = foot("starter.footer.address")
	}
	if st.Kind == KindOptional {
		b.WriteString("        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" color=\"#5B6472\" padding=\"0\">" + addr +
			"<br />" + foot("starter.footer.optional") + " <a href=\"{{ unsubscribe_url }}\" style=\"color:#5B6472\">" + foot("starter.footer.turnOff") +
			"</a> · <a href=\"{{ preferences_url }}\" style=\"color:#5B6472\">" + foot("starter.footer.manage") + "</a></mj-text>\n")
	} else if st.why != "" {
		b.WriteString("        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" color=\"#5B6472\" padding=\"0\">" + addr +
			"<br />" + esc(st.why) + "</mj-text>\n")
	} else {
		b.WriteString("        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" color=\"#5B6472\" padding=\"0\">" + addr +
			"<br />" + foot("starter.footer.essential") + "</mj-text>\n")
	}
	b.WriteString("      </mj-column>\n    </mj-section>\n  </mj-body>\n</mjml>\n")
	return b.String()
}
