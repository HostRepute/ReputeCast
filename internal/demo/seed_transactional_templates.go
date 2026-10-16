package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional templates for the editor and publish boards (X03): Acme
// Cloud's password reset (v3 live, Variables open), its welcome email (a
// draft with an undeclared and an unused variable), the trial reminder (v1
// live, a draft v2 that breaks the calls of the last 7 days) and the weekly
// usage summary (ready to publish v7). Templates X02's seed already made are
// left as they are. Also: Beta Program's welcome prompt is dismissed, so
// List health offers the Welcome series for Product Updates.
func init() { seedTransactionalTemplates = seedTransactionalTemplatesImpl }

// txSeedTemplate is one template: its published versions, then the draft.
type txSeedTemplate struct {
	key, name, description, category string
	versions                         []txSeedVersion
	// draft is the content after the last version; nil when it matches.
	draft     *txSeedContent
	draftBy   string
	draftAt   time.Duration
	testData  []transactional.TransactionalTestData
	liveSince time.Duration
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

const acmeAddress = "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105"

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
	fmt.Fprintf(&b, "        <mj-text font-family=\"'IBM Plex Sans', system-ui, sans-serif\" font-size=\"12px\" line-height=\"1.6\" color=\"#5B6472\" padding=\"0\">%s<br />%s</mj-text>\n", html.EscapeString(acmeAddress), foot)
	b.WriteString("      </mj-column>\n    </mj-section>\n  </mj-body>\n</mjml>\n")
	return b.String()
}

// txDraftHash fingerprints content the way the template API does
// (internal/transactional draftHash), so a draft saved unchanged still
// matches its live version.
func txDraftHash(c txSeedContent, compiled string) string {
	h := sha256.New()
	vars, _ := json.Marshal(c.vars)
	for _, part := range []string{c.subject, c.preheader, "visual", c.mjml, compiled, "", string(vars), "", "", fmt.Sprint(false, false)} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func txCompile(c txSeedContent) (string, error) {
	res := mjml.Compile(mjml.SetHeadMeta(c.mjml, c.subject, c.preheader))
	if len(res.Errors) > 0 {
		return "", fmt.Errorf("mjml: %s", res.Errors[0].Message)
	}
	return res.HTML, nil
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
	welcomeVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("dashboard_url", "url", true, "https://app.acme.io/", "", false)}
	welcome := txSeedContent{subject: "Welcome to Acme Cloud, {{ first_name | default: \"there\" }}", preheader: "Your workspace is ready.", vars: welcomeVars,
		mjml: acmeCloudMJML("Welcome to Acme Cloud", []string{`Hi {{ first_name | default: "there" }},`, "Your workspace is ready. Everything you set up is waiting on your dashboard."}, "Open your dashboard", "{{ dashboard_url }}", "Reply to this email if you need a hand; a person reads every reply.", false)}
	welcomeDraft := welcome
	welcomeDraft.vars = append(append([]transactional.TransactionalVariable{}, welcomeVars...), tv("plan_name", "text", false, "Pro", "your plan", false))
	welcomeDraft.mjml = acmeCloudMJML("Welcome to Acme Cloud", []string{`Hi {{ first_name | default: "there" }},`, "Thanks for joining {{ company_name }}. Your workspace is ready, and your first invoice arrives on the 1st."}, "Open your dashboard", "{{ dashboard_url }}", "Reply to this email if you need a hand; a person reads every reply.", false)
	trialVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("days_left", "number", true, "3", "", false), tv("upgrade_url", "url", true, "https://app.acme.io/billing/plans", "", false)}
	trial := txSeedContent{subject: "Your Acme Cloud trial ends in {{ days_left }} days", preheader: "Upgrade to keep your projects.", vars: trialVars,
		mjml: acmeCloudMJML("Your trial ends in {{ days_left }} days", []string{`Hi {{ first_name | default: "there" }}, upgrade before then to keep your projects and your teammates.`}, "Upgrade now", "{{ upgrade_url }}", "", true)}
	trialDraft := txSeedContent{subject: "Your Acme Cloud trial ends on {{ trial_end_date }}", preheader: "Choose a plan to keep your projects.",
		vars: []transactional.TransactionalVariable{tv("first_name", "text", false, "Priya", "there", false), tv("trial_end_date", "date", true, "18 Oct 2026", "", false), tv("upgrade_url", "url", true, "https://app.acme.io/billing/plans", "", false)},
		mjml: acmeCloudMJML("Your trial ends on {{ trial_end_date }}", []string{`Hi {{ first_name | default: "there" }}, choose a plan before then to keep your projects and your teammates.`}, "Choose a plan", "{{ upgrade_url }}", "", true)}
	usageVars := []transactional.TransactionalVariable{tv("first_name", "text", false, "Sam", "there", false), tv("projects", "number", true, "3", "", false), tv("deploys", "number", true, "41", "", false), tv("usage_url", "url", true, "https://app.acme.io/usage", "", false)}
	usage := func(heading string) txSeedContent {
		return txSeedContent{subject: "Your week in Acme Cloud", preheader: "{{ projects }} projects, {{ deploys }} deploys.", vars: usageVars,
			mjml: acmeCloudMJML(heading, []string{`Hi {{ first_name | default: "there" }}, here is what your team shipped this week.`}, "See your usage", "{{ usage_url }}", "", true)}
	}
	return []txSeedTemplate{
		{key: "password-reset", name: "Password reset", description: "A reset link that expires", category: "Account", liveSince: 9 * day,
			versions: []txSeedVersion{
				{reset("Reset your password", ""), "marcus@acme.io", 120 * day, "First version"},
				{reset("Reset your password", ignore), "marcus@acme.io", 60 * day, "Added what to do when you didn't ask"},
				{reset("Reset your password", ignore), "linh@acme.io", 9 * day, "Footer address and the 30 minute expiry"},
			},
			testData: []transactional.TransactionalTestData{
				{Name: "New user", Data: map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/sample-token", "expires_in": "30 minutes"}},
				{Name: "No first name", Data: map[string]any{"reset_url": "https://app.acme.io/reset/sample-token", "expires_in": "30 minutes"}},
			}},
		{key: "welcome", name: "Welcome to Acme Cloud", description: "The first email after sign-up", category: "Account", liveSince: 14 * day,
			versions: []txSeedVersion{
				{welcome, "marcus@acme.io", 200 * day, "First version"},
				{welcome, "marcus@acme.io", 150 * day, "New dashboard link"},
				{welcome, "sofia@acme.io", 90 * day, "Shorter subject"},
				{welcome, "sofia@acme.io", 40 * day, "A person reads every reply"},
				{welcome, "linh@acme.io", 14 * day, "Footer address"},
			},
			draft: &welcomeDraft, draftBy: "sofia@acme.io", draftAt: 50 * time.Minute,
			testData: []transactional.TransactionalTestData{{Name: "New user", Data: map[string]any{"first_name": "Priya", "dashboard_url": "https://app.acme.io/"}}}},
		{key: "trial-ending", name: "Trial ends in 3 days", description: "A reminder before the trial ends", category: "Reminders", liveSince: 43 * day,
			versions: []txSeedVersion{{trial, "marcus@acme.io", 43 * day, "First version"}},
			draft:    &trialDraft, draftBy: "daniel@acme.io", draftAt: 25 * time.Minute,
			testData: []transactional.TransactionalTestData{
				{Name: "Trial user", Data: map[string]any{"first_name": "Priya", "trial_end_date": "18 Oct 2026", "upgrade_url": "https://app.acme.io/billing/plans"}},
				{Name: "No first name", Data: map[string]any{"trial_end_date": "18 Oct 2026", "upgrade_url": "https://app.acme.io/billing/plans"}},
			}},
		{key: "usage-summary", name: "Weekly usage summary", description: "Every Monday: projects and deploys", category: "Product updates", liveSince: 21 * day,
			versions: func() []txSeedVersion {
				out := []txSeedVersion{}
				for i, by := range []string{"marcus@acme.io", "marcus@acme.io", "sofia@acme.io", "sofia@acme.io", "linh@acme.io", "sofia@acme.io"} {
					change := "First version"
					if i > 0 {
						change = "Copy edits"
					}
					out = append(out, txSeedVersion{usage("Your week in Acme Cloud"), by, time.Duration(180-30*i) * day, change})
				}
				return out
			}(),
			draft: func() *txSeedContent {
				c := usage("Your week in Acme Cloud: {{ projects }} projects, {{ deploys }} deploys")
				return &c
			}(), draftBy: "sofia@acme.io", draftAt: 2 * time.Hour,
			testData: []transactional.TransactionalTestData{
				{Name: "Busy week", Data: map[string]any{"first_name": "Sam", "projects": 3, "deploys": 41, "usage_url": "https://app.acme.io/usage"}},
				{Name: "Quiet week", Data: map[string]any{"first_name": "Sam", "projects": 1, "deploys": 0, "usage_url": "https://app.acme.io/usage"}},
				{Name: "No first name", Data: map[string]any{"projects": 2, "deploys": 7, "usage_url": "https://app.acme.io/usage"}},
			}},
	}
}

func seedTransactionalTemplatesImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	if err := s.Q.TxnSettingsEnsure(ctx, ws.ID); err != nil {
		return err
	}
	categories := map[string]uuid.UUID{}
	cats, err := s.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, c := range cats {
		categories[c.Name] = c.ID
	}
	wantCats := []struct{ name, kind, desc string }{
		{"Account", "essential", "Sign-in, password and account emails"},
		{"Reminders", "optional", "Trial and renewal reminders"},
		{"Product updates", "optional", "Usage summaries and product news"},
	}
	for _, c := range wantCats {
		if _, ok := categories[c.name]; ok {
			continue
		}
		id := uuid.Must(uuid.NewV7())
		if err := s.Q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: id, WorkspaceID: ws.ID, Name: c.name, Kind: c.kind, Description: c.desc, CreatedAt: s.Ago(400 * 24 * time.Hour)}); err != nil {
			return err
		}
		categories[c.name] = id
	}
	user := func(email string) *uuid.UUID {
		if u, ok := s.Users[email]; ok {
			id := u.ID
			return &id
		}
		return nil
	}
	for _, t := range txSeedTemplates() {
		if taken, err := s.Q.TxnTemplateKeyTaken(ctx, store.TxnTemplateKeyTakenParams{WorkspaceID: ws.ID, Key: t.key, ID: uuid.Nil}); err != nil {
			return err
		} else if taken {
			continue // X02's seed made it.
		}
		if err := seedTxTemplate(ctx, s, t, categories[t.category], user); err != nil {
			return fmt.Errorf("%s: %w", t.key, err)
		}
	}
	// The trial reminder's calls of the last 7 days still send days_left:
	// publishing the draft as it is would break them.
	if err := seedTrialCalls(ctx, s); err != nil {
		return err
	}
	// List health offers the Welcome series for Product Updates: Beta
	// Program's prompt was dismissed.
	if beta, ok := s.Lists["Beta Program"]; ok {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO welcome_prompt_dismissals (workspace_id, list_id, dismissed_by, dismissed_at) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
			ws.ID, beta, user("linh@acme.io"), s.Ago(3*24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

func seedTxTemplate(ctx context.Context, s *Seeder, t txSeedTemplate, category uuid.UUID, user func(string) *uuid.UUID) error {
	ws := s.WS()
	id := uuid.Must(uuid.NewV7())
	first := t.versions[0]
	created := s.Ago(first.ago + time.Hour)
	draft := t.versions[len(t.versions)-1].content
	if t.draft != nil {
		draft = *t.draft
	}
	draftHTML, err := txCompile(draft)
	if err != nil {
		return err
	}
	testData, _ := json.Marshal(t.testData)
	vars, _ := json.Marshal(draft.vars)
	origin, _ := json.Marshal(map[string]any{})
	cat := category
	if err := s.Q.TxnTemplateInsert(ctx, store.TxnTemplateInsertParams{ID: id, WorkspaceID: ws.ID, Key: t.key, Name: t.name, Description: t.description, CategoryID: &cat,
		Subject: draft.subject, Preheader: draft.preheader, EditorMode: "visual", Mjml: draft.mjml, Html: draftHTML, PlainText: "",
		Variables: vars, TestData: testData, ReplyTo: "", TrackOpens: false, TrackClicks: false,
		DraftHash: txDraftHash(draft, draftHTML), Now: created, CreatedBy: user(first.by), Origin: origin}); err != nil {
		return err
	}
	for i, v := range t.versions {
		vHTML, err := txCompile(v.content)
		if err != nil {
			return err
		}
		vVars, _ := json.Marshal(v.content.vars)
		if err := s.Q.TxnVersionInsert(ctx, store.TxnVersionInsertParams{TemplateID: id, Version: int32(i + 1), Subject: v.content.subject, Preheader: v.content.preheader,
			EditorMode: "visual", Mjml: v.content.mjml, Html: vHTML, PlainText: "", Variables: vVars, TestData: testData,
			ContentHash: txDraftHash(v.content, vHTML), SizeBytes: int32(len(vHTML)), Change: v.change, PublishedBy: user(v.by), PublishedAt: s.Ago(v.ago)}); err != nil {
			return err
		}
	}
	live := int32(len(t.versions))
	last := t.versions[len(t.versions)-1]
	liveVars, _ := json.Marshal(last.content.vars)
	if err := s.Q.TxnTemplatePublish(ctx, store.TxnTemplatePublishParams{WorkspaceID: ws.ID, ID: id, Version: &live, Now: s.Ago(last.ago), LiveBy: user(last.by), Variables: liveVars}); err != nil {
		return err
	}
	// first_published_at is when v1 went live; TxnTemplatePublish stamped the last one.
	revision := int32(len(t.versions))
	savedBy, savedAt := user(last.by), s.Ago(last.ago)
	if t.draft != nil {
		revision++
		savedBy, savedAt = user(t.draftBy), s.Ago(t.draftAt)
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE transactional_templates SET first_published_at = $2, variables = $3, draft_revision = $4, draft_saved_by = $5, draft_saved_at = $6 WHERE id = $1`,
		id, s.Ago(first.ago), vars, revision, savedBy, savedAt); err != nil {
		return err
	}
	return s.Q.TxnTemplateInsertDraftRevision(ctx, store.TxnTemplateInsertDraftRevisionParams{TemplateID: id, Revision: revision, Subject: draft.subject, Preheader: draft.preheader,
		Mjml: draft.mjml, Html: draftHTML, Variables: vars, SavedBy: savedBy, SavedAt: savedAt, Label: "", Source: "manual"})
}

// seedTrialCalls are the trial reminder's calls of the last 6 days: 71, all
// sending days_left. The latest went to ops@contoso.example at 10:39.
func seedTrialCalls(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	var id uuid.UUID
	var version int32
	var category *uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT id, coalesce(live_version, 1), category_id FROM transactional_templates WHERE workspace_id = $1 AND key = 'trial-ending' AND deleted_at IS NULL`, ws.ID).Scan(&id, &version, &category)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // no trial reminder to call
	}
	if err != nil {
		return err
	}
	var have int
	if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM transactional_messages WHERE template_id = $1`, id).Scan(&have); err != nil || have > 0 {
		return err
	}
	people := []string{"ops@contoso.example", "dana@fabrikam.example", "lee@northwind.example", "ana@litware.example", "joe@adatum.example", "mia@tailspin.example"}
	firsts := []string{"Ops", "Dana", "Lee", "Ana", "Joe", "Mia"}
	// The latest call: 10:39 today, in the workspace's timezone.
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	y, mo, d := s.Now.In(loc).Date()
	latest := time.Date(y, mo, d, 10, 39, 0, 0, loc)
	if latest.After(s.Now) {
		latest = latest.Add(-24 * time.Hour)
	}
	for i := range 71 {
		at := latest.Add(-time.Duration(i) * 6 * 24 * time.Hour / 71)
		email := people[i%len(people)]
		data, _ := json.Marshal(map[string]any{"first_name": firsts[i%len(firsts)], "days_left": 3, "upgrade_url": "https://app.acme.io/billing/plans"})
		mid := uuid.Must(uuid.NewV7())
		if err := s.Q.TxnInsertMessage(ctx, store.TxnInsertMessageParams{ID: mid, WorkspaceID: ws.ID, Email: email, MailboxProvider: "other", Status: "delivered", QueuedAt: at, Kind: "api"}); err != nil {
			return err
		}
		v := version
		if err := s.Q.TxnInsertMessageLog(ctx, store.TxnInsertMessageLogParams{MessageID: mid, WorkspaceID: ws.ID, Source: "api", TemplateID: &id, TemplateKey: "trial-ending",
			TemplateName: "Trial ends in 3 days", TemplateVersion: &v, CategoryID: category, CategoryName: "Reminders", CategoryKind: "optional",
			IdempotencyKey: fmt.Sprintf("trial-%04d", i), Subject: "Your Acme Cloud trial ends in 3 days", FromEmail: "notify@acme.io", FromName: "Acme Cloud",
			ToName: firsts[i%len(firsts)], Cc: []string{}, Bcc: []string{}, Tags: []string{"trial"}, Metadata: json.RawMessage(`{}`), Data: data,
			DataKeys: []string{"days_left", "first_name", "upgrade_url"}, Sensitive: []string{}, Attachments: json.RawMessage(`[]`), SizeBytes: 9800,
			RejectDetail: json.RawMessage(`{}`), Links: []string{}, Route: json.RawMessage(`{}`), AcceptedAt: at}); err != nil {
			return err
		}
	}
	return nil
}
