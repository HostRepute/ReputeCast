package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func init() { seedWebsiteImport = seedWebsiteImportImpl }

// An import of posts waiting for its review (M61, G-26; board
// WebsiteImport): Sofia Rossi uploaded acme-weekly-substack-export.zip (4.2
// MB) this morning, read in 18 seconds. 15 posts: 12 to import (4 paid, 2
// members-only), 3 drafts skipped; tiers mapped by name, Sofia's byline
// mapped to her, Jordan Ellis kept as written. The board draws tags and
// authors on the Substack posts; a real Substack export has neither (the
// importer says so), the sample keeps the board's.
type demoStaged struct {
	title, slug     string
	day             string // "2026-03-12", "" for a draft
	visibility, tag string
	tier, author    string
	images          int
	problem, detail string
}

var demoStagedPosts = []demoStaged{
	{"Why we moved our queue to Postgres", "why-we-moved-our-queue-to-postgres", "2026-03-12", "public", "Engineering", "", "Sofia Rossi", 5, "", ""},
	{"The 2025 deliverability report", "the-2025-deliverability-report", "2025-12-02", "paid", "Deliverability", "Paid subscribers", "Sofia Rossi", 4, "embed_link", "YouTube"},
	{"Ask Acme #12: warming up a new domain", "ask-acme-12-warming-up-a-new-domain", "2025-10-21", "members", "Ask Acme", "", "Sofia Rossi", 3, "", ""},
	{"Shipping feature flags without fear", "shipping-feature-flags-without-fear", "2025-09-09", "public", "Engineering", "", "Jordan Ellis", 4, "image_alt", "https://substackcdn.com/image/fetch/f_auto/acme/flags-rollout.png"},
	{"Founders' letter: year two", "founders-letter-year-two", "2025-07-01", "paid", "", "Founding members", "Sofia Rossi", 2, "", ""},
	{"How we test email rendering", "how-we-test-email-rendering", "2025-05-14", "public", "Engineering", "", "Sofia Rossi", 3, "slug_taken", "how-we-test-email-rendering-2"},
	{"Inside a 550 5.7.1 bounce", "inside-a-550-5-7-1-bounce", "2025-03-18", "public", "Deliverability", "", "Jordan Ellis", 4, "", ""},
	{"Ask Acme #11: BIMI in practice", "ask-acme-11-bimi-in-practice", "2025-02-04", "members", "Ask Acme", "", "Sofia Rossi", 3, "", ""},
	{"Rate limits, explained", "rate-limits-explained", "2024-12-10", "paid", "Engineering", "Paid subscribers", "Sofia Rossi", 3, "embed_link", "YouTube"},
	{"Our on-call handbook", "our-on-call-handbook", "2024-10-22", "paid", "Engineering", "Paid subscribers", "Sofia Rossi", 2, "", ""},
	{"DMARC reports without the pain", "dmarc-reports-without-the-pain", "2024-08-27", "public", "Deliverability", "", "Jordan Ellis", 3, "", ""},
	{"Hello from Acme Weekly", "hello-from-acme-weekly", "2024-06-04", "public", "", "", "Sofia Rossi", 2, "", ""},
	{"Q1 planning notes", "q1-planning-notes", "", "public", "", "", "Sofia Rossi", 0, "", ""},
	{"Postmortem template", "postmortem-template", "", "members", "", "", "Sofia Rossi", 0, "", ""},
	{"Spring launch checklist", "spring-launch-checklist", "", "public", "", "", "Sofia Rossi", 0, "", ""},
}

// demoExportSize is the export's size: 4.2 MB.
const demoExportSize int64 = 4404019

func seedWebsiteImportImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	sofia := s.Users["sofia@acme.io"]
	tiers := map[string]string{}
	for _, name := range []string{"Weekly Digest Insider", "Founding member"} {
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL`, ws.ID, name).Scan(&id); err != nil {
			return fmt.Errorf("the tier %s: %w", name, err)
		}
		tiers[name] = id.String()
	}
	options, _ := json.Marshal(map[string]any{"publish": "as_published", "images": "copy",
		"tiers":  map[string]string{"Paid subscribers": tiers["Weekly Digest Insider"], "Founding members": tiers["Founding member"]},
		"author": map[string]string{"Sofia Rossi": sofia.ID.String(), "Jordan Ellis": "keep"}})
	uploaded := s.designDay(0, 10, 12)
	id := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO website_imports (id, workspace_id, source, status, options, stats, file_name, file_size, created_by, created_at, updated_at)
		VALUES ($1,$2,'substack','ready',$3,'{}',$4,$5,$6,$7,$8)`,
		id, ws.ID, options, "acme-weekly-substack-export.zip", demoExportSize, sofia.ID, uploaded, uploaded.Add(18*time.Second)); err != nil {
		return err
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		ny = time.UTC
	}
	for i, p := range demoStagedPosts {
		// The ids keep the export's order: newest first, drafts last.
		row := ID()
		var published *time.Time
		status, decision := "draft", "skip"
		if p.day != "" {
			d, err := time.ParseInLocation("2006-01-02", p.day, ny)
			if err != nil {
				return err
			}
			d = d.Add(9 * time.Hour)
			published, status, decision = &d, "published", "import"
		}
		tags, refs := []string{}, []string{}
		if p.tag != "" {
			tags = append(tags, p.tag)
		}
		if p.tier != "" {
			refs = append(refs, p.tier)
		}
		problems := []map[string]string{}
		if p.problem != "" {
			problems = append(problems, map[string]string{"code": p.problem, "detail": p.detail})
		}
		slug := p.slug
		if p.problem == "slug_taken" {
			slug = p.detail
		}
		var body strings.Builder
		body.WriteString("<p>" + p.title + ".</p>")
		for n := range p.images {
			alt := fmt.Sprintf(` alt="%s, figure %d"`, p.title, n+1)
			src := fmt.Sprintf("https://substackcdn.com/image/fetch/f_auto/acme/%s-%d.png", p.slug, n+1)
			if p.problem == "image_alt" && n == 0 {
				alt, src = ` alt=""`, p.detail
			}
			body.WriteString(`<figure><img src="` + src + `"` + alt + `></figure>`)
		}
		if p.problem == "embed_link" {
			body.WriteString(`<iframe src="https://www.youtube.com/embed/acme` + fmt.Sprint(i) + `"></iframe>`)
		}
		pb, _ := json.Marshal(problems)
		seo, _ := json.Marshal(map[string]string{"sourceUrl": "/p/" + p.slug})
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_import_posts (id, import_id, source_id, title, slug, published_at, status, visibility, tier_refs, tags,
			author, excerpt, seo, legacy_paths, html, images, problems, decision)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			row, id, fmt.Sprint(118190+i*7), p.title, slug, published, status, p.visibility, refs, tags, p.author, p.title+".", seo,
			[]string{"p/" + p.slug}, body.String(), p.images, pb, decision); err != nil {
			return fmt.Errorf("staged post %s: %w", p.slug, err)
		}
	}
	after, _ := json.Marshal(map[string]any{"source": "substack", "file_name": "acme-weekly-substack-export.zip", "file_size": demoExportSize})
	_, err = s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name, result, after)
		VALUES ($1,$2,$3,'user',$4,$5,'website.import_uploaded','website_import',$6,'acme-weekly-substack-export.zip','allowed',$7)`,
		ID(), ws.ID, uploaded, sofia.ID, sofia.Name, id.String(), after)
	return err
}
