package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// Conditional content, polls and surveys (design expansion, X11; brief
// reputecast-design-addon-create.md §4):
//
//   - Weekly Digest #143 gets its three conditional blocks (We've missed
//     you, Beta corner, Start a free trial) and the topics poll in every
//     version (x11Digest143, called while the accessibility sample writes
//     its versions), and the count of its six versions at 10:27;
//   - Weekly Digest #141 gets its NPS poll and 2,412 answers: NPS 26,
//     318 comments, 74 changed answers, 9 from the web archive and 1,377
//     automated visits ignored;
//   - the survey "Beta feedback · October 2026", linked from the sent
//     campaign "Beta Program: October check-in" whose poll answers its
//     first question: 688 started, 571 completed.
func init() { seedPolls = seedPollsImpl }

// Block ids of #143's conditional blocks and its poll.
const (
	x11Missed = "missed"
	x11Beta   = "beta"
	x11Trial  = "trial"
	x11Topics = "topics"
)

// x11Topics143 is the topics poll of Weekly Digest #143.
var x11Topics143 = polls.PollConfig{ID: x11Topics, Kind: polls.KindChoice, Question: "What should we write about next?",
	Answers: []polls.PollAnswerConfig{{Label: "Postgres internals", Tag: "wants:postgres"}, {Label: "Feature flag patterns", Tag: "wants:flags"},
		{Label: "On-call stories", Tag: "wants:oncall"}, {Label: "Rust in production", Tag: "wants:rust"}},
	Layout: "stacked", Save: polls.SaveTag, Close: polls.PollClose{Mode: polls.CloseDays, Days: 7}, After: "thanks",
	Thanks: "Thanks! We read every answer, and the next issue follows the winner.", NonContacts: "count", Results: true}

// x11NPS141 is the NPS poll of Weekly Digest #141.
var x11NPS141 = polls.PollConfig{ID: "nps141", Kind: polls.KindScale, Question: "How likely are you to recommend the Weekly Digest to a colleague?",
	Save: polls.SaveField, Field: "nps_score", Close: polls.PollClose{Mode: polls.CloseDays, Days: 14}, After: "thanks",
	Thanks: "Thanks! Your answer helps us decide what to write next.", NonContacts: "count", NPS: true, Low: "Not likely", High: "Very likely"}

func x11Rule(id string, c visibility.VisibilityCondition) string {
	r := visibility.VisibilityRule{ID: id, Show: visibility.ShowOnly, Match: visibility.MatchAll, Conditions: []visibility.VisibilityCondition{c}}
	return html.EscapeString(r.Encode())
}

// x11Digest143 adds the brief's blocks to a version of Weekly Digest #143:
// the "We've missed you" note after the hero (the phase-2 sample's
// sections keep their places), the Beta corner, the free trial banner and
// the topics poll before the social links.
func x11Digest143(s *Seeder, src string) string {
	plex := composerPlexFont
	missed := `    <mj-section rc-name="We&#39;ve missed you" rc-visible="` + x11Rule(x11Missed, visibility.VisibilityCondition{Kind: visibility.KindTier, Op: "is", Value: "at_risk"}) +
		`" background-color="#ffffff" padding="16px 24px 0px">
      <mj-column>
        <mj-text font-size="14px" line-height="1.55" color="#374151" container-background-color="#F0FDFA" padding="12px 14px">We've missed you. Here's the short version of the month, in two minutes. Too much email? <a href="{{ preferences_url }}" style="color:#0F766E">Choose how often you hear from us</a>.</mj-text>
      </mj-column>
    </mj-section>
`
	beta := `    <mj-section rc-name="Beta corner" rc-visible="` + x11Rule(x11Beta, visibility.VisibilityCondition{Kind: visibility.KindList, Op: "includes", Value: s.Lists["Beta Program"].String(), Label: "Beta Program"}) +
		`" background-color="#ffffff" padding="8px 24px 16px">
      <mj-column background-color="#F5F3FF" padding="16px 18px" border-radius="8px">
        <mj-text font-size="12px" font-weight="700" color="#5B21B6" padding="0px 0px 6px">Beta corner</mj-text>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Flag rules by region are in beta</mj-text>
        <mj-text font-size="14px" line-height="1.55" color="#374151" padding="0px 0px 12px">Target a rollout by country or data region. Try it before general release, and tell us what's confusing.</mj-text>
        <mj-button href="https://acme.io/beta/region-rules" align="left" background-color="#0F766E" color="#ffffff" font-family="` + plex + `" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">Try region rules</mj-button>
      </mj-column>
    </mj-section>
`
	trial := `    <mj-section rc-name="Start a free trial" rc-visible="` + x11Rule(x11Trial, visibility.VisibilityCondition{Kind: visibility.KindField, Field: "plan", Op: "is_empty"}) +
		`" background-color="#ffffff" padding="8px 24px 16px">
      <mj-column background-color="#0B1220" padding="18px 20px" border-radius="8px">
        <mj-text font-size="15px" line-height="1.5" color="#ffffff" padding="0px 0px 12px">Run your own flags on Acme Cloud, free for 14 days.</mj-text>
        <mj-button href="https://acme.io/cloud/trial" align="left" background-color="#ffffff" color="#0B1220" font-family="` + plex + `" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">Start a free trial</mj-button>
      </mj-column>
    </mj-section>
`
	poll := "    " + strings.ReplaceAll(polls.Section(x11Topics143, nil), "\n", "\n    ") + "\n"
	if i := strings.Index(src, "    <!-- rc:intro -->"); i >= 0 {
		src = src[:i] + missed + src[i:]
	}
	if i := strings.Index(src, `    <mj-raw><table align="center"`); i >= 0 {
		src = src[:i] + beta + trial + poll + src[i:]
	}
	return src
}

// x11Versions143 are #143's versions as the brief counts them (6 of the 8
// combinations): the blocks each leaves out, and how many get it.
var x11Versions143 = []struct {
	hidden string
	n      int64
}{
	{"beta,missed,trial", 9812}, {"beta,missed", 16940}, {"beta,trial", 1166}, {"beta", 2284}, {"missed,trial", 812}, {"missed", 590},
}

func seedPollsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	if _, ok := s.Lists["Weekly Digest"]; !ok {
		return nil
	}
	// The fields the sample's answers fill.
	for i, f := range []struct{ key, label, typ, opts string }{
		{"nps_score", "NPS score", "number", `[]`},
		{"research_ok", "Research OK", "select", `["Yes","No"]`},
		{"role", "Role", "select", `["Engineer","Engineering manager","Product","Other"]`},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type, options, display, position, created_at) VALUES ($1,$2,$3,$4,$5,$6,'',$7,$8)
			ON CONFLICT (workspace_id, key) DO NOTHING`, ID(), ws.ID, f.key, f.label, f.typ, json.RawMessage(f.opts), 10+i, s.designDay(9, 10, 0)); err != nil {
			return err
		}
	}
	steps := []func(context.Context, *Seeder) error{x11Count143, x11Poll141, x11Survey}
	for _, step := range steps {
		if err := step(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// x11Count143 keeps the count of #143's versions from 10:27, so the
// composer and Review & send show the brief's numbers until the rules or
// the audience change (then the demo's own contacts are counted). Its
// topics poll exists as a test send would have created it.
func x11Count143(ctx context.Context, s *Seeder) error {
	id, ok := s.Campaigns["Weekly Digest #143"]
	if !ok {
		return nil
	}
	ws := s.WS()
	var src string
	var audience json.RawMessage
	if err := s.Tx.QueryRow(ctx, `SELECT mjml, audience FROM campaigns WHERE id = $1`, id).Scan(&src, &audience); err != nil {
		return err
	}
	if !strings.Contains(src, "rc-visible") {
		return nil
	}
	rules, aud := visibility.CountKeys(src, audience)
	type stored struct {
		Hidden     string `json:"hidden"`
		Recipients int64  `json:"recipients"`
	}
	var vs []stored
	shown := map[string]int64{x11Missed: 0, x11Beta: 0, x11Trial: 0}
	var total int64
	for _, v := range x11Versions143 {
		vs = append(vs, stored{v.hidden, v.n})
		total += v.n
		hidden := visibility.ParseHidden(v.hidden)
		for b := range shown {
			if !hidden[b] {
				shown[b] += v.n
			}
		}
	}
	vb, _ := json.Marshal(vs)
	sb, _ := json.Marshal(shown)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO visibility_counts (campaign_id, workspace_id, rules_hash, audience_hash, total, versions, shown, factors, counted_at, duration_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'{}',$8,640) ON CONFLICT (campaign_id) DO UPDATE SET rules_hash = EXCLUDED.rules_hash, audience_hash = EXCLUDED.audience_hash,
		total = EXCLUDED.total, versions = EXCLUDED.versions, shown = EXCLUDED.shown, counted_at = EXCLUDED.counted_at`,
		id, ws.ID, rules, aud, total, vb, sb, s.sampleToday(10, 27)); err != nil {
		return err
	}
	_, err := (&polls.Service{Now: func() time.Time { return s.sampleToday(10, 41) }}).Sync(ctx, s.Q, ws.ID, id, src, nil, ws.Timezone)
	return err
}

// x11Logo is the library's Acme logo, "" without it.
func x11Logo(ctx context.Context, s *Seeder) string {
	var logoID uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM media_assets WHERE workspace_id = $1 AND file_name = 'acme-logo.png' LIMIT 1`, s.WS().ID).Scan(&logoID); err == nil {
		return media.URL(logoID, "acme-logo.png")
	}
	return ""
}

// x11Digest141 is Weekly Digest #141 as the archive shows it, with its NPS
// poll before the footer.
func x11Digest141(ctx context.Context, s *Seeder) (string, error) {
	logo := x11Logo(ctx, s)
	src := formsIssueMJML(formsIssue{campaign: "Weekly Digest #141", slug: "the-case-for-boring-infrastructure", label: "#141",
		title:   "The case for boring infrastructure",
		excerpt: "Why we moved three services back to Postgres, a reader's guide to feature flags, and the community links of the week."}, logo)
	poll := "    " + strings.ReplaceAll(polls.Section(x11NPS141, nil), "\n", "\n    ") + "\n"
	return strings.Replace(src, composerDigestFooter, poll+composerDigestFooter, 1), nil
}

// x11Contact is a contact who can answer: their tier and language.
type x11Contact struct {
	id   uuid.UUID
	tier string
	lang string
}

func x11Members(ctx context.Context, s *Seeder, list uuid.UUID) ([]x11Contact, error) {
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.tier, lower(split_part(replace(coalesce(c.language, ''), '_', '-'), '-', 1))
		FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $2 AND m.status = 'subscribed'
		WHERE c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL ORDER BY c.id`, s.WS().ID, list)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (x11Contact, error) {
		var c x11Contact
		err := r.Scan(&c.id, &c.tier, &c.lang)
		return c, err
	})
}

func (s *Seeder) contactByEmail(ctx context.Context, email string) (uuid.UUID, bool) {
	var id uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, s.WS().ID, email).Scan(&id)
	return id, err == nil
}

// x11Answer is one seeded poll answer.
type x11Answer struct {
	contact  uuid.UUID // Nil: someone who isn't a contact
	answer   int
	previous int // -1: never changed
	comment  string
	at       time.Time
	changed  time.Time
}

// x11NamedComments are the comments the PollResults board shows by name.
var x11NamedComments = []struct {
	email, comment string
	answer         int
	day, h, m      int
}{
	{"priya.n@gmail.com", "The Postgres pieces are the best thing in my inbox.", 10, 0, 9, 14},
	{"omar.haddad@gmail.com", "Short enough to read on the train. Keep the links section.", 9, 0, 9, 31},
	{"d.alvarez@yahoo.com", "Good, but every week is a lot. Monthly would suit me.", 6, 0, 11, 2},
}

// Comments of the other answers, by group.
var (
	x11PromoterComments = []string{"The on-call charts were the best part.", "I forward this to my whole team every week.", "Clear, short and always useful.",
		"The links section alone is worth it.", "Love the deep dives on Postgres.", "Best engineering newsletter I read.", "Keep the migration stories coming.",
		"Concise and practical, thank you.", "The incident write-ups are gold.", "Always one thing I can use the same day."}
	x11PassiveComments = []string{"Good, though some weeks feel long.", "More code samples, please.", "I skim most of it, but the deep dives are great.",
		"Useful, but I'd like fewer product announcements.", "Solid. A bit Postgres-heavy lately.", "Fine as is; I read it on Fridays."}
	x11DetractorComments = []string{"Too much about feature flags lately.", "It arrives at a bad time for me in Europe.", "Too long to read every week.",
		"Mostly product news now, less engineering.", "I'd rather get it once a month.", "The links don't load well on my phone."}
)

// x11Poll141 adds #141's NPS poll and its answers as of today (the brief's
// Wed 15 Oct, 10:00): 2,412 answers from the Weekly Digest list.
func x11Poll141(ctx context.Context, s *Seeder) error {
	id, ok := s.Campaigns["Weekly Digest #141"]
	if !ok {
		return nil
	}
	ws := s.WS()
	src, err := x11Digest141(ctx, s)
	if err != nil {
		return err
	}
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return fmt.Errorf("compile Weekly Digest #141: %s", res.Errors[0].Message)
	}
	var started time.Time
	if err := s.Tx.QueryRow(ctx, `UPDATE campaigns SET mjml = $2, html = $3, plain_text = $4 WHERE id = $1 RETURNING started_at`,
		id, src, res.HTML, contentcheck.PlainText(res.HTML)).Scan(&started); err != nil {
		return err
	}
	synced, err := (&polls.Service{Now: func() time.Time { return started.Add(-time.Hour) }}).Sync(ctx, s.Q, ws.ID, id, src, &started, ws.Timezone)
	if err != nil || len(synced) == 0 {
		return err
	}
	pollID := synced[0].ID
	members, err := x11Members(ctx, s, s.Lists["Weekly Digest"])
	if err != nil {
		return err
	}
	rng := audRand(141, 2412)
	// Who answered, by tier and language (x11Allocate): Active 1,932,
	// Cooling 341, At risk 125, Sunset 14; English 2,108, Spanish 201,
	// German 82, Vietnamese 21; NPS 31, 12, −8 by tier and 27, 19, 22 by
	// language, 26 overall (1,086 promoters, 458 detractors).
	type cell = x11Cell
	byCell := map[cell][]uuid.UUID{}
	cellOf := map[uuid.UUID]cell{}
	named := map[uuid.UUID]bool{}
	for _, nc := range x11NamedComments {
		if cid, ok := s.contactByEmail(ctx, nc.email); ok {
			named[cid] = true
		}
	}
	sam, hasSam := s.contactByEmail(ctx, "sam.rivera@outlook.com")
	if hasSam {
		named[sam] = true
	}
	for _, m := range members {
		// Readers of the board's four languages only: the language
		// breakdown has no "Not set" or French row.
		if !slices.Contains(x11Tiers, m.tier) || !slices.Contains(x11Langs, m.lang) {
			continue
		}
		k := cell{m.tier, m.lang}
		cellOf[m.id] = k
		if !named[m.id] {
			byCell[k] = append(byCell[k], m.id)
		}
	}
	// Scores: exact counts of the board (0: 41 … 10: 597).
	counts := []int{41, 12, 18, 27, 39, 128, 193, 352, 516, 489, 597}
	pool := map[string][]int{}
	for score, n := range counts {
		g := "detractor"
		if score >= 9 {
			g = "promoter"
		} else if score >= 7 {
			g = "passive"
		}
		for i := 0; i < n; i++ {
			pool[g] = append(pool[g], score)
		}
	}
	for _, g := range pool {
		rng.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	}
	take := func(g string, want int) int {
		// The named answers take their own scores first.
		for i, v := range pool[g] {
			if v == want {
				pool[g] = append(pool[g][:i], pool[g][i+1:]...)
				return v
			}
		}
		v := pool[g][0]
		pool[g] = pool[g][1:]
		return v
	}
	var answers []x11Answer
	at := func(minutes float64) time.Time { return started.Add(time.Duration(minutes * float64(time.Minute))) }
	// The board's named answers.
	for _, nc := range x11NamedComments {
		cid, ok := s.contactByEmail(ctx, nc.email)
		if !ok {
			continue
		}
		g := "promoter"
		if nc.answer < 7 {
			g = "detractor"
		}
		t := s.designDay(5, nc.h, nc.m)
		answers = append(answers, x11Answer{contact: cid, answer: take(g, nc.answer), previous: -1, comment: nc.comment, at: t})
	}
	if hasSam {
		// "Changed from 7 to 9" at 09:22, with a comment.
		answers = append(answers, x11Answer{contact: sam, answer: take("promoter", 9), previous: 7, comment: "More of the on-call charts, please. They're the first thing I share with my team.",
			at: s.designDay(5, 9, 11), changed: s.designDay(5, 9, 22)})
	}
	// The named answers count in their readers' cells.
	used := map[cell][2]int{}
	usedN := map[cell]int{}
	for _, a := range answers {
		k, ok := cellOf[a.contact]
		if !ok {
			continue
		}
		u := used[k]
		if a.answer >= 9 {
			u[0]++
		} else if a.answer <= 6 {
			u[1]++
		}
		used[k] = u
		usedN[k]++
	}
	avail := map[cell]int{}
	for k, ids := range byCell {
		avail[k] = len(ids)
	}
	quota, groups, err := x11Allocate(avail, usedN, used)
	if err != nil {
		return fmt.Errorf("poll 141: %w", err)
	}
	for _, t := range x11Tiers {
		for _, l := range x11Langs {
			k := cell{t, l}
			ids := byCell[k]
			rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
			p, d := groups[k][0], groups[k][1]
			for i := 0; i < quota[k]; i++ {
				g := "passive"
				switch {
				case i < p:
					g = "promoter"
				case i < p+d:
					g = "detractor"
				}
				answers = append(answers, x11Answer{contact: ids[i], answer: take(g, -1), previous: -1})
			}
		}
	}
	// When: 61% in the first 6 hours, 24 after 72 hours, the rest between.
	rng.Shuffle(len(answers), func(i, j int) { answers[i], answers[j] = answers[j], answers[i] })
	for i := range answers {
		if !answers[i].at.IsZero() {
			continue
		}
		switch r := rng.Float64(); {
		case i%100 < 61:
			answers[i].at = at(4 + r*r*350)
		case i < 24:
			answers[i].at = at(72*60 + 30 + r*36*60)
		default:
			answers[i].at = at(6*60 + r*66*60)
		}
	}
	x11FixWindows(answers, started, rng)
	// 74 changed their answer (Sam among them); 318 comments (151 from
	// promoters, 96 from passives, 71 from detractors), the named ones and
	// the one from the archive included.
	changedLeft := 74
	comments := map[string]int{"promoter": 151, "passive": 96, "detractor": 70}
	for _, a := range answers {
		if a.comment != "" {
			comments[x11Group(a.answer)]--
		}
		if a.previous >= 0 {
			changedLeft--
		}
	}
	for i := range answers {
		a := &answers[i]
		if a.comment != "" || a.previous >= 0 {
			continue
		}
		g := x11Group(a.answer)
		if comments[g] > 0 && rng.Float64() < 0.16 {
			comments[g]--
			switch g {
			case "promoter":
				a.comment = x11PromoterComments[rng.IntN(len(x11PromoterComments))]
			case "passive":
				a.comment = x11PassiveComments[rng.IntN(len(x11PassiveComments))]
			default:
				a.comment = x11DetractorComments[1+rng.IntN(len(x11DetractorComments)-1)]
			}
		} else if changedLeft > 0 && rng.Float64() < 0.04 {
			changedLeft--
			a.previous = x11Neighbour(a.answer, rng)
			a.changed = a.at.Add(time.Duration(5+rng.IntN(600)) * time.Minute)
		}
	}
	for g, n := range comments {
		for i := range answers {
			if n == 0 {
				break
			}
			a := &answers[i]
			if a.comment == "" && x11Group(a.answer) == g && a.previous < 0 {
				a.comment = map[string][]string{"promoter": x11PromoterComments, "passive": x11PassiveComments, "detractor": x11DetractorComments[1:]}[g][i%5]
				n--
			}
		}
	}
	for i := range answers {
		if changedLeft == 0 {
			break
		}
		a := &answers[i]
		if a.previous < 0 && a.comment == "" {
			a.previous = x11Neighbour(a.answer, rng)
			a.changed = a.at.Add(time.Duration(5+rng.IntN(600)) * time.Minute)
			changedLeft--
		}
	}
	// Nine from the web archive, counted apart; one with a comment.
	for i, v := range []int{3, 8, 9, 10, 7, 6, 9, 10, 8} {
		a := x11Answer{answer: v, previous: -1, at: s.designDay(4, 7+i, 40)}
		if i == 0 {
			a.comment = "Too much about feature flags lately."
		}
		answers = append(answers, a)
	}
	if err := x11WriteAnswers(ctx, s, pollID, answers, x11NPS141.Field); err != nil {
		return err
	}
	for _, v := range []struct {
		reason string
		n      int64
	}{{"gateway", 1129}, {"prefetch", 248}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO poll_automated_visits (poll_id, reason, n, last_at) VALUES ($1,$2,$3,$4)`, pollID, v.reason, v.n, s.designDay(1, 6, 12)); err != nil {
			return err
		}
	}
	return nil
}

// x11Cell is a group of readers: an engagement tier and a language.
type x11Cell struct{ tier, lang string }

var (
	x11Tiers = []string{"active", "cooling", "at_risk", "sunset"}
	x11Langs = []string{"vi", "de", "es", "en"}
)

// x11Allocate splits #141's answers over tiers and languages: how many
// answers each group of readers gives (at most avail) and how many of them
// are promoters and detractors, so that the board's counts by tier and by
// language hold and each NPS rounds to the board's (Active 31, Cooling 12,
// At risk −8; English 27, Spanish 19, German 22), the named answers
// (taken, picked) included. Only the nets (promoters less detractors) are
// fixed; how many readers of a group are passive is free, which leaves
// room for the demo's German readers, most of them at risk.
func x11Allocate(avail, taken map[x11Cell]int, picked map[x11Cell][2]int) (map[x11Cell]int, map[x11Cell][2]int, error) {
	tierQ := map[string]int{"active": 1932, "cooling": 341, "at_risk": 125, "sunset": 14}
	langQ := map[string]int{"vi": 21, "de": 82, "es": 201, "en": 2108}
	tierNet := map[string]int{"active": 599, "cooling": 41, "at_risk": -10, "sunset": -2}
	langNet := map[string]int{"vi": 3, "de": 18, "es": 38, "en": 569}
	promoters, detractors := 1086, 458
	for k, n := range taken {
		tierQ[k.tier] -= n
		langQ[k.lang] -= n
		p := picked[k]
		tierNet[k.tier] -= p[0] - p[1]
		langNet[k.lang] -= p[0] - p[1]
		promoters, detractors = promoters-p[0], detractors-p[1]
	}
	// How many answers each group gives: the other languages over the
	// tiers in order, English the rest of each tier.
	quota := map[x11Cell]int{}
	for _, l := range x11Langs[:3] {
		for _, t := range x11Tiers {
			k := x11Cell{t, l}
			n := min(langQ[l], tierQ[t], avail[k])
			quota[k] = n
			langQ[l] -= n
			tierQ[t] -= n
		}
		if langQ[l] > 0 {
			return nil, nil, fmt.Errorf("%d more %s readers are needed", langQ[l], l)
		}
	}
	for _, t := range x11Tiers {
		k := x11Cell{t, "en"}
		if tierQ[t] > avail[k] {
			return nil, nil, fmt.Errorf("%d %s English readers are needed, %d available", tierQ[t], t, avail[k])
		}
		quota[k] = tierQ[t]
		langQ["en"] -= tierQ[t]
	}
	if langQ["en"] != 0 {
		return nil, nil, fmt.Errorf("English answers off by %d", langQ["en"])
	}
	// Each group's net: the other languages' one point at a time, where
	// English keeps the most room; English takes what each tier has left.
	net := map[x11Cell]int{}
	enNet := func(t string) int {
		n := tierNet[t]
		for _, l := range x11Langs[:3] {
			n -= net[x11Cell{t, l}]
		}
		return n
	}
	for _, l := range x11Langs[:3] {
		step := 1
		if langNet[l] < 0 {
			step = -1
		}
		for range x11Abs(langNet[l]) {
			best, room := "", 0
			for _, t := range x11Tiers {
				k := x11Cell{t, l}
				if x11Abs(net[k]+step) > quota[k] {
					continue
				}
				if r := quota[x11Cell{t, "en"}] - x11Abs(enNet(t)-step); best == "" || r > room {
					best, room = t, r
				}
			}
			if best == "" {
				return nil, nil, fmt.Errorf("the %s readers can't give an NPS of their own", l)
			}
			net[x11Cell{best, l}] += step
		}
	}
	sum := 0
	for _, t := range x11Tiers {
		k := x11Cell{t, "en"}
		net[k] = enNet(t)
		if x11Abs(net[k]) > quota[k] {
			return nil, nil, fmt.Errorf("the %s English readers can't give a net of %d", t, net[k])
		}
	}
	for _, n := range net {
		sum += n
	}
	if sum != promoters-detractors {
		return nil, nil, fmt.Errorf("promoters and detractors off by %d", promoters-detractors-sum)
	}
	// Then pairs of a promoter and a detractor, which leave the nets as
	// they are, over the groups by the room they have.
	cells := make([]x11Cell, 0, len(quota))
	for _, t := range x11Tiers {
		for _, l := range x11Langs {
			cells = append(cells, x11Cell{t, l})
		}
	}
	pd := map[x11Cell][2]int{}
	pairs, spare := promoters+detractors, 0
	for _, k := range cells {
		pairs -= x11Abs(net[k])
		spare += (quota[k] - x11Abs(net[k])) / 2
	}
	if pairs < 0 || pairs%2 != 0 || pairs/2 > spare {
		return nil, nil, fmt.Errorf("%d promoters and %d detractors don't fit", promoters, detractors)
	}
	pairs /= 2
	given, rest := 0, make([]struct {
		k    x11Cell
		frac float64
	}, 0, len(cells))
	for _, k := range cells {
		room := (quota[k] - x11Abs(net[k])) / 2
		share := 0.0
		if spare > 0 {
			share = float64(room) * float64(pairs) / float64(spare)
		}
		n := int(share)
		pd[k] = [2]int{max(net[k], 0) + n, max(-net[k], 0) + n}
		given += n
		if n < room {
			rest = append(rest, struct {
				k    x11Cell
				frac float64
			}{k, share - float64(n)})
		}
	}
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].frac > rest[j].frac })
	for _, r := range rest {
		if given == pairs {
			break
		}
		p := pd[r.k]
		pd[r.k] = [2]int{p[0] + 1, p[1] + 1}
		given++
	}
	if given != pairs {
		return nil, nil, fmt.Errorf("%d promoter and detractor pairs are left over", pairs-given)
	}
	return quota, pd, nil
}

func x11Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func x11Group(score int) string {
	switch {
	case score >= 9:
		return "promoter"
	case score >= 7:
		return "passive"
	}
	return "detractor"
}

// x11Neighbour is the answer a changed answer had before: a point away.
func x11Neighbour(score int, rng *rand.Rand) int {
	if score == 0 || (score < 10 && rng.IntN(2) == 0) {
		return score + 1
	}
	return score - 1
}

// x11FixWindows moves answers so 61% fall in the first 6 hours and 24
// after 72 hours, exactly.
func x11FixWindows(answers []x11Answer, started time.Time, rng *rand.Rand) {
	six, late := started.Add(6*time.Hour), started.Add(72*time.Hour)
	wantSix := int(float64(len(answers))*0.61 + 0.5)
	idx := make([]int, len(answers))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return answers[idx[a]].at.Before(answers[idx[b]].at) })
	for rank, i := range idx {
		t := answers[i].at
		switch {
		case rank < wantSix && !t.Before(six):
			answers[i].at = started.Add(time.Duration(4+rng.IntN(350)) * time.Minute)
		case rank >= wantSix && rank < len(idx)-24 && (t.Before(six) || !t.Before(late)):
			answers[i].at = six.Add(time.Duration(1+rng.IntN(66*60-2)) * time.Minute)
		case rank >= len(idx)-24 && t.Before(late):
			answers[i].at = late.Add(time.Duration(30+rng.IntN(36*60)) * time.Minute)
		}
	}
}

// x11WriteAnswers writes a poll's answers, their history and the field
// they fill on the contacts.
func x11WriteAnswers(ctx context.Context, s *Seeder, pollID uuid.UUID, answers []x11Answer, field string) error {
	ws := s.WS()
	var rows, events [][]any
	var fieldIDs []uuid.UUID
	var fieldValues []int32
	for _, a := range answers {
		aid := ID()
		kind := polls.AnswererContact
		var contact *uuid.UUID
		if a.contact != uuid.Nil {
			c := a.contact
			contact = &c
			fieldIDs, fieldValues = append(fieldIDs, c), append(fieldValues, int32(a.answer))
		} else {
			kind = polls.AnswererNonContact
		}
		first, changes, updated := a.answer, 0, a.at
		if a.previous >= 0 {
			first, changes, updated = a.previous, 1, a.changed
		}
		var commented *time.Time
		if a.comment != "" {
			t := updated.Add(40 * time.Second)
			commented, updated = &t, t
		}
		rows = append(rows, []any{aid, pollID, ws.ID, contact, kind, a.answer, first, changes, a.comment, commented, "Gmail", a.at, updated})
		events = append(events, []any{aid, pollID, "answered", int32(first), nil, a.at})
		if a.previous >= 0 {
			events = append(events, []any{aid, pollID, "changed", int32(a.answer), int32(a.previous), a.changed})
		}
		if commented != nil {
			events = append(events, []any{aid, pollID, "commented", int32(a.answer), nil, *commented})
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"poll_answers"}, []string{"id", "poll_id", "workspace_id", "contact_id", "kind", "answer", "first_answer", "changes", "comment",
		"commented_at", "client", "answered_at", "updated_at"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("poll answers: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"poll_answer_events"}, []string{"answer_id", "poll_id", "kind", "answer", "previous", "at"}, pgx.CopyFromRows(events)); err != nil {
		return fmt.Errorf("poll answer events: %w", err)
	}
	if field == "" {
		return nil
	}
	_, err := s.Tx.Exec(ctx, `UPDATE contacts c SET attributes = jsonb_set(c.attributes, ARRAY[$3::text], to_jsonb(v.answer), true)
		FROM unnest($1::uuid[], $2::int[]) AS v(id, answer) WHERE c.id = v.id`, fieldIDs, fieldValues, field)
	return err
}

// x11SurveyQuestions are the six questions of "Beta feedback · October 2026".
var x11SurveyQuestions = []polls.SurveyQuestion{
	{Key: "q1", Type: polls.QuestionSingle, Text: "How often do you use feature flags?", Required: true,
		Answers: []polls.SurveyOption{{Key: 1, Label: "Daily"}, {Key: 2, Label: "Weekly"}, {Key: 3, Label: "Monthly"}, {Key: 4, Label: "Not yet"}}, Save: polls.SurveySave{To: "none"}},
	{Key: "q2", Type: polls.QuestionMulti, Text: "Which beta features have you tried?", Required: true,
		Answers: []polls.SurveyOption{{Key: 1, Label: "Feature flags"}, {Key: 2, Label: "Audit log exports"}, {Key: 3, Label: "Usage alerts"}, {Key: 4, Label: "API v2"}}, Save: polls.SurveySave{To: "none"}},
	{Key: "q3", Type: polls.QuestionScale, Text: "How likely are you to recommend Acme to a colleague?", Required: true, NPS: true, Low: "Not likely", High: "Very likely",
		Answers: []polls.SurveyOption{}, Save: polls.SurveySave{To: "none"}},
	{Key: "q4", Type: polls.QuestionLong, Text: "What's the one thing we should fix before general release?", Answers: []polls.SurveyOption{}, Save: polls.SurveySave{To: "none"}},
	{Key: "q5", Type: polls.QuestionSingle, Text: "Can we contact you about your answers?", Required: true,
		Answers: []polls.SurveyOption{{Key: 1, Label: "Yes"}, {Key: 2, Label: "No"}}, Save: polls.SurveySave{To: "field", Field: "research_ok"}},
	{Key: "q6", Type: polls.QuestionSingle, Text: "Your role", Required: true, RequiredText: "Pick the one closest to your job.",
		Answers: []polls.SurveyOption{{Key: 1, Label: "Engineer"}, {Key: 2, Label: "Engineering manager"}, {Key: 3, Label: "Product"}, {Key: 4, Label: "Other"}},
		Save:    polls.SurveySave{To: "field", Field: "role"}},
}

// x11FixTexts are answers to question 4.
var x11FixTexts = []string{
	"Flag rules by region are confusing in the UI.", "Let me schedule audit log exports to S3, not just run them once.",
	"Usage alerts fire twice when a limit is crossed at midnight UTC.", "The API v2 docs need a webhook migration example.",
	"Bulk edit for flags across environments.", "Show who changed a flag in the flag list itself.", "Rate limits on API v2 are hard to find in the docs.",
	"The export download link expires too quickly.", "Let alerts go to Slack, not only email.", "A dry run for rollouts would help a lot.",
	"Faster search in the audit log.", "Make the SDK setup page shorter.", "Flag stale-ness warnings, please.", "Clearer billing for usage alerts.",
}

// x11Respondent is one response of the sample survey.
type x11Respondent struct {
	contact   uuid.UUID
	email     string
	answers   map[string]any
	pollAt    time.Time
	completed *time.Time
}

// x11Survey adds the survey "Beta feedback · October 2026", the campaign
// "Beta Program: October check-in" that links to it through its poll, and
// their answers as of today.
func x11Survey(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	beta, ok := s.Lists["Beta Program"]
	if !ok {
		return nil
	}
	sofia := s.Users["sofia@acme.io"]
	loc := s.newYork()
	// Before the email, which goes out the day after Weekly Digest #141
	// and before the October Product Update (still the latest send).
	created, published := s.designDay(4, 9, 5), s.designDay(4, 9, 40)
	closes := s.designDay(-16, 23, 59)
	sid := ID()
	settings := polls.SurveySettings{Close: polls.SurveyClose{Mode: "date", Date: closes.In(loc).Format("2006-01-02"), Time: "23:59"}, Timezone: "America/New_York",
		OneResponse: true, AcceptNonContacts: false, Thanks: "Your answers are saved. You can change them until the survey closes."}
	qb, _ := json.Marshal(x11SurveyQuestions)
	sb, _ := json.Marshal(settings)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO surveys (id, workspace_id, slug, name, description, status, questions, settings, closes_at, published_at, published_by,
		created_by, updated_by, created_at, updated_at) VALUES ($1,$2,'beta-feedback-oct-2026','Beta feedback · October 2026',$3,'open',$4,$5,$6,$7,$8,$8,$8,$9,$7)`,
		sid, ws.ID, "Six questions, about three minutes. Your answers go to Acme's product team and are linked to your beta account.", qb, sb, closes, published, sofia.ID, created); err != nil {
		return err
	}
	var number int64
	if err := s.Tx.QueryRow(ctx, `SELECT number FROM surveys WHERE id = $1`, sid).Scan(&number); err != nil {
		return err
	}
	for _, a := range []struct {
		at     time.Time
		action string
		after  map[string]any
	}{
		{created, "survey.created", map[string]any{"name": "Beta feedback · October 2026", "slug": "beta-feedback-oct-2026"}},
		{published, "survey.published", map[string]any{"status": "open"}},
	} {
		if err := readinessAudit(ctx, s, a.at, sofia, a.action, "survey", sid.String(), "Beta feedback · October 2026", a.after, map[string]any{"permission": "forms:edit"}, "198.51.100.23"); err != nil {
			return err
		}
	}
	// The campaign: a poll of question 1 that continues to the survey.
	if err := seedSent(ctx, s, sentCampaign{name: "Beta Program: October check-in", audience: "Beta Program", at: 4, hour: 10,
		perProvider: map[string]int64{"ses": 1885, "postal": 209}, delivered: 99.5, humanOpens: 61.4, machineOpens: 15.2, clicks: 18.3, bounce: 0.3, complaint: 0, unsubscribes: 0.05},
		map[string]any{"lists": []uuid.UUID{beta}}); err != nil {
		return err
	}
	cid := s.Campaigns["Beta Program: October check-in"]
	q1 := polls.PollConfig{ID: "beta1q", Kind: polls.KindChoice, Question: "How often do you use feature flags?",
		Answers: []polls.PollAnswerConfig{{Label: "Daily"}, {Label: "Weekly"}, {Label: "Monthly"}, {Label: "Not yet"}}, Layout: "row", Save: polls.SaveNone,
		Close: polls.PollClose{Mode: polls.CloseDate, Date: settings.Close.Date}, After: "survey", Survey: sid.String(), NonContacts: "refuse"}
	src := `<mjml>
` + fmt.Sprintf(composerDigestHead, "Six questions about the beta", "Start with one click: how often do you use feature flags?") + `
  <mj-body width="600px" background-color="#F3F4F6">
` + strings.ReplaceAll(composerDigestHeader, "@@acme-logo.png@@", x11Logo(ctx, s)) + `
    <mj-section background-color="#ffffff" padding="24px 24px 8px">
      <mj-column>
        <mj-text font-size="15px" line-height="1.55">Hi {{ first_name | default: "there" }}, thanks for testing the beta. Six quick questions, starting with this one:</mj-text>
      </mj-column>
    </mj-section>
    ` + strings.ReplaceAll(polls.Section(q1, nil), "\n", "\n    ") + `
` + composerDigestFooter + `
  </mj-body>
</mjml>
`
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return fmt.Errorf("compile the beta check-in: %s", res.Errors[0].Message)
	}
	var started time.Time
	if err := s.Tx.QueryRow(ctx, `UPDATE campaigns SET subject = 'Six questions about the beta', preheader = 'Start with one click: how often do you use feature flags?',
		mjml = $2, html = $3, plain_text = $4 WHERE id = $1 RETURNING started_at`, cid, src, res.HTML, contentcheck.PlainText(res.HTML)).Scan(&started); err != nil {
		return err
	}
	synced, err := (&polls.Service{Now: func() time.Time { return started.Add(-time.Hour) }}).Sync(ctx, s.Q, ws.ID, cid, src, &started, ws.Timezone)
	if err != nil || len(synced) == 0 {
		return err
	}
	members, err := x11Members(ctx, s, beta)
	if err != nil {
		return err
	}
	rng := audRand(688, 571)
	rng.Shuffle(len(members), func(i, j int) { members[i], members[j] = members[j], members[i] })
	// The respondents the board names come first: the newest completed
	// responses.
	today := func(h, m int) *time.Time { t := s.sampleToday(h, m); return &t }
	yesterday := func(h, m int) *time.Time { t := s.designDay(1, h, m); return &t }
	named := []struct {
		email   string
		answers map[string]any
		at      *time.Time
	}{
		{"priya.n@gmail.com", map[string]any{"q1": 1, "q2": []int{1, 2}, "q3": 9, "q4": "Flag rules by region are confusing in the UI.", "q5": 1, "q6": 1}, today(9, 42)},
		{"jonas.weber@example.de", map[string]any{"q1": 2, "q2": []int{1, 2, 4}, "q3": 8, "q4": "Let me schedule audit log exports to S3, not just run them once.", "q5": 1, "q6": 2}, today(8, 15)},
		{"omar.haddad@gmail.com", map[string]any{"q1": 2, "q2": []int{1, 3}, "q3": 10, "q4": "Usage alerts fire twice when a limit is crossed at midnight UTC.", "q5": 2, "q6": 1}, yesterday(21, 3)},
		{"kim.tran@example.com", map[string]any{"q1": 3, "q2": []int{1}, "q3": 7, "q5": 1, "q6": 3}, yesterday(18, 20)},
	}
	// Kim Tran: a Beta Program reader named for the board (no new contact).
	if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET first_name = 'Kim', last_name = 'Tran', email = 'kim.tran@example.com'
		WHERE id = (SELECT c.id FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = $2
			WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.status = 'active' AND c.email NOT IN ('priya.n@gmail.com', 'omar.haddad@gmail.com', 'sam.rivera@outlook.com')
			ORDER BY c.email DESC LIMIT 1)
		AND NOT EXISTS (SELECT 1 FROM contacts WHERE workspace_id = $1 AND email = 'kim.tran@example.com')`, ws.ID, beta); err != nil {
		return err
	}
	var resp []x11Respondent
	taken := map[uuid.UUID]bool{}
	for _, n := range named {
		id, ok := s.contactByEmail(ctx, n.email)
		if !ok {
			continue
		}
		taken[id] = true
		resp = append(resp, x11Respondent{contact: id, email: n.email, answers: n.answers, completed: n.at, pollAt: n.at.Add(-time.Duration(4+len(resp)) * time.Minute)})
	}
	nNamed := len(resp)
	for _, m := range members {
		if len(resp) >= 688 {
			break
		}
		if taken[m.id] {
			continue
		}
		resp = append(resp, x11Respondent{contact: m.id, answers: map[string]any{}})
	}
	if len(resp) < 688 {
		return fmt.Errorf("survey: 688 respondents wanted, %d Beta Program readers", len(resp))
	}
	x11SurveyAnswers(resp, nNamed, rng)
	// When: the poll answers from the send on Mon 13 Oct at 10:00; the
	// completed responses up to this morning, the named ones the newest.
	newest := s.sampleToday(9, 0)
	if nNamed > 0 {
		newest = *resp[nNamed-1].completed
	}
	for i := nNamed; i < len(resp); i++ {
		r := &resp[i]
		r.pollAt = started.Add(time.Duration(3+rng.IntN(int(newest.Sub(started).Minutes())-60)) * time.Minute)
		if i < 571 {
			done := r.pollAt.Add(time.Duration(2+rng.IntN(9)) * time.Minute)
			if !done.Before(newest) {
				done = newest.Add(-time.Duration(1+rng.IntN(600)) * time.Minute)
			}
			r.completed = &done
		}
	}
	return x11WriteSurvey(ctx, s, sid, synced[0].ID, resp)
}

// x11SurveyAnswers fills the answers of the respondents after the named
// ones: the first 571 complete the survey, the other 117 answered
// question 1 in the email and stopped.
func x11SurveyAnswers(resp []x11Respondent, named int, rng *rand.Rand) {
	// fill gives resp[from:to] the values counts says (value i+offset
	// counts[i] times), less the named respondents' own answers when the
	// range starts after them.
	fill := func(counts []int, offset, from, to int, key string) {
		left := append([]int(nil), counts...)
		if from == named {
			for _, r := range resp[:named] {
				if v, ok := r.answers[key].(int); ok && v-offset >= 0 && v-offset < len(left) {
					left[v-offset]--
				}
			}
		}
		var vals []int
		for k, n := range left {
			for i := 0; i < n; i++ {
				vals = append(vals, k+offset)
			}
		}
		rng.Shuffle(len(vals), func(i, j int) { vals[i], vals[j] = vals[j], vals[i] })
		for i := from; i < to && i-from < len(vals); i++ {
			resp[i].answers[key] = vals[i-from]
		}
	}
	// Question 1 among the completed: Daily 158 · Weekly 241 · Monthly 97 ·
	// Not yet 75; among the 117 who stopped, in about the same proportions.
	fill([]int{158, 241, 97, 75}, 1, named, 571, "q1")
	fill([]int{30, 47, 24, 16}, 1, 571, len(resp), "q1")
	fill([]int{342, 229}, 1, named, 571, "q5")
	fill([]int{361, 118, 64, 28}, 1, named, 571, "q6")
	// Question 3: 290 promoters, 225 passives, 56 detractors (NPS 41).
	fill([]int{3, 2, 3, 5, 8, 15, 20, 98, 127, 151, 139}, 0, named, 571, "q3")
	// Question 2: Feature flags 498 · Audit log exports 287 · Usage alerts
	// 164 · API v2 121, at least one each.
	want := []int{498, 287, 164, 121}
	for _, r := range resp[:named] {
		for _, k := range r.answers["q2"].([]int) {
			want[k-1]--
		}
	}
	rest := resp[named:571]
	order := rng.Perm(len(rest))
	picks := make([][]int, len(rest))
	for i, j := range order {
		if i < want[0] {
			picks[j] = append(picks[j], 1)
		}
	}
	// Those without feature flags tried audit log exports.
	for j := range rest {
		if len(picks[j]) == 0 {
			picks[j] = append(picks[j], 2)
			want[1]--
		}
	}
	for k := 1; k < 4; k++ {
		order := rng.Perm(len(rest))
		n := want[k]
		for _, j := range order {
			if n == 0 {
				break
			}
			if !slices.Contains(picks[j], k+1) {
				picks[j] = append(picks[j], k+1)
				n--
			}
		}
	}
	for j := range rest {
		slices.Sort(picks[j])
		rest[j].answers["q2"] = picks[j]
	}
	// Question 4: 412 answers (the named ones included).
	texts := 412
	for _, r := range resp[:named] {
		if _, ok := r.answers["q4"]; ok {
			texts--
		}
	}
	for i := named; i < 571 && texts > 0; i++ {
		if rng.Float64() < 0.75 || 571-i <= texts {
			resp[i].answers["q4"] = x11FixTexts[4+rng.IntN(len(x11FixTexts)-4)]
			texts--
		}
	}
}

// x11WriteSurvey writes the poll answers of the check-in, the survey's
// responses and the fields they fill.
func x11WriteSurvey(ctx context.Context, s *Seeder, surveyID, pollID uuid.UUID, resp []x11Respondent) error {
	ws := s.WS()
	var answers []x11Answer
	var rows [][]any
	research := map[uuid.UUID]string{}
	role := map[uuid.UUID]string{}
	labels := func(q polls.SurveyQuestion, v any) string { return polls.AnswerText(q, v) }
	for _, r := range resp {
		q1, _ := r.answers["q1"].(int)
		answers = append(answers, x11Answer{contact: r.contact, answer: q1, previous: -1, at: r.pollAt})
		ab, _ := json.Marshal(r.answers)
		updated := r.pollAt
		if r.completed != nil {
			updated = *r.completed
			research[r.contact] = labels(x11SurveyQuestions[4], r.answers["q5"])
			role[r.contact] = labels(x11SurveyQuestions[5], r.answers["q6"])
		}
		rows = append(rows, []any{ID(), surveyID, ws.ID, r.contact, polls.AnswererContact, ab, "poll", r.pollAt, r.completed, updated})
	}
	if err := x11WriteAnswers(ctx, s, pollID, answers, ""); err != nil {
		return err
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"survey_responses"}, []string{"id", "survey_id", "workspace_id", "contact_id", "kind", "answers", "source", "started_at",
		"completed_at", "updated_at"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("survey responses: %w", err)
	}
	var ids []uuid.UUID
	var res, roles []string
	for id, v := range research {
		ids, res, roles = append(ids, id), append(res, v), append(roles, role[id])
	}
	_, err := s.Tx.Exec(ctx, `UPDATE contacts c SET attributes = c.attributes || jsonb_build_object('research_ok', v.research, 'role', v.role)
		FROM unnest($1::uuid[], $2::text[], $3::text[]) AS v(id, research, role) WHERE c.id = v.id`, ids, res, roles)
	return err
}

// newYork is the sample's time zone.
func (s *Seeder) newYork() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return loc
}
