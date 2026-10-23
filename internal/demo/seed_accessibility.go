package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/readiness"
)

func init() { seedAccessibility = seedAccessibilityImpl }

// The accessibility checks of the design expansion (brief §4,
// "Accessibility policy and gallery" and Weekly Digest #143): the policy
// (Warn, 12 px, set by Linh Nguyen on 6 Oct), Weekly Digest #143 with its
// six issues and its Review & send run, the button of the Welcome series
// v4 step 3 B, the saved template "Acme digest v3" with two issues, and the
// titles of the pages #143 links to.

const (
	digest143Subject   = "Scheduled rollouts, faster exports, and 6 weeks left on the v1 API"
	digest143Preheader = "Plan a flag change for 2 a.m. and sleep through it."
)

func seedAccessibilityImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	if _, err := s.Tx.Exec(ctx, `INSERT INTO accessibility_policies (workspace_id, on_serious, min_text_px, updated_by, updated_at) VALUES ($1,'warn',12,$2,$3)
		ON CONFLICT (workspace_id) DO NOTHING`, ws.ID, linh.ID, s.designDay(9, 10, 5)); err != nil {
		return err
	}
	for url, t := range accessibilityTitles {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO accessibility_link_titles (url, title, lang, status, fetched_at) VALUES ($1,$2,$3,200,$4)
			ON CONFLICT (url) DO UPDATE SET title = EXCLUDED.title, lang = EXCLUDED.lang, fetched_at = EXCLUDED.fetched_at`, url, t.Title, t.Lang, s.Ago(2*time.Hour)); err != nil {
			return err
		}
	}
	steps := []func(context.Context, *Seeder) error{seedDigest143, accessibilityWelcomeButton, accessibilityDigestTemplate}
	for _, step := range steps {
		if err := step(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// digest143MJML is Weekly Digest #143 in English at a version: the intro
// gained "in any timezone" in v8 and the CSV card its new sentence in v9
// (after German was translated from v6).
func digest143MJML(version int) string {
	intro := `Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time, in any timezone. <a href="https://docs.acme.io/scheduled-rollouts" style="color:#0F766E">Click here</a> to see how it works.`
	if version < 8 {
		intro = strings.Replace(intro, " ahead of time, in any timezone.", " ahead of time.", 1)
	}
	csv := "Exports of up to a million rows now finish in under a minute."
	if version < 9 {
		csv = "Exports now run up to 3× faster."
	}
	button := func(href, label string) string {
		return `<mj-button href="` + href + `" align="left" background-color="#0F766E" color="#ffffff" font-family="` + composerPlexFont + `" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">` + label + `</mj-button>`
	}
	card := func(alt, title, text, href, label, padding string) string {
		return `      <mj-column width="50%" padding="` + padding + `">
        <mj-image src="@@article-card.png@@" alt="` + alt + `" width="268px" height="120px" fluid-on-mobile="true" padding="0px 0px 10px"></mj-image>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px"><h3 style="margin:0;font-size:17px;line-height:1.3;font-weight:700">` + title + `</h3></mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">` + text + `</mj-text>
        ` + button(href, label) + `
      </mj-column>
`
	}
	social := func(href, file, alt string) string {
		return `<td style="padding:0px 6px"><a href="` + href + `"><img src="@@` + file + `@@" alt="` + alt + `" width="28" height="28" style="display:block;border:0"></a></td>`
	}
	return `<mjml>
` + fmt.Sprintf(composerDigestHead, composerXMLText(digest143Subject), composerXMLText(digest143Preheader)) + `
  <mj-body width="600px" background-color="#F3F4F6">
    <!-- rc:header-logo -->
    <mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="40%" vertical-align="middle">
        <mj-image src="@@acme-logo.png@@" href="https://acme.io" alt="Acme" width="96px" height="28px" align="left" padding="0px"></mj-image>
      </mj-column>
      <mj-column width="60%" vertical-align="middle">
        <mj-text align="right" font-family="` + composerPlexFont + `" font-size="12px" color="#5B6472"><a href="https://acme.io/blog" style="color:#5B6472;text-decoration:none">Blog</a> · <a href="https://docs.acme.io" style="color:#5B6472;text-decoration:none">Docs</a> · <a href="https://acme.io/changelog" style="color:#5B6472;text-decoration:none">Changelog</a></mj-text>
      </mj-column>
    </mj-section>
    <!-- rc:hero -->
    <mj-section background-color="#ffffff" padding="20px 24px 0px">
      <mj-column>
        <mj-image src="@@scheduled-rollouts.png@@" width="552px" height="200px" padding="0px"></mj-image>
      </mj-column>
    </mj-section>
    <!-- rc:intro -->
    <mj-section background-color="#ffffff" padding="16px 24px 8px">
      <mj-column>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" color="#0B1220" padding="0px 0px 10px"><h1 style="margin:0;font-size:26px;line-height:1.2;font-weight:700">Scheduled rollouts are here</h1></mj-text>
        <mj-text font-size="15px" line-height="1.55">` + intro + `</mj-text>
      </mj-column>
    </mj-section>
    <!-- rc:cards -->
    <mj-section background-color="#ffffff" padding="16px 24px">
` + card("The schedule picker in the flag editor", "Scheduled rollouts", "Pick the minute a flag changes. Every scheduled change shows in the audit log.", "https://docs.acme.io/scheduled-rollouts", "Read the docs", "0px 8px 0px 0px") +
		card("The CSV export dialog", "Faster CSV exports", csv, "https://acme.io/changelog", "See the changelog", "0px 0px 0px 8px") + `    </mj-section>
    <!-- rc:v1-API -->
    <mj-section background-color="#ffffff" padding="8px 24px">
      <mj-column>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">6 weeks left on the v1 API</mj-text>
        <mj-text font-size="15px" line-height="1.55">The v1 API stops answering on 30 November. Migration guide: <a href="https://docs.acme.io/migrate-v2" style="color:#0F766E">https://docs.acme.io/migrate-v2</a></mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="12px 24px 20px">
      <mj-column>
        <mj-button href="https://docs.acme.io/migrate-v2/plan" align="center" background-color="#5EBFB0" color="#ffffff" font-family="` + composerPlexFont + `" font-size="15px" font-weight="700" border-radius="6px" inner-padding="13px 22px" padding="0px">Plan your migration</mj-button>
      </mj-column>
    </mj-section>
    <mj-raw><table align="center" cellpadding="0" cellspacing="0" border="0" style="margin:0 auto;background:#ffffff;width:600px;max-width:100%"><tr><td align="center" style="padding:4px 0px 20px"><table align="center" cellpadding="0" cellspacing="0" border="0"><tr>` +
		social("https://x.com/acme", "social-x.png", "Acme on X") + social("https://www.linkedin.com/company/acme", "social-linkedin.png", "Acme on LinkedIn") + social("https://github.com/acme", "social-github.png", "Acme on GitHub") +
		`</tr></table></td></tr></table></mj-raw>
    <!-- rc:footer -->
    <mj-section background-color="#F3F4F6" padding="18px 24px">
      <mj-column>
        <mj-text align="center" font-family="` + composerPlexFont + `" font-size="11px" line-height="1.6" color="#5B6472">Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105<br><a href="{{ unsubscribe_url }}" style="color:#0F766E;text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#0F766E;text-decoration:none">Email preferences</a> · <a href="{{ view_in_browser_url }}" style="color:#0F766E;text-decoration:none">View in browser</a></mj-text>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>
`
}

// demoImages maps the image library's file names to their URLs.
func demoImages(ctx context.Context, s *Seeder) (func(string) string, error) {
	r := &readinessSeed{s: s, ws: s.WS(), images: map[string]string{}}
	if err := r.loadImages(ctx); err != nil {
		return nil, err
	}
	return r.withImages, nil
}

// seedDigest143 creates Weekly Digest #143 (a draft for Tue 21 Oct 09:00,
// written by Sofia Rossi, English v9) unless another seeder already did,
// and its Review & send run of 14:18.
func seedDigest143(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	sofia, linh := s.Users["sofia@acme.io"], s.Users["linh@acme.io"]
	withImages, err := demoImages(ctx, s)
	if err != nil {
		return err
	}
	id, ok := s.Campaigns["Weekly Digest #143"]
	created := !ok
	if created {
		id = ID()
		n := int64(31604)
		aud, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{s.Lists["Weekly Digest"]}})
		sched := s.designDay(-6, 9, 0)
		schedule := json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, sched.Format(time.RFC3339)))
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count, schedule, thresholds, reply_to,
			created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,'Weekly Digest #143','draft',$3,$4,$5,$6,$7,$8,$9,'hello@acme.io',$10,$10,$11,$11)`,
			id, ws.ID, digest143Subject, digest143Preheader, s.Senders["digest@news.acme.io"], aud, n, schedule, json.RawMessage(readinessThresholds),
			sofia.ID, s.designDay(2, 15, 40)); err != nil {
			return err
		}
		s.Campaigns["Weekly Digest #143"] = id
		if err := planRoutes(ctx, s, id, n, map[string]float64{"ses": 0.9, "postal": 0.1}); err != nil {
			return err
		}
	}
	history := []struct {
		day, h, m int
		label     string
	}{
		{2, 15, 40, "created from template “Acme digest v3”"}, {2, 15, 58, "hero image"}, {2, 16, 20, "headline and intro"},
		{1, 10, 5, "two cards"}, {1, 11, 30, "v1 API notice"}, {1, 15, 50, "migration button"},
		{0, 8, 40, "social links"}, {0, 9, 5, "intro"}, {0, 9, 12, "“Faster CSV exports” card"},
	}
	var src, html string
	if !created {
		// Another seeder wrote its content; only the run is ours.
		if err := s.Tx.QueryRow(ctx, `SELECT mjml, html FROM campaigns WHERE id = $1`, id).Scan(&src, &html); err != nil {
			return err
		}
		history = nil
	}
	for i, v := range history {
		version := i + 1
		at := s.designDay(v.day, v.h, v.m)
		if v.day == 0 {
			at = s.sampleToday(v.h, v.m)
		}
		// The conditional blocks and the poll of the X11 sample are in every
		// version (seed_polls.go).
		src = withImages(x11Digest143(s, digest143MJML(version)))
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile Weekly Digest #143: %s", res.Errors[0].Message)
		}
		html, _ = accessibility.Finalize(res.HTML, "en")
		source := "autosave"
		if version == 1 {
			source = "template"
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT DO NOTHING`,
			id, version, digest143Subject, digest143Preheader, src, html, contentcheck.PlainText(html), sofia.ID, at, v.label, source, len(html)); err != nil {
			return err
		}
	}
	last := s.sampleToday(9, 12)
	if created {
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET mjml = $2, html = $3, plain_text = $4, content_hash = $5, version = $6, updated_by = $7, updated_at = $8, autosaved_at = $9
			WHERE id = $1`, id, src, html, contentcheck.PlainText(html), campaigns.ContentHash(digest143Subject, digest143Preheader, src, html, nil), len(history), sofia.ID, last,
			s.sampleToday(11, 20)); err != nil {
			return err
		}
	}
	// Review & send ran at 14:18, before scheduling.
	r := &readinessSeed{s: s, ws: ws, images: map[string]string{}}
	checked, compiled := s.sampleToday(14, 18), last
	var sched time.Time
	var version int32
	if err := s.Tx.QueryRow(ctx, `SELECT coalesce((schedule->>'at')::timestamptz, now()), version FROM campaigns WHERE id = $1`, id).Scan(&sched, &version); err != nil {
		return err
	}
	f := r.baseFacts(checked)
	f.Campaign = readiness.CampaignFacts{ID: id.String(), Name: "Weekly Digest #143", Subject: digest143Subject, Preheader: digest143Preheader, Version: int(version),
		CompiledAt: &compiled, SenderName: "Acme Newsletter", SenderEmail: "digest@news.acme.io", SenderDomain: "news.acme.io", StartAt: sched, ScheduleMode: "at"}
	f.Audience = readiness.AudienceFacts{BaseLabel: "Weekly Digest list", GroupLabel: "Weekly Digest", BaseKind: readiness.BaseList, BaseName: "Weekly Digest", Lists: []string{"Weekly Digest"}, Base: 31988, Suppressed: 331,
		Role: 27, Duplicates: 26, Final: 31604, Unengaged: 412, UnengagedDays: 180,
		Mailboxes: map[string]int64{"gmail": 14538, "outlook": 6637, "yahoo": 2844, "icloud": 2528, "other": 5057}}
	f.Route = readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{r.ses(28444, 90), r.postal(3160, 10)}}
	links := []contentcheck.ContentCheckLink{}
	for i, l := range [][3]string{
		{"https://acme.io", "Acme", "image"}, {"https://acme.io/blog", "Blog", "link"}, {"https://docs.acme.io", "Docs", "link"}, {"https://acme.io/changelog", "Changelog", "link"},
		{"https://docs.acme.io/scheduled-rollouts", "Click here", "link"}, {"https://docs.acme.io/scheduled-rollouts", "Read the docs", "button"},
		{"https://acme.io/changelog", "See the changelog", "button"}, {"https://docs.acme.io/migrate-v2", "https://docs.acme.io/migrate-v2", "link"},
		{"https://docs.acme.io/migrate-v2/plan", "Plan your migration", "button"}, {"https://x.com/acme", "Acme on X", "image"},
		{"https://www.linkedin.com/company/acme", "Acme on LinkedIn", "image"}, {"https://github.com/acme", "Acme on GitHub", "image"},
		{"{{ unsubscribe_url }}", "Unsubscribe", "link"}, {"{{ preferences_url }}", "Email preferences", "link"}, {"{{ view_in_browser_url }}", "View in browser", "link"},
	} {
		host := strings.TrimPrefix(strings.SplitN(strings.TrimPrefix(l[0], "https://"), "/", 2)[0], "www.")
		ms := 140 + (i*53)%380
		if l[0] == "https://docs.acme.io/migrate-v2" {
			ms = 620
		}
		links = append(links, contentcheck.ContentCheckLink{URL: l[0], Host: host, Text: l[1], Kind: l[2], Path: []int{}, Probed: !strings.Contains(l[0], "{{"), Status: 200, Millis: ms})
	}
	f.Content = readiness.ContentFacts{Checked: true, Address: "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105", InlineCSSBytes: 11 * 1024,
		URIBlocklists: readiness.URIBlocklists, OwnDomain: "acme.io",
		Report: contentcheck.ContentCheckReport{Verdict: readiness.VerdictReady, SizeBytes: 66 * 1024, ClipBytes: contentcheck.ClipBytes, Images: 7, Links: 15,
			SubjectHeuristics: []string{}, Issues: []contentcheck.ContentCheckIssue{}, Sections: []contentcheck.ContentCheckSection{}, PlainText: strings.Repeat("x", 8*1024),
			CheckedAt: checked, ImageShare: 0.38, Words: 980, LinkList: links}}
	_, _, err = r.storeRun(ctx, id, f, checked, &linh.ID, readiness.TriggerManual)
	return err
}

// demoAccessibilityFacts are the accessibility checks of a sample run: the
// checks on the campaign's content, with the run's own counts of images and
// links (its content facts are the boards' numbers).
func demoAccessibilityFacts(ctx context.Context, r *readinessSeed, id uuid.UUID, f readiness.Facts) readiness.Facts {
	var src, html, name string
	if err := r.s.Tx.QueryRow(ctx, `SELECT mjml, html, name FROM campaigns WHERE id = $1`, id).Scan(&src, &html, &name); err != nil {
		return f
	}
	rep, err := accessibility.Check([]accessibility.Version{{Lang: "en", MJML: src, HTML: html, Source: true}},
		accessibility.Options{Policy: accessibility.PolicyWarn, MinTextPx: accessibility.DefaultMinTextPx, Titles: accessibilityTitles, Now: func() time.Time { return f.Now }})
	if err != nil {
		f.Accessibility = accessibility.ReadinessFacts{Ran: true, Problem: "The MJML doesn't compile. The other checks finished; the verdict waits for this one."}
		return f
	}
	missingAlt, badLinks := 0, 0
	for _, is := range rep.Issues {
		switch is.Kind {
		case accessibility.KindAltText:
			missingAlt++
		case accessibility.KindLinkNoText, accessibility.KindLinkText:
			badLinks++
		}
	}
	if n := f.Content.Report.Images; n > 0 {
		rep.Counts.Images, rep.Counts.ImagesOK = n, n-missingAlt
	}
	if n := len(f.Content.Report.LinkList); n > 0 {
		rep.Counts.Links, rep.Counts.LinksOK = n, n-badLinks
	}
	if name == "Weekly Digest #142" || name == "Weekly Digest #143" {
		// As on the boards (SendReadinessAccessibility, AccessibilityStates).
		rep.Counts.LayoutTables, rep.Counts.RoleAdded = 16, 1
	}
	f.Accessibility = accessibility.ReadinessFacts{Ran: true, Checked: true, Report: rep}
	return f
}

// accessibilityWelcomeButton gives the Welcome series v4's step 3 B its
// pale button: white on #5EBFB0 (2.1:1), the serious issue the publish
// dialog and the policy's impact name. The live v3 doesn't have the step.
func accessibilityWelcomeButton(ctx context.Context, s *Seeder) error {
	_, err := s.Tx.Exec(ctx, `UPDATE automation_email_versions v SET mjml = replace(v.mjml, 'background-color="#0F766E"', 'background-color="#5EBFB0"'),
		html = replace(v.html, 'background:#0F766E', 'background:#5EBFB0')
		FROM automation_emails e WHERE e.id = v.email_id AND e.workspace_id = $1 AND e.name = $2`, s.WS().ID, `Step 3 "A 5-minute walkthrough"`)
	return err
}

// accessibilityDigestTemplate gives the saved template "Acme digest v3" its
// two issues (brief §4): the logo links to acme.io without alt text, so the
// link has no text, and both cards say "Read more".
func accessibilityDigestTemplate(ctx context.Context, s *Seeder) error {
	var id uuid.UUID
	var src string
	err := s.Tx.QueryRow(ctx, `SELECT id, mjml FROM templates WHERE workspace_id = $1 AND name = 'Acme digest v3'`, s.WS().ID).Scan(&id, &src)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // the composer's seeder didn't create it
	}
	if err != nil {
		return err
	}
	next := strings.Replace(src, `alt="Acme" width="96px"`, `href="https://acme.io" width="96px"`, 1)
	next = strings.Replace(next, `>Read the docs</mj-button>`, `>Read more</mj-button>`, 1)
	next = strings.Replace(next, `>See the changelog</mj-button>`, `>Read more</mj-button>`, 1)
	res := mjml.Compile(next)
	if len(res.Errors) > 0 {
		return fmt.Errorf("compile Acme digest v3: %s", res.Errors[0].Message)
	}
	_, err = s.Tx.Exec(ctx, `UPDATE templates SET mjml = $2, html = $3, size_bytes = $4 WHERE id = $1`, id, next, res.HTML, len(res.HTML))
	return err
}

// sampleToday is a time of the design expansion's sample day (Wed 15 Oct
// 2026; its clock reads 14:20 in New York) relative to now: hour:minute is
// as far before now as it is before 14:20, so today's items read the same
// at any hour and none is in the future. Earlier days use designDay.
func (s *Seeder) sampleToday(hour, minute int) time.Time {
	return s.Ago(time.Duration(14*60+20-(hour*60+minute)) * time.Minute)
}
