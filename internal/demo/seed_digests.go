package demo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/digests"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

func init() { seedDigests = seedDigestsImpl }

// digestName is the sample's recurring digest.
const digestName = "Engineering blog roundup"

// seedDigestsImpl seeds "Engineering blog roundup" (brief §4): Daniel Park
// built it in draft mode the day the blog was connected, Sofia Rossi turned
// it on to send automatically that afternoon. Weekly at 08:00 New York time,
// prepared at 05:00, late limit 24 hours, to Product Updates from Acme
// Product. The first occurrence was skipped (nothing new since it was turned
// on), #1 and #2 went out after their approvals, and #3 is collecting with
// two new posts. Its weekday is two days after the demo's "today", as the
// sample's Friday is two days after its Wednesday.
func seedDigestsImpl(ctx context.Context, s *Seeder) error {
	siteID, ok := s.Extra["integration:blog.acme.io"].(uuid.UUID)
	if !ok {
		return nil // the integrations hub's sample isn't there
	}
	ws := s.WS()
	sofia, daniel, marcus, linh := s.Users["sofia@acme.io"], s.Users["daniel@acme.io"], s.Users["marcus@acme.io"], s.Users["linh@acme.io"]
	product := s.Lists["Product Updates"]
	sender, err := composerEnsureSender(ctx, s, "news.acme.io", "Acme Product", "product@news.acme.io")
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	clamp := func(t time.Time) time.Time {
		if t.After(s.Now) {
			return s.Now.Add(-time.Minute)
		}
		return t
	}
	created, turnedOn := s.designDay(20, 10, 40), s.designDay(20, 15, 12)
	skippedSend, send1, send2, send3 := s.designDay(19, 8, 0), s.designDay(12, 8, 0), s.designDay(5, 8, 0), s.designDay(-2, 8, 0)

	// The posts' images, copied into the library (out of the library's
	// sample: hidden there, still served to the issues).
	images := map[string]uuid.UUID{}
	urls := map[string]string{}
	entries, err := digestAssets.ReadDir("assets/digests")
	if err != nil {
		return err
	}
	for _, e := range entries {
		data, err := digestAssets.ReadFile("assets/digests/" + e.Name())
		if err != nil {
			return err
		}
		ctype, w, h, alpha, err := media.Inspect(data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		sum := sha256.Sum256(data)
		at := created
		for _, p := range blogPosts {
			if p.image == e.Name() {
				at = clamp(s.designDay(p.seenDays, p.seenH, p.seenM))
			}
		}
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at, hidden_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)
			ON CONFLICT (workspace_id, sha256) DO UPDATE SET file_name = EXCLUDED.file_name RETURNING id`,
			ID(), ws.ID, e.Name(), ctype, len(data), w, h, alpha, sum[:], data, sofia.ID, at).Scan(&id); err != nil {
			return err
		}
		images[e.Name()], urls[e.Name()] = id, media.URL(id, e.Name())
	}

	// The template: the Blog digest starter as Sofia left it (v9, autosaved
	// at 10:24): the Acme logo, "Engineering blog" in the header, the intro
	// about the engineering blog, the Feed items block with the fallback
	// image, and the footer saying why readers get it.
	// The workspace as the earlier seeders left it (its footer address).
	current, err := s.Q.GetWorkspace(ctx, ws.ID)
	if err != nil {
		return err
	}
	src, _ := templates.StarterMJML(templates.BlogDigestStarter, templates.WorkspaceAddress(&current))
	var logo uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM media_assets WHERE workspace_id = $1 AND file_name = 'acme-logo.png' LIMIT 1`, ws.ID).Scan(&logo); err == nil {
		src = strings.Replace(src, `src="/media/starters/logo.png" alt="Your logo"`, `src="`+media.URL(logo, "acme-logo.png")+`" alt="Acme"`, 1)
	}
	src = strings.Replace(src, ">{{ feed.title }}</mj-text>", ">Engineering blog</mj-text>", 1)
	src = strings.Replace(src, "here's what's new on the blog this week.", "here's what's new on the Acme engineering blog this week.", 1)
	block := digests.DefaultBlock()
	block.Fallback, block.FallbackName = urls["blog-fallback.png"], "blog-fallback.png"
	if old, ok := digestMarker(src); ok {
		src = strings.Replace(src, old, digests.MarkerFor(block), 1)
	}
	src = strings.Replace(src, `<br><a href="{{ unsubscribe_url }}"`, `<br>You get this because you're on Product Updates.<br><a href="{{ unsubscribe_url }}"`, 1)
	subject := "{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from the Acme blog"
	preheader := "New on blog.acme.io · issue {{ issue.number }}"
	audience, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{product}})
	tplID, digestID := ID(), ID()
	sched := digests.DigestSchedule{Frequency: digests.FreqWeekly, Weekday: int(send3.In(loc).Weekday()), Time: "08:00", Timezone: ws.Timezone,
		MonthRule: "day", MonthDay: 1, Count: 3, WindowFrom: "08:00", WindowUntil: "18:00"}
	schedJSON, _ := json.Marshal(sched)
	autosaved := clamp(s.designDay(0, 10, 24))
	tplHTML, err := compileHTML(src)
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count, mjml, html, plain_text,
		schedule, thresholds, tracking, content_hash, version, digest_id, digest_role, created_by, updated_by, created_at, updated_at, autosaved_at)
		VALUES ($1,$2,$3,'draft',$4,$5,$6,$7,14110,$8,$9,$10,'{"mode":"now"}','{}','{"opens":true,"clicks":true}',$11,9,NULL,'template',$12,$13,$14,$15,$15)`,
		tplID, ws.ID, digestName, subject, preheader, sender, audience, src, tplHTML, contentcheck.PlainText(tplHTML),
		campaigns.ContentHash(subject, preheader, src, tplHTML, nil), daniel.ID, sofia.ID, created, autosaved); err != nil {
		return err
	}
	for _, v := range []struct {
		n     int
		by    uuid.UUID
		at    time.Time
		label string
	}{
		{1, daniel.ID, created, "created from the Blog digest starter"},
		{2, daniel.ID, clamp(s.designDay(20, 11, 5)), "Acme logo and the blog's name in the header"},
		{3, daniel.ID, clamp(s.designDay(20, 11, 18)), "autosaved"},
		{4, sofia.ID, clamp(s.designDay(19, 9, 30)), "Feed items: cards, up to 6 posts"},
		{5, sofia.ID, clamp(s.designDay(14, 16, 10)), "autosaved"},
		{6, sofia.ID, clamp(s.designDay(7, 10, 45)), "subject with feed tokens"},
		{7, sofia.ID, clamp(s.designDay(3, 14, 20)), "autosaved"},
		{8, sofia.ID, clamp(s.designDay(1, 15, 2)), "intro and footer reworded"},
		{9, sofia.ID, autosaved, "autosaved"},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'template',$11)`, tplID, v.n, subject, preheader, src, tplHTML, contentcheck.PlainText(tplHTML), v.by, v.at, v.label, len(tplHTML)); err != nil {
			return err
		}
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO digests (id, workspace_id, name, template_id, state, mode, source, feed_url, site_id, categories, tags, max_age_days,
		start_mode, start_latest, schedule, prepare_hours, late_hours, archive, owner_id, owner_since, first_on_at, feed_status, feed_kind, feed_title,
		feed_items, feed_bytes, feed_etag, last_read_at, last_read_via, last_ok_at, created_by, created_at, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,'on','auto','wordpress',$5,$6,'{Engineering,Infrastructure}','{}',14,'after_on',3,$7,3,24,true,$8,$9,$9,'ok','rss',$10,
		10,49152,$11,$12,'plugin',$12,$13,$14,$8,$9)`,
		digestID, ws.ID, digestName, tplID, BlogFeedURL, siteID, schedJSON, sofia.ID, turnedOn, blogFeedTitle, BlogFeedETag, clamp(s.designDay(0, 10, 5)),
		daniel.ID, created); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET digest_id = $2 WHERE id = $1`, tplID, digestID); err != nil {
		return err
	}

	// The feed's items, as the reads found them.
	itemIDs := map[int]uuid.UUID{}
	for _, p := range blogPosts {
		id := ID()
		itemIDs[p.id] = id
		published := clamp(s.designDay(p.daysAgo, p.hour, p.min))
		seen := clamp(s.designDay(p.seenDays, p.seenH, p.seenM))
		if seen.Before(published) {
			seen = published
		}
		var mediaID *uuid.UUID
		mediaURL, imageURL, width, height := "", "", 0, 0
		if p.image != "" {
			m := images[p.image]
			mediaID, mediaURL, imageURL, width, height = &m, urls[p.image], blogImageURL(p.image), 552, 240
		}
		tags := p.tags
		if tags == nil {
			tags = []string{}
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO digest_items (id, digest_id, key, url, title, author, published_at, categories, tags, excerpt, image_url, image_alt,
			media_id, media_url, media_width, media_height, first_seen_at, first_seen_via, last_seen_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
			id, digestID, p.guid(), p.url(), p.title, p.author, published, []string{p.category}, tags, p.excerpt, imageURL, p.imageAlt,
			mediaID, mediaURL, width, height, seen, p.seenVia, clamp(s.designDay(0, 10, 5))); err != nil {
			return err
		}
	}

	// The template as the issues used it, for their content.
	d := store.Digest{ID: digestID, WorkspaceID: ws.ID, Name: digestName, TemplateID: &tplID, FeedTitle: blogFeedTitle, FeedUrl: BlogFeedURL, Schedule: schedJSON,
		PrepareHours: 3, LateHours: 24}
	tpl := store.DigestCampaignTemplateRow{ID: tplID, Subject: subject, Preheader: preheader, Mjml: src, SenderID: &sender, Audience: audience, Version: 9,
		Thresholds: json.RawMessage(`{}`), RouteOverride: nil}
	postsOf := func(ids ...int) []digests.BlockItem {
		out := []digests.BlockItem{}
		for _, pid := range ids {
			for _, p := range blogPosts {
				if p.id == pid {
					at := clamp(s.designDay(p.daysAgo, p.hour, p.min))
					it := digests.BlockItem{ID: itemIDs[p.id].String(), Title: p.title, URL: p.url(), Author: p.author, Published: &at, Categories: []string{p.category},
						Excerpt: p.excerpt, ImageAlt: p.imageAlt}
					if p.image != "" {
						it.Image = urls[p.image]
					}
					out = append(out, it)
				}
			}
		}
		return out
	}

	// The skipped first occurrence: nothing new since it was turned on.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO digest_issues (id, digest_id, workspace_id, number, status, mode, send_at, prepare_at, deadline, skipped_at, skip_reason, created_at, updated_at)
		VALUES ($1,$2,$3,NULL,'skipped','auto',$4,$5,$6,$5,'no_items',$7,$5)`,
		ID(), digestID, ws.ID, skippedSend, skippedSend.Add(-3*time.Hour), skippedSend.Add(24*time.Hour), turnedOn); err != nil {
		return err
	}
	// #1 and #2: prepared at 05:00, approved, sent at 08:00.
	sentIssues := []struct {
		number             int
		daysAgo            time.Duration
		send               time.Time
		posts              []int
		perProvider        map[string]int64
		humanOpens, clicks float64
		approver           store.User
		approvedAt         time.Time
	}{
		{1, 12, send1, []int{396, 391}, map[string]int64{"ses": 12484, "postal": 1387}, 40.3, 8.1, linh, s.designDay(12, 7, 55)},
		{2, 5, send2, []int{409, 405, 402}, map[string]int64{"ses": 12512, "postal": 1390}, 38.9, 7.4, marcus, s.designDay(5, 6, 41)},
	}
	for _, si := range sentIssues {
		name := fmt.Sprintf("%s #%d", digestName, si.number)
		if err := seedSent(ctx, s, sentCampaign{name: name, audience: "Product Updates", at: si.daysAgo, hour: 8, perProvider: si.perProvider, delivered: 99.2,
			humanOpens: si.humanOpens, machineOpens: 17.6, clicks: si.clicks, bounce: 0.6, complaint: 0.03, unsubscribes: 0.11}, map[string]any{"lists": []uuid.UUID{product}}); err != nil {
			return err
		}
		campaignID := s.Campaigns[name]
		// Each route's median time to delivery, as the report seeds give their sent campaigns
		// (the report reads it: the demo's messages carry no delivery times).
		if _, err := s.Tx.Exec(ctx, `UPDATE campaign_routes cr SET delivery_p50_ms = CASE p.adapter WHEN 'ses' THEN 1750 + $2 WHEN 'postal' THEN 3300 + $2 ELSE 2450 + $2 END
			FROM providers p WHERE p.id = cr.provider_id AND cr.campaign_id = $1`, campaignID, int32(si.number*40)); err != nil {
			return err
		}
		send := si.send
		content, err := digests.BuildIssue(d, tpl, postsOf(si.posts...), si.number, &send, ws.Timezone)
		if err != nil {
			return fmt.Errorf("issue #%d: %w", si.number, err)
		}
		prepared := si.send.Add(-3 * time.Hour)
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET subject = $2, preheader = $3, sender_id = $4, mjml = $5, html = $6, plain_text = $7, digest_id = $8, digest_role = 'issue',
			created_by = $9, updated_by = $9, started_by = $9, created_at = $10, updated_at = $11,
			schedule = $12 WHERE id = $1`,
			campaignID, content.Subject, content.Preheader, sender, content.MJML, content.HTML, contentcheck.PlainText(content.HTML), digestID, sofia.ID, prepared,
			si.send.Add(16*time.Minute), json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":%q}`, si.send.UTC().Format(time.RFC3339), ws.Timezone))); err != nil {
			return err
		}
		if err := issueEngagement(ctx, s, campaignID, name, content.HTML); err != nil {
			return err
		}
		issueID := ID()
		if _, err := s.Tx.Exec(ctx, `INSERT INTO digest_issues (id, digest_id, workspace_id, number, status, mode, send_at, prepare_at, deadline, campaign_id, prepared_at,
			started_at, sent_at, item_count, basis, created_at, updated_at)
			VALUES ($1,$2,$3,$4,'sent','auto',$5,$6,$7,$8,$6,$5,$9,$10,$11,$12,$9)`,
			issueID, digestID, ws.ID, si.number, si.send, prepared, si.send.Add(24*time.Hour), campaignID, si.send.Add(16*time.Minute), len(content.Items), content.Basis,
			si.send.AddDate(0, 0, -7)); err != nil {
			return err
		}
		for i, it := range content.Items {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO digest_issue_items (issue_id, item_id, position) VALUES ($1,$2,$3)`, issueID, uuid.MustParse(it.ID), i+1); err != nil {
				return err
			}
		}
		// The request made at preparation in Sofia's name, and its approval.
		snap := approvals.Snapshot{Version: 1, Subject: content.Subject, Preheader: content.Preheader, SenderID: sender.String(), From: "Acme Product <product@news.acme.io>",
			Links: []string{}, Sections: []string{}, Audience: audience, AudienceLabels: []string{"Product Updates"}, AudienceCount: sum(si.perProvider),
			Route: []approvals.SnapshotShare{
				{ProviderID: s.Providers["ses"].String(), Name: "Amazon SES", Short: "SES", Adapter: "ses", Percent: 90, Recipients: si.perProvider["ses"], Status: "healthy", StatusLabel: "Healthy"},
				{ProviderID: s.Providers["postal"].String(), Name: "Postal", Short: "Postal", Adapter: "postal", Percent: 10, Recipients: si.perProvider["postal"], Status: "warming_up", StatusLabel: "Warming up"}},
			SplitLabel: "Broadcasts", Schedule: json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":%q}`, si.send.UTC().Format(time.RFC3339), ws.Timezone)),
			Readiness: approvals.SnapshotReadiness{Verdict: "ready", RanAt: prepared},
			Rules: []approvals.ApprovalRuleMatch{
				{Key: approvals.RuleAudienceMin, Label: fmt.Sprintf("Audience of %s is at least 10,000", thousands(sum(si.perProvider))), Short: "audience ≥ 10,000", Badge: "audience ≥ 10,000"},
				{Key: approvals.RuleWarmingProvider, Label: "Route includes Postal, which is warming up", Short: "warming-up provider", Badge: "Warming-up provider"}},
			AuthorRole: "Editor"}
		sb, _ := json.Marshal(snap)
		reqID := ID()
		approvedAt := si.approvedAt
		if _, err := s.Tx.Exec(ctx, `INSERT INTO approval_requests (id, workspace_id, subject_type, campaign_id, requested_by, requested_at, note, status, matched_rules,
			approvals_required, snapshot, campaign_version, decided_by, decided_at) VALUES ($1,$2,'campaign',$3,$4,$5,'','approved',$6,1,$7,1,$8,$9)`,
			reqID, ws.ID, campaignID, sofia.ID, prepared, []string{approvals.RuleAudienceMin, approvals.RuleWarmingProvider}, sb, si.approver.ID, approvedAt); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO approval_decisions (id, request_id, approver_id, decision, comment, created_at) VALUES ($1,$2,$3,'approve','',$4)`,
			ID(), reqID, si.approver.ID, approvedAt); err != nil {
			return err
		}
	}
	// #3, collecting: the posts since #2 are its posts so far.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO digest_issues (id, digest_id, workspace_id, status, mode, send_at, prepare_at, deadline, created_at, updated_at)
		VALUES ($1,$2,$3,'collecting','auto',$4,$5,$6,$7,$7)`,
		ID(), digestID, ws.ID, send3, send3.Add(-3*time.Hour), send3.Add(24*time.Hour), send2.Add(time.Minute)); err != nil {
		return err
	}

	// The history the overview tells: Daniel built it, Sofia turned it on,
	// the first occurrence was skipped.
	for _, e := range []struct {
		at                  time.Time
		actor               *store.User
		actorName, action   string
		before, after, meta map[string]any
	}{
		{created, &daniel, daniel.Name, "digest.created", nil, map[string]any{"name": digestName, "source": "wordpress", "feed": BlogFeedURL, "mode": "draft"}, nil},
		{turnedOn, &sofia, sofia.Name, "digest.turned_on", map[string]any{"state": "off", "mode": "draft"}, map[string]any{"state": "on", "mode": "auto", "owner": sofia.Name},
			map[string]any{"permission": "campaigns:send"}},
		{skippedSend.Add(-3 * time.Hour), nil, "System · " + digestName + ", turned on by " + sofia.Name, "digest.issue_skipped", map[string]any{"status": "collecting"},
			map[string]any{"status": "skipped", "reason": "no_items"}, map[string]any{"summary": "no new posts"}},
	} {
		before, _ := json.Marshal(e.before)
		after, _ := json.Marshal(e.after)
		meta, _ := json.Marshal(e.meta)
		if e.before == nil {
			before = nil
		}
		if e.meta == nil {
			meta = []byte(`{}`)
		}
		actorType, actorID := "system", (*uuid.UUID)(nil)
		if e.actor != nil {
			actorType, actorID = "user", &e.actor.ID
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			before, after, result, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,'digest',$8,$9,$10,$11,'allowed',$12)`,
			ID(), ws.ID, e.at, actorType, actorID, e.actorName, e.action, digestID.String(), digestName, before, after, meta); err != nil {
			return err
		}
	}
	return nil
}

// issueEngagement gives a sent issue what the report seed gives the sent
// campaigns seeded before it: opens and clicks hour by hour over 72 hours,
// and the clicks of each link in its HTML. The counters stay as seeded.
func issueEngagement(ctx context.Context, s *Seeder, id uuid.UUID, name, html string) error {
	c := &reportCampaignSeed{id: id, name: name, html: html}
	if err := reportLoadCells(ctx, s, c, nil); err != nil {
		return err
	}
	if len(c.cells) == 0 {
		return nil
	}
	if err := reportHourly(ctx, s, c, false); err != nil {
		return err
	}
	return reportLinks(ctx, s, c, false)
}

// compileHTML compiles a template's MJML.
func compileHTML(src string) (string, error) {
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return "", fmt.Errorf("the digest template doesn't compile: %s", res.Errors[0].Message)
	}
	return res.HTML, nil
}

// digestMarker finds the Feed items block's marker in a template.
func digestMarker(src string) (string, bool) {
	i := strings.Index(src, "<mj-raw><!-- rc:feed-items ")
	if i < 0 {
		return "", false
	}
	j := strings.Index(src[i:], "</mj-raw>")
	if j < 0 {
		return "", false
	}
	return src[i : i+j+len("</mj-raw>")], true
}

func sum(m map[string]int64) int64 {
	var n int64
	for _, v := range m {
		n += v
	}
	return n
}

// thousands writes 13871 as "13,871".
func thousands(n int64) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
