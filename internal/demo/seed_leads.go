package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/leads/meta/metatest"
)

// Lead ads from Meta (M66): Acme Marketing, a Meta app whose system user
// reaches three Pages, two of them chosen, with three lead forms (one
// paused) and 41 leads in the last 30 days.

const (
	LeadAdsAppID     = "481516234200"
	LeadAdsAppSecret = "demo-lead-ads-app-secret-7f3c"   //nolint:gosec // the in-process Graph fake's, never Meta's
	LeadAdsToken     = "demo-lead-ads-system-user-token" //nolint:gosec // the in-process Graph fake's, never Meta's
)

type leadAdsPage struct {
	id, name, token string
	chosen          bool
	forms           []meta.Form
}

func leadAdsPages() []leadAdsPage {
	webinar := meta.Form{ID: "1180000000001", Name: "Spring webinar signup", Status: "ACTIVE", Locale: "EN_US", Checkboxes: []meta.Checkbox{},
		Questions: []meta.Question{{Key: "email", Label: "Email", Type: "EMAIL"}, {Key: "full_name", Label: "Full name", Type: "FULL_NAME"},
			{Key: "company_name", Label: "Company name", Type: "COMPANY_NAME"}, {Key: "job_title", Label: "Job title", Type: "JOB_TITLE"},
			{Key: "what_to_learn", Label: "What do you want to learn?", Type: "CUSTOM"}}}
	guide := meta.Form{ID: "1180000000002", Name: "Field Guide download", Status: "ACTIVE", Locale: "EN_US",
		Questions:  []meta.Question{{Key: "email", Label: "Email", Type: "EMAIL"}, {Key: "first_name", Label: "First name", Type: "FIRST_NAME"}},
		Disclaimer: "Acme sends the Field Guide to the address you give.",
		Checkboxes: []meta.Checkbox{{Key: "marketing_optin", Text: "Yes, send me the Weekly Digest and product updates from Acme by email. I can unsubscribe at any time.", Required: false}}}
	beta := meta.Form{ID: "1180000000003", Name: "Beta waitlist", Status: "ACTIVE", Locale: "EN_US", Checkboxes: []meta.Checkbox{},
		Questions: []meta.Question{{Key: "email", Label: "Email", Type: "EMAIL"}, {Key: "full_name", Label: "Full name", Type: "FULL_NAME"}}}
	return []leadAdsPage{
		{id: "102030405060001", name: "Acme Inc", token: "demo-page-token-acme-inc", chosen: true, forms: []meta.Form{webinar, guide}},
		{id: "102030405060002", name: "Acme Labs", token: "demo-page-token-acme-labs", chosen: true, forms: []meta.Form{beta}},
		{id: "102030405060003", name: "Acme Careers", token: "demo-page-token-acme-careers", forms: []meta.Form{}},
	}
}

// LeadAdsFake is the Graph API the demo's connection answers from: its
// Pages and forms, accepting the credentials the demo holds.
func LeadAdsFake() *metatest.Fake {
	f := metatest.New(LeadAdsAppID, LeadAdsAppSecret, LeadAdsToken)
	f.Permissive = true
	for _, p := range leadAdsPages() {
		f.AddPage(p.id, p.name, p.token)
		for _, form := range p.forms {
			f.AddForm(p.id, form)
		}
	}
	return f
}

func init() { seedLeads = seedLeadsImpl }

func seedLeadsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"].ID
	conn := uuid.Must(uuid.NewV7())
	creds, _ := json.Marshal(meta.Credentials{AppID: LeadAdsAppID, AppSecret: LeadAdsAppSecret, Token: LeadAdsToken})
	sealed, err := s.Box.Seal(creds, "lead_connection:"+conn.String()+":credentials")
	if err != nil {
		return err
	}
	token, hash := authn.NewToken()
	cb, err := s.Box.SealString(token, "lead_connection:"+conn.String()+":callback_token")
	if err != nil {
		return err
	}
	vt, err := s.Box.SealString("rc-leads-demoVerifyToken2026", "lead_connection:"+conn.String()+":verify_token")
	if err != nil {
		return err
	}
	created := s.designDay(45, 10, 30)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO lead_connections (id, workspace_id, platform, name, demo, app_id, credentials_sealed, callback_token_hash,
		callback_token_sealed, verify_token_sealed, status, last_event_at, verified_at, subscribed_at, checked_at, created_by, created_at, updated_at)
		VALUES ($1,$2,'meta','Acme Marketing',true,$3,$4,$5,$6,$7,'connected',$8,$9,$9,$10,$11,$9,$9)`,
		conn, ws.ID, LeadAdsAppID, sealed, hash, cb, vt, s.Ago(12*time.Minute), created, s.Ago(4*time.Minute), linh); err != nil {
		return err
	}
	lists := map[string]uuid.UUID{"Weekly Digest": s.Lists["Weekly Digest"], "Product Updates": s.Lists["Product Updates"], "Beta Program": s.Lists["Beta Program"]}
	type formSeed struct {
		mapping  map[string]string
		lists    []uuid.UUID
		tags     []string
		mode     string
		status   string
		leads    int
		older    int
		consentK string
	}
	seeds := map[string]formSeed{
		"Spring webinar signup": {mapping: map[string]string{"email": "email", "full_name": "full_name", "company_name": "company"},
			lists: []uuid.UUID{lists["Weekly Digest"]}, tags: []string{"webinar-2026"}, mode: "double", status: "active", leads: 24},
		"Field Guide download": {mapping: map[string]string{"email": "email", "first_name": "first_name"},
			lists: []uuid.UUID{lists["Weekly Digest"], lists["Product Updates"]}, tags: []string{"field-guide"}, mode: "single", status: "active", leads: 17, consentK: "marketing_optin"},
		"Beta waitlist": {mapping: map[string]string{"email": "email", "full_name": "full_name"},
			lists: []uuid.UUID{lists["Beta Program"]}, mode: "double", status: "paused", older: 6},
	}
	// The board's Recent leads name these people (Priya, Tom and Sam);
	// the other leads go to the workspace's newest active contacts.
	byEmail := map[string]*uuid.UUID{}
	for _, email := range []string{"priya.n@gmail.com", "t.nowak@gmail.com", "sam.rivera@outlook.com"} {
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, ws.ID, email).Scan(&id); err == nil {
			byEmail[email] = &id
		}
	}
	var contacts []uuid.UUID
	rows, err := s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' AND erased_at IS NULL ORDER BY created_at DESC LIMIT 60`, ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		contacts = append(contacts, id)
	}
	rows.Close()
	next := 0
	contact := func() *uuid.UUID {
		if len(contacts) == 0 {
			return nil
		}
		id := contacts[next%len(contacts)]
		next++
		return &id
	}
	named := func(email string) *uuid.UUID {
		if id := byEmail[email]; id != nil {
			return id
		}
		return contact()
	}
	lead, formN := 0, 0
	for _, p := range leadAdsPages() {
		if !p.chosen {
			continue
		}
		page := uuid.Must(uuid.NewV7())
		pt, err := s.Box.SealString(p.token, "lead_page:"+page.String()+":token")
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO lead_pages (id, workspace_id, connection_id, page_id, name, page_token_sealed, chosen, subscribed_at, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,true,$7,$7,$7)`, page, ws.ID, conn, p.id, p.name, pt, created); err != nil {
			return err
		}
		for _, f := range p.forms {
			sd := seeds[f.Name]
			qs, _ := json.Marshal(f.Questions)
			cbs, _ := json.Marshal(f.Checkboxes)
			mp, _ := json.Marshal(sd.mapping)
			text := f.Disclaimer
			if sd.consentK != "" {
				text = f.Checkboxes[0].Text
			}
			version := ""
			if text != "" {
				version = "meta:demo" + f.ID[len(f.ID)-4:]
			}
			if sd.tags == nil {
				sd.tags = []string{}
			}
			form := leadAdsFormID(s, f.Name)
			// A minute apart, so the page lists them in the board's order.
			formAt := created.Add(time.Duration(formN) * time.Minute)
			formN++
			var last *time.Time
			if sd.leads > 0 {
				last = new(leadAdsEntries(s, f.Name, sd.leads, 0, sd.mode, func(string) *uuid.UUID { return nil }, func() *uuid.UUID { return nil })[0].at)
			}
			if _, err := s.Tx.Exec(ctx, `INSERT INTO lead_forms (id, workspace_id, page_id, form_id, name, meta_status, questions, checkboxes, disclaimer, status, mapped_at,
				mapping, consent_mode, consent_key, consent_text, consent_version, list_ids, tags, last_lead_at, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,'ACTIVE',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$10,$10)`,
				form, ws.ID, page, f.ID, f.Name, qs, cbs, f.Disclaimer, sd.status, formAt, mp, sd.mode, sd.consentK, text, version, sd.lists, sd.tags, last); err != nil {
				return err
			}
			for _, e := range leadAdsEntries(s, f.Name, sd.leads, sd.older, sd.mode, named, contact) {
				lead++
				if _, err := s.Tx.Exec(ctx, `INSERT INTO lead_entries (id, workspace_id, form_id, leadgen_id, contact_id, outcome, source, lead_at, created_at)
					VALUES ($1,$2,$3,$4,$5,$6,'webhook',$7,$7)`, uuid.Must(uuid.NewV7()), ws.ID, form, fmt.Sprintf("9200000000%04d", lead), e.who, e.outcome, e.at); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// leadAdsEntry is one recorded lead of the demo.
type leadAdsEntry struct {
	at      time.Time
	outcome string
	who     *uuid.UUID
}

// leadAdsEntries are a form's leads, newest first: the board's Recent
// leads (Priya waiting for confirmation 12 minutes ago, Tom's new contact,
// Sam's update, a consent box not ticked, a suppressed address, and
// yesterday's test lead from Meta's testing tool), then the rest of the
// last 30 days, then the older ones.
func leadAdsEntries(s *Seeder, form string, leads, older int, mode string, named func(string) *uuid.UUID, contact func() *uuid.UUID) []leadAdsEntry {
	var out []leadAdsEntry
	switch form {
	case "Spring webinar signup":
		out = append(out, leadAdsEntry{s.Ago(12 * time.Minute), "pending", named("priya.n@gmail.com")},
			leadAdsEntry{s.Ago(198 * time.Minute), "blocked", contact()})
	case "Field Guide download":
		out = append(out, leadAdsEntry{s.Ago(28 * time.Minute), "created", named("t.nowak@gmail.com")},
			leadAdsEntry{s.Ago(65 * time.Minute), "updated", named("sam.rivera@outlook.com")},
			leadAdsEntry{s.Ago(100 * time.Minute), "declined", nil},
			leadAdsEntry{s.designDay(1, 17, 30), "test", nil})
	}
	counted := 0
	for _, e := range out {
		if e.outcome != "test" {
			counted++
		}
	}
	// The rest of the 30 days, older than the test lead (at most 36 hours
	// old): a day and a half back to about 15 days.
	for i := 0; counted < leads; i++ {
		outcome := "updated"
		if mode == "double" && i%3 == 0 {
			outcome = "pending"
		} else if i%4 == 1 {
			outcome = "created"
		}
		out = append(out, leadAdsEntry{s.Ago(time.Duration(36+i*15) * time.Hour), outcome, contact()})
		counted++
	}
	for i := range older {
		out = append(out, leadAdsEntry{s.Ago(time.Duration(35+i) * 24 * time.Hour), "updated", contact()})
	}
	return out
}

// leadAdsFormID is a lead form's id, made by whichever seeder needs it
// first: Growth's sample counts Field Guide download's joins before the
// form is stored.
func leadAdsFormID(s *Seeder, name string) uuid.UUID {
	key := "leadForm:" + name
	if id, ok := s.Extra[key].(uuid.UUID); ok {
		return id
	}
	id := uuid.Must(uuid.NewV7())
	s.Extra[key] = id
	return id
}
