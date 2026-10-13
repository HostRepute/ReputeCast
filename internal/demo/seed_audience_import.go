package demo

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// audImportOptions is imports.options with the wizard's defaults.
func audImportOptions(lists []map[string]string) map[string]any {
	return map[string]any{
		"lists": lists, "importTags": true, "importEngagement": true, "pullSuppressions": true, "recordHistory": true,
		"evidence": "trust", "onConflict": "fill", "skipRows": []int{},
		"risky": map[string]string{"role": "exclude_broadcasts", "disposable": "skip", "typo": "fix", "stale": "pending", "trap": "skip"},
	}
}

// audSeedImports creates the import history (an event-signup audience from
// Mailchimp and the Brevo migration, both on 18 Sep) and the Mailchimp
// draft that waits in the importer.
func audSeedImports(ctx context.Context, s *Seeder, a *audSeed) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	digest, updates, beta := s.Lists["Weekly Digest"], s.Lists["Product Updates"], s.Lists["Beta Program"]
	brevoAt := s.designDay(24, 16, 40)
	mcAt := s.designDay(24, 14, 5)
	insert := func(id any, source, status, step, file string, conn, opts, stats, progress map[string]any, report, skipped []byte, created, started, finished time.Time) error {
		_, err := s.Tx.Exec(ctx, `INSERT INTO imports (id, workspace_id, source, status, step, file_name, connection, options, stats, progress, report_csv, skipped_csv,
			created_by, created_at, staged_at, started_at, finished_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17)`,
			id, ws.ID, source, status, step, file, mustJSONB(conn), mustJSONB(opts), mustJSONB(stats), mustJSONB(progress), report, skipped,
			linh.ID, created, started.Add(-time.Minute), started, finished)
		return err
	}

	// 2,118 people who signed up at an event and never confirmed.
	mcStats := map[string]any{"rows": 2118, "imported": 0, "merged": 0, "suppressed": 0, "suppressedBy": map[string]int{}, "pending": 2118, "skipped": 0,
		"reasons": map[string]int{}, "lists": []map[string]any{{"name": "Weekly Digest", "contacts": 31104}}, "durationMs": 21400,
		"pushedTo": []string{}, "confirmsSent": 0}
	mcConn := map[string]any{"account": "Acme Inc", "dataCenter": "us21", "checkedAt": mcAt.Add(-9 * time.Minute), "fields": []any{},
		"lists": []map[string]any{{"id": "8e1d07", "name": "Conference 2025 signups", "subscribed": 0, "unsubscribed": 0, "cleaned": 0, "pending": 2118,
			"transactional": 0, "complained": 0, "fields": 4, "tags": 2, "tagExamples": []string{"conference", "booth"}, "doubleOptIn": true, "since": "2025"}}}
	if err := insert(a.mailchimpOld, "mailchimp", "done", "done", "", mcConn,
		audImportOptions([]map[string]string{{"sourceId": "8e1d07", "target": digest.String()}}), mcStats,
		map[string]any{"total": 2118, "processed": 2118, "batch": 5, "batches": 5, "batchSize": 500, "pending": 2118},
		nil, nil, mcAt.Add(-7*time.Minute), mcAt.Add(-21*time.Second), mcAt); err != nil {
		return err
	}

	// The migration from Brevo: 47,365 rows, 46,118 contacts.
	brevoStats := map[string]any{"rows": 47365, "imported": 39402, "merged": 0, "suppressed": 1033,
		"suppressedBy": map[string]int{"unsubscribe": 180, "hard_bounce": 812, "complaint": 41}, "pending": 6716, "skipped": 214,
		"reasons":    map[string]int{"invalid": 131, "disposable": 52, "trap": 31, "typo_fixed": 18, "role": 40},
		"lists":      []map[string]any{{"name": "Weekly Digest", "contacts": 31176}, {"name": "Product Updates", "contacts": 13719}, {"name": "Beta Program", "contacts": 2034}},
		"durationMs": 252000, "pushedTo": []string{"Amazon SES", "Postmark"}, "confirmsSent": 6716}
	brevoConn := map[string]any{"account": "Acme Newsletter (Brevo)", "checkedAt": brevoAt.Add(-11 * time.Minute), "fields": []any{},
		"lists": []map[string]any{
			{"id": "2", "name": "Newsletter", "subscribed": 30940, "unsubscribed": 141, "cleaned": 612, "pending": 0, "transactional": 0, "complained": 33, "fields": 11, "tags": 0, "tagExamples": []string{}, "doubleOptIn": true, "since": "2019"},
			{"id": "7", "name": "Product news", "subscribed": 13502, "unsubscribed": 32, "cleaned": 171, "pending": 0, "transactional": 0, "complained": 7, "fields": 11, "tags": 0, "tagExamples": []string{}, "doubleOptIn": true, "since": "2021"},
			{"id": "9", "name": "Beta testers", "subscribed": 1890, "unsubscribed": 7, "cleaned": 29, "pending": 0, "transactional": 0, "complained": 1, "fields": 11, "tags": 0, "tagExamples": []string{}, "doubleOptIn": true, "since": "2026"},
		}}
	if err := insert(a.brevoImport, "brevo", "done", "done", "old-list-···@brevo-export.csv", brevoConn,
		audImportOptions([]map[string]string{{"sourceId": "2", "target": digest.String()}, {"sourceId": "7", "target": updates.String()}, {"sourceId": "9", "target": beta.String()}}),
		brevoStats, map[string]any{"total": 47365, "processed": 47365, "batch": 95, "batches": 95, "batchSize": 500, "imported": 39402, "suppressed": 1033, "pending": 6716, "skipped": 214},
		nil, nil, brevoAt.Add(-9*time.Minute), brevoAt.Add(-252*time.Second), brevoAt); err != nil {
		return err
	}
	logs := []struct {
		at      time.Duration
		kind, m string
	}{
		{-252 * time.Second, "start", "Import started · 47,365 rows · 3 lists from Brevo → Weekly Digest, Product Updates, Beta Program · consent: trust double opt-in"},
		{-130 * time.Second, "retry", "Brevo returned 429 Too Many Requests on batch 48; waited 2 s and succeeded"},
		{-4 * time.Second, "sync", "Pushed 1,033 new suppressions to Amazon SES and Postmark"},
		{0, "done", "Import finished · 46,118 contacts · 1,033 suppressions"},
	}
	for _, l := range logs {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO import_log (id, import_id, occurred_at, kind, message) VALUES ($1,$2,$3,$4,$5)`, ID(), a.brevoImport, brevoAt.Add(l.at), l.kind, l.m); err != nil {
			return err
		}
	}
	return nil
}

// audImportReports writes the downloadable reports of the two finished
// imports, in the importer's format: every row with its outcome, and the
// skipped rows on their own.
func audImportReports(ctx context.Context, s *Seeder, a *audSeed) error {
	var pending, brevo []string
	for _, c := range a.contacts {
		switch {
		case c.pending:
			pending = append(pending, c.email)
		case c.importID != nil && *c.importID == a.brevoImport:
			brevo = append(brevo, c.email)
		}
	}
	var blocked []audSupp
	for _, x := range a.supps {
		if x.brevo {
			blocked = append(blocked, x)
		}
	}
	report := func(write func(w *csv.Writer)) []byte {
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"row", "email", "source_status", "outcome", "reason"})
		write(w)
		w.Flush()
		return b.Bytes()
	}
	mc := report(func(w *csv.Writer) {
		for i, e := range pending {
			_ = w.Write([]string{fmt.Sprint(i + 1), e, "pending", "pending", ""})
		}
	})
	statusOf := map[string]string{"unsubscribe": "unsubscribed", "hard_bounce": "cleaned", "complaint": "complained"}
	skippedReason := func(i int) string {
		switch {
		case i < 131:
			return "invalid"
		case i < 183:
			return "disposable"
		}
		return "trap"
	}
	skippedValue := func(i int) string {
		switch skippedReason(i) {
		case "invalid":
			return []string{"jane.doe@", "mark@@acme.io", "no email", "carlos..m@gmail.com"}[i%4]
		case "disposable":
			return fmt.Sprintf("tmp%03d@mailinator.com", i)
		}
		return "redacted"
	}
	row := 0
	brevoReport := report(func(w *csv.Writer) {
		for _, x := range blocked {
			row++
			_ = w.Write([]string{fmt.Sprint(row), x.email, statusOf[x.reason], "suppressed", x.reason})
		}
		for i := 0; i < 214; i++ {
			row++
			_ = w.Write([]string{fmt.Sprint(row), skippedValue(i), "subscribed", "skipped", skippedReason(i)})
		}
		for i := 0; i < 46118 && len(brevo) > 0; i++ {
			row++
			outcome := "imported"
			if i < 6716 {
				outcome = "pending"
			}
			_ = w.Write([]string{fmt.Sprint(row), brevo[i%len(brevo)], "subscribed", outcome, ""})
		}
	})
	var skipped bytes.Buffer
	w := csv.NewWriter(&skipped)
	_ = w.Write([]string{"row", "value", "source_status", "reason"})
	for i := 0; i < 214; i++ {
		_ = w.Write([]string{fmt.Sprint(len(blocked) + i + 1), skippedValue(i), "subscribed", skippedReason(i)})
	}
	w.Flush()
	if _, err := s.Tx.Exec(ctx, `UPDATE imports SET report_csv = $2 WHERE id = $1`, a.mailchimpOld, mc); err != nil {
		return err
	}
	_, err := s.Tx.Exec(ctx, `UPDATE imports SET report_csv = $2, skipped_csv = $3 WHERE id = $1`, a.brevoImport, brevoReport, skipped.Bytes())
	return err
}

func mustJSONB(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// audStageDraft creates the Mailchimp draft of the importer boards: the
// "Acme Customers" audience, connected at 09:44, with its 13,739 members
// already read: 11,844 new, 318 already in ReputeCast (Sam, Priya, Diego,
// Hannah, Omar, Grace and 311 more, plus j.park who is suppressed), 1,259
// unsubscribed, cleaned or complained, 46 invalid and 272 risky.
func audStageDraft(ctx context.Context, s *Seeder, a *audSeed) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	id := a.mailchimpDraft
	checked := audToday(s, 9, 44)
	g := &audGen{s: s, rng: audRand(13739, 9), emails: a.emails, now: s.Now}
	fields := []map[string]string{
		{"key": "EMAIL", "label": "EMAIL", "type": "email"}, {"key": "FNAME", "label": "FNAME", "type": "text"}, {"key": "LNAME", "label": "LNAME", "type": "text"},
		{"key": "COMPANY", "label": "COMPANY", "type": "text"}, {"key": "PLAN", "label": "PLAN", "type": "text"}, {"key": "MRR", "label": "MRR", "type": "number"},
		{"key": "INDUSTRY", "label": "INDUSTRY", "type": "text"}, {"key": "SIGNUP_SRC", "label": "SIGNUP_SRC", "type": "text"}, {"key": "TRIAL_END", "label": "TRIAL_END", "type": "date"},
		{"key": "TIMEZONE", "label": "TIMEZONE", "type": "timezone"}, {"key": "OPTIN", "label": "OPTIN_TIME · OPTIN_IP", "type": "consent"}, {"key": "TAGS", "label": "TAGS", "type": "tags"},
		{"key": "ADDRESS", "label": "ADDRESS", "type": "address"}, {"key": "PHONE", "label": "PHONE", "type": "phone"},
	}
	conn := map[string]any{"account": "Acme Inc", "dataCenter": "us21", "checkedAt": checked, "fields": fields, "stagedLists": []string{"3f9a2c"},
		"lists": []map[string]any{{"id": "3f9a2c", "name": "Acme Customers", "subscribed": 12480, "unsubscribed": 1033, "cleaned": 212, "pending": 0,
			"transactional": 0, "complained": 14, "fields": 14, "tags": 9, "tagExamples": []string{"customer", "churn-risk", "trial"}, "doubleOptIn": true, "since": "2024"}}}
	creds, err := s.Box.Seal([]byte(`{"apiKey":"5f0c9e3a7b2d4e6f8a1b3c5d7e9f0a2b-us21"}`), "import:"+id.String()+":credentials")
	if err != nil {
		return err
	}
	opts := audImportOptions([]map[string]string{{"sourceId": "3f9a2c", "target": s.Lists["Product Updates"].String()}})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO imports (id, workspace_id, source, status, step, connection, credentials_enc, options, created_by, created_at, staged_at, updated_at)
		VALUES ($1,$2,'mailchimp','draft','source',$3,$4,$5,$6,$7,$8,$8)`,
		id, ws.ID, mustJSONB(conn), creds, mustJSONB(opts), linh.ID, checked.Add(-2*time.Minute), checked.Add(40*time.Second)); err != nil {
		return err
	}

	type row struct {
		email, status string
		data          map[string]string
		tags          []string
		eng           map[string]any
	}
	optin := func(t string, ip string) map[string]any {
		at, _ := time.Parse("2006-01-02 15:04", t)
		return map[string]any{"optin_at": at.UTC(), "optin_ip": ip, "rating": 3 + g.rng.IntN(3), "changed": s.designDay(12, 10, 0).UTC()}
	}
	person := func(email, first, last string, extra map[string]string) map[string]string {
		d := map[string]string{"EMAIL": email, "FNAME": first, "LNAME": last}
		for k, v := range extra {
			d[k] = v
		}
		return d
	}
	byBoard := a.byBoard
	rows := []row{
		{byBoard["sam"].email, "subscribed", person(byBoard["sam"].email, "Sam", "Rivera", map[string]string{"COMPANY": "Northwind Labs", "PLAN": "team", "MRR": "29",
			"INDUSTRY": "SaaS", "SIGNUP_SRC": "pricing-page", "TRIAL_END": "2026-10-30", "TIMEZONE": "America/Chicago", "OPTIN_TIME": "2025-03-02 14:11",
			"OPTIN_IP": "203.0.113.9", "ADDRESS": "500 Market St Suite 200, San Francisco, CA", "PHONE": "+1 415 5550134 ext 21"}),
			[]string{"customer", "trial"}, optin("2025-03-02 14:11", "203.0.113.9")},
		{byBoard["priya"].email, "subscribed", person(byBoard["priya"].email, "Priya", "Natarajan", map[string]string{"COMPANY": "Berg Consulting", "PLAN": "starter",
			"MRR": "9", "INDUSTRY": "Agency", "SIGNUP_SRC": "docs", "TRIAL_END": "2026-11-04", "TIMEZONE": "Europe/Berlin", "OPTIN_TIME": "2025-03-02 09:20",
			"OPTIN_IP": "198.51.100.23", "ADDRESS": "14 Rue Lafayette, 75009 Paris", "PHONE": "+49 30 1234567890"}),
			[]string{"customer", "churn-risk"}, optin("2025-03-02 09:20", "198.51.100.23")},
		{byBoard["diego"].email, "subscribed", person(byBoard["diego"].email, "Diego", "Alvarez", map[string]string{"PLAN": "business", "MRR": "79",
			"INDUSTRY": "Ecommerce", "SIGNUP_SRC": "webinar", "TIMEZONE": "America/Chicago"}), []string{"customer"}, optin("2024-11-11 18:02", "")},
		{byBoard["hannah"].email, "subscribed", person(byBoard["hannah"].email, "Hannah", "Berg", map[string]string{"COMPANY": "Berg Consulting"}), []string{"customer"}, optin("2026-06-22 10:00", "")},
		{byBoard["omar"].email, "subscribed", person(byBoard["omar"].email, "Omar", "Haddad", map[string]string{"COMPANY": "Haddad Studio"}), []string{}, optin("2025-09-30 12:30", "")},
		{byBoard["grace"].email, "subscribed", person(byBoard["grace"].email, "Grace", "Okafor", nil), []string{},
			map[string]any{"last_open": time.Date(2024, 3, 2, 7, 41, 0, 0, time.UTC), "changed": time.Date(2024, 3, 2, 7, 41, 0, 0, time.UTC), "rating": 1}},
		{"j.park@gmail.com", "subscribed", person("j.park@gmail.com", "Jin", "Park", nil), []string{"customer"}, optin("2025-05-14 08:12", "")},
		// The first invalid, role, typo and disposable rows are the preview's examples.
		{"sam.rivera@", "subscribed", person("sam.rivera@", "Sam", "Rivera", nil), []string{}, nil},
		{"priya..n@gmail.com", "subscribed", person("priya..n@gmail.com", "Priya", "N", nil), []string{}, nil},
		{`"hannah berg"`, "subscribed", person(`"hannah berg"`, "Hannah", "Berg", nil), []string{}, nil},
		{"info@northwind-labs.co", "subscribed", person("info@northwind-labs.co", "", "", map[string]string{"COMPANY": "Northwind Labs"}), []string{}, nil},
		{"sales@kestrel.dev", "subscribed", person("sales@kestrel.dev", "", "", map[string]string{"COMPANY": "Kestrel"}), []string{}, nil},
		{"mike.ross@gmial.com", "subscribed", person("mike.ross@gmial.com", "Mike", "Ross", nil), []string{}, nil},
		{"laura.k@outlok.com", "subscribed", person("laura.k@outlok.com", "Laura", "Klein", nil), []string{}, nil},
		{"tomoko@yaho.com", "subscribed", person("tomoko@yaho.com", "Tomoko", "Sato", nil), []string{}, nil},
		{"trial-user-3381@mailinator.com", "subscribed", person("trial-user-3381@mailinator.com", "", "", nil), []string{}, nil},
		{"qa.signup@guerrillamail.com", "subscribed", person("qa.signup@guerrillamail.com", "", "", nil), []string{}, nil},
	}
	var rest []row
	newPerson := func(domain string) (string, map[string]string) {
		first, last := g.name()
		e := g.email(first, last, domain)
		return e, person(e, first, last, nil)
	}
	// 311 more contacts already in ReputeCast.
	n := 0
	for _, c := range a.contacts {
		if n == 311 {
			break
		}
		if !audIsGenerated(c) || c.pending || c.lists&audInUpdates == 0 || g.rng.IntN(20) != 0 {
			continue
		}
		tags := []string{}
		if g.rng.IntN(2) == 0 {
			tags = append(tags, "customer")
		}
		rest = append(rest, row{c.email, "subscribed", person(c.email, c.first, c.last, nil), tags, optin("2025-06-01 10:00", "")})
		n++
	}
	well := []string{"gmail.com", "gmail.com", "gmail.com", "outlook.com", "hotmail.com", "yahoo.com", "icloud.com", "me.com", "aol.com", "live.com", "gmx.de", "web.de", "proton.me", "fastmail.com"}
	pick := func() string { return well[g.rng.IntN(len(well))] }
	for _, st := range []struct {
		status string
		n      int
	}{{"unsubscribed", 1033}, {"cleaned", 212}, {"complained", 14}} {
		for i := 0; i < st.n; i++ {
			e, d := newPerson(pick())
			rest = append(rest, row{e, st.status, d, []string{}, map[string]any{"rating": 1 + g.rng.IntN(3), "changed": s.designDay(30+g.rng.IntN(600), 12, 0).UTC()}})
		}
	}
	// Invalid: 11 more without a domain, 8 more with a double dot, 6 more
	// that aren't addresses, 18 on domains with no MX record.
	for i := 0; i < 11; i++ {
		first, last := g.name()
		v := audASCII(first) + "." + audASCII(last) + "@"
		rest = append(rest, row{v, "subscribed", person(v, first, last, nil), []string{}, nil})
	}
	for i := 0; i < 8; i++ {
		first, last := g.name()
		v := audASCII(first) + ".." + audASCII(last)[:1] + "@" + pick()
		rest = append(rest, row{v, "subscribed", person(v, first, last, nil), []string{}, nil})
	}
	for i := 0; i < 6; i++ {
		first, last := g.name()
		v := []string{first + " " + last, "n/a", audASCII(first) + " at " + audASCII(last) + ".com", "-", "unknown", "test"}[i]
		rest = append(rest, row{v, "subscribed", person(v, first, last, nil), []string{}, nil})
	}
	noMX := []string{"acme-old-mail.invalid", "northwind-legacy.invalid", "mailhost-closed.invalid"}
	for i := 0; i < 18; i++ {
		e, d := newPerson(noMX[i%len(noMX)])
		rest = append(rest, row{e, "subscribed", d, []string{}, optin("2024-08-01 10:00", "")})
	}
	// Risky: 25 more role addresses, 17 more disposable, 22 more typos,
	// 187 stale and 14 trap-like.
	roleLocals := []string{"info", "sales", "support", "hello", "contact", "office", "team", "billing", "marketing", "admin"}
	for i := 0; i < 25; i++ {
		d := audCompanyDomains[g.rng.IntN(len(audCompanyDomains)-4)].domain
		e := roleLocals[i%len(roleLocals)] + "@" + strings.Replace(d, ".", "-hq.", 1)
		if a.emails[e] {
			e = fmt.Sprintf("%s%d@%s", roleLocals[i%len(roleLocals)], i, d)
		}
		a.emails[e] = true
		rest = append(rest, row{e, "subscribed", person(e, "", "", nil), []string{}, optin("2025-01-10 10:00", "")})
	}
	disposable := []string{"mailinator.com", "guerrillamail.com", "10minutemail.com", "yopmail.com", "maildrop.cc", "getnada.com"}
	for i := 0; i < 17; i++ {
		e, d := newPerson(disposable[i%len(disposable)])
		rest = append(rest, row{e, "subscribed", d, []string{}, nil})
	}
	typos := []string{"gmial.com", "outlok.com", "yaho.com", "gmai.com", "hotmial.com", "gnail.com", "iclod.com", "outllok.com", "yahooo.com"}
	for i := 0; i < 22; i++ {
		e, d := newPerson(typos[i%len(typos)])
		rest = append(rest, row{e, "subscribed", d, []string{}, optin("2025-04-01 10:00", "")})
	}
	for i := 0; i < 187; i++ {
		e, d := newPerson(pick())
		changed := time.Date(2024, time.Month(1+g.rng.IntN(9)), 1+g.rng.IntN(28), 10, 0, 0, 0, time.UTC)
		rest = append(rest, row{e, "subscribed", d, []string{}, map[string]any{"rating": 1, "changed": changed}})
	}
	traps := []string{"spamtrap", "honeypot", "spam-trap", "blackhole", "trapaddress"}
	for i := 0; i < 14; i++ {
		e := fmt.Sprintf("%s%02d@%s", traps[i%len(traps)], i+1, []string{"mail-sink.net", "trapnet.org", "listcheck.io"}[i%3])
		rest = append(rest, row{e, "subscribed", person(e, "", "", nil), []string{}, nil})
	}
	// 11,844 new contacts, subscribed.
	industries := []string{"SaaS", "Agency", "Ecommerce", "Fintech", "Healthcare", "Education", "Media", "Nonprofit"}
	plans := []string{"free", "free", "starter", "starter", "team", "business"}
	sources := []string{"pricing-page", "docs", "webinar", "blog", "partner"}
	tagPool := []string{"customer", "trial", "churn-risk", "webinar-2025", "partner", "enterprise-lead", "newsletter", "beta-waitlist", "event-booth"}
	for i := 0; i < 11844; i++ {
		e, d := newPerson(pick())
		plan := plans[g.rng.IntN(len(plans))]
		d["PLAN"] = plan
		d["MRR"] = map[string]string{"free": "0", "starter": "9", "team": "29", "business": "79"}[plan]
		d["INDUSTRY"] = industries[g.rng.IntN(len(industries))]
		d["SIGNUP_SRC"] = sources[g.rng.IntN(len(sources))]
		cc := audCountries[g.rng.IntN(len(audCountries))]
		d["TIMEZONE"] = cc.tz
		if plan == "free" && g.rng.IntN(3) == 0 {
			d["TRIAL_END"] = s.designDay(-g.rng.IntN(30), 0, 0).Format("2006-01-02")
		}
		at := s.designDay(40+g.rng.IntN(700), 8+g.rng.IntN(10), g.rng.IntN(60))
		d["OPTIN_TIME"] = at.UTC().Format("2006-01-02 15:04")
		d["OPTIN_IP"] = fmt.Sprintf("198.51.100.%d", 1+g.rng.IntN(250))
		var tags []string
		for _, t := range tagPool {
			if g.rng.IntN(9) == 0 {
				tags = append(tags, t)
			}
		}
		if tags == nil {
			tags = []string{}
		}
		eng := map[string]any{"optin_at": at.UTC(), "optin_ip": d["OPTIN_IP"], "rating": 2 + g.rng.IntN(4), "changed": s.designDay(1+g.rng.IntN(300), 12, 0).UTC()}
		if g.rng.IntN(3) > 0 {
			eng["last_open"] = s.designDay(1+g.rng.IntN(120), 9+g.rng.IntN(10), g.rng.IntN(60)).UTC()
		}
		rest = append(rest, row{e, "subscribed", d, tags, eng})
	}
	g.rng.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	rows = append(rows, rest...)
	if len(rows) != 13739 {
		return fmt.Errorf("staged %d rows, want 13,739", len(rows))
	}

	buf := make([][]any, 0, len(rows))
	domains := map[string]bool{}
	for i, r := range rows {
		data, _ := json.Marshal(r.data)
		eng := []byte(`{}`)
		if r.eng != nil {
			eng, _ = json.Marshal(r.eng)
		}
		buf = append(buf, []any{id, int32(i + 1), r.email, r.status, "3f9a2c", data, r.tags, eng})
		if e, problem := emailaddr.Check(r.email); problem == "" {
			domains[emailaddr.Domain(e)] = true
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"import_rows"}, []string{"import_id", "row_no", "email", "status", "list_ref", "data", "tags", "engagement"},
		pgx.CopyFromRows(buf)); err != nil {
		return err
	}
	// MX answers for every domain the preview checks, so it never waits on
	// DNS: the three closed domains have no MX record.
	for d := range domains {
		if emailaddr.MailboxProvider(d) != emailaddr.Other {
			continue
		}
		hosting := ""
		for _, c := range audCompanyDomains {
			if c.domain == d {
				hosting = c.hosting
			}
		}
		has := !strings.HasSuffix(d, ".invalid")
		if _, err := s.Tx.Exec(ctx, `INSERT INTO mx_cache (domain, has_mx, provider, checked_at) VALUES ($1,$2,$3,now()) ON CONFLICT (domain) DO NOTHING`, d, has, hosting); err != nil {
			return err
		}
	}
	return nil
}
