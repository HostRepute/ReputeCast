package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
	"github.com/hostrepute/reputecast/internal/website"
)

//go:embed assets/products/*.png
var productAssets embed.FS

func init() { seedPagesProducts = seedPagesProductsImpl }

// Landing and sales pages and digital products of the expansion's sample
// (publish-sell brief §4, PagesList, PublicSalesPage, Products,
// ProductEditor and their states): six pages (the Field Guide's sales page
// with its draft changes, the DNS records checklist, the Newsletter Launch
// Kit whose countdown ended on 6 Oct, the DMARC live session, About and
// Daniel Park's draft), three products (the Field Guide at $29, the Launch
// Kit, pay what you want from $5, Daniel's Warm-up planner draft), 311
// live orders since 1 Oct (214 Field Guide, 97 Launch Kit, $7,494 gross,
// three refunds and a dispute) and two test orders, the delivery template,
// the lead magnet, the webinar and waitlist forms, 30 days of views and
// what each page brought in, and the "Field Guide follow-up" automation.
type sellSeed struct {
	s      *Seeder
	ws     store.Workspace
	rng    *rand.Rand
	linh   uuid.UUID
	sofia  uuid.UUID
	marcus uuid.UUID
	daniel uuid.UUID
	covers map[string]string
	files  map[string]uuid.UUID
	forms  map[string]uuid.UUID
	pages  map[string]uuid.UUID
	prods  map[string]uuid.UUID
	digest uuid.UUID
	// The delivery template and its Purchases category.
	template uuid.UUID
	category uuid.UUID
	live     uuid.UUID // the live payment connection
	test     uuid.UUID
	store    products.Store
	orders   []*sellOrder
	// The webinar's start (17:00 UTC on 29 Oct in the sample) and the
	// Launch Kit's opening (6 Oct, 09:00 New York), anchored to today.
	event, opens time.Time
	// The webinar's list and the Launch Kit waitlist.
	dmarc, waitlist uuid.UUID
	// taken: sample contacts the buyers' stories already use.
	taken map[uuid.UUID]bool
}

// sellDay is a moment of the sample's calendar, where today is 15 Oct.
func (w *sellSeed) day(m time.Month, d, h, mi int) time.Time {
	days := int(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC).Sub(time.Date(2026, m, d, 0, 0, 0, 0, time.UTC)).Hours() / 24)
	return w.s.designDay(days, h, mi)
}

func seedPagesProductsImpl(ctx context.Context, s *Seeder) error {
	w := &sellSeed{s: s, ws: s.WS(), rng: rand.New(rand.NewPCG(19, 29)), //nolint:gosec // G404: the demo's sample numbers, the same every time
		covers: map[string]string{}, files: map[string]uuid.UUID{}, forms: map[string]uuid.UUID{}, pages: map[string]uuid.UUID{}, prods: map[string]uuid.UUID{},
		linh: s.Users["linh@acme.io"].ID, sofia: s.Users["sofia@acme.io"].ID, marcus: s.Users["marcus@acme.io"].ID, daniel: s.Users["daniel@acme.io"].ID,
		digest: s.Lists["Weekly Digest"], store: products.Store{Dir: s.Cfg.FilesDir()}, taken: map[uuid.UUID]bool{}}
	day := w.day(10, 29, 12, 0)
	w.event = time.Date(day.Year(), day.Month(), day.Day(), 17, 0, 0, 0, time.UTC)
	w.opens = w.day(10, 6, 9, 0)
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM payment_connections WHERE workspace_id = $1 AND mode = 'live' AND disconnected_at IS NULL LIMIT 1`, w.ws.ID).Scan(&w.live); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM payment_connections WHERE workspace_id = $1 AND mode = 'test' AND disconnected_at IS NULL LIMIT 1`, w.ws.ID).Scan(&w.test); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// Planner statistics for the tables the sample's people are picked
	// against (sampleFree): filled in this transaction, they have none, and
	// the picks would scan each of them once per contact.
	if _, err := s.Tx.Exec(ctx, `ANALYZE contacts, list_memberships, suppressions, automation_runs, paid_subscriptions, earn_checkouts, form_submissions,
		poll_answers, survey_responses, unsubscribe_feedback, website_members`); err != nil {
		return err
	}
	for _, step := range []func(context.Context) error{w.coverImages, w.deliveryTemplate, w.listsAndForms, w.sampleFiles, w.productsAndPages,
		w.makeOrders, w.writeOrders, w.pageNumbers, w.signups, w.followUp, w.reminders, w.audit, w.settle} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

// coverImages puts the products' covers in the image library.
func (w *sellSeed) coverImages(ctx context.Context) error {
	entries, err := productAssets.ReadDir("assets/products")
	if err != nil {
		return err
	}
	for i, e := range entries {
		data, err := productAssets.ReadFile("assets/products/" + e.Name())
		if err != nil {
			return err
		}
		ctype, wd, ht, alpha, err := media.Inspect(data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		sum := sha256.Sum256(data)
		var id uuid.UUID
		if err := w.s.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (workspace_id, sha256) DO UPDATE SET file_name = EXCLUDED.file_name RETURNING id`,
			ID(), w.ws.ID, e.Name(), ctype, len(data), wd, ht, alpha, sum[:], data, w.sofia, w.day(9, 21, 10, i)).Scan(&id); err != nil {
			return err
		}
		w.covers[strings.TrimSuffix(e.Name(), ".png")] = media.URL(id, e.Name())
	}
	return nil
}

// deliveryTemplate makes the Purchases category and product-delivery v1,
// published on 1 Oct by Sofia Rossi.
func (w *sellSeed) deliveryTemplate(ctx context.Context) error {
	if SkipTransactional {
		// The Transactional tab's own tests start from an empty tab: the
		// products have no delivery template and their orders no emails.
		return nil
	}
	s := w.s
	q := store.New(s.Tx)
	w.category = ID()
	if err := q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: w.category, WorkspaceID: w.ws.ID, Name: products.CategoryName, Kind: transactional.KindEssential,
		Description: "Buyers' emails: the download links of digital products", CreatedBy: &w.sofia, CreatedAt: w.day(10, 1, 8, 40)}); err != nil {
		return err
	}
	d, testData, err := transactional.StarterDemo(products.TemplateKey, "The Acme newsletter", "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105")
	if err != nil {
		return err
	}
	w.template = ID()
	created, published := w.day(10, 1, 8, 41), w.day(10, 1, 8, 52)
	hash := transactional.DemoHash(d, nil, "", false, false)
	if err := q.TxnTemplateInsert(ctx, store.TxnTemplateInsertParams{ID: w.template, WorkspaceID: w.ws.ID, Key: products.TemplateKey, Name: "Product delivery",
		Description: "A digital product's download link, sent after a purchase", CategoryID: &w.category, Subject: d.Subject, Preheader: d.Preheader, EditorMode: "visual",
		Mjml: d.MJML, Html: d.HTML, PlainText: "", Variables: mustJSON(d.Variables), TestData: mustJSON(testData), ReplyTo: "", TrackOpens: false, TrackClicks: false,
		DraftHash: hash, Now: created, CreatedBy: &w.sofia, Origin: json.RawMessage(`{"source":"products"}`)}); err != nil {
		return err
	}
	v := int32(1)
	if err := q.TxnVersionInsert(ctx, store.TxnVersionInsertParams{TemplateID: w.template, Version: v, Subject: d.Subject, Preheader: d.Preheader, EditorMode: "visual",
		Mjml: d.MJML, Html: d.HTML, PlainText: "", Variables: mustJSON(d.Variables), TestData: mustJSON(testData), ReplyTo: "", ContentHash: hash,
		SizeBytes: int32(len(d.HTML)), Change: "First version", PublishedBy: &w.sofia, PublishedAt: published}); err != nil {
		return err
	}
	if err := q.TxnTemplatePublish(ctx, store.TxnTemplatePublishParams{WorkspaceID: w.ws.ID, ID: w.template, Version: &v, Now: published, LiveBy: &w.sofia,
		Variables: mustJSON(d.Variables)}); err != nil {
		return err
	}
	return q.TxnTemplateInsertDraftRevision(ctx, store.TxnTemplateInsertDraftRevisionParams{TemplateID: w.template, Revision: 1, Subject: d.Subject, Preheader: d.Preheader,
		Mjml: d.MJML, Html: d.HTML, Variables: mustJSON(d.Variables), SavedBy: &w.sofia, SavedAt: created, Label: "created", Source: "starter"})
}

// templateRef is the delivery template in a product's settings: none when
// the demo leaves Transactional out.
func (w *sellSeed) templateRef() string {
	if w.template == uuid.Nil {
		return ""
	}
	return w.template.String()
}

// templateID is the products row's delivery template.
func (w *sellSeed) templateID() *uuid.UUID {
	if w.template == uuid.Nil {
		return nil
	}
	return &w.template
}

// listsAndForms: the DMARC live session's list and date field, the Launch
// Kit waitlist, and the three forms the pages use (the lead magnet form
// offers the checklist after confirmation). Both lists are private: their
// people agreed to one session's emails or one announcement, so "All
// subscribers" leaves them out.
func (w *sellSeed) listsAndForms(ctx context.Context) error {
	s := w.s
	w.dmarc, w.waitlist = ID(), ID()
	dmarc, waitlist := w.dmarc, w.waitlist
	for _, l := range []struct {
		id         uuid.UUID
		name, desc string
		double     bool
		at         time.Time
	}{
		{dmarc, "DMARC live · " + w.event.Format("2 Jan"), "Registrants of the live session on " + w.event.Format("2 January") + ": the joining link and the reminders.", false, w.day(10, 7, 15, 20)},
		{waitlist, "Launch Kit waitlist", "People waiting for the Newsletter Launch Kit to open.", true, w.day(9, 28, 11, 5)},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, public, created_at) VALUES ($1,$2,$3,$4,$5,false,$6)`,
			l.id, w.ws.ID, l.name, l.desc, l.double, l.at); err != nil {
			return err
		}
		s.Lists[l.name] = l.id
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type, created_at) VALUES ($1,$2,'dmarc_live_at','DMARC live session','date',$3)
		ON CONFLICT DO NOTHING`, ID(), w.ws.ID, w.day(10, 7, 15, 22)); err != nil {
		return err
	}
	spam, _ := json.Marshal(map[string]any{"honeypot": true, "min_seconds": 3, "rate_limit_per_ip": 5, "block_disposable": true})
	email := map[string]any{"key": "email", "required": true}
	content := func(heading, button string) map[string]any {
		return map[string]any{"heading": heading, "description": "", "button": button, "footnote": "", "listsLabel": "",
			"emailPlaceholder": "you@company.com", "namePlaceholder": "Sam"}
	}
	for _, f := range []struct {
		key, name, slug, public string
		list                    uuid.UUID
		double                  bool
		fields                  []map[string]any
		content                 map[string]any
		consent                 string
		at                      time.Time
		by                      uuid.UUID
		views, subs, conf       int64
	}{
		{"dns", "DNS checklist", "dns-checklist", "frm_d4c9a1", w.digest, true, []map[string]any{email, {"key": "first_name"}}, content("Get the checklist", "Send me the checklist"),
			"Send me the DNS records checklist and the Weekly Digest. Unsubscribe any time.", w.day(9, 22, 9, 30), w.sofia, 6412, 1148, 1032},
		{"dmarc", "DMARC live", "dmarc-live", "frm_e7b302", dmarc, false, []map[string]any{email, {"key": "first_name"}}, content("Save your seat", "Register"),
			"Send me the joining link and reminders for this session.", w.day(10, 8, 10, 2), w.marcus, 2930, 486, 486},
		{"waitlist", "Launch Kit waitlist", "launch-kit-waitlist", "frm_a91f55", waitlist, true, []map[string]any{email}, content("Join the waitlist", "Join the waitlist"),
			"Email me when the Newsletter Launch Kit opens.", w.day(9, 29, 9, 12), w.sofia, 3618, 1406, 1298},
	} {
		id := ID()
		w.forms[f.key] = id
		fields, _ := json.Marshal(f.fields)
		body, _ := json.Marshal(f.content)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, kind, slug, public_id, list_ids, default_list_ids, list_choice, double_opt_in,
			fields, content, spam_protection, consent_text, consent_version, allowed_origins, after_submit, redirect_url, status, published_at,
			views, submissions, confirmed, blocked, created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,$3,'embedded',$4,$5,$6,$6,false,$7,$8,$9,$10,$11,1,'{news.acme.io}','message','','live',$12,$13,$14,$15,0,$16,$16,$12,$12)`,
			id, w.ws.ID, f.name, f.slug, f.public, []uuid.UUID{f.list}, f.double, fields, body, spam, f.consent, f.at, f.views, f.subs, f.conf, f.by); err != nil {
			return fmt.Errorf("form %s: %w", f.name, err)
		}
		if err := formsDailyNumbers(ctx, s, id, formsDemoForm{name: f.name, views: f.views, submissions: f.subs, confirmed: f.conf}); err != nil {
			return err
		}
	}
	return nil
}

// sampleFiles stores the products' files and the checklist on the data
// volume: small stand-ins with the real files' kinds, listed with the
// sizes the boards show.
func (w *sellSeed) sampleFiles(ctx context.Context) error {
	pdf := []byte("%PDF-1.4\n% ReputeCast demo file\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
	zip := []byte("PK\x05\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	files := []struct {
		key, owner, name, kind, ctype string
		data                          []byte
		size                          int64
		pages                         *int32
		at                            time.Time
		by                            uuid.UUID
	}{
		{"fg", "product", "deliverability-field-guide.pdf", "pdf", "application/pdf", pdf, 4404019, i32p(86), w.day(10, 1, 8, 30), w.sofia},
		{"lk", "product", "newsletter-launch-kit.zip", "zip", "application/zip", zip, 39845888, nil, w.day(10, 5, 16, 10), w.sofia},
		{"wp", "product", "warm-up-planner.xlsx", "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", append([]byte(nil), zip...), 215040, nil, w.day(10, 14, 15, 5), w.daniel},
		{"dns", "form", "dns-records-checklist.pdf", "pdf", "application/pdf", append(append([]byte(nil), pdf...), '\n'), 389120, i32p(2), w.day(9, 22, 9, 20), w.sofia},
	}
	for _, f := range files {
		key, sum, err := w.store.PutBytes(w.ws.ID, f.data)
		if err != nil {
			return fmt.Errorf("file %s: %w", f.name, err)
		}
		id := ID()
		w.files[f.key] = id
		// The owner is set once products and forms exist (productsAndPages).
		if _, err := w.s.Tx.Exec(ctx, `INSERT INTO download_files (id, workspace_id, owner_kind, owner_id, name, size_bytes, kind, content_type, pages, sha256, storage_key, position, uploaded_by, uploaded_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,0,$12,$13)`, id, w.ws.ID, f.owner, uuid.Nil, f.name, f.size, f.kind, f.ctype, f.pages, sum, key, f.by, f.at); err != nil {
			return err
		}
	}
	_, err := w.s.Tx.Exec(ctx, `UPDATE forms SET download_file_id = $2, download_days = 7 WHERE id = $1`, w.forms["dns"], w.files["dns"])
	if err != nil {
		return err
	}
	_, err = w.s.Tx.Exec(ctx, `UPDATE download_files SET owner_id = $2 WHERE id = $1`, w.files["dns"], w.forms["dns"])
	return err
}

func i32p(n int32) *int32 { return &n }

// productsAndPages: the three products and the six pages, with the Field
// Guide page's versions.
func (w *sellSeed) productsAndPages(ctx context.Context) error {
	s := w.s
	for _, k := range []string{"fg", "lk", "wp"} {
		w.prods[k] = ID()
	}
	for _, k := range []string{"field-guide", "dns-checklist", "launch-kit", "dmarc-live", "about", "sponsor"} {
		w.pages[k] = ID()
	}
	// Pages first: products point at their sales pages.
	type pageVersion struct {
		html     string
		label    string
		at       time.Time
		by       uuid.UUID
		live     bool
		settings map[string]any
	}
	set := func(seoTitle, seoDesc string, nav bool, label string) map[string]any {
		return map[string]any{"seoTitle": seoTitle, "seoDescription": seoDesc, "socialImage": "", "visibility": "public", "tiers": []string{},
			"hideHeader": false, "showInNav": nav, "navLabel": label}
	}
	focus := func(m map[string]any) map[string]any { m["hideHeader"] = true; return m }
	fgSet := set("The Deliverability Field Guide · Acme", "Eight chapters on getting into the inbox and staying there, from the people who run Acme's sending. PDF, 86 pages.", true, "Field Guide")
	fgDraftSet := set("The Deliverability Field Guide · Acme", "Eight practical chapters on getting your newsletter into the inbox and keeping it there. PDF, 86 pages, with a checklist after each.", true, "Field Guide")
	fgLive, fgDraft := fieldGuideHTML(false, w.prods["fg"].String(), w.covers["field-guide"]), fieldGuideHTML(true, w.prods["fg"].String(), w.covers["field-guide"])
	pagesSeed := []struct {
		key, name, template string
		created             time.Time
		by                  uuid.UUID
		versions            []pageVersion
		views               int64
	}{
		{"field-guide", "The Deliverability Field Guide", pages.TemplateSales, w.day(9, 24, 10, 15), w.sofia, nil, 9840},
		{"dns-checklist", "DNS records checklist", pages.TemplateLeadMagnet, w.day(9, 19, 14, 40), w.sofia, []pageVersion{
			{dnsChecklistHTML(w.forms["dns"].String(), w.covers["dns-checklist"]), "created", w.day(9, 19, 14, 40), w.sofia, false, focus(set("", "SPF, DKIM, DMARC, the return path and BIMI, in the order to add them.", false, ""))},
			{dnsChecklistHTML(w.forms["dns"].String(), w.covers["dns-checklist"]), "saved", w.day(9, 22, 9, 40), w.sofia, true, focus(set("", "SPF, DKIM, DMARC, the return path and BIMI, in the order to add them.", false, ""))},
		}, 6412},
		{"launch-kit", "Newsletter Launch Kit", pages.TemplateLaunch, w.day(9, 27, 15, 0), w.sofia, []pageVersion{
			{launchKitHTML(w.prods["lk"].String(), w.forms["waitlist"].String(), w.covers["launch-kit"], w.opens), "created", w.day(9, 27, 15, 0), w.sofia, false, focus(set("", "Twelve templates and a checklist to get your first issues out the door.", false, ""))},
			{launchKitHTML(w.prods["lk"].String(), w.forms["waitlist"].String(), w.covers["launch-kit"], w.opens), "saved", w.day(9, 29, 9, 20), w.sofia, true, focus(set("", "Twelve templates and a checklist to get your first issues out the door.", false, ""))},
		}, 3618},
		{"dmarc-live", "Live: Fix your DMARC setup in 30 minutes", pages.TemplateWebinar, w.day(10, 7, 15, 30), w.marcus, []pageVersion{
			{dmarcLiveHTML(w.forms["dmarc"].String(), w.event), "created", w.day(10, 7, 15, 30), w.marcus, false, focus(set("", "Marcus Chen takes a real domain from p=none to p=quarantine, live.", false, ""))},
			{dmarcLiveHTML(w.forms["dmarc"].String(), w.event), "saved", w.day(10, 8, 10, 5), w.marcus, true, focus(set("", "Marcus Chen takes a real domain from p=none to p=quarantine, live.", false, ""))},
		}, 2930},
		{"about", "About", pages.TemplateAbout, w.day(9, 26, 11, 0), w.linh, []pageVersion{
			{aboutHTML(w.formsNewsletter(ctx)), "created", w.day(9, 26, 11, 0), w.linh, false, set("", "Who writes the Acme newsletter, and what you get every Tuesday.", true, "")},
			{aboutHTML(w.formsNewsletter(ctx)), "saved", w.day(9, 29, 10, 12), w.linh, true, set("", "Who writes the Acme newsletter, and what you get every Tuesday.", true, "")},
		}, 2104},
		{"sponsor", "Sponsor the Weekly Digest", pages.TemplateBlank, w.day(10, 14, 16, 20), w.daniel, []pageVersion{
			{sponsorHTML(), "created", w.day(10, 14, 16, 20), w.daniel, false, set("", "", false, "")},
			{sponsorHTML(), "saved", w.day(10, 15, 10, 30), w.daniel, false, set("", "", false, "")},
			{sponsorHTML(), "autosave", w.day(10, 15, 10, 41), w.daniel, false, set("", "", false, "")},
		}, 0},
	}
	// The Field Guide page: nine versions, v8 published on 1 Oct, v9 the
	// draft with the changes the publish dialog lists.
	fg := []pageVersion{}
	for i := 1; i <= 9; i++ {
		v := pageVersion{html: fgLive, label: "saved", at: w.day(9, 23+min(i, 7), 11, 0), by: w.sofia, settings: fgSet}
		switch i {
		case 1:
			v.label = "created"
		case 8:
			v.live, v.at = true, w.day(10, 1, 9, 0)
		case 9:
			v.html, v.settings, v.at = fgDraft, fgDraftSet, w.day(10, 14, 10, 41)
		}
		fg = append(fg, v)
	}
	pagesSeed[0].versions = fg
	for _, p := range pagesSeed {
		id := w.pages[p.key]
		last := p.versions[len(p.versions)-1]
		var liveV *int32
		var livePub *time.Time
		var liveBy *uuid.UUID
		status := pages.StatusDraft
		for i, v := range p.versions {
			if v.live {
				n := int32(i + 1)
				liveV, livePub, liveBy, status = &n, &v.at, &v.by, pages.StatusPublished
			}
		}
		html := pages.UniqueIDs(pages.Sanitize(last.html))
		sj, _ := json.Marshal(last.settings)
		var sd struct {
			SEOTitle, SEODescription, NavLabel string
			ShowInNav, HideHeader              bool
		}
		_ = json.Unmarshal(sj, &sd)
		var liveSlug *string
		if liveV != nil {
			slug := p.key
			liveSlug = &slug
		}
		autosaves := 0
		if last.label == "autosave" {
			autosaves = 1
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO pages (id, workspace_id, name, slug, template, seo_title, seo_description, social_image, visibility, tiers, hide_header,
			show_in_nav, nav_label, project, html, version, saved_at, saved_by, autosaves, status, live_version, live_slug, published_at, published_by, first_published_at,
			created_by, created_at, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'','public','{}',$8,$9,$10,'{}',$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$19,$21,$22,$14,$13)`,
			id, w.ws.ID, p.name, p.key, p.template, sd.SEOTitle, sd.SEODescription, sd.HideHeader, sd.ShowInNav, sd.NavLabel, html, len(p.versions), last.at, last.by,
			autosaves, status, liveV, liveSlug, livePub, liveBy, p.by, p.created); err != nil {
			return fmt.Errorf("page %s: %w", p.key, err)
		}
		for i, v := range p.versions {
			var pubAt *time.Time
			var pubBy *uuid.UUID
			if v.live {
				pubAt, pubBy = &v.at, &v.by
			}
			vj, _ := json.Marshal(v.settings)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO page_versions (page_id, version, name, slug, settings, project, html, label, saved_by, saved_at, published_at, published_by)
				VALUES ($1,$2,$3,$4,$5,'{}',$6,$7,$8,$9,$10,$11)`, id, i+1, p.name, p.key, vj, pages.UniqueIDs(pages.Sanitize(v.html)), v.label, v.by, v.at, pubAt, pubBy); err != nil {
				return fmt.Errorf("page %s v%d: %w", p.key, i+1, err)
			}
		}
	}
	// The products.
	live := func(name, slug, short, long, cover, alt, kind string, price, minimum, suggested int64, page uuid.UUID, offer bool) json.RawMessage {
		l := products.Live{Name: name, Slug: slug, ShortDesc: short, LongDesc: long, CoverURL: cover, CoverAlt: alt, PriceKind: kind, PriceCents: price, MinCents: minimum,
			SuggestedCents: suggested, ShowInShop: true, OfferOn: offer, OfferText: "Also send me the Weekly Digest", LinkDays: 7, LinkDownloads: 5,
			DeliveryTemplate: w.templateRef(), RevokeOnRefund: true}
		if page != uuid.Nil {
			l.SalesPageID = page.String()
		}
		if offer {
			l.OfferListID = w.digest.String()
		}
		b, _ := json.Marshal(l)
		return b
	}
	fgLong := "Get your newsletter into the inbox and keep it there. Eight practical chapters on authentication, warm-up, list hygiene and what to do when a mailbox provider starts pushing back."
	fgAlt := "Cover of The Deliverability Field Guide: a dark book cover with indigo bands and the title in white"
	lkAlt := "Cover of the Newsletter Launch Kit: four tiles in indigo and white under the title"
	stripe := func(product, price, kind string, amount, minimum, suggested int64) json.RawMessage {
		b, _ := json.Marshal(map[string]any{"live": map[string]any{"product": product, "price": price, "kind": kind, "amount": amount, "min": minimum, "suggested": suggested}})
		return b
	}
	for _, p := range []struct {
		key, name, slug, short, long, cover, alt, kind string
		price, minimum, suggested                      int64
		page                                           uuid.UUID
		offer                                          bool
		status                                         string
		live                                           json.RawMessage
		stripe                                         json.RawMessage
		published                                      *time.Time
		by                                             uuid.UUID
		created                                        time.Time
	}{
		{"fg", "The Deliverability Field Guide", "deliverability-field-guide", "Eight chapters on getting into the inbox and staying there. PDF, 86 pages.",
			fgLong + " Each chapter ends with a one-page checklist.", w.covers["field-guide"], fgAlt, products.PriceOneTime, 2900, 0, 0, w.pages["field-guide"], true, products.StatusPublished,
			live("The Deliverability Field Guide", "deliverability-field-guide", "Eight chapters on getting into the inbox and staying there. PDF, 86 pages.", fgLong,
				w.covers["field-guide"], fgAlt, products.PriceOneTime, 2900, 0, 0, w.pages["field-guide"], true),
			stripe("prod_Qx7FieldGuide01", "price_1Q9FieldGuide2900", "one_time", 2900, 0, 0), tp(w.day(10, 1, 9, 0)), w.sofia, w.day(9, 24, 10, 5)},
		{"lk", "Newsletter Launch Kit", "newsletter-launch-kit", "12 MJML templates and a launch checklist. ZIP, 38 MB.",
			"Twelve MJML templates (welcome, digest, launch and re-engagement) and a 30-step checklist from domain setup to your third issue. Pay what you want, from $5.",
			w.covers["launch-kit"], lkAlt, products.PricePWYW, 0, 500, 1500, w.pages["launch-kit"], true, products.StatusPublished,
			live("Newsletter Launch Kit", "newsletter-launch-kit", "12 MJML templates and a launch checklist. ZIP, 38 MB.",
				"Twelve MJML templates (welcome, digest, launch and re-engagement) and a 30-step checklist from domain setup to your third issue. Pay what you want, from $5.",
				w.covers["launch-kit"], lkAlt, products.PricePWYW, 0, 500, 1500, w.pages["launch-kit"], true),
			stripe("prod_Qx7LaunchKit001", "price_1Q9LaunchKitPWYW", "pwyw", 0, 500, 1500), tp(w.day(10, 6, 9, 0)), w.sofia, w.day(9, 27, 14, 30)},
		{"wp", "Warm-up planner", "warm-up-planner", "A four-week, day-by-day sending plan for a new domain or IP.",
			"Enter your list size and providers, and the planner sets each day's volume, the numbers to check and when to slow down. Built from chapter 2 of The Deliverability Field Guide.",
			w.covers["warm-up-planner"], "Cover of the Warm-up planner: a white sheet with a four-week grid", products.PriceOneTime, 1200, 0, 0, uuid.Nil, false, products.StatusDraft,
			nil, json.RawMessage(`{}`), nil, w.daniel, w.day(10, 14, 14, 50)},
	} {
		var page *uuid.UUID
		if p.page != uuid.Nil {
			pg := p.page
			page = &pg
		}
		var offerList *uuid.UUID
		if p.offer {
			offerList = &w.digest
		}
		var liveSlug *string
		var publishedBy *uuid.UUID
		if p.published != nil {
			sl := p.slug
			liveSlug, publishedBy = &sl, &p.by
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO products (id, workspace_id, name, slug, short_desc, long_desc, cover_url, cover_alt, price_kind, price_cents, min_cents,
			suggested_cents, sales_page_id, show_in_shop, offer_on, offer_list_id, offer_text, link_days, link_downloads, delivery_template_id, revoke_on_refund, status,
			live, live_slug, stripe, currency, published_at, published_by, first_published_at, created_by, created_at, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,true,$14,$15,'Also send me the Weekly Digest',7,5,$16,true,$17,$18,$19,$20,'usd',$21,$22,$21,$23,$24,$23,$25)`,
			w.prods[p.key], w.ws.ID, p.name, p.slug, p.short, p.long, p.cover, p.alt, p.kind, p.price, p.minimum, p.suggested, page, p.offer, offerList, w.templateID(),
			p.status, p.live, liveSlug, p.stripe, p.published, publishedBy, p.by, p.created, w.day(10, 14, 15, 5)); err != nil {
			return fmt.Errorf("product %s: %w", p.key, err)
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE download_files SET owner_id = $2 WHERE id = $1`, w.files[p.key], w.prods[p.key]); err != nil {
			return err
		}
	}
	// The header links the Field Guide and About pages (seed_website.go
	// put them in the navigation).
	return nil
}

// formsNewsletter is the workspace's main signup form (the About page's band).
func (w *sellSeed) formsNewsletter(ctx context.Context) string {
	var id uuid.UUID
	if err := w.s.Tx.QueryRow(ctx, `SELECT id FROM forms WHERE workspace_id = $1 AND slug = 'footer-mini-form'`, w.ws.ID).Scan(&id); err != nil {
		return ""
	}
	return id.String()
}

// sellOrder is one order of the sample.
type sellOrder struct {
	id         uuid.UUID
	number     int64
	product    string
	at         time.Time
	email      string
	name       string
	contact    *uuid.UUID
	buyer      string
	amount     int64
	source     string
	ticked     bool
	test       bool
	refundedAt *time.Time
	dispute    bool
	// links: the purchase link and what happened to it.
	used      int32
	expired   bool
	newLinkBy *uuid.UUID
	newLinkAt *time.Time
	message   bool
	delivered time.Duration
}

// makeOrders lays out the 311 live orders and the two test orders.
func (w *sellSeed) makeOrders(ctx context.Context) error {
	fixed := map[int64]*sellOrder{
		1107: {product: "fg", at: w.day(10, 3, 11, 20), email: "sam.rivera@outlook.com", name: "Sam Rivera", buyer: "contact", source: "page", used: 2, expired: true, message: true},
		1288: {product: "fg", at: w.day(10, 12, 21, 15), email: "j.kowalski@example.pl", name: "Jan Kowalski", buyer: "none", source: "page", dispute: true, used: 3, message: true},
		1302: {product: "fg", at: w.day(10, 14, 19, 2), email: "d.okoro@example.org", name: "David Okoro", buyer: "none", source: "page", expired: true, newLinkBy: &w.sofia, newLinkAt: tp(w.day(10, 15, 8, 0)), message: true},
		1309: {product: "fg", at: w.day(10, 15, 8, 30), email: "lee.wong@example.com", name: "Lee Wong", buyer: "none", source: "page", refundedAt: tp(w.day(10, 15, 9, 5)), message: true},
		1310: {product: "lk", at: w.day(10, 15, 9, 47), email: "marco.bianchi@example.it", name: "Marco Bianchi", buyer: "none", source: "page", amount: 2000, message: true},
		1311: {product: "fg", at: w.day(10, 15, 10, 12), email: "ana.ruiz@example.com", name: "Ana Ruiz", buyer: "joined", source: "page", ticked: true, used: 1, message: true, delivered: 1400 * time.Millisecond},
	}
	// The other orders' times: between the fixed ones, the Field Guide from
	// its launch on 1 Oct, the Launch Kit from 6 Oct, 09:00.
	start := w.day(10, 1, 9, 4)
	lkOpen := w.day(10, 6, 9, 0)
	anchors := []int64{1000, 1107, 1288, 1302, 1309}
	times := map[int64]time.Time{1000: start}
	for n, o := range fixed {
		times[n] = o.at
	}
	all := map[int64]*sellOrder{}
	for i := 0; i < len(anchors)-1; i++ {
		a, b := anchors[i], anchors[i+1]
		ta, tb := times[a], times[b]
		for n := a + 1; n < b; n++ {
			f := float64(n-a) / float64(b-a)
			// Busier at launches: front-load each stretch a little.
			f = f * (0.7 + 0.3*f)
			at := ta.Add(time.Duration(f * float64(tb.Sub(ta))))
			all[n] = &sellOrder{at: at.Add(time.Duration(w.rng.IntN(600)) * time.Second)}
		}
	}
	// 1310 sits between 1309 and 1311; no other number is missing.
	for n, o := range fixed {
		all[n] = o
	}
	nums := make([]int64, 0, len(all))
	for n := range all {
		nums = append(nums, n)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	if len(nums) != 311 {
		return fmt.Errorf("sell seed: %d orders, want 311", len(nums))
	}
	// Products: the Launch Kit gets 96 of the free slots after it opened.
	var afterOpen []int64
	for _, n := range nums {
		if _, isFixed := fixed[n]; isFixed {
			continue
		}
		if all[n].at.After(lkOpen) {
			afterOpen = append(afterOpen, n)
		}
	}
	if len(afterOpen) < 96 {
		return fmt.Errorf("sell seed: %d orders after the Launch Kit opened", len(afterOpen))
	}
	w.rng.Shuffle(len(afterOpen), func(i, j int) { afterOpen[i], afterOpen[j] = afterOpen[j], afterOpen[i] })
	lk := map[int64]bool{}
	for _, n := range afterOpen[:96] {
		lk[n] = true
	}
	var fgFree, lkFree []int64
	for _, n := range nums {
		o := all[n]
		o.number = n
		o.id = ID()
		if _, isFixed := fixed[n]; isFixed {
			continue
		}
		if lk[n] {
			o.product = "lk"
			lkFree = append(lkFree, n)
		} else {
			o.product = "fg"
			fgFree = append(fgFree, n)
		}
	}
	// Sources: FG 25 from the shop, LK 16.
	pick := func(list []int64, k int) map[int64]bool {
		cp := append([]int64(nil), list...)
		w.rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
		out := map[int64]bool{}
		for _, n := range cp[:k] {
			out[n] = true
		}
		return out
	}
	fgShop, lkShop := pick(fgFree, 25), pick(lkFree, 16)
	for _, n := range fgFree {
		all[n].source = "page"
		if fgShop[n] {
			all[n].source = "shop"
		}
		all[n].amount = 2900
	}
	for _, n := range lkFree {
		all[n].source = "page"
		if lkShop[n] {
			all[n].source = "shop"
		}
	}
	for _, o := range fixed {
		if o.product == "fg" {
			o.amount = 2900
		}
	}
	// Launch Kit amounts: pay what you want, $1,071 from its page (81
	// orders, 1310's $20 among them) and $217 from the shop (16).
	if err := w.pwyw(all, lkFree, false, 107100-2000); err != nil {
		return err
	}
	if err := w.pwyw(all, lkFree, true, 21700); err != nil {
		return err
	}
	// Buyers: FG 131 contacts, 44 joined at checkout, 39 not contacts; LK
	// 61, 17 and 19. The fixed orders count in. The contacts and those who
	// joined the Weekly Digest at checkout are sample contacts already on
	// it, as the store sample does: the Audience boards keep their 48,213
	// contacts and the three lists their numbers.
	contacts, err := w.existingContacts(ctx, (131-1)+61+(44-1)+17)
	if err != nil {
		return err
	}
	kinds := func(list []int64, contact, joined int) {
		cp := append([]int64(nil), list...)
		w.rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
		for i, n := range cp {
			o := all[n]
			switch {
			case i < contact:
				o.buyer = "contact"
			case i < contact+joined:
				o.buyer, o.ticked = "joined", true
			default:
				o.buyer = "none"
			}
		}
	}
	kinds(fgFree, 131-1, 44-1)
	kinds(lkFree, 61, 17)
	first := []string{"Alex", "Jordan", "Priya", "Mateo", "Yuki", "Chloe", "Noah", "Amara", "Lukas", "Sofia", "Ravi", "Elena", "Owen", "Hana", "Tomás", "Ingrid", "Kofi", "Mei", "Felix", "Zara"}
	last := []string{"Martin", "Okafor", "Silva", "Novak", "Tanaka", "Dubois", "Kim", "Haddad", "Berg", "Moreau", "Iyer", "Costa", "Walsh", "Sato", "Ruiz", "Lund", "Mensah", "Chen", "Weber", "Patel"}
	domains := []string{"example.com", "example.org", "example.net", "example.io", "example.co.uk", "example.de"}
	ci := 0
	for _, n := range nums {
		o := all[n]
		switch {
		case o.email != "":
			// A fixed order: its buyer is known (Lee Wong and Ana Ruiz are
			// sample contacts by those names).
			if o.buyer == "contact" || o.buyer == "joined" {
				id, err := w.sampleAs(ctx, o.email, o.name)
				if err != nil {
					return err
				}
				o.contact = &id
			}
		case o.buyer == "contact" || o.buyer == "joined":
			if ci >= len(contacts) {
				return fmt.Errorf("sell seed: %d sample contacts for the buyers", len(contacts))
			}
			c := contacts[ci]
			ci++
			o.email, o.name, o.contact = c.email, c.name, &c.id
		default:
			f, l := first[w.rng.IntN(len(first))], last[w.rng.IntN(len(last))]
			o.name = f + " " + l
			o.email = strings.ToLower(strings.NewReplacer("á", "a", "é", "e", "ó", "o", "ú", "u", "í", "i").Replace(f[:1]+"."+l)) + fmt.Sprint(n) + "@" + domains[w.rng.IntN(len(domains))]
		}
		if o.buyer == "joined" {
			if err := w.joinedAtCheckout(ctx, *o.contact, o.at, o.product, o.number == 1311); err != nil {
				return err
			}
		}
		// Downloads: most buyers download once or twice within a day.
		if o.used == 0 && o.number != 1310 && o.number != 1309 {
			o.used = int32(1 + w.rng.IntN(2))
		}
	}
	// The "downloads used up" order: a Launch Kit buyer with all 5 used.
	for _, n := range lkFree {
		if all[n].buyer == "none" && all[n].at.Before(w.day(10, 13, 0, 0)) {
			all[n].email, all[n].name, all[n].used = "k.ito@example.net", "Kaito Ito", 5
			break
		}
	}
	// Refunds: 1309 and two other Field Guide buyers who are contacts.
	refunds := 0
	for _, n := range fgFree {
		o := all[n]
		if o.contact != nil && o.at.After(w.day(10, 4, 0, 0)) && refunds < 2 {
			o.refundedAt = tp(o.at.Add(26 * time.Hour))
			refunds++
		}
	}
	for _, n := range nums {
		w.orders = append(w.orders, all[n])
	}
	// The test orders (Stripe in test mode, Linh checking the whole path).
	w.orders = append(w.orders,
		&sellOrder{id: ID(), number: 1312, product: "lk", at: w.day(10, 15, 10, 31), email: "linh@acme.io", name: "Linh Nguyen", buyer: "none", amount: 1500, source: "page", test: true, message: true},
		&sellOrder{id: ID(), number: 1313, product: "fg", at: w.day(10, 15, 11, 2), email: "linh@acme.io", name: "Linh Nguyen", buyer: "none", amount: 2900, source: "page", test: true, used: 1, message: true},
	)
	return nil
}

// pwyw gives the Launch Kit's orders from its page (shop false) or the
// shop amounts that add up to total.
func (w *sellSeed) pwyw(all map[int64]*sellOrder, list []int64, shop bool, total int64) error {
	var mine []*sellOrder
	for _, n := range list {
		o := all[n]
		if (o.source == "shop") == shop {
			mine = append(mine, o)
		}
	}
	if len(mine) == 0 {
		return nil
	}
	choices := []int64{500, 1000, 1000, 1500, 1500, 1500, 1500, 2000, 2500}
	var sum int64
	for _, o := range mine {
		o.amount = choices[w.rng.IntN(len(choices))]
		sum += o.amount
	}
	// Nudge amounts by $1 until the total is right, keeping each at $5 or more.
	for i := 0; sum != total && i < 100000; i++ {
		o := mine[i%len(mine)]
		switch {
		case sum < total && o.amount < 5000:
			o.amount += 100
			sum += 100
		case sum > total && o.amount > 500:
			o.amount -= 100
			sum -= 100
		}
	}
	if sum != total {
		return fmt.Errorf("sell seed: pay what you want adds to %d, want %d", sum, total)
	}
	return nil
}

// writeOrders writes the orders, their checkouts, links, downloads, events
// and the pages' sales.
func (w *sellSeed) writeOrders(ctx context.Context) error {
	s := w.s
	var orders, checkouts, links, downloads, events, convs [][]any
	names := map[string]string{"fg": "The Deliverability Field Guide", "lk": "Newsletter Launch Kit"}
	pageOf := map[string]uuid.UUID{"fg": w.pages["field-guide"], "lk": w.pages["launch-kit"]}
	pathOf := map[string]string{"fg": "news.acme.io/field-guide", "lk": "news.acme.io/launch-kit"}
	ev := func(order uuid.UUID, kind string, at time.Time, by *uuid.UUID, detail map[string]any) {
		events = append(events, []any{ID(), order, kind, at, by, mustJSON(detail)})
	}
	for _, o := range w.orders {
		mode, conn := "live", w.live
		if o.test {
			mode, conn = "test", w.test
		}
		var connRef *uuid.UUID
		if conn != uuid.Nil {
			c := conn
			connRef = &c
		}
		checkout := ID()
		started := o.at.Add(-time.Duration(40+w.rng.IntN(140)) * time.Second)
		session := fmt.Sprintf("cs_%s_%s", mode, strings.ReplaceAll(o.id.String(), "-", "")[:24])
		var page *uuid.UUID
		url := "news.acme.io/products"
		if o.source == "page" {
			p := pageOf[o.product]
			page = &p
			url = pathOf[o.product]
		}
		consent := map[string]any{"offered": true, "ticked": o.ticked, "text": "Also send me the Weekly Digest", "listId": w.digest.String(), "page": url,
			"ip": "203.0.113.77", "at": started}
		checkouts = append(checkouts, []any{checkout, w.ws.ID, w.prods[o.product], mode, connRef, session, o.email, o.contact, mustJSON(consent), o.source, page,
			"https://" + url, "completed", o.id, started, o.at})
		status := products.OrderPaid
		var refunded int64
		revokeReason, disputeStatus, disputeReason := "", "", ""
		var revokedAt, disputedAt, dueBy *time.Time
		if o.refundedAt != nil {
			status, refunded, revokeReason, revokedAt = products.OrderRefunded, o.amount, "refund", o.refundedAt
		}
		if o.dispute {
			status, disputeStatus, disputeReason = products.OrderDisputed, "open", "fraudulent"
			disputedAt, dueBy = tp(w.day(10, 14, 6, 12)), tp(w.day(10, 28, 19, 59))
			revokeReason, revokedAt = "dispute", disputedAt
		}
		disputeID, disputeFee := "", int64(0)
		if o.dispute {
			disputeID, disputeFee = "dp_1Q9Kowalski1288", 1500
		}
		pi := "pi_" + strings.ReplaceAll(o.id.String(), "-", "")[:24]
		orders = append(orders, []any{o.id, w.ws.ID, o.number, w.prods[o.product], names[o.product], mode, connRef, checkout, o.email, o.name, o.contact, o.buyer,
			o.amount, int64(0), "usd", o.product == "lk", status, refunded, o.refundedAt, disputeID, disputeStatus, disputeReason, disputeFee, disputedAt, dueBy,
			revokedAt, revokeReason, "stripe", session, pi, "ch_" + pi[3:], o.source, page, "https://" + url, mustJSON(consent), "", started, o.at, o.at})
		// The sale on its page (or the shop).
		if !o.test {
			convs = append(convs, []any{ID(), w.ws.ID, page, o.source, "sale", o.contact, o.id, o.amount, o.at})
		}
		// The purchase link, then a new one for 1302.
		link := ID()
		expires := o.at.Add(7 * 24 * time.Hour)
		if o.expired && o.newLinkAt != nil {
			expires = o.newLinkAt.Add(-30 * time.Minute)
		}
		var superseded *time.Time
		if o.newLinkAt != nil {
			superseded = o.newLinkAt
		}
		var msg *uuid.UUID
		if o.message {
			sent := o.at.Add(2 * time.Second)
			if o.number == 1311 {
				// "delivered via Postmark at 10:12:41", 1.4 s after it was sent.
				sent = o.at.Add(39600 * time.Millisecond)
			}
			msg = w.deliveryMessage(ctx, o, sent, o.delivered)
		}
		used := o.used
		if used > 5 {
			used = 5
		}
		links = append(links, []any{link, o.id, "purchase", nil, o.at.Add(time.Second), expires, int32(5), used, msg, "sent", superseded})
		for i := int32(0); i < used; i++ {
			at := o.at.Add(time.Duration(50+int(i)*3600*(1+w.rng.IntN(20))) * time.Second)
			var ip []byte
			switch o.number {
			case 1311:
				at = o.at.Add(time.Minute)
			case 1288:
				at = []time.Time{o.at.Add(4 * time.Minute), o.at.Add(31 * time.Minute), w.day(10, 13, 8, 2)}[i]
				ip = []byte{0x12, 0x88, byte(min(i, 1))}
			}
			downloads = append(downloads, []any{ID(), o.id, link, w.files[o.product], ip, at})
		}
		ev(o.id, "checkout_started", started, nil, map[string]any{"page": url, "ticked": o.ticked})
		ev(o.id, "paid", o.at, nil, map[string]any{"stripe": "checkout.session.completed"})
		if o.newLinkAt != nil {
			ev(o.id, "buyer_asked", o.newLinkAt.Add(-19*time.Minute), nil, map[string]any{"note": "reply to the delivery email: “The link says it has expired”"})
			nl := ID()
			nmsg := w.deliveryMessage(ctx, o, *o.newLinkAt, 1300*time.Millisecond)
			links = append(links, []any{nl, o.id, "person", o.newLinkBy, *o.newLinkAt, o.newLinkAt.Add(7 * 24 * time.Hour), int32(5), int32(0), nmsg, "sent", nil})
			ev(o.id, "new_link", *o.newLinkAt, o.newLinkBy, map[string]any{"email": o.email, "note": "the buyer, by replying to the delivery email"})
		}
		if o.refundedAt != nil {
			var by *uuid.UUID
			if o.number == 1309 {
				by = &w.sofia
			}
			ev(o.id, "refunded", *o.refundedAt, by, map[string]any{"stripe": "charge.refunded", "amount": o.amount, "currency": "usd", "byStripe": true})
			ev(o.id, "access_revoked", *o.refundedAt, nil, map[string]any{"reason": "refund", "auto": true})
		}
		if o.dispute {
			ev(o.id, "dispute_opened", *disputedAt, nil, map[string]any{"stripe": "charge.dispute.created", "reason": "fraudulent", "respondBy": dueBy})
			ev(o.id, "access_revoked", *disputedAt, nil, map[string]any{"reason": "dispute", "auto": true})
		}
	}
	copyRows := func(table string, cols []string, rows [][]any) error {
		if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
		return nil
	}
	if err := copyRows("product_checkouts", []string{"id", "workspace_id", "product_id", "mode", "connection_id", "session_id", "email", "contact_id", "consent", "source",
		"source_page_id", "source_url", "status", "order_id", "created_at", "completed_at"}, checkouts); err != nil {
		return err
	}
	if err := copyRows("product_orders", []string{"id", "workspace_id", "number", "product_id", "product_name", "mode", "connection_id", "checkout_id", "buyer_email",
		"buyer_name", "contact_id", "buyer", "amount_cents", "tax_cents", "currency", "pay_what_you_want", "status", "refunded_cents", "refunded_at", "dispute_id",
		"dispute_status", "dispute_reason", "dispute_fee_cents", "disputed_at", "dispute_due_by", "access_revoked_at", "access_revoke_reason", "provider", "session_id",
		"payment_intent", "charge_id", "source", "source_page_id", "source_url", "consent", "attributed_by", "checkout_started_at", "paid_at", "created_at"}, orders); err != nil {
		return err
	}
	if err := copyRows("product_order_links", []string{"id", "order_id", "reason", "created_by", "created_at", "expires_at", "downloads_allowed", "downloads_used",
		"message_id", "delivery", "superseded_at"}, links); err != nil {
		return err
	}
	if err := copyRows("product_order_downloads", []string{"id", "order_id", "link_id", "file_id", "ip_hash", "at"}, downloads); err != nil {
		return err
	}
	if err := copyRows("product_order_events", []string{"id", "order_id", "kind", "at", "actor_id", "detail"}, events); err != nil {
		return err
	}
	if err := copyRows("page_conversions", []string{"id", "workspace_id", "page_id", "source", "kind", "contact_id", "order_id", "amount_cents", "occurred_at"}, convs); err != nil {
		return err
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO product_order_numbers (workspace_id, last) VALUES ($1, 1313)`, w.ws.ID)
	return err
}

// deliveryMessage is a delivery email in Transactional › Messages,
// delivered by Postmark.
func (w *sellSeed) deliveryMessage(ctx context.Context, o *sellOrder, at time.Time, took time.Duration) *uuid.UUID {
	postmark := w.s.Providers["postmark"]
	if postmark == uuid.Nil || w.template == uuid.Nil {
		return nil
	}
	if took == 0 {
		took = time.Duration(900+w.rng.IntN(1500)) * time.Millisecond
	}
	id := ID()
	if _, err := w.s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, contact_id, email, provider_id, provider_message_id, mailbox_provider, status, queued_at, sent_at, delivered_at)
		VALUES ($1, $2, 'transactional', $3, $4, $5, $6, $7, 'delivered', $8, $9, $10)`, id, w.ws.ID, o.contact, o.email, postmark, "pm-"+id.String(),
		emailaddr.MailboxProvider(emailaddr.Domain(o.email)), at, at.Add(took/3), at.Add(took)); err != nil {
		return nil
	}
	names := map[string]string{"fg": "The Deliverability Field Guide", "lk": "Newsletter Launch Kit"}
	if _, err := w.s.Tx.Exec(ctx, `INSERT INTO transactional_messages (message_id, workspace_id, source, template_id, template_key, template_name, template_version,
		category_id, category_name, category_kind, subject, from_email, from_name, reply_to, to_name, tags, metadata, data_keys, sensitive, test, accepted_at, rendered_at, provider_since)
		VALUES ($1, $2, 'api', $3, 'product-delivery', 'Product delivery', 1, $4, 'Purchases', 'essential', $5, 'no-reply@notify.acme.io', 'The Acme newsletter',
		'support@acme.io', $6, '{products}', $7, '{product_name,download_url,file,expires_on,downloads_allowed,order_number,amount}', '{download_url}', $8, $9, $9, $9)`,
		id, w.ws.ID, w.template, w.category, "Your download: "+names[o.product], o.name, mustJSON(map[string]string{"source": "products", "order": fmt.Sprint(o.number)}),
		o.test, at); err != nil {
		return nil
	}
	return &id
}

// pageNumbers: 30 days of views and what the pages brought in besides
// sales (signups on About and the checklist, registrations for the
// session), and the lead magnet's links.
func (w *sellSeed) pageNumbers(ctx context.Context) error {
	s := w.s
	type spread struct {
		page  uuid.UUID
		from  time.Time
		views int64
	}
	shop := uuid.Nil
	spreads := []spread{
		{w.pages["field-guide"], w.day(10, 1, 0, 0), 9840},
		{w.pages["dns-checklist"], w.day(9, 22, 0, 0), 6412},
		{w.pages["launch-kit"], w.day(9, 29, 0, 0), 3618},
		{w.pages["dmarc-live"], w.day(10, 8, 0, 0), 2930},
		{w.pages["about"], w.day(9, 29, 0, 0), 2104},
		{shop, w.day(10, 1, 0, 0), 3105},
	}
	var rows [][]any
	for _, sp := range spreads {
		var days []time.Time
		for d := sp.from; !d.After(w.day(10, 15, 0, 0)); d = d.AddDate(0, 0, 1) {
			days = append(days, d)
		}
		weights := make([]float64, len(days))
		for i := range days {
			// A launch spike, then a steady tail.
			weights[i] = 1 + 3/float64(1+i)
		}
		parts := Split(sp.views, weights)
		for i, d := range days {
			day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
			rows = append(rows, []any{w.ws.ID, day, website.PagePage, sp.page, parts[i]})
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"website_view_days"}, []string{"workspace_id", "day", "page_kind", "page_id", "views"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("page views: %w", err)
	}
	// Signups and registrations.
	var convs [][]any
	add := func(page uuid.UUID, kind string, n int, from time.Time) {
		span := w.day(10, 15, 9, 0).Sub(from)
		for i := 0; i < n; i++ {
			at := from.Add(time.Duration(w.rng.Int64N(int64(span))))
			pid := page
			convs = append(convs, []any{ID(), w.ws.ID, &pid, "page", kind, nil, nil, int64(0), at})
		}
	}
	add(w.pages["dns-checklist"], "signup", 1148, w.day(9, 22, 9, 40))
	add(w.pages["about"], "signup", 38, w.day(9, 29, 10, 12))
	// The session's registrations come with their registrants (signups).
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"page_conversions"}, []string{"id", "workspace_id", "page_id", "source", "kind", "contact_id", "order_id", "amount_cents", "occurred_at"},
		pgx.CopyFromRows(convs)); err != nil {
		return fmt.Errorf("page conversions: %w", err)
	}
	// The checklist's links: one per confirmed signup, 1,019 downloaded.
	var lm [][]any
	from := w.day(9, 22, 10, 0)
	span := w.day(10, 15, 9, 0).Sub(from)
	for i := 0; i < 1032; i++ {
		at := from.Add(time.Duration(w.rng.Int64N(int64(span))))
		downloads := int32(1)
		if i >= 1019 {
			downloads = 0
		}
		lm = append(lm, []any{ID(), w.ws.ID, w.forms["dns"], w.files["dns"], nil, at, at.Add(7 * 24 * time.Hour), downloads})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"lead_magnet_links"}, []string{"id", "workspace_id", "form_id", "file_id", "contact_id", "created_at", "expires_at", "downloads"},
		pgx.CopyFromRows(lm)); err != nil {
		return fmt.Errorf("lead magnet links: %w", err)
	}
	return nil
}

// followUp is the "Field Guide follow-up" automation: Bought a product
// (the Field Guide), two days, then a "How's the guide going?" email. 175
// contacts entered; three left after a refund.
func (w *sellSeed) followUp(ctx context.Context) error {
	s := w.s
	email := ID()
	c := automations.DemoEmail("How's the guide going?", "One question, and the chapter most readers start with.", "How's the guide going?", "Open chapter 1",
		"Thanks for buying The Deliverability Field Guide, {{ first_name }}. Most readers start with chapter 1: authentication takes an afternoon and fixes the most.",
		"Reply to this email if a chapter leaves you stuck. Sofia reads every reply.")
	created := w.day(10, 1, 10, 5)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,1,$4,$5,$5)`,
		email, w.ws.ID, `Field Guide follow-up · "How's the guide going?"`, w.sofia, created); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,$7,$8)`,
		email, c.Subject, c.Preheader, c.MJML, c.HTML, c.Text, w.sofia, created); err != nil {
		return err
	}
	graph := automations.AutomationGraph{
		Nodes: []autoFlowNode{
			autoNode("n1", automations.NodeTrigger, 370, 24, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerBoughtProduct, ProductID: w.prods["fg"].String(), Reentry: "once"}}),
			autoNode("n2", automations.NodeDelay, 370, 144, autoFlowData{Delay: autoDur(2, "days")}),
			autoNode("n3", automations.NodeEmail, 370, 215, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: email.String(), EmailVersion: 1, Subject: "How's the guide going?",
				SendWindow: &automations.AutomationSendWindow{Enabled: true, Start: "08:00", End: "20:00"}}}),
			autoNode("n4", automations.NodeExit, 370, 360, autoFlowData{}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", ""), autoEdge("n3", "n4", "")},
	}
	flow := ID()
	if err := autoInsertFlow(ctx, s, autoFlowSeed{id: flow, name: "Field Guide follow-up", description: "Two days after buying The Deliverability Field Guide: how it's going.",
		status: "live", live: 1, created: created, updated: w.day(10, 1, 10, 20), createdBy: w.sofia,
		versions: []autoVersionSeed{{graph: graph, by: w.sofia, created: created, published: tp(w.day(10, 1, 10, 20)), summary: "First version"}}}); err != nil {
		return err
	}
	// The runs: every Field Guide buyer who is a contact.
	var rows [][]any
	for _, o := range w.orders {
		if o.product != "fg" || o.contact == nil || o.test {
			continue
		}
		status, node, reason := "completed", "n4", ""
		var finished *time.Time
		var next *time.Time
		switch {
		case o.refundedAt != nil:
			status, node, reason, finished = "exited", "n2", "refunded", o.refundedAt
		case o.at.After(w.day(10, 13, 10, 0)):
			status, node = "waiting", "n2"
			n := o.at.Add(48 * time.Hour)
			next = &n
		default:
			f := o.at.Add(48 * time.Hour)
			finished = &f
		}
		state := mustJSON(map[string]any{"vars": map[string]any{"order": map[string]any{"product_name": "The Deliverability Field Guide", "amount": "$29.00",
			"number": o.number, "currency": "USD"}}})
		rows = append(rows, []any{ID(), flow, int32(1), *o.contact, node, status, state, o.at, o.at, next, finished, reason})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"automation_runs"}, []string{"id", "automation_id", "version", "contact_id", "node_id", "status", "state", "entered_at",
		"node_entered_at", "next_at", "finished_at", "exit_reason"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("follow-up runs: %w", err)
	}
	return nil
}

// audit: what the people and Stripe did.
func (w *sellSeed) audit(ctx context.Context) error {
	s := w.s
	entries := []auditSeed{
		{at: w.day(10, 1, 9, 0), actorType: "user", actor: "sofia@acme.io", action: "product.published", targetType: "product", targetID: w.prods["fg"].String(), target: "The Deliverability Field Guide",
			after: map[string]any{"price": "$29", "status": "published"}},
		{at: w.day(10, 1, 9, 0), actorType: "user", actor: "sofia@acme.io", action: "page.published", targetType: "page", targetID: w.pages["field-guide"].String(), target: "The Deliverability Field Guide",
			after: map[string]any{"path": "/field-guide", "version": 8}},
		{at: w.day(10, 6, 9, 0), actorType: "user", actor: "sofia@acme.io", action: "product.published", targetType: "product", targetID: w.prods["lk"].String(), target: "Newsletter Launch Kit",
			after: map[string]any{"price": "pay what you want, from $5", "status": "published"}},
		{at: w.day(10, 8, 10, 5), actorType: "user", actor: "marcus@acme.io", action: "page.published", targetType: "page", targetID: w.pages["dmarc-live"].String(), target: "Live: Fix your DMARC setup in 30 minutes",
			after: map[string]any{"path": "/dmarc-live", "version": 2}},
		{at: w.day(10, 15, 8, 0), actorType: "user", actor: "sofia@acme.io", action: "order.link_sent", targetType: "product_order", targetID: "", target: "Order 1302 · The Deliverability Field Guide · $29.00",
			after: map[string]any{"to": "d.okoro@example.org"}},
		{at: w.day(10, 15, 9, 5).Add(14 * time.Second), actorType: "system", actor: "", action: "order.refunded", targetType: "product_order", targetID: "", target: "Order 1309 · The Deliverability Field Guide · $29.00",
			after: map[string]any{"refunded_cents": 2900, "status": "refunded", "access": "revoked"},
			meta:  map[string]any{"source": "stripe", "summary": "Refunded in Stripe · download access revoked, as the product says", "event": "charge.refunded"}},
		{at: w.day(10, 14, 6, 12), actorType: "system", actor: "", action: "order.disputed", targetType: "product_order", targetID: "", target: "Order 1288 · The Deliverability Field Guide · $29.00",
			after: map[string]any{"dispute": "open", "reason": "fraudulent", "access": "revoked"}, meta: map[string]any{"source": "stripe", "event": "charge.dispute.created"}},
	}
	for _, o := range w.orders {
		for i := range entries {
			if strings.HasPrefix(entries[i].target, fmt.Sprintf("Order %d ", o.number)) {
				entries[i].targetID = o.id.String()
			}
		}
	}
	for _, e := range entries {
		if err := s.insertAudit(ctx, w.ws.ID, e); err != nil {
			return err
		}
	}
	return nil
}
