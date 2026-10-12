package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func init() { seedComposer = seedComposerImpl }

//go:embed assets/composer/*.png
var composerAssets embed.FS

// FlashSaleBloat is the number of pasted spans per plan card; it puts the
// flash sale at about 98 KB, inside the "approaching Gmail's clipping
// limit" warning of the ComposerWarnings board.
const FlashSaleBloat = 42

// seedComposerImpl gives the demo campaigns their content and history:
// Weekly Digest #142 (the Composer board, 14 versions, an A/B subject
// test), the flash sale that trips every inline warning (ComposerWarnings),
// October Product Update ("Copy of …" in the gallery), the empty draft, the
// saved template and sections of ComposerEmpty, and the image library.
func seedComposerImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh, marcus := s.Users["linh@acme.io"], s.Users["marcus@acme.io"]

	// Senders: the sending-domains seeder may have created them already.
	digestSender, err := ensureSender(ctx, s, "news.acme.io", "Acme Newsletter", "digest@news.acme.io")
	if err != nil {
		return err
	}
	helloSender, err := ensureSender(ctx, s, "news.acme.io", "Acme", "hello@news.acme.io")
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET default_sender_id = coalesce(default_sender_id, $2),
		branding = CASE WHEN branding ? 'address' THEN branding ELSE branding || '{"address":"Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105"}'::jsonb END
		WHERE id = $1`, ws.ID, digestSender); err != nil {
		return err
	}
	// The custom field the token list offers ({{ plan | default: "free" }}).
	if _, err := s.Tx.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type, options) VALUES ($1,$2,'plan','Plan','select','["free","starter","team","business","enterprise"]')
		ON CONFLICT (workspace_id, key) DO NOTHING`, ID(), ws.ID); err != nil {
		return err
	}
	// The link domain the flash sale points at is on two URI blocklists.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO link_domain_listings (domain, lists, source, checked_at) VALUES ('go.acme-promo.link', ARRAY['SURBL','Spamhaus DBL'], 'demo', $1)
		ON CONFLICT (domain) DO UPDATE SET lists = EXCLUDED.lists`, s.Ago(20*time.Minute)); err != nil {
		return err
	}

	// Image library.
	urls := map[string]string{}
	entries, err := composerAssets.ReadDir("assets/composer")
	if err != nil {
		return err
	}
	for i, e := range entries {
		data, err := composerAssets.ReadFile("assets/composer/" + e.Name())
		if err != nil {
			return err
		}
		ctype, w, h, alpha, err := media.Inspect(data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		sum := sha256.Sum256(data)
		id := ID()
		var got uuid.UUID
		if err := s.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (workspace_id, sha256) DO UPDATE SET hidden_at = NULL RETURNING id`,
			id, ws.ID, e.Name(), ctype, len(data), w, h, alpha, sum[:], data, linh.ID, s.Ago(time.Duration(30-i)*24*time.Hour)).Scan(&got); err != nil {
			return err
		}
		urls[e.Name()] = media.URL(got, e.Name())
	}
	withImages := func(src string) string {
		for name, u := range urls {
			src = strings.ReplaceAll(src, "@@"+name+"@@", u)
		}
		return src
	}

	type version struct {
		at        time.Time
		by        uuid.UUID
		label     string
		source    string
		subject   string
		preheader string
		mjml      string
	}
	compiled := map[string]string{}
	compile := func(src string) (string, error) {
		if h, ok := compiled[src]; ok {
			return h, nil
		}
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return "", fmt.Errorf("compile demo MJML: %s", res.Errors[0].Message)
		}
		compiled[src] = res.HTML
		return res.HTML, nil
	}
	saveHistory := func(campaign uuid.UUID, vs []version, ab json.RawMessage, autosaved bool) error {
		for i, v := range vs {
			html, err := compile(v.mjml)
			if err != nil {
				return err
			}
			if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, ab_test, saved_by, saved_at, label, source, size_bytes)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
				campaign, i+1, v.subject, v.preheader, v.mjml, html, contentcheck.PlainText(html), ab, v.by, v.at, v.label, v.source, len(html)); err != nil {
				return err
			}
		}
		last := vs[len(vs)-1]
		html, _ := compile(last.mjml)
		var autosavedAt *time.Time
		if autosaved {
			autosavedAt = &last.at
		}
		_, err := s.Tx.Exec(ctx, `UPDATE campaigns SET subject = $2, preheader = $3, mjml = $4, html = $5, plain_text = $6, ab_test = coalesce($7, ab_test),
			content_hash = $8, version = $9, updated_by = $10, updated_at = greatest(updated_at, $11), autosaved_at = $12 WHERE id = $1`,
			campaign, last.subject, last.preheader, last.mjml, html, contentcheck.PlainText(html), ab,
			campaigns.ContentHash(last.subject, last.preheader, last.mjml, html, ab), len(vs), last.by, last.at, autosavedAt)
		return err
	}

	// Weekly Digest #142.
	const digestSubject = "Three launches, one deprecation, and the roadmap for Q4"
	const digestPreheader = "Feature flags are GA, audit log exports, and what we're retiring in November."
	if id, ok := s.Campaigns["Weekly Digest #142"]; ok {
		early := withImages(digestMJML(1, "This week at Acme", "Launches, guides and the changelog."))
		mid := withImages(digestMJML(9, digestSubject, digestPreheader))
		final := withImages(digestMJML(14, digestSubject, digestPreheader))
		labels := []struct {
			day, h, m int
			by        uuid.UUID
			label     string
		}{
			{2, 16, 20, linh.ID, "created from template “Acme digest v3”"},
			{2, 16, 34, linh.ID, "hero copy"},
			{2, 16, 41, linh.ID, "subject and preheader"},
			{2, 17, 2, linh.ID, "header links"},
			{1, 10, 15, marcus.ID, "hero image"},
			{1, 10, 22, marcus.ID, "article card images"},
			{1, 10, 30, marcus.ID, "button colours"},
			{1, 11, 5, marcus.ID, "footer links"},
			{1, 15, 12, marcus.ID, "article cards, links"},
			{1, 16, 5, marcus.ID, "A/B subject B"},
			{0, 9, 12, linh.ID, "retirement notice"},
			{0, 9, 30, linh.ID, "migration guide link"},
			{0, 9, 35, linh.ID, "2-column section images"},
			{0, 9, 38, linh.ID, "2-column section padding"},
		}
		var vs []version
		for i, l := range labels {
			src, subject, pre := early, "This week at Acme", "Launches, guides and the changelog."
			if i >= 2 {
				subject, pre = digestSubject, digestPreheader
				src = withImages(digestMJML(1, subject, pre))
			}
			if i >= 8 {
				src = mid
			}
			if i == len(labels)-1 {
				src = final
			}
			source := "autosave"
			if i == 0 {
				source = "template"
			}
			vs = append(vs, version{at: s.designDay(l.day, l.h, l.m), by: l.by, label: l.label, source: source, subject: subject, preheader: pre, mjml: src})
		}
		ab := json.RawMessage(`{"kind":"subject","variants":[{"key":"A"},{"key":"B","subject":"What shipped this week, and what we're retiring"}],"test_percent":20,"metric":"human_opens","duration_hours":2}`)
		if err := saveHistory(id, vs, ab, true); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET sender_id = $2 WHERE id = $1`, id, digestSender); err != nil {
			return err
		}
	}

	// Flash sale: the inline warnings state.
	if id, ok := s.Campaigns["Flash sale: 48 hours only"]; ok {
		const subject, preheader = "FLASH SALE!!! 48 HOURS ONLY", "Annual plans 30% off until Thursday midnight."
		withFooter := withImages(flashSaleMJML("Flash sale", preheader, true, FlashSaleBloat))
		vs := []version{
			{at: s.designDay(1, 17, 40), by: linh.ID, label: "created", source: "create", subject: "Flash sale", preheader: preheader, mjml: withImages(flashSaleMJML("Flash sale", preheader, true, 0))},
			{at: s.designDay(1, 17, 52), by: linh.ID, label: "hero and button", subject: "Flash sale", preheader: preheader, mjml: withImages(flashSaleMJML("Flash sale", preheader, true, 0))},
			{at: s.designDay(0, 9, 2), by: linh.ID, label: "plan cards pasted from the pricing page", subject: "Flash sale", preheader: preheader, mjml: withFooter},
			{at: s.designDay(0, 9, 18), by: linh.ID, label: "second row of plans", subject: "Flash sale", preheader: preheader, mjml: withFooter},
			{at: s.designDay(0, 9, 26), by: linh.ID, label: "subject", subject: subject, preheader: preheader, mjml: withImages(flashSaleMJML(subject, preheader, true, FlashSaleBloat))},
			{at: s.designDay(0, 9, 31), by: linh.ID, label: "footer removed", subject: subject, preheader: preheader, mjml: withImages(flashSaleMJML(subject, preheader, false, FlashSaleBloat))},
		}
		for i := range vs {
			if vs[i].source == "" {
				vs[i].source = "autosave"
			}
		}
		if err := saveHistory(id, vs, nil, true); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET sender_id = $2 WHERE id = $1`, id, helloSender); err != nil {
			return err
		}
	}

	// October Product Update: sent, so only its final content matters.
	if id, ok := s.Campaigns["October Product Update"]; ok {
		src := withImages(octoberMJML())
		var started time.Time
		if err := s.Tx.QueryRow(ctx, `SELECT coalesce(started_at, now()) FROM campaigns WHERE id = $1`, id).Scan(&started); err != nil {
			return err
		}
		v := version{at: started.Add(-26 * time.Hour), by: linh.ID, label: "subject A/B test", source: "autosave", subject: "October at Acme: what shipped", preheader: "Feature flags preview, faster exports and the Postal warm-up.", mjml: src}
		if err := saveHistory(id, []version{v}, nil, false); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET sender_id = $2 WHERE id = $1`, id, digestSender); err != nil {
			return err
		}
	}

	// The empty draft: created, never edited ("New draft · not saved yet").
	if id, ok := s.Campaigns["Untitled campaign"]; ok {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, saved_by, saved_at, label, source)
			SELECT id, 1, '', '', '', '', created_by, created_at, '', 'create' FROM campaigns WHERE id = $1`, id); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET sender_id = $2, autosaved_at = NULL WHERE id = $1`, id, digestSender); err != nil {
			return err
		}
	}

	// Gallery: the saved template and the saved sections.
	tpl := withImages(digestMJML(1, "", ""))
	tplHTML, err := compile(tpl)
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,'template','Acme digest v3','Saved template',$3,$4,$5,$6,$7,$7,$8,$9)`,
		ID(), ws.ID, tpl, tplHTML, json.RawMessage(`[{"h":10,"w":40},{"h":34},{"row":[{"h":28},{"h":28}]}]`), len(tplHTML), linh.ID, s.designDay(40, 11, 0), s.designDay(5, 14, 20)); err != nil {
		return err
	}
	for i, sec := range savedSections {
		thumb, _ := json.Marshal(mjml.Thumb("<mjml><mj-body>" + sec.mjml + "</mj-body></mjml>"))
		html, err := compile("<mjml><mj-body>" + sec.mjml + "</mj-body></mjml>")
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,'section',$3,$4,$5,$6,$7,$8,$9,$9,$10,$10)`,
			ID(), ws.ID, sec.name, sec.description, sec.mjml, html, thumb, len(html), linh.ID, s.Ago(time.Duration(60-i*9)*24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

// ensureSender returns the sender for email, creating its sending domain and
// the sender when the senders seeder hasn't.
func ensureSender(ctx context.Context, s *Seeder, domain, name, email string) (uuid.UUID, error) {
	if id, ok := s.Senders[email]; ok {
		return id, nil
	}
	ws := s.WS()
	domainID, ok := s.Domains[domain]
	if !ok {
		var err error
		if err = s.Tx.QueryRow(ctx, `SELECT id FROM sending_domains WHERE workspace_id = $1 AND domain = $2`, ws.ID, domain).Scan(&domainID); err != nil {
			domainID = ID()
			if _, err := s.Tx.Exec(ctx, `INSERT INTO sending_domains (id, workspace_id, domain, status, auth_summary, last_checked_at, created_at)
				VALUES ($1,$2,$3,'verified','{"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}',$4,$5)`,
				domainID, ws.ID, domain, s.Ago(5*time.Minute), s.Ago(230*24*time.Hour)); err != nil {
				return uuid.Nil, err
			}
		}
		s.Domains[domain] = domainID
	}
	var id uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM senders WHERE workspace_id = $1 AND email = $2`, ws.ID, email).Scan(&id); err == nil {
		s.Senders[email] = id
		return id, nil
	}
	id = ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO senders (id, workspace_id, domain_id, name, email, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, ws.ID, domainID, name, email, s.Ago(229*24*time.Hour)); err != nil {
		return uuid.Nil, err
	}
	s.Senders[email] = id
	return id, nil
}
