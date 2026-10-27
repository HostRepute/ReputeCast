package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// SampleWorkspace fills a new workspace with a compact Acme sample for its
// owner: "Explore with demo data" on the first-run Overview (review PR-13).
// Three lists, a few hundred contacts across mailbox providers, demo
// connections of Amazon SES, Postmark and Postal (demo connections never
// send: nothing leaves the instance), the routing of the ProvidersRouting
// board and a draft campaign. Nothing else changes: no demo users, no
// shared password, no other workspace.
func SampleWorkspace(ctx context.Context, tx pgx.Tx, box *secrets.Box, ws store.Workspace, owner store.User, now time.Time) error {
	s := &Seeder{
		Tx: tx, Q: store.New(tx), Box: box, Now: now,
		Users: map[string]store.User{"linh@acme.io": owner}, Workspaces: map[string]store.Workspace{"acme-newsletter": ws}, Roles: map[string]store.Role{},
		Lists: map[string]uuid.UUID{}, Providers: map[string]uuid.UUID{}, Campaigns: map[string]uuid.UUID{},
		Senders: map[string]uuid.UUID{}, Domains: map[string]uuid.UUID{}, Extra: map[string]any{},
	}
	lists := []struct{ name, desc string }{
		{"Weekly Digest", "The best of the week, every Tuesday."},
		{"Product Updates", "New features and changes, about twice a month."},
		{"Beta Program", "Early builds for testers."},
	}
	for _, l := range lists {
		id := ID()
		if _, err := tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, created_at) VALUES ($1,$2,$3,$4,true,$5)`,
			id, ws.ID, l.name, l.desc, now.Add(-200*24*time.Hour)); err != nil {
			return fmt.Errorf("sample lists: %w", err)
		}
		s.Lists[l.name] = id
	}
	if err := sampleContacts(ctx, s, 240); err != nil {
		return err
	}
	checked := now.Add(-time.Minute)
	for _, p := range acmeProviders(checked) {
		id := ID()
		if err := insertProvider(ctx, s, id, ws.ID, p, owner.ID, checked); err != nil {
			return fmt.Errorf("sample providers: %w", err)
		}
		s.Providers[p.key] = id
	}
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	beta := s.Lists["Beta Program"]
	rules := []struct {
		stream, name, conditions, splits string
		priority                         int
		fallback                         *uuid.UUID
	}{
		{"broadcast", "small, high-value list", `{"lists":["` + beta.String() + `"]}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 10, nil},
		{"broadcast", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":90},{"provider_id":"` + postal.String() + `","weight":10}]`, 100, nil},
		{"automation", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 100, &postmark},
		{"system", "", `{}`, `[{"provider_id":"` + postmark.String() + `","weight":100}]`, 100, &ses},
	}
	for _, r := range rules {
		if _, err := tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, ID(), ws.ID, r.stream, r.priority, r.name, json.RawMessage(r.conditions), json.RawMessage(r.splits), r.fallback,
			owner.ID, now); err != nil {
			return fmt.Errorf("sample routing: %w", err)
		}
	}
	for stream, policy := range map[string]string{"broadcast": "pause_and_ask", "automation": "automatic", "system": "automatic"} {
		if _, err := tx.Exec(ctx, `INSERT INTO stream_policies (workspace_id, stream, failover) VALUES ($1,$2,$3)
			ON CONFLICT (workspace_id, stream) DO UPDATE SET failover = EXCLUDED.failover`, ws.ID, stream, policy); err != nil {
			return err
		}
	}
	aud, _ := json.Marshal(map[string]any{"lists": []string{s.Lists["Weekly Digest"].String()}})
	if _, err := tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, mjml, html, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,'Weekly Digest #142','draft','This week at Acme','Three changes worth your time',$3,'','',$4,$4,$5,$5)`, ID(), ws.ID, aud, owner.ID, now); err != nil {
		return fmt.Errorf("sample campaign: %w", err)
	}
	return nil
}

// sampleContacts adds n contacts with the sample's mix of mailbox
// providers and engagement tiers, subscribed to the lists.
func sampleContacts(ctx context.Context, s *Seeder, n int) error {
	ws := s.WS()
	rng := rand.New(rand.NewPCG(42, 142))
	domains := []struct{ domain, mbp string }{
		{"gmail.com", "gmail"}, {"gmail.com", "gmail"}, {"gmail.com", "gmail"}, {"outlook.com", "outlook"}, {"hotmail.com", "outlook"},
		{"yahoo.com", "yahoo"}, {"icloud.com", "icloud"}, {"example.org", "other"},
	}
	first := []string{"Sam", "Priya", "Lucas", "Mei", "Omar", "Ana", "Jonas", "Aisha", "Tom", "Yuki", "Elena", "Kofi"}
	last := []string{"Rivera", "Shah", "Martin", "Chen", "Haddad", "Silva", "Berg", "Khan", "Walsh", "Sato", "Rossi", "Mensah"}
	tiers := []string{"active", "active", "active", "cooling", "cooling", "at_risk", "sunset"}
	weekly, updates, beta := s.Lists["Weekly Digest"], s.Lists["Product Updates"], s.Lists["Beta Program"]
	contacts := make([][]any, 0, n)
	members := [][]any{}
	for i := 0; i < n; i++ {
		id := ID()
		d := domains[rng.IntN(len(domains))]
		f, l := first[rng.IntN(len(first))], last[rng.IntN(len(last))]
		email := fmt.Sprintf("%s.%s%d@%s", lower(f), lower(l), i, d.domain)
		created := s.Now.Add(-time.Duration(30+rng.IntN(300)) * 24 * time.Hour)
		contacts = append(contacts, []any{id, ws.ID, email, f, l, "active", tiers[rng.IntN(len(tiers))], "import", d.mbp, created, created})
		members = append(members, []any{weekly, id, "subscribed", "import", created, created})
		if rng.IntN(3) == 0 {
			members = append(members, []any{updates, id, "subscribed", "import", created, created})
		}
		if rng.IntN(12) == 0 {
			members = append(members, []any{beta, id, "subscribed", "import", created, created})
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source",
		"mailbox_provider", "created_at", "updated_at"}, pgx.CopyFromRows(contacts)); err != nil {
		return fmt.Errorf("sample contacts: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "created_at"},
		pgx.CopyFromRows(members)); err != nil {
		return fmt.Errorf("sample memberships: %w", err)
	}
	return nil
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
