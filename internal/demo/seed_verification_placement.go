package demo

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification"
)

func init() { seedVerificationPlacement = seedVerificationPlacementImpl }

// seedVerificationPlacementImpl seeds address verification and placement
// tests (design expansion X14, boards VerificationServices, VerificationRun,
// PlacementTests, PlacementResult and PlacementSeeds). The sample day is
// Wednesday 15 Oct, 10:30: designDay(0, …) is "today".
//
// Verification: ZeroBounce (default for bulk and imports) and Kickbox
// (default for signup forms), both connected by Marcus on Thu 2 Oct; the
// Sunset tier checked with ZeroBounce on Tue 14 Oct (2,904 valid, 287
// invalid and suppressed, 318 risky, 106 unknown), the Mailchimp import's
// new addresses this morning, and Kickbox's real-time checks on three
// forms. The demo accounts answer from a recording and never reach the
// services.
//
// Placement tests: ten seed mailboxes added by Linh on Thu 2 Oct (iCloud's
// app-specific password refused since Tue 14 Oct, 21:40) and the six tests
// of the list, with every copy of Weekly Digest #142 · SES vs Postal as the
// result board shows it. The seeds are demo mailboxes: tests started from
// the demo find their copies in a simulated mailbox, never a real one.
func seedVerificationPlacementImpl(ctx context.Context, s *Seeder) error {
	if err := seedVerificationDemo(ctx, s); err != nil {
		return fmt.Errorf("verification: %w", err)
	}
	if err := seedPlacementDemo(ctx, s); err != nil {
		return fmt.Errorf("placement: %w", err)
	}
	return nil
}

// ── Verification ─────────────────────────────────────────────────────

type verifyDemoResult struct {
	result, reason, code string
}

func seedVerificationDemo(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	q := store.New(s.Tx)
	marcus, linh := s.Users["marcus@acme.io"], s.Users["linh@acme.io"]
	connected := s.designDay(13, 9, 40)
	read := s.designDay(0, 10, 12)
	zb, kb := ID(), ID()
	for _, v := range []struct {
		id                 uuid.UUID
		service, region    string
		price              int64
		bulk, forms        bool
		balance            int64
		usedAt             time.Time
		usedBy             *uuid.UUID
		usedWhat           string
		successAt          time.Time
		successWhat        string
		testedAt           time.Time
		connectedAt, updAt time.Time
	}{
		{zb, "zerobounce", "us", 8000, true, false, 36656, s.designDay(0, 9, 12), &linh.ID, "import preview", s.designDay(0, 9, 14), "import preview",
			read, connected, connected.Add(4 * time.Minute)},
		{kb, "kickbox", "", 10000, false, true, 9214, s.designDay(0, 10, 2), nil, "signup form", s.designDay(0, 10, 2), "signup form",
			s.designDay(0, 8, 0), connected.Add(20 * time.Minute), connected.Add(22 * time.Minute)},
	} {
		creds, err := verification.SealKey(s.Box, ws.ID, v.service, "demo-"+v.service+"-key-not-real", true)
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO verification_services (id, workspace_id, service, region, credentials_enc, price_micros, default_bulk, default_forms,
				balance, balance_at, last_used_at, last_used_by, last_used_what, last_success_at, last_success_what, tested_at, test_ok,
				connected_by, connected_at, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,true,$17,$18,$17,$19)`,
			v.id, ws.ID, v.service, v.region, creds, v.price, v.bulk, v.forms, v.balance, read, v.usedAt, v.usedBy, v.usedWhat, v.successAt, v.successWhat,
			v.testedAt, marcus.ID, v.connectedAt, v.updAt); err != nil {
			return err
		}
	}

	// Seven contacts the boards name: the Sunset tier's addresses on the
	// suppression list and among the risky ones, and Grace Okafor (valid).
	rename := []struct{ mailbox, email, first, last string }{
		{"yahoo", "d.moreau@yahoo.com", "Delphine", "Moreau"},
		{"yahoo", "k.lindgren@aol.com", "Karl", "Lindgren"},
		{"outlook", "r.tanaka@hotmail.com", "Rina", "Tanaka"},
		{"other", "m.keller@kellerdesign.ch", "Max", "Keller"},
		{"other", "contact@lindqvist-studio.se", "", ""},
		{"other", "z.wagner@guerrillamail.com", "Zoe", "Wagner"},
	}
	named := map[string]verifyDemoResult{
		"d.moreau@yahoo.com":          {"invalid", "invalid_address", "invalid / mailbox_not_found"},
		"k.lindgren@aol.com":          {"invalid", "invalid_address", "invalid / no_dns_entries"},
		"r.tanaka@hotmail.com":        {"invalid", "spam_trap", "spamtrap"},
		"m.keller@kellerdesign.ch":    {"risky", "catch_all", "catch-all"},
		"contact@lindqvist-studio.se": {"risky", "role", "do_not_mail / role_based"},
		"z.wagner@guerrillamail.com":  {"risky", "disposable", "do_not_mail / disposable"},
		"g.okafor@outlook.com":        {"valid", "", "valid"},
	}
	for i, r := range rename {
		var id uuid.UUID
		err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND tier = 'sunset' AND erased_at IS NULL AND status = 'active'
			AND mailbox_provider = $2 AND email <> 'g.okafor@outlook.com' ORDER BY id OFFSET $3 LIMIT 1`, ws.ID, r.mailbox, i).Scan(&id)
		if err != nil {
			return fmt.Errorf("a sunset contact at %s: %w", r.mailbox, err)
		}
		// The generated audience may already hold the address: keep its
		// first letters (the boards mask the rest).
		email := r.email
		for n := 2; ; n++ {
			var taken bool
			if err := s.Tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE workspace_id = $1 AND email = $2)`, ws.ID, email).Scan(&taken); err != nil {
				return err
			}
			if !taken {
				break
			}
			local, domain, _ := strings.Cut(r.email, "@")
			email = fmt.Sprintf("%s%d@%s", local, n, domain)
		}
		if email != r.email {
			named[email] = named[r.email]
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET email = $2, first_name = $3, last_name = $4 WHERE id = $1`, id, email, r.first, r.last); err != nil {
			return err
		}
	}

	// The Sunset tier, checked on Tue 14 Oct from 15:02 to 15:09.
	rows, err := s.Tx.Query(ctx, `SELECT id, email FROM contacts WHERE workspace_id = $1 AND tier = 'sunset' AND erased_at IS NULL ORDER BY id`, ws.ID)
	if err != nil {
		return err
	}
	type sunsetContact struct {
		id    uuid.UUID
		email string
	}
	var sunset []sunsetContact
	for rows.Next() {
		var c sunsetContact
		if err := rows.Scan(&c.id, &c.email); err != nil {
			rows.Close()
			return err
		}
		sunset = append(sunset, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// The rest in order: 283 invalid, 1 spam trap, 250 catch-all, 54
	// disposable, 11 role, 106 unknown, then valid.
	plan := []struct {
		n int
		r verifyDemoResult
	}{
		{170, verifyDemoResult{"invalid", "invalid_address", "invalid / mailbox_not_found"}},
		{113, verifyDemoResult{"invalid", "invalid_address", "invalid / no_dns_entries"}},
		{1, verifyDemoResult{"invalid", "spam_trap", "spamtrap"}},
		{250, verifyDemoResult{"risky", "catch_all", "catch-all"}},
		{54, verifyDemoResult{"risky", "disposable", "do_not_mail / disposable"}},
		{11, verifyDemoResult{"risky", "role", "do_not_mail / role_based"}},
		{106, verifyDemoResult{"unknown", "", "unknown"}},
	}
	results := make([]verifyDemoResult, len(sunset))
	k := 0
	for i, c := range sunset {
		if r, ok := named[c.email]; ok {
			results[i] = r
			continue
		}
		for k < len(plan) && plan[k].n == 0 {
			k++
		}
		if k < len(plan) {
			results[i] = plan[k].r
			plan[k].n--
			continue
		}
		results[i] = verifyDemoResult{"valid", "", "valid"}
	}
	counts := map[string]int{}
	reasons := map[string]int{}
	for _, r := range results {
		counts[r.result]++
		if r.reason != "" {
			reasons[r.reason]++
		}
	}
	jobID := ID()
	started, finished := s.designDay(1, 15, 2), s.designDay(1, 15, 9)
	rb, _ := json.Marshal(reasons)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO verification_jobs (id, workspace_id, service_id, service, kind, target, label, origin, status, skip_recent,
			total, checked, valid, invalid, risky, unknown, suppressed, reasons, price_micros, balance_before, balance_after, started_by, created_at, started_at, finished_at)
		VALUES ($1,$2,$3,'zerobounce','tier','{"kind":"tier","tiers":["sunset"]}','Sunset','list_health','finished',true,$4,$4,$5,$6,$7,$8,$6,$9,8000,52115,48500,$10,$11,$11,$12)`,
		jobID, ws.ID, zb, len(sunset), counts["valid"], counts["invalid"], counts["risky"], counts["unknown"], rb, marcus.ID, started, finished); err != nil {
		return err
	}
	items := make([][]any, len(sunset))
	ids := make([]uuid.UUID, len(sunset))
	res, rsn, codes, tags := make([]string, len(sunset)), make([]string, len(sunset)), make([]string, len(sunset)), make([]string, len(sunset))
	for i, c := range sunset {
		r := results[i]
		at := started.Add(time.Duration(i) * (finished.Sub(started) / time.Duration(len(sunset))))
		items[i] = []any{jobID, c.email, c.id, int32(i + 1), true, r.result, r.reason, r.code, at}
		ids[i], res[i], rsn[i], codes[i] = c.id, r.result, r.reason, r.code
		switch r.result {
		case "risky":
			tags[i] = "verify:" + strings.ReplaceAll(r.reason, "_", "-")
		case "unknown":
			tags[i] = "verify:unknown"
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"verification_job_items"}, []string{"job_id", "email", "contact_id", "position", "done", "result", "reason", "code", "checked_at"},
		pgx.CopyFromRows(items)); err != nil {
		return err
	}
	if err := q.VerifContactChecksUpsert(ctx, store.VerifContactChecksUpsertParams{WorkspaceID: ws.ID, Service: "zerobounce", JobID: &jobID, CheckedAt: finished,
		ContactIds: ids, Results: res, Reasons: rsn, Codes: codes}); err != nil {
		return err
	}
	var tagIDs []uuid.UUID
	var tagVals []string
	for i, t := range tags {
		if t != "" {
			tagIDs, tagVals = append(tagIDs, ids[i]), append(tagVals, t)
		}
	}
	if err := q.VerifSetTags(ctx, store.VerifSetTagsParams{WorkspaceID: ws.ID, ContactIds: tagIDs, Tags: tagVals}); err != nil {
		return err
	}
	// Invalid addresses on the suppression list for all mail, source ZeroBounce.
	var supIDs, evIDs []uuid.UUID
	var notes []string
	for i, c := range sunset {
		r := results[i]
		if r.result != "invalid" {
			continue
		}
		reason := "invalid"
		if r.reason == "spam_trap" {
			reason = "spam_trap"
		}
		code := "ZeroBounce: " + r.code
		_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{ID: ID(), WorkspaceID: ws.ID, Email: c.email, Reason: reason, Source: verification.SuppressionSource("zerobounce"),
			OriginalCode: code, Detail: "Verification · Sunset", CreatedBy: &marcus.ID, Scope: "all", Context: "Verification · Sunset tier", CreatedAt: finished})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		supIDs, evIDs, notes = append(supIDs, c.id), append(evIDs, ID()), append(notes, code)
	}
	if _, err := q.VerifSuppressContacts(ctx, store.VerifSuppressContactsParams{WorkspaceID: ws.ID, Ids: supIDs}); err != nil {
		return err
	}
	if err := q.VerifConsentEvents(ctx, store.VerifConsentEventsParams{Ids: evIDs, ContactIds: supIDs, Notes: notes, Actor: &marcus.ID, At: finished}); err != nil {
		return err
	}

	// The Mailchimp import's new addresses, this morning at 09:12.
	importAt := s.designDay(0, 9, 12)
	importReasons, _ := json.Marshal(map[string]int{"invalid_address": 401, "spam_trap": 2, "catch_all": 352, "disposable": 38, "role": 27})
	var importID *uuid.UUID
	var imp uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM imports WHERE workspace_id = $1 AND source = 'mailchimp' AND status = 'draft' ORDER BY created_at DESC LIMIT 1`, ws.ID).Scan(&imp); err == nil {
		importID = &imp
	}
	importJob := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO verification_jobs (id, workspace_id, service_id, service, kind, target, label, origin, import_id, status, skip_recent,
			total, checked, valid, invalid, risky, unknown, suppressed, reasons, price_micros, balance_before, balance_after, started_by, created_at, started_at, finished_at)
		VALUES ($1,$2,$3,'zerobounce','import',$4,$5,'import_preview',$6,'finished',false,11844,11844,10912,403,417,112,0,$7,8000,48500,36656,$8,$9,$9,$10)`,
		importJob, ws.ID, zb, mustJSONB(map[string]any{"kind": "import", "importId": importIDString(importID)}), "Mailchimp “Acme Customers”", importID, importReasons,
		linh.ID, importAt, importAt.Add(2*time.Minute+14*time.Second)); err != nil {
		return err
	}
	if importID != nil {
		if err := seedImportAnswers(ctx, s, ws.ID, *importID, importJob, importAt.Add(2*time.Minute+14*time.Second)); err != nil {
			return err
		}
	}

	// Kickbox checks three forms as people sign up, since Thu 2 Oct.
	forms := map[string]string{"subscribe_page": "subscribe_page"}
	for _, name := range []string{"Newsletter signup", "Footer mini form"} {
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM forms WHERE workspace_id = $1 AND name = $2`, ws.ID, name).Scan(&id); err != nil {
			return fmt.Errorf("form %s: %w", name, err)
		}
		forms[name] = id.String()
	}
	for _, key := range []string{forms["Newsletter signup"], forms["Footer mini form"], forms["subscribe_page"]} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO verification_form_checks (workspace_id, form_key, enabled, updated_by, updated_at) VALUES ($1,$2,true,$3,$4)`,
			ws.ID, key, marcus.ID, connected.Add(25*time.Minute)); err != nil {
			return err
		}
	}
	// Per form: checks, valid, invalid, risky, unknown, no answer over the 14 days.
	totals := []struct {
		key                                             string
		checks, valid, invalid, risky, unknown, noReply int
	}{
		{forms["Newsletter signup"], 338, 315, 11, 9, 1, 2},
		{forms["Footer mini form"], 126, 118, 4, 3, 1, 0},
		{forms["subscribe_page"], 48, 45, 2, 1, 0, 0},
	}
	const days = 14
	for _, t := range totals {
		left := []int{t.valid, t.invalid, t.risky, t.unknown, t.noReply}
		for d := 0; d < days; d++ {
			day := s.designDay(days-1-d, 0, 0)
			// Spread each total over the days, the remainder on the last ones.
			share := func(total int) int {
				base := total / days
				if d >= days-total%days {
					base++
				}
				return base
			}
			v, inv, rk, un, nr := share(left[0]), share(left[1]), share(left[2]), share(left[3]), share(left[4])
			n := v + inv + rk + un + nr
			if n == 0 {
				continue
			}
			if _, err := s.Tx.Exec(ctx, `INSERT INTO verification_realtime_daily (workspace_id, service, form_key, day, checks, valid, invalid, risky, unknown, no_answer)
				VALUES ($1,'kickbox',$2,$3::date,$4,$5,$6,$7,$8,$9)`, ws.ID, t.key, day.Format("2006-01-02"), n, v, inv, rk, un, nr); err != nil {
				return err
			}
		}
	}
	return nil
}

func importIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// ── Placement tests ──────────────────────────────────────────────────

type placementDemoSeed struct {
	provider, email, auth string
	id                    uuid.UUID
}

// placementDemoCell is one copy: where it landed and when.
type placementDemoCell struct {
	kind      string // inbox, tab, spam, na (not arrived), late (found late, inbox), nc (not checked)
	tab       string
	secs      int
	deferrals int
}

func pdIn(tab string, secs int) placementDemoCell {
	return placementDemoCell{kind: "inbox", tab: tab, secs: secs}
}
func pdTab(secs int) placementDemoCell {
	return placementDemoCell{kind: "tab", tab: "Promotions", secs: secs}
}
func pdSpam(secs int) placementDemoCell { return placementDemoCell{kind: "spam", secs: secs} }
func pdNA(deferrals int) placementDemoCell {
	return placementDemoCell{kind: "na", deferrals: deferrals}
}

type placementDemoProvider struct {
	key, short, host, ip, selector, spf string
	warmDay                             int
}

type placementDemoTest struct {
	name, kind, content, note, key, personalization, through, stream, origin, variant, language, subject string
	version                                                                                              int
	contentAt                                                                                            *time.Time
	startedBy                                                                                            string
	at                                                                                                   time.Time
	providers                                                                                            []placementDemoProvider
	cells                                                                                                [][]placementDemoCell // per provider, per seed
	fromName, fromEmail                                                                                  string
	// tokens fixes some copies' tokens (provider index, seed index → token).
	tokens map[[2]int]string
	// pmids fixes provider message ids.
	pmids map[[2]int]string
}

func seedPlacementDemo(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	added := s.designDay(13, 9, 40)
	seeds := []placementDemoSeed{
		{seedbox.Gmail, "acme.seed01@gmail.com", seedbox.AuthPassword, ID()},
		{seedbox.Gmail, "acme.seed02@gmail.com", seedbox.AuthPassword, ID()},
		{seedbox.GoogleWorkspace, "seed@gws.acmeseed.io", seedbox.AuthOAuth, ID()},
		{seedbox.Outlook, "acme.seed@outlook.com", seedbox.AuthOAuth, ID()},
		{seedbox.Microsoft365, "seed@m365.acmeseed.io", seedbox.AuthOAuth, ID()},
		{seedbox.Yahoo, "acme.seed@yahoo.com", seedbox.AuthPassword, ID()},
		{seedbox.AOL, "acme.seed@aol.com", seedbox.AuthPassword, ID()},
		{seedbox.ICloud, "acme.seed@icloud.com", seedbox.AuthPassword, ID()},
		{seedbox.GMX, "acme.seed@gmx.de", seedbox.AuthPassword, ID()},
		{seedbox.WebDE, "acme.seed@web.de", seedbox.AuthPassword, ID()},
	}
	icloudFailed := s.designDay(1, 21, 40)
	for i, sd := range seeds {
		p, _ := seedbox.ProviderOf(sd.provider)
		creds, err := placement.SealCredentials(s.Box, sd.id, "", true)
		if err != nil {
			return err
		}
		folders, _ := json.Marshal(placement.DemoFolders(sd.provider, sd.auth))
		status, statusAt, detail := "connected", added, ""
		if sd.provider == seedbox.ICloud {
			status, statusAt, detail = "failed", icloudFailed, "Apple rejected the app-specific password (AUTHENTICATIONFAILED)"
		}
		username := ""
		if sd.auth == seedbox.AuthPassword {
			username = sd.email
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO placement_seeds (id, workspace_id, email, provider, auth, imap_host, imap_port, username, credentials_enc,
				status, status_at, status_detail, folders, checked_at, added_by, added_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			sd.id, ws.ID, sd.email, sd.provider, sd.auth, p.Host, p.Port, username, creds, status, statusAt, detail, folders, added, linh.ID,
			added.Add(time.Duration(i)*time.Minute)); err != nil {
			return err
		}
	}

	ses := placementDemoProvider{key: "ses", short: "SES", host: "a8-25.smtp-out.amazonses.com", ip: "54.240.8.25", selector: "7kq2n3x", spf: "bounce.news.acme.io"}
	postal := func(day int) placementDemoProvider {
		return placementDemoProvider{key: "postal", short: "Postal", host: "mta.acme.io", ip: "203.0.113.25", selector: "postal-2026", spf: "rp.news.acme.io", warmDay: day}
	}
	postmark := placementDemoProvider{key: "postmark", short: "Postmark", host: "mta-out.pstmrk.it", ip: "50.31.156.6", selector: "pm2026", spf: "pm-bounces.notify.acme.io"}
	tue14, tue7, tue30, thu2 := s.designDay(1, 9, 0), s.designDay(8, 9, 0), s.designDay(15, 9, 0), s.designDay(13, 15, 0)
	nc := placementDemoCell{kind: "nc"}
	tests := []placementDemoTest{
		{name: "Feature flags preview · SES", kind: "campaign", content: "Feature flags preview", note: "scheduled", contentAt: &thu2, through: "provider",
			stream: "broadcast", origin: "placement", subject: "Feature flags are here: try them before everyone else", startedBy: "sofia@acme.io",
			at: s.designDay(13, 14, 40), providers: []placementDemoProvider{ses}, fromName: "Acme Beta", fromEmail: "beta@news.acme.io",
			cells: [][]placementDemoCell{{pdTab(10), pdTab(12), pdIn("Primary", 9), pdIn("Focused", 13), pdIn("Other", 11), pdIn("", 19), pdIn("", 22), pdIn("", 16), pdIn("", 14), pdIn("", 15)}}},
		{name: "Weekly Digest #140 · Postal", kind: "campaign", content: "Weekly Digest #140", note: "sent", contentAt: &tue30, through: "provider",
			stream: "broadcast", origin: "warmup", subject: "Weekly Digest #140: the September roundup", startedBy: "linh@acme.io",
			at: s.designDay(10, 10, 15), providers: []placementDemoProvider{postal(2)}, fromName: "Acme Newsletter", fromEmail: "digest@news.acme.io",
			cells: [][]placementDemoCell{{pdTab(41), pdTab(47), pdSpam(36), pdSpam(58), pdIn("Other", 63), pdSpam(140), pdSpam(121), pdNA(0), pdIn("", 62),
				{kind: "late", secs: 34*60 + 5, deferrals: 2}}}},
		{name: "Weekly Digest #141 · Postal", kind: "campaign", content: "Weekly Digest #141", note: "sent", contentAt: &tue7, through: "provider",
			stream: "broadcast", origin: "warmup", subject: "Weekly Digest #141: what shipped in the first week of October", startedBy: "linh@acme.io",
			at: s.designDay(7, 11, 30), providers: []placementDemoProvider{postal(5)}, fromName: "Acme Newsletter", fromEmail: "digest@news.acme.io",
			cells: [][]placementDemoCell{{pdTab(33), pdTab(40), pdIn("Primary", 22), pdSpam(51), pdIn("Other", 47), pdSpam(95), pdSpam(88), pdIn("", 39), pdIn("", 44), pdNA(2)}}},
		{name: "Weekly Digest #142 · SES vs Postal", kind: "campaign", content: "Weekly Digest #142", note: "scheduled", contentAt: &tue14, through: "compare",
			stream: "broadcast", origin: "review", variant: "A", language: "en", subject: "Three launches, one deprecation, and the roadmap for Q4",
			startedBy: "sofia@acme.io", at: s.designDay(3, 16, 20), providers: []placementDemoProvider{ses, postal(9)}, fromName: "Acme Newsletter", fromEmail: "digest@news.acme.io",
			cells: [][]placementDemoCell{
				{pdTab(9), pdTab(11), pdIn("Primary", 8), pdIn("Focused", 14), pdIn("Other", 12), pdIn("", 21), pdIn("", 24), pdIn("", 17), pdIn("", 15), pdIn("", 16)},
				{pdTab(26), pdTab(31), pdIn("Primary", 19), pdSpam(44), pdIn("Other", 52), pdSpam(130), pdIn("", 108), pdIn("", 41), pdIn("", 37), pdNA(3)},
			},
			tokens: map[[2]int]string{{1, 3}: "pt-7f3a91c2"}, pmids: map[[2]int]string{{1, 3}: "2481906", {1, 9}: "2481913"}},
		{name: "Flash sale: 48 hours only · SES", kind: "campaign", content: "Flash sale: 48 hours only", note: "draft", through: "provider",
			stream: "broadcast", origin: "placement", subject: "FLASH SALE!!! 48 HOURS ONLY", startedBy: "linh@acme.io",
			at: s.designDay(2, 15, 10), providers: []placementDemoProvider{ses}, fromName: "Acme", fromEmail: "hello@news.acme.io",
			cells: [][]placementDemoCell{{pdSpam(12), pdTab(9), pdSpam(10), pdSpam(15), pdSpam(13), pdSpam(18), pdSpam(16), pdSpam(14), pdSpam(11), pdSpam(13)}}},
		{name: "Password reset · Postmark", kind: "transactional", content: "Password reset", note: "transactional", key: "password-reset", version: 3,
			personalization: "New user", through: "route", stream: "transactional", origin: "template", subject: "Reset your Acme password",
			startedBy: "marcus@acme.io", at: s.designDay(0, 10, 5), providers: []placementDemoProvider{postmark}, fromName: "Acme Cloud", fromEmail: "no-reply@notify.acme.io",
			cells: [][]placementDemoCell{{pdIn("Primary", 5), pdIn("Primary", 6), pdIn("Primary", 4), pdIn("Focused", 7), pdIn("Focused", 9), pdIn("", 8), pdIn("", 4), nc, pdIn("", 3), pdIn("", 5)}}},
	}
	type use struct {
		test  uuid.UUID
		at    time.Time
		label string
	}
	lastUsed := map[uuid.UUID]use{}
	for ti, t := range tests {
		id, err := seedPlacementTest(ctx, s, t, ti, seeds)
		if err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
		label := t.key + " test"
		if t.kind == "campaign" {
			n, _, _ := strings.Cut(t.content, ": ")
			label = n + " test"
		}
		for si, sd := range seeds {
			if t.cells[0][si].kind != "nc" {
				lastUsed[sd.id] = use{id, t.at, label}
			}
		}
	}
	for _, sd := range seeds {
		if u, ok := lastUsed[sd.id]; ok {
			if _, err := s.Tx.Exec(ctx, `UPDATE placement_seeds SET last_used_at = $2, last_test_id = $3, last_test_label = $4 WHERE id = $1`,
				sd.id, u.at, u.test, u.label); err != nil {
				return err
			}
		}
	}
	return nil
}

// placementDemoToken is a copy's sample test token, fixed by its place.
func placementDemoToken(test, provider, seed int) string {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "placement-demo:%d:%d:%d", test, provider, seed)
	return fmt.Sprintf("pt-%08x", h.Sum32())
}

var placementDemoAccepted = map[string][2]string{
	seedbox.Gmail: {"250 2.0.0", "OK"}, seedbox.GoogleWorkspace: {"250 2.0.0", "OK"},
	seedbox.Outlook: {"250 2.6.0", "Queued mail for delivery"}, seedbox.Microsoft365: {"250 2.6.0", "Queued mail for delivery"},
	seedbox.Yahoo: {"250", "ok dirdel"}, seedbox.AOL: {"250", "ok dirdel"}, seedbox.ICloud: {"250 2.0.0", "Ok: queued"},
	seedbox.GMX: {"250", "Requested mail action okay, completed"}, seedbox.WebDE: {"250", "Requested mail action okay, completed"},
}

var placementDemoDeferred = map[string]string{
	seedbox.WebDE: "Temporary rejection, try again later (mx-ha03.web.de)", seedbox.GMX: "Temporary rejection, try again later (mx-ha02.gmx.net)",
	seedbox.ICloud: "Service temporarily unavailable (mx6.mail.icloud.com)",
}

func placementMailbox(provider string) string {
	switch {
	case seedbox.IsGoogle(provider):
		return "gmail"
	case seedbox.IsMicrosoft(provider):
		return "outlook"
	case provider == seedbox.Yahoo || provider == seedbox.AOL:
		return "yahoo"
	case provider == seedbox.ICloud:
		return "icloud"
	}
	return "other"
}

// seedPlacementTest writes one test, its copies, the copies' messages and
// the providers' events.
func seedPlacementTest(ctx context.Context, s *Seeder, t placementDemoTest, ti int, seeds []placementDemoSeed) (uuid.UUID, error) {
	ws := s.WS()
	id := ID()
	starter := s.Users[t.startedBy]
	var contentID *uuid.UUID
	switch t.kind {
	case "campaign":
		c, ok := s.Campaigns[t.content]
		if !ok {
			return id, fmt.Errorf("no campaign %s", t.content)
		}
		contentID = &c
	case "transactional":
		var c uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM transactional_templates WHERE workspace_id = $1 AND key = $2`, ws.ID, t.key).Scan(&c); err != nil {
			return id, fmt.Errorf("transactional template %s: %w", t.key, err)
		}
		contentID = &c
	}
	pids := make([]uuid.UUID, len(t.providers))
	snap := map[string]map[string]any{}
	for i, p := range t.providers {
		pid := s.Providers[p.key]
		pids[i] = pid
		var name, adapter, label string
		var config json.RawMessage
		if err := s.Tx.QueryRow(ctx, `SELECT name, adapter, label, config FROM providers WHERE id = $1`, pid).Scan(&name, &adapter, &label, &config); err != nil {
			return id, err
		}
		var cfg struct {
			Region    string `json:"region"`
			ServerURL string `json:"server_url"`
		}
		_ = json.Unmarshal(config, &cfg)
		detail := label
		switch {
		case cfg.Region != "":
			detail = cfg.Region
		case cfg.ServerURL != "":
			detail = strings.TrimPrefix(strings.TrimPrefix(cfg.ServerURL, "https://"), "http://")
		}
		ps := map[string]any{"name": name, "short": p.short, "adapter": adapter, "detail": detail, "status": "healthy", "cost": 0.0}
		if adapter == "ses" {
			ps["cost"] = 0.0001
		}
		if p.warmDay > 0 {
			ps["status"], ps["warmDay"], ps["warmDays"], ps["ip"] = "warming_up", p.warmDay, 21, p.ip
		}
		snap[pid.String()] = ps
	}
	type copyRow struct {
		id                       uuid.UUID
		seed                     placementDemoSeed
		pi, si, round            int
		cell                     placementDemoCell
		sendAt, handed, received time.Time
		token, pmid              string
	}
	var copies []copyRow
	for si, sd := range seeds {
		for r := range t.providers {
			pi := r
			round := 1
			if t.through == "compare" {
				pi = (si + r) % len(t.providers)
				round = r + 1
			}
			cell := t.cells[pi][si]
			sendAt := t.at.Add(time.Duration(round-1) * placement.CompareGap)
			c := copyRow{id: ID(), seed: sd, pi: pi, si: si, round: round, cell: cell, sendAt: sendAt, handed: sendAt}
			c.token = placementDemoToken(ti, pi, si)
			if tok, ok := t.tokens[[2]int{pi, si}]; ok {
				c.token = tok
			}
			c.pmid = fmt.Sprintf("%s-%08x", t.providers[pi].key, fnv32(c.token))
			if t.providers[pi].key == "postal" {
				c.pmid = fmt.Sprintf("%d", 2481000+ti*40+pi*20+si)
			}
			if v, ok := t.pmids[[2]int{pi, si}]; ok {
				c.pmid = v
			}
			c.received = c.handed.Add(time.Duration(cell.secs) * time.Second)
			copies = append(copies, c)
		}
	}
	var last time.Time
	allFound := true
	for _, c := range copies {
		if c.cell.kind == "nc" {
			continue
		}
		if c.handed.After(last) {
			last = c.handed
		}
		if c.cell.kind == "na" || c.cell.kind == "late" {
			allFound = false
		}
	}
	wait := last.Add(placement.WaitFor)
	finished, lateUntil := wait, last.Add(placement.LateWithin)
	if allFound {
		finished = t.at
		for _, c := range copies {
			if c.cell.kind != "nc" && c.received.After(finished) {
				finished = c.received
			}
		}
		lateUntil = finished
	}
	deleteAt := finished.Add(24 * time.Hour)
	var deletedAt *time.Time
	if deleteAt.Before(s.Now) {
		deletedAt = &deleteAt
	}
	html := `<p>Hi {{ first_name | default: "there" }},</p><p>` + t.subject + `</p><p><a href="https://acme.io/blog">Read more</a></p>`
	if t.kind == "transactional" {
		html = `<p>Hi Sam,</p><p>Reset your password within 30 minutes: <a href="https://news.acme.io/">Reset password</a></p>`
	}
	links, _ := json.Marshal([]map[string]any{{"Position": 1, "URL": "https://acme.io/blog", "Label": "Read more"}})
	pb, _ := json.Marshal(snap)
	var deletedVal any
	if deletedAt != nil {
		deletedVal = *deletedAt
	}
	var deleteAtVal any = deleteAt
	if deletedAt != nil {
		deleteAtVal = nil
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO placement_tests (id, workspace_id, name, content_kind, content_id, content_name, content_note, content_key, content_version,
			content_at, variant, language, personalization, through, stream, provider_ids, from_name, from_email, origin, status, subject, html, text_body,
			links, providers, started_by, started_at, last_sent_at, wait_until, late_until, finished_at, delete_after, delete_at, copies_deleted_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,'finished',$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,'24h',$31,$32)`,
		id, ws.ID, t.name, t.kind, contentID, t.content, t.note, t.key, t.version, t.contentAt, t.variant, t.language, t.personalization, t.through, t.stream,
		pids, t.fromName, t.fromEmail, t.origin, t.subject, html, t.subject, links, pb, starter.ID, t.at, last, wait, lateUntil, finished, deleteAtVal, deletedVal); err != nil {
		return id, err
	}
	for _, c := range copies {
		p := t.providers[c.pi]
		pid := pids[c.pi]
		sp, _ := seedbox.ProviderOf(c.seed.provider)
		pos := int32(c.si + 1)
		if c.cell.kind == "nc" {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO placement_copies (id, test_id, workspace_id, seed_id, seed_email, seed_provider, position, provider_id, provider_name,
					token, round, send_at, status, not_checked) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'not_checked','signin_failed')`,
				c.id, id, ws.ID, c.seed.id, c.seed.email, c.seed.provider, pos, pid, snap[pid.String()]["name"], c.token, c.round, c.sendAt); err != nil {
				return id, err
			}
			continue
		}
		status, folder, folderName, tab := "found", "", "", ""
		var receivedAt, foundAt *time.Time
		late := false
		switch c.cell.kind {
		case "inbox", "late":
			folder, folderName, tab = seedbox.RoleInbox, sp.InboxName, c.cell.tab
			late = c.cell.kind == "late"
		case "tab":
			folder, folderName, tab = seedbox.RoleTab, c.cell.tab, c.cell.tab
		case "spam":
			folder, folderName = seedbox.RoleSpam, sp.SpamName
		case "na":
			status = "not_arrived"
		}
		auth, hop := json.RawMessage(`{}`), json.RawMessage(`{}`)
		headers := ""
		ref := json.RawMessage(`{}`)
		if status == "found" {
			rcv := c.received
			found := rcv.Add(20 * time.Second)
			receivedAt, foundAt = &rcv, &found
			headers = placement.DemoHeaders(placement.DemoMail{SeedProvider: c.seed.provider, Host: p.host, IP: p.ip, From: t.fromEmail,
				MessageID: fmt.Sprintf("%s.%02d.%s", c.token, c.si+1, p.key), Token: c.token, Selector: p.selector, SPFDomain: p.spf, At: rcv})
			auth, hop = placement.DemoAuth(headers), placement.DemoHop(headers)
			path := folderName
			if folder == seedbox.RoleSpam && seedbox.IsGoogle(c.seed.provider) {
				path = "[Gmail]/Spam"
			}
			ref, _ = json.Marshal(seedbox.Ref{Folder: path})
		}
		var deleted any
		if deletedAt != nil && status == "found" {
			deleted = *deletedAt
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO placement_copies (id, test_id, workspace_id, seed_id, seed_email, seed_provider, position, provider_id, provider_name,
				token, round, send_at, status, folder, folder_name, tab, late, handed_at, received_at, found_at, provider_message_id, auth, hop, headers, ref, deleted_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
			c.id, id, ws.ID, c.seed.id, c.seed.email, c.seed.provider, pos, pid, snap[pid.String()]["name"], c.token, c.round, c.sendAt, status, folder, folderName,
			tab, late, c.handed, receivedAt, foundAt, c.pmid, auth, hop, headers, ref, deleted); err != nil {
			return id, err
		}
		// The message the provider sent, and its events.
		mstatus := "delivered"
		var delivered *time.Time
		accepted := c.handed.Add(2 * time.Second)
		if status == "found" {
			delivered = &accepted
			if c.cell.kind == "late" {
				lateAt := c.received.Add(-2 * time.Second)
				delivered = &lateAt
			}
		} else if c.cell.deferrals > 0 {
			mstatus = "deferred"
		} else {
			mstatus = "sent"
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, email, provider_id, provider_message_id, mailbox_provider, status, queued_at, sent_at, delivered_at, kind)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,$10,'placement')`,
			c.id, ws.ID, t.stream, c.seed.email, pid, c.pmid, placementMailbox(c.seed.provider), mstatus, c.handed, delivered); err != nil {
			return id, err
		}
		event := func(typ string, at time.Time, code, diag string) error {
			_, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, provider_id, type, occurred_at, mailbox_provider, original_type, original_code,
					diagnostic, recipient, provider_message_id, source, received_at, dedupe_key)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'Placement test copy',$6,$13)`,
				ID(), ws.ID, c.id, pid, typ, at, placementMailbox(c.seed.provider), typ, code, diag, c.seed.email, c.pmid, c.id.String()+":"+typ+":"+at.Format(time.RFC3339))
			return err
		}
		deferAt := []time.Duration{3 * time.Second, 6*time.Minute + 5*time.Second, 12*time.Minute + 11*time.Second}
		for d := 0; d < c.cell.deferrals && d < len(deferAt); d++ {
			if err := event("deferred", c.handed.Add(deferAt[d]), "421", placementDemoDeferred[c.seed.provider]); err != nil {
				return id, err
			}
		}
		if delivered != nil {
			acc := placementDemoAccepted[c.seed.provider]
			if err := event("delivered", *delivered, acc[0], acc[1]); err != nil {
				return id, err
			}
		}
	}
	return id, nil
}

func fnv32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// seedImportAnswers gives the Mailchimp draft's check its answers, so the
// preview shows ZeroBounce's share (VerificationStates, "After checking"):
// ReputeCast's 27 role and 19 disposable rows, which ZeroBounce agrees
// with, and 11,798 of the new addresses: 401 invalid, 2 spam traps, 352
// catch-all, 19 more disposable, 112 unknown and 10,912 valid.
func seedImportAnswers(ctx context.Context, s *Seeder, ws, importID, job uuid.UUID, at time.Time) error {
	rows, err := s.Tx.Query(ctx, `SELECT r.email, r.status, coalesce((r.engagement->>'rating')::int, 0) FROM import_rows r
		WHERE r.import_id = $1 AND NOT EXISTS (SELECT 1 FROM contacts c WHERE c.workspace_id = $2 AND c.email = lower(r.email))
		ORDER BY r.row_no`, importID, ws)
	if err != nil {
		return err
	}
	var role, disposable, fresh []string
	for rows.Next() {
		var email, status string
		var rating int
		if err := rows.Scan(&email, &status, &rating); err != nil {
			rows.Close()
			return err
		}
		e, problem := emailaddr.Check(email)
		if problem != "" || status != "subscribed" {
			continue
		}
		domain := emailaddr.Domain(e)
		_, typo := emailaddr.TypoFix(domain)
		switch {
		case emailaddr.TrapLike(e) || typo || strings.HasSuffix(domain, ".invalid"):
		case emailaddr.IsDisposable(domain):
			disposable = append(disposable, e)
		case emailaddr.IsRole(e):
			role = append(role, e)
		case rating >= 2:
			fresh = append(fresh, e)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// The new addresses in a fixed, spread-out order.
	hash := func(v string) uint32 {
		h := fnv.New32a()
		_, _ = h.Write([]byte(v))
		return h.Sum32()
	}
	slices.SortFunc(fresh, func(a, b string) int { return cmp.Compare(hash(a), hash(b)) })
	if len(fresh) > 11798 {
		fresh = fresh[:11798]
	}
	type answer struct{ result, reason, code string }
	var buf [][]any
	add := func(email string, a answer) {
		buf = append(buf, []any{job, email, int32(len(buf) + 1), true, a.result, a.reason, a.code, at})
	}
	for _, e := range role {
		add(e, answer{"risky", "role", "do_not_mail / role_based"})
	}
	for _, e := range disposable {
		add(e, answer{"risky", "disposable", "do_not_mail / disposable"})
	}
	plan := []struct {
		n int
		a answer
	}{
		{401, answer{"invalid", "invalid_address", "invalid / mailbox_not_found"}},
		{2, answer{"invalid", "spam_trap", "spamtrap"}},
		{352, answer{"risky", "catch_all", "catch-all"}},
		{19, answer{"risky", "disposable", "do_not_mail / disposable"}},
		{112, answer{"unknown", "", "unknown / mail_server_did_not_respond"}},
	}
	i := 0
	for _, p := range plan {
		for k := 0; k < p.n && i < len(fresh); k++ {
			add(fresh[i], p.a)
			i++
		}
	}
	for ; i < len(fresh); i++ {
		add(fresh[i], answer{"valid", "", "valid"})
	}
	_, err = s.Tx.CopyFrom(ctx, pgx.Identifier{"verification_job_items"}, []string{"job_id", "email", "position", "done", "result", "reason", "code", "checked_at"},
		pgx.CopyFromRows(buf))
	return err
}
