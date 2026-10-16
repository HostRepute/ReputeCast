package demo

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/transactional"
)

// Acme Cloud's six transactional templates as the boards show them: their
// content, variables, test data and published versions. seedTransactional
// (seed_transactional.go) writes them with the rest of the transactional
// demo; this file holds what they say. Only trial-ending has a draft that
// differs from its live version (brief §4: "v1 live, v2 draft"): Daniel
// Park's v2, whose required trial_end_date none of the last 7 days' 71 calls
// sent.
//
// The seeder step registered here does one thing more: Beta Program's
// welcome prompt is dismissed, so List health offers the Welcome series for
// Product Updates.
func init() { seedTransactionalTemplates = seedWelcomePromptDismissal }

// txSeedTemplate is one template: its published versions, then the draft.
type txSeedTemplate struct {
	key, name, description, category string
	versions                         []txSeedVersion
	// draft is the content after the last version; nil when it matches.
	draft    *txSeedContent
	draftBy  string
	draftAgo time.Duration
	testData []transactional.TransactionalTestData
	// sender is the template's From (an email of seed_transactional.go's
	// senders); empty uses the workspace's default Transactional sender.
	sender string
	opens  bool
}

type txSeedVersion struct {
	content txSeedContent
	by      string
	ago     time.Duration
	change  string
}

type txSeedContent struct {
	subject, preheader string
	mjml               string
	vars               []transactional.TransactionalVariable
}

// demo is the content as the template API keeps it, compiled.
func (c txSeedContent) demo() (transactional.DemoTemplate, error) {
	d := transactional.DemoTemplate{Subject: c.subject, Preheader: c.preheader, MJML: c.mjml, Variables: c.vars}
	return d, d.Compile()
}

// acmeCloudMJML is Acme Cloud's product mail layout, as on the boards.
func acmeCloudMJML(heading string, paragraphs []string, button, href, after string, optional bool) string {
	var b strings.Builder
	b.WriteString("<mjml>\n  <mj-head>\n    <mj-attributes>\n      <mj-all font-family=\"Georgia, 'Times New Roman', serif\" />\n      <mj-text font-size=\"15px\" line-height=\"1.6\" color=\"#1F2937\" padding=\"0 0 14px\" />\n    </mj-attributes>\n  </mj-head>\n")
	b.WriteString("  <mj-body background-color=\"#F3F4F6\" width=\"600px\">\n")
	b.WriteString("    <mj-section background-color=\"#ffffff\" padding=\"20px 28px\" border-bottom=\"1px solid #EEF0F3\">\n      <mj-column>\n        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-weight=\"700\" font-size=\"15px\" color=\"#0B1220\" padding=\"0\">Acme Cloud</mj-text>\n      </mj-column>\n    </mj-section>\n")
	b.WriteString("    <mj-section background-color=\"#ffffff\" padding=\"26px 28px 8px\">\n      <mj-column>\n")
	fmt.Fprintf(&b, "        <mj-text font-size=\"24px\" line-height=\"1.25\" color=\"#0B1220\" font-weight=\"700\">%s</mj-text>\n", heading)
	for _, p := range paragraphs {
		fmt.Fprintf(&b, "        <mj-text>%s</mj-text>\n", p)
	}
	if button != "" {
		fmt.Fprintf(&b, "        <mj-button href=\"%s\" background-color=\"#0F766E\" color=\"#ffffff\" font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"14px\" font-weight=\"600\" border-radius=\"6px\" inner-padding=\"11px 18px\" align=\"left\" padding=\"8px 0 18px\">%s</mj-button>\n", href, button)
	}
	if after != "" {
		fmt.Fprintf(&b, "        <mj-text font-size=\"14px\" color=\"#4B5563\">%s</mj-text>\n", after)
	}
	b.WriteString("      </mj-column>\n    </mj-section>\n")
	b.WriteString("    <!-- rc:transactional-footer -->\n    <mj-section background-color=\"#F3F4F6\" padding=\"16px 28px\">\n      <mj-column>\n")
	foot := "You get this email because of your Acme Cloud account. Account emails can't be turned off."
	if optional {
		foot = `You get these emails because you have an Acme Cloud account. <a href="{{ unsubscribe_url }}" style="color:#5B6472">Turn these emails off</a> · <a href="{{ preferences_url }}" style="color:#5B6472">Manage notifications</a>`
	}
	fmt.Fprintf(&b, "        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" line-height=\"1.6\" color=\"#5B6472\" padding=\"0\">%s<br />%s</mj-text>\n", html.EscapeString(SendersFooterAddress), foot)
	b.WriteString("      </mj-column>\n    </mj-section>\n  </mj-body>\n</mjml>\n")
	return b.String()
}

func tv(name, typ string, required bool, sample, def string, sensitive bool) transactional.TransactionalVariable {
	return transactional.TransactionalVariable{Name: name, Type: typ, Required: required, Sample: sample, Default: def, Sensitive: sensitive}
}

func txSeedTemplates() []txSeedTemplate {
	day := 24 * time.Hour
	resetVars := []transactional.TransactionalVariable{
		tv("first_name", "text", false, "Sam", "there", false),
		tv("reset_url", "url", true, "https://app.acme.io/reset/sample-token", "", true),
		tv("expires_in", "text", true, "30 minutes", "", false),
	}
	reset := func(heading, after string) txSeedContent {
		return txSeedContent{subject: "Reset your Acme Cloud password", preheader: "This link works for {{ expires_in }}.", vars: resetVars,
			mjml: acmeCloudMJML(heading, []string{`Hi {{ first_name | default: "there" }},`, "We got a request to reset the password for your Acme Cloud account. The link below works for {{ expires_in }}."},
				"Reset password", "{{ reset_url }}", after, false)}
	}
	ignore := "If you didn't ask for this, you can ignore this email. Your password stays the same."
	magicVars := []transactional.TransactionalVariable{
		tv("sign_in_url", "url", true, "https://app.acme.io/magic/sample-token", "", true),
		tv("expires_in", "text", true, "15 minutes", "", false),
	}
	magic := func(after string) txSeedContent {
		return txSeedContent{subject: "Your sign-in link for Acme Cloud", preheader: "It works once, for {{ expires_in }}.", vars: magicVars,
			mjml: acmeCloudMJML("Sign in to Acme Cloud", []string{"Use the button below to sign in. The link works once, for {{ expires_in }}."}, "Sign in", "{{ sign_in_url }}", after, false)}
	}
	welcomeVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("dashboard_url", "url", true, "https://app.acme.io/", "", false)}
	welcome := txSeedContent{subject: "Welcome to Acme Cloud, {{ first_name | default: \"there\" }}", preheader: "Your workspace is ready.", vars: welcomeVars,
		mjml: acmeCloudMJML("Welcome to Acme Cloud", []string{`Hi {{ first_name | default: "there" }},`, "Your workspace is ready. Everything you set up is waiting on your dashboard."}, "Open your dashboard", "{{ dashboard_url }}", "Reply to this email if you need a hand; a person reads every reply.", false)}
	invoiceVars := []transactional.TransactionalVariable{
		tv("invoice_number", "text", true, "INV-2026-0418", "", false),
		tv("amount", "text", true, "$49.00", "", false),
		tv("paid_on", "date", true, "15 Oct 2026", "", false),
		tv("plan_name", "text", false, "Pro", "Acme Cloud", false),
		tv("invoice_url", "url", true, "https://app.acme.io/billing/INV-2026-0418", "", false),
	}
	invoice := func(subject string) txSeedContent {
		return txSeedContent{subject: subject, preheader: "Thanks for your payment. The PDF is attached.", vars: invoiceVars,
			mjml: acmeCloudMJML("Thanks for your payment", []string{"We received {{ amount }} for invoice {{ invoice_number }} on {{ paid_on }}. Your {{ plan_name | default: \"Acme Cloud\" }} plan stays active."},
				"View the invoice", "{{ invoice_url }}", "The invoice is attached as a PDF.", false)}
	}
	trialVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("days_left", "number", true, "3", "", false), tv("upgrade_url", "url", true, "https://app.acme.io/billing/plans", "", false)}
	trial := txSeedContent{subject: "Your Acme Cloud trial ends in {{ days_left }} days", preheader: "Upgrade to keep your projects.", vars: trialVars,
		mjml: acmeCloudMJML("Your trial ends in {{ days_left }} days", []string{`Hi {{ first_name | default: "there" }}, upgrade before then to keep your projects and your teammates.`}, "Upgrade now", "{{ upgrade_url }}", "", true)}
	trialDraft := txSeedContent{subject: "Your Acme Cloud trial ends on {{ trial_end_date }}", preheader: "Choose a plan to keep your projects.",
		vars: []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("trial_end_date", "date", true, "18 Oct 2026", "", false), tv("upgrade_url", "url", true, "https://app.acme.io/billing/plans", "", false)},
		mjml: acmeCloudMJML("Your trial ends on {{ trial_end_date }}", []string{`Hi {{ first_name | default: "there" }}, choose a plan before then to keep your projects and your teammates.`}, "Choose a plan", "{{ upgrade_url }}", "", true)}
	usageVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Sam", "there", false), tv("projects", "number", true, "3", "", false), tv("deploys", "number", true, "41", "", false), tv("usage_url", "url", true, "https://app.acme.io/usage", "", false)}
	usage := txSeedContent{subject: "Your week in Acme Cloud", preheader: "{{ projects }} projects, {{ deploys }} deploys.", vars: usageVars,
		mjml: acmeCloudMJML("Your week in Acme Cloud", []string{`Hi {{ first_name | default: "there" }}, here is what your team shipped this week.`}, "See your usage", "{{ usage_url }}", "", true)}
	return []txSeedTemplate{
		{key: "password-reset", name: "Password reset", description: "A reset link that expires", category: "Account",
			versions: []txSeedVersion{
				{reset("Reset your password", ""), "marcus@acme.io", 120 * day, "First version"},
				{reset("Reset your password", ignore), "marcus@acme.io", 60 * day, "Added what to do when you didn't ask"},
				{reset("Reset your password", ignore), "linh@acme.io", 9 * day, "Footer address and the 30 minute expiry"},
			},
			testData: []transactional.TransactionalTestData{
				{Name: "New user", Data: map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/sample-token", "expires_in": "30 minutes"}},
				{Name: "No first name", Data: map[string]any{"reset_url": "https://app.acme.io/reset/sample-token", "expires_in": "30 minutes"}},
			}},
		{key: "magic-link", name: "Magic sign-in link", description: "A one-time link instead of a password", category: "Account",
			versions: []txSeedVersion{
				{magic(""), "marcus@acme.io", 110 * day, "First version"},
				{magic("If you didn't try to sign in, you can ignore this email."), "sofia@acme.io", 20 * day, "What to do when you didn't try to sign in"},
			},
			testData: []transactional.TransactionalTestData{{Name: "Sign in", Data: map[string]any{"sign_in_url": "https://app.acme.io/magic/sample-token", "expires_in": "15 minutes"}}}},
		{key: "welcome", name: "Welcome to Acme Cloud", description: "The first email after sign-up", category: "Account",
			versions: []txSeedVersion{
				{welcome, "marcus@acme.io", 200 * day, "First version"},
				{welcome, "marcus@acme.io", 150 * day, "New dashboard link"},
				{welcome, "sofia@acme.io", 90 * day, "Shorter subject"},
				{welcome, "sofia@acme.io", 40 * day, "A person reads every reply"},
				{welcome, "linh@acme.io", 14 * day, "Footer address"},
			},
			testData: []transactional.TransactionalTestData{{Name: "New user", Data: map[string]any{"first_name": "Priya", "dashboard_url": "https://app.acme.io/"}}}},
		{key: "invoice-paid", name: "Invoice paid", description: "A receipt, with the PDF attached", category: "Billing", sender: "billing@notify.acme.io",
			versions: []txSeedVersion{
				{invoice("Your Acme Cloud invoice"), "linh@acme.io", 100 * day, "First version"},
				{invoice("Invoice {{ invoice_number }} paid · {{ amount }}"), "sofia@acme.io", 70 * day, "The invoice and the amount in the subject"},
				{invoice("Invoice {{ invoice_number }} paid · {{ amount }}"), "sofia@acme.io", 35 * day, "plan_name optional"},
				{invoice("Invoice {{ invoice_number }} paid · {{ amount }}"), "sofia@acme.io", 8 * day, "The PDF goes along"},
			},
			testData: []transactional.TransactionalTestData{{Name: "Pro plan", Data: map[string]any{"invoice_number": "INV-2026-0418", "amount": "$49.00", "paid_on": "15 Oct 2026", "plan_name": "Pro", "invoice_url": "https://app.acme.io/billing/INV-2026-0418"}}}},
		{key: "trial-ending", name: "Trial ends in 3 days", description: "A reminder before the trial ends", category: "Reminders",
			versions: []txSeedVersion{{trial, "marcus@acme.io", 43 * day, "First version"}},
			draft:    &trialDraft, draftBy: "daniel@acme.io", draftAgo: 25 * time.Minute,
			testData: []transactional.TransactionalTestData{
				{Name: "Trial user", Data: map[string]any{"first_name": "Priya", "trial_end_date": "18 Oct 2026", "upgrade_url": "https://app.acme.io/billing/plans"}},
				{Name: "No first name", Data: map[string]any{"trial_end_date": "18 Oct 2026", "upgrade_url": "https://app.acme.io/billing/plans"}},
			}},
		{key: "usage-summary", name: "Weekly usage summary", description: "Every Monday: projects and deploys", category: "Product updates", opens: true,
			versions: func() []txSeedVersion {
				out := []txSeedVersion{}
				for i, by := range []string{"marcus@acme.io", "marcus@acme.io", "sofia@acme.io", "sofia@acme.io", "linh@acme.io", "sofia@acme.io"} {
					change := "First version"
					if i > 0 {
						change = "Copy edits"
					}
					out = append(out, txSeedVersion{usage, by, time.Duration(180-30*i) * day, change})
				}
				return out
			}(),
			testData: []transactional.TransactionalTestData{
				{Name: "Busy week", Data: map[string]any{"first_name": "Sam", "projects": 3, "deploys": 41, "usage_url": "https://app.acme.io/usage"}},
				{Name: "Quiet week", Data: map[string]any{"first_name": "Sam", "projects": 1, "deploys": 0, "usage_url": "https://app.acme.io/usage"}},
				{Name: "No first name", Data: map[string]any{"projects": 2, "deploys": 7, "usage_url": "https://app.acme.io/usage"}},
			}},
	}
}

// seedWelcomePromptDismissal: Beta Program's welcome prompt was dismissed,
// so List health offers the Welcome series for Product Updates.
func seedWelcomePromptDismissal(ctx context.Context, s *Seeder) error {
	beta, ok := s.Lists["Beta Program"]
	if !ok {
		return nil
	}
	var by any
	if u, ok := s.Users["linh@acme.io"]; ok {
		by = u.ID
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO welcome_prompt_dismissals (workspace_id, list_id, dismissed_by, dismissed_at) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
		s.WS().ID, beta, by, s.Ago(3*24*time.Hour))
	return err
}
