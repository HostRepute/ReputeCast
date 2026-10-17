package demo

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedScoringGrowth = seedScoringGrowthImpl }

// Contact scoring and growth sources (audience-insights brief §4): Sofia
// Rossi's eight rules with every contact's score and the events behind it,
// how each contact first joined and left, the quality of each source, the
// "Sales-ready handoff" automation and the "Hot on paid plans" segment.
//
// The sample's numbers are its own: the 1.0 contacts keep their lists,
// tiers, dates and sources, and the growth and scoring tables are seeded
// to the boards' numbers on top of them (DESIGN_INVENTORY lists where the
// two disagree). Everything is relative to now, like the rest of the demo.

// sgContact is one Acme Newsletter contact with what the seeder decides
// about its growth source and score.
type sgContact struct {
	id                   uuid.UUID
	email                string
	tier, source         string
	created              time.Time
	importID             *uuid.UUID
	lastHuman, lastClick *time.Time
	paid                 bool
	lists                int // subscribed lists (audInDigest…)
	pending              bool
	// leftList: lists the contact left in 1.0's sample while staying on another.
	leftList int
	board    string

	g  sgGrowth
	sc sgScore
}

// sgWorld carries the loaded contacts and the references both halves need.
type sgWorld struct {
	ctx      context.Context
	s        *Seeder
	ws       store.Workspace
	loc      *time.Location
	now      time.Time
	rng      *rand.Rand
	contacts []*sgContact
	byEmail  map[string]*sgContact
	listBit  map[uuid.UUID]int
	bitList  map[int]uuid.UUID

	brevoImport, mcImport uuid.UUID
	brevoAt, mcAt         time.Time
	forms                 map[string]uuid.UUID // by name
	apiKey                uuid.UUID
	zapier                uuid.UUID
	// periodStart is the first day of Growth's 30 days; liveSince is when
	// "Sales-ready handoff" went live.
	periodStart time.Time
	liveSince   time.Time
	// used marks contacts a growth role took, so no two roles share one;
	// special marks the roles given after that (Beta Program's movers).
	used    map[*sgContact]string
	special map[*sgContact]string
	// samLinks are Sam's events to point at his emails later.
	samLinks []sgLink
}

func seedScoringGrowthImpl(ctx context.Context, s *Seeder) error {
	start := time.Now()
	w, err := sgLoad(ctx, s)
	if err != nil {
		return err
	}
	if err := w.growth(ctx); err != nil {
		return fmt.Errorf("growth: %w", err)
	}
	grown := time.Now()
	if err := w.scoring(ctx); err != nil {
		return fmt.Errorf("scoring: %w", err)
	}
	if err := w.uses(ctx); err != nil {
		return fmt.Errorf("scoring uses: %w", err)
	}
	if s.Log != nil {
		s.Log.Info("scoring and growth sample", "growth", grown.Sub(start).Round(time.Millisecond), "scoring", time.Since(grown).Round(time.Millisecond))
	}
	sam, links := w.named("sam").id, w.samLinks
	s.After(func(ctx context.Context, s *Seeder) error { return linkSam(ctx, s, sam, links) })
	return sgEngineeringBlog(ctx, s)
}

// sgWorkflowKey is the Zap that adds beta applicants from Typeform.
const sgWorkflowKey = "283551240"

func sgLoad(ctx context.Context, s *Seeder) (*sgWorld, error) {
	ws := s.WS()
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	w := &sgWorld{ctx: ctx, s: s, ws: ws, loc: loc, now: s.Now, rng: audRand(1323, 2026), byEmail: map[string]*sgContact{},
		listBit: map[uuid.UUID]int{s.Lists["Weekly Digest"]: audInDigest, s.Lists["Product Updates"]: audInUpdates, s.Lists["Beta Program"]: audInBeta},
		bitList: map[int]uuid.UUID{audInDigest: s.Lists["Weekly Digest"], audInUpdates: s.Lists["Product Updates"], audInBeta: s.Lists["Beta Program"]},
		used:    map[*sgContact]string{}, special: map[*sgContact]string{}}
	l := w.now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	w.periodStart = today.AddDate(0, 0, -29)
	w.liveSince = s.designDay(1, 15, 30)
	if f, ok := s.Extra["forms"].(map[string]uuid.UUID); ok {
		w.forms = f
	} else {
		return nil, errors.New("the forms seeder hasn't run")
	}
	if id, ok := s.Extra["key:Website signup form"].(uuid.UUID); ok {
		w.apiKey = id
	}
	if id, ok := s.Extra["integration:Acme ops"].(uuid.UUID); ok {
		w.zapier = id
	}
	if err := s.Tx.QueryRow(ctx, `SELECT id, created_at FROM imports WHERE workspace_id = $1 AND source = 'brevo' AND status = 'done' ORDER BY created_at LIMIT 1`, ws.ID).
		Scan(&w.brevoImport, &w.brevoAt); err != nil {
		return nil, fmt.Errorf("brevo import: %w", err)
	}
	if err := s.Tx.QueryRow(ctx, `SELECT id, created_at FROM imports WHERE workspace_id = $1 AND source = 'mailchimp' AND status = 'done' ORDER BY created_at LIMIT 1`, ws.ID).
		Scan(&w.mcImport, &w.mcAt); err != nil {
		return nil, fmt.Errorf("mailchimp import: %w", err)
	}
	// The import day is when the contacts arrived (the board's 16:40).
	_ = s.Tx.QueryRow(ctx, `SELECT created_at FROM contacts WHERE workspace_id = $1 AND source = 'import:brevo' ORDER BY created_at LIMIT 1`, ws.ID).Scan(&w.brevoAt)
	_ = s.Tx.QueryRow(ctx, `SELECT created_at FROM contacts WHERE workspace_id = $1 AND source = 'import:mailchimp' ORDER BY created_at LIMIT 1`, ws.ID).Scan(&w.mcAt)

	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.email, c.tier, c.source, c.created_at, c.import_id, c.last_human_open_at, c.last_click_at,
		       lower(coalesce(c.attributes ->> 'plan', '')),
		       coalesce(array_agg(lm.list_id) FILTER (WHERE lm.status = 'subscribed'), '{}'),
		       coalesce(bool_or(lm.status = 'pending'), false),
		       coalesce(array_agg(lm.list_id) FILTER (WHERE lm.status = 'unsubscribed'), '{}')
		FROM contacts c LEFT JOIN list_memberships lm ON lm.contact_id = c.id
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL
		GROUP BY c.id ORDER BY c.email`, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	board := map[string]string{}
	for _, b := range audBoardContacts {
		board[b.email] = b.key
	}
	for rows.Next() {
		c := &sgContact{}
		var plan string
		var lists []uuid.UUID
		var left []uuid.UUID
		if err := rows.Scan(&c.id, &c.email, &c.tier, &c.source, &c.created, &c.importID, &c.lastHuman, &c.lastClick, &plan, &lists, &c.pending, &left); err != nil {
			return nil, err
		}
		for _, l := range left {
			c.leftList |= w.listBit[l]
		}
		c.paid = plan == "team" || plan == "business"
		for _, l := range lists {
			c.lists |= w.listBit[l]
		}
		c.board = board[c.email]
		w.contacts = append(w.contacts, c)
		w.byEmail[c.email] = c
	}
	return w, rows.Err()
}

// named returns a board contact by key ("sam").
func (w *sgWorld) named(key string) *sgContact {
	for _, b := range audBoardContacts {
		if b.key == key {
			return w.byEmail[b.email]
		}
	}
	return nil
}

// day is the start of the day d days before today in the workspace timezone.
func (w *sgWorld) day(d int) time.Time {
	return w.periodStart.AddDate(0, 0, 29-d)
}

// shuffled returns the contacts that pass keep, in a seeded random order.
func (w *sgWorld) shuffled(keep func(*sgContact) bool) []*sgContact {
	var out []*sgContact
	for _, c := range w.contacts {
		if keep(c) {
			out = append(out, c)
		}
	}
	w.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// take marks n contacts of a pool as used for a role and returns them.
func (w *sgWorld) take(pool []*sgContact, n int, role string) ([]*sgContact, error) {
	var out []*sgContact
	for _, c := range pool {
		if len(out) == n {
			break
		}
		if _, taken := w.used[c]; taken {
			continue
		}
		w.used[c] = role
		out = append(out, c)
	}
	if len(out) < n {
		return out, fmt.Errorf("%s: %d of %d contacts", role, len(out), n)
	}
	return out, nil
}

// takeFree is take for the roles after the sources are set: any contact
// but the board's and the last 30 days' signups, once.
func (w *sgWorld) takeFree(pool []*sgContact, n int, role string) ([]*sgContact, error) {
	var out []*sgContact
	for _, c := range pool {
		if len(out) == n {
			break
		}
		if _, taken := w.special[c]; taken {
			continue
		}
		switch w.used[c] {
		case "named", sgZapier, sgBeta, "nl-beta", "person-beta", sgPerson, sgAPI, sgSub, sgFooter, sgNL:
			continue
		}
		w.special[c] = role
		out = append(out, c)
	}
	if len(out) < n {
		return out, fmt.Errorf("%s: %d of %d contacts", role, len(out), n)
	}
	return out, nil
}

// sgCopy writes rows with COPY.
func sgCopy(ctx context.Context, tx pgx.Tx, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("%s: %w", table, err)
	}
	return nil
}

// sgWeekday weights a day: weekends get about half the traffic.
func (w *sgWorld) sgWeekday(t time.Time) float64 {
	switch t.In(w.loc).Weekday() {
	case time.Saturday:
		return 0.5
	case time.Sunday:
		return 0.45
	}
	return 1
}

// sgAt is a moment on day start (00:00, workspace time) between two hours,
// never later than a few minutes ago.
func (w *sgWorld) sgAt(start time.Time, fromHour, toHour int) time.Time {
	mins := fromHour*60 + w.rng.IntN((toHour-fromHour)*60)
	t := start.Add(time.Duration(mins) * time.Minute).Add(time.Duration(w.rng.IntN(60)) * time.Second)
	if limit := w.now.Add(-8 * time.Minute); t.After(limit) {
		t = limit.Add(-time.Duration(w.rng.IntN(int(time.Hour/time.Second))) * time.Second)
		if t.Before(start) {
			t = start.Add(time.Duration(w.rng.IntN(60)) * time.Second)
		}
	}
	return t
}
