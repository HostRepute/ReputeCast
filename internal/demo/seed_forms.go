package demo

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/publicurl"
)

func init() { seedForms = seedFormsImpl }

//go:embed assets/forms/acme-mark.svg
var formsLogo []byte

// FormsDNSOverlay answers the demo's custom domain for the public pages
// (news.acme.io) with a CNAME to this instance, so its DNS check passes in
// development like the board's "Published · news.acme.io/subscribe".
func FormsDNSOverlay(r dnscheck.Resolver, urls publicurl.Builder) dnscheck.Resolver {
	host := "localhost"
	if u, err := url.Parse(urls.Base()); err == nil && u.Hostname() != "" {
		host = strings.ToLower(u.Hostname())
	}
	z := dnscheck.NewZone()
	z.Add("news.acme.io", "CNAME", host+".")
	return dnscheck.Overlay(z, nil, r)
}

// formsDemoForm is one of the three forms of the FormBuilder board.
type formsDemoForm struct {
	name, kind, slug, publicID string
	lists, defaults            []string // list names
	listChoice                 bool
	fields                     []map[string]any
	content                    map[string]any
	consentVersion             int
	created                    int // days ago
	// The last 30 days, as the builder's numbers show them.
	views, submissions, confirmed, blocked int64
}

// seedFormsImpl seeds the Forms & public pages boards: the publisher
// branding of FormsStates, the three forms of FormBuilder with their
// settings and 30-day numbers, the copy of the public pages
// (PublicSubscribe, PublicConfirm, PublicPreferences, PublicUnsubscribe)
// and the archive of PublicArchive, built from the sent campaigns. Sam
// Rivera, the example subscriber of the public pages, is the audience
// seeder's contact; nothing about them changes here.
func seedFormsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	list := func(name string) uuid.UUID { return s.Lists[name] }
	digest, updates, beta := list("Weekly Digest"), list("Product Updates"), list("Beta Program")
	ids := func(names []string) []uuid.UUID {
		out := []uuid.UUID{}
		for _, n := range names {
			out = append(out, list(n))
		}
		return out
	}

	// The publisher's colours are stored with the derived ones the boards use.
	sum := sha256.Sum256(formsLogo)
	logoHash := hex.EncodeToString(sum[:8])
	branding, _ := json.Marshal(map[string]any{
		"publisherName": "Acme", "accent": "#4338CA", "background": "#F5F3FF", "accentHover": "#3730A3", "accentSoft": "#DDD6FE",
		"accentText": "#4338CA", "onAccent": "#FFFFFF", "font": "plex", "poweredBy": true, "privacyUrl": "https://acme.io/privacy",
		"logo": map[string]string{"name": "acme-mark.svg", "type": "image/svg+xml", "hash": logoHash},
	})
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET branding = branding || $2::jsonb WHERE id = $1`, ws.ID, branding); err != nil {
		return err
	}

	// The three forms of the builder.
	consent := "I agree to receive emails from Acme. Unsubscribe any time with one click. See the privacy policy."
	email := map[string]any{"key": "email", "required": true}
	content := func(heading, description, button, footnote string) map[string]any {
		return map[string]any{"heading": heading, "description": description, "button": button, "footnote": footnote, "listsLabel": "What would you like?",
			"emailPlaceholder": "you@company.com", "namePlaceholder": "Sam", "popupDelay": 8, "popupScroll": 0, "popupRepeatDays": 14}
	}
	demoForms := []formsDemoForm{
		{name: "Newsletter signup", kind: "embedded", slug: "newsletter-signup", publicID: "frm_8c2e1a",
			lists: []string{"Weekly Digest", "Product Updates"}, defaults: []string{"Weekly Digest"}, listChoice: true,
			fields: []map[string]any{email, {"key": "first_name"}},
			content: content("The Acme newsletter", "One email a week with what shipped, what's next and the occasional deep dive. 48,000 readers.",
				"Subscribe", "We'll send a confirmation link first. No spam, ever."),
			consentVersion: 3, created: 210, views: 14208, submissions: 812, confirmed: 731, blocked: 1106},
		{name: "Footer mini form", kind: "embedded", slug: "footer-mini-form", publicID: "frm_3f91d0",
			lists: []string{"Weekly Digest"}, defaults: []string{"Weekly Digest"},
			fields:         []map[string]any{email},
			content:        content("Get the Weekly Digest", "What shipped and what's next, every Tuesday.", "Subscribe", "One email a week. Unsubscribe any time."),
			consentVersion: 2, created: 160, views: 9412, submissions: 388, confirmed: 349, blocked: 512},
		{name: "Beta apply", kind: "hosted", slug: "beta-apply", publicID: "frm_b7a412",
			lists: []string{"Beta Program"}, defaults: []string{"Beta Program"},
			fields: []map[string]any{email, {"key": "first_name", "required": true}, {"key": "company"}},
			content: content("Apply for the Acme beta", "Early access to features before they ship. We read every application and reply within a week.",
				"Apply", "We'll send a confirmation link first."),
			consentVersion: 1, created: 64, views: 1904, submissions: 96, confirmed: 90, blocked: 37},
	}
	spam, _ := json.Marshal(map[string]any{"honeypot": true, "min_seconds": 3, "rate_limit_per_ip": 5, "block_disposable": true, "turnstile": true})
	origins := []string{"acme.io", "docs.acme.io", "blog.acme.io"}
	formIDs := map[string]uuid.UUID{}
	for _, f := range demoForms {
		id := ID()
		fields, _ := json.Marshal(f.fields)
		body, _ := json.Marshal(f.content)
		created := s.Ago(time.Duration(f.created) * 24 * time.Hour)
		published := created.Add(26 * time.Hour)
		updated := s.designDay(2, 16, 40)
		// Lifetime totals: the builder shows 30 days, the API keeps both.
		if _, err := s.Tx.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, kind, slug, public_id, list_ids, default_list_ids, list_choice, double_opt_in,
			fields, content, spam_protection, consent_text, consent_version, allowed_origins, after_submit, redirect_url, status, published_at,
			views, submissions, confirmed, blocked, created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10,$11,$12,$13,$14,$15,'message','','live',$16,$17,$18,$19,$20,$21,$21,$22,$23)`,
			id, ws.ID, f.name, f.kind, f.slug, f.publicID, ids(f.lists), ids(f.defaults), f.listChoice, fields, body, spam, consent, f.consentVersion,
			origins, published, f.views*5, f.submissions*5, f.confirmed*5, f.blocked*5, linh.ID, created, updated); err != nil {
			return fmt.Errorf("form %s: %w", f.name, err)
		}
		formIDs[f.name] = id
		if err := formsDailyNumbers(ctx, s, id, f); err != nil {
			return err
		}
	}
	s.Extra["forms"] = formIDs

	// The public pages: the subscribe page is the Newsletter signup form;
	// each page's copy is the board's.
	pages, _ := json.Marshal(map[string]any{
		"subscribe": map[string]any{
			"heading":     "The Acme newsletter",
			"description": "One email a week with what shipped, what's next and the occasional deep dive. Read by 48,000 people building on Acme.",
			"footnote":    "We'll send a confirmation link first. No spam, ever. You can change lists or pause emails from any message.",
			"listText": map[string]string{
				digest.String():  "Every Tuesday morning. {subscribers} subscribers.",
				updates.String(): "Only when something ships. About twice a month.",
			},
			"index": true,
		},
		"confirm": map[string]any{
			"listText": map[string]string{
				digest.String():  "Tuesdays at 09:00 your time.",
				updates.String(): "Only when something ships, about twice a month.",
				beta.String():    "Early access invitations and feedback requests. Rare.",
			},
			"offerText": map[string]string{updates.String(): "release news", beta.String(): "early access invitations"},
			"inboxTip":  true,
		},
		"preferences": map[string]any{
			"lists": []string{},
			"listText": map[string]string{
				digest.String():  "Tuesdays. What shipped, what's next.",
				updates.String(): "Only when something ships, about twice a month.",
				beta.String():    "Early access invitations and feedback requests. Rare.",
			},
			"frequency": true, "pause": true, "firstName": true,
		},
		"unsubscribe": map[string]any{
			"fewer": true, "fewerTitle": "Too many emails? Get fewer instead.",
			"fewerText":   "Switch to one monthly best-of and nothing else. Most people who pick this stay subscribed.",
			"fewerButton": "Send me the monthly best-of only", "reasons": true,
		},
		"archive":      map[string]any{"enabled": true, "lists": []string{digest.String(), updates.String()}, "subscribeBox": true, "index": true},
		"previewEmail": "sam.rivera@outlook.com",
	})
	host := "localhost"
	if s.Cfg != nil {
		if u, err := url.Parse(s.Cfg.HTTP.PublicURL); err == nil && u.Hostname() != "" {
			host = strings.ToLower(u.Hostname())
		}
	}
	turnstileSecret, err := s.Box.SealString("0x4AAAAAAAdemo-secret-not-real", "turnstile:"+ws.ID.String()+":secret")
	if err != nil {
		return err
	}
	verified := s.Ago(180 * 24 * time.Hour)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO public_page_settings (workspace_id, hosted_form_id, pages, custom_domain, domain_status, domain_detail,
		domain_checked_at, domain_verified_at, turnstile_site_key, turnstile_secret_enc, turnstile_status, turnstile_checked_at, logo, logo_type, updated_by, updated_at)
		VALUES ($1,$2,$3,'news.acme.io','verified',$4,$5,$6,'0x4AAAAAAADnPIDROrmt1Wwj',$7,'ok',$8,$9,'image/svg+xml',$10,$11)
		ON CONFLICT (workspace_id) DO NOTHING`,
		ws.ID, formIDs["Newsletter signup"], pages, "CNAME points at "+host+".", s.Ago(3*time.Hour), verified, turnstileSecret,
		s.Ago(50*time.Hour), formsLogo, linh.ID, s.designDay(2, 16, 40)); err != nil {
		return err
	}
	return formsArchive(ctx, s, digest, updates)
}

// formsDailyNumbers spreads a form's 30-day numbers over the days: more
// views around Tuesday's digest, and the last two days' submissions still
// partly waiting for the confirmation click (the "pending" line).
func formsDailyNumbers(ctx context.Context, s *Seeder, form uuid.UUID, f formsDemoForm) error {
	const days = 30
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	today := s.Now.In(loc)
	weights := make([]float64, days)
	confirmWeights := make([]float64, days)
	for i := range days {
		d := today.AddDate(0, 0, -i)
		w := 1.0
		switch d.Weekday() {
		case time.Tuesday:
			w = 1.6
		case time.Wednesday:
			w = 1.25
		case time.Saturday, time.Sunday:
			w = 0.7
		}
		weights[i] = w
		confirmWeights[i] = w
		switch i {
		case 0:
			weights[i] *= 0.45 // today isn't over
			confirmWeights[i] = weights[i] * 0.25
		case 1:
			confirmWeights[i] = weights[i] * 0.6
		}
	}
	views := Split(f.views, weights)
	subs := Split(f.submissions, weights)
	conf := Split(f.confirmed, confirmWeights)
	blocked := Split(f.blocked, weights)
	for i := range days {
		if conf[i] > subs[i] {
			// Keep confirmations within the day's submissions (moves the rest to older days).
			extra := conf[i] - subs[i]
			conf[i] = subs[i]
			for j := days - 1; j > i && extra > 0; j-- {
				room := subs[j] - conf[j]
				take := min(room, extra)
				conf[j] += take
				extra -= take
			}
		}
		d := today.AddDate(0, 0, -i)
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO form_daily (form_id, day, views, submissions, confirmed, blocked) VALUES ($1,$2,$3,$4,$5,$6)`,
			form, day, views[i], subs[i], conf[i], blocked[i]); err != nil {
			return err
		}
	}
	return nil
}

// formsIssue is an archived issue of the PublicArchive board.
type formsIssue struct {
	campaign, slug, title, excerpt, label string
	list                                  uuid.UUID
}

// formsArchive files the sent campaigns in the archive: the five issues of
// the board with their titles and excerpts, then the imported Brevo digests.
// Sent campaigns without content (the seeded history) get an issue body
// built like the Weekly Digest template.
func formsArchive(ctx context.Context, s *Seeder, digest, updates uuid.UUID) error {
	ws := s.WS()
	issues := []formsIssue{
		{campaign: "October Product Update", slug: "what-shipped-in-october", list: updates,
			title:   "What shipped in October, and what we're retiring",
			excerpt: "Feature flags are generally available, audit log exports stream to S3 or your SIEM, and the v1 API retires on 30 November. Plus the Q4 roadmap."},
		{campaign: "Weekly Digest #141", slug: "the-case-for-boring-infrastructure", list: digest, label: "#141",
			title:   "The case for boring infrastructure",
			excerpt: "Why we moved three services back to Postgres, a reader's guide to feature flags, and the community links of the week."},
		{campaign: "Weekly Digest #140", slug: "shipping-on-fridays-revisited", list: digest, label: "#140",
			title:   "Shipping on Fridays, revisited",
			excerpt: "Our incident data from a year of Friday deploys, how the on-call rotation changed, and a new template for post-mortems."},
		{campaign: "Weekly Digest #139", slug: "reading-the-audit-log-like-a-detective", list: digest, label: "#139",
			title:   "Reading the audit log like a detective",
			excerpt: "Three real investigations, the queries behind them, and why we keep 13 months of history."},
		{campaign: "Welcome to news.acme.io", slug: "we-have-moved-to-news-acme-io", list: digest, label: "special",
			title:   "We've moved: the newsletter now comes from news.acme.io",
			excerpt: "New sending domain, same team. What to add to your contacts and how to tell us what you want to receive."},
	}
	for i, h := range formsBrevoHistory {
		num := 138 - i
		issues = append(issues, formsIssue{campaign: fmt.Sprintf("Weekly Digest #%d", num), slug: fmt.Sprintf("%d-%s", num, slugWords(h[0])),
			list: digest, label: fmt.Sprintf("#%d", num), title: h[0], excerpt: h[1]})
	}
	logo := ""
	var logoID uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM media_assets WHERE workspace_id = $1 AND file_name = 'acme-logo.png' LIMIT 1`, ws.ID).Scan(&logoID); err == nil {
		logo = media.URL(logoID, "acme-logo.png")
	}
	for _, it := range issues {
		id, ok := s.Campaigns[it.campaign]
		if !ok {
			continue
		}
		var at time.Time
		if err := s.Tx.QueryRow(ctx, `SELECT coalesce(started_at, created_at) FROM campaigns WHERE id = $1`, id).Scan(&at); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO archive_issues (campaign_id, workspace_id, slug, list_id, title, excerpt, label, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) ON CONFLICT (campaign_id) DO NOTHING`,
			id, ws.ID, it.slug, it.list, it.title, it.excerpt, it.label, at.Add(2*time.Hour)); err != nil {
			return fmt.Errorf("archive %s: %w", it.campaign, err)
		}
		res := mjml.Compile(formsIssueMJML(it, logo))
		if len(res.Errors) > 0 {
			return fmt.Errorf("archive %s: %s", it.campaign, res.Errors[0].Message)
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET html = $2, plain_text = $3 WHERE id = $1 AND html = ''`,
			id, res.HTML, contentcheck.PlainText(res.HTML)); err != nil {
			return err
		}
	}
	return nil
}

// slugWords is a short slug of a title.
func slugWords(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if b.Len() >= 60 {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// formsIssueMJML is an archived digest: the Weekly Digest header and footer
// around the issue's lead and the changelog highlights.
func formsIssueMJML(it formsIssue, logo string) string {
	esc := func(s string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
	}
	header := strings.ReplaceAll(composerDigestHeader, "@@acme-logo.png@@", logo)
	if logo == "" {
		header = ""
	}
	return `<mjml>
` + fmt.Sprintf(composerDigestHead, esc(it.title), esc(it.excerpt)) + `
  <mj-body width="600px" background-color="#F3F4F6">
` + header + `
    <mj-section background-color="#ffffff" padding="24px 24px 8px">
      <mj-column>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + esc(it.title) + `</mj-text>
        <mj-text font-size="15px" line-height="1.55" padding="0px 0px 12px">Hi {{ first_name | default: "there" }}, ` + esc(lowerFirst(it.excerpt)) + `</mj-text>
        <mj-text font-size="15px" line-height="1.55">Reply to this email if there's something you'd like us to cover next time. We read every answer.</mj-text>
      </mj-column>
    </mj-section>
` + composerSavedSections[0].mjml + `
` + composerDigestFooter + `
  </mj-body>
</mjml>
`
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s // an acronym ("SSO for…")
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}

// formsBrevoHistory titles the Weekly Digests sent from Brevo before the
// migration (#138 down to #100).
var formsBrevoHistory = [][2]string{
	{"Postgres 18 in production: the first month", "What changed in our query plans, the two settings we tuned, and a migration checklist you can borrow."},
	{"What our support queue taught us about onboarding", "The five questions every new team asks, and the screens we rebuilt so they don't have to."},
	{"A field guide to rate limits", "How our limits work, how to read the headers, and the retry loop we recommend."},
	{"The quiet launch of regional data residency", "EU and Australian regions are live. Here's how to move a workspace and what stays the same."},
	{"How we test migrations before they ship", "Shadow traffic, production snapshots and the one rule that has saved us three times this year."},
	{"Ten small things that make the dashboard faster", "Most of them took less than a day. Together they cut the median page load by 40%."},
	{"Designing APIs you can deprecate", "Versioning, sunset headers and the email we send a year before anything breaks."},
	{"Inside our incident review", "The template, the meeting and why nobody is ever the root cause."},
	{"Webhooks that survive a bad day", "Signed payloads, 72-hour retries and the dead-letter view that shipped this week."},
	{"The summer roadmap, in one page", "Feature flags, audit log exports and SSO for every plan, with the dates we're aiming for."},
	{"Why we write design docs before code", "The two-page format we use, and the questions a good review answers."},
	{"Search got smarter: what changed under the hood", "Typo tolerance, filters you can combine and results that arrive in under 50 ms."},
	{"Measuring what customers actually use", "The events we track, the ones we stopped tracking, and how we keep it private."},
	{"Our first year of public status updates", "Every incident we posted, what we learned writing them, and the new subscription options."},
	{"Feature flags without the mess", "Naming, ownership and the cleanup job that removes stale flags for you."},
	{"Exports, imports and the art of not losing data", "Checksums, resumable uploads and the importer's new dry run."},
	{"How the billing rewrite went", "Three months, zero double charges, and the tests that made us confident."},
	{"Notes from our customer advisory board", "What twelve teams asked for, and the three requests already in progress."},
	{"Keeping p99 latency honest", "Our dashboards, our alerts and the slow query that hid for six weeks."},
	{"SSO for every plan", "SAML and OIDC are no longer an Enterprise feature. Here's how to turn them on."},
	{"The spring cleaning release", "Forty small fixes, two removed settings and a faster settings page."},
	{"What we learned from 1,000 support calls", "The patterns, the surprises and the help articles we rewrote."},
	{"Choosing boring dependencies", "Our checklist for adding a library, and the two we removed this quarter."},
	{"Audit logs are now searchable", "Filter by person, action or date, and export what you find."},
	{"A better way to invite your team", "Roles at invite time, expiring links and invitations that survive a typo."},
	{"Retries, idempotency and you", "Why every write endpoint takes an idempotency key now, with examples."},
	{"Our open-source contributions this quarter", "Patches upstream, two libraries we released and how to get involved."},
	{"Behind the new pricing page", "Fewer plans, clearer limits and the calculator customers asked for."},
	{"The case for fewer dashboards", "We deleted half of ours. Here's what replaced them."},
	{"Testing in production, carefully", "Dark launches, kill switches and the review that comes before either."},
	{"Year in review: what shipped in 2025", "Twelve months of releases on one page, with the numbers behind them."},
	{"Holiday deploy freeze, and what we do instead", "How we keep shipping safely in December without stopping completely."},
	{"Faster cold starts for every region", "The caching change that cut first-request time by two thirds."},
	{"Postmortem: the 14-minute outage", "What failed, how we noticed and the three changes we made the same week."},
	{"How we choose what to build next", "The scoring sheet, the customer calls and the things we said no to."},
	{"Dark mode is (almost) here", "A preview of the new theme and the accessibility checks it had to pass."},
	{"Scaling the team without slowing down", "Smaller teams, clearer ownership and the meetings we cancelled."},
	{"The API changelog gets a home", "Every change, dated and searchable, with an RSS feed for your tooling."},
	{"Issue 100: thank you for reading", "A hundred issues in, a look back at the first one and what readers asked us for."},
}
