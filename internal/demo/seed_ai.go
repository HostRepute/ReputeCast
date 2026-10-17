package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The AI assistant (design expansion, phase 3 brief §4, "AI assistant"):
// Acme Newsletter connected Anthropic's Claude Sonnet 5.5 (Marcus Chen, Mon
// 6 Oct), at $2.00 and $10.00 per million tokens from the price list of
// 1 Oct 2026, with a $25.00 cap Linh Nguyen set. This month's 412 requests
// add up to $7.84 (1.62M input and 460K output tokens), by feature and by
// member as on AISettings, and end with the activity log's six rows. Linh
// kept a summary on October Product Update's report. Acme Engineering Blog
// has no model connected.
//
// The connection is the demo's sample: it answers from recordings in
// development and never calls Anthropic, whose key it doesn't have.
func init() { seedAI = seedAIImpl }

// AIDemoKey is the sample connection's key: a placeholder, sealed like a
// real one, that no provider would accept.
const AIDemoKey = "sk-ant-demo-recorded-0000" //nolint:gosec // a placeholder, documented as demo data

// aiAt is a moment of the AI sample, whose "now" is 10:32 on Wed 15 Oct
// (the log's latest row is 10:31): daysAgo days before, at h:m.
func (s *Seeder) aiAt(daysAgo, h, m int) time.Time {
	return s.Ago(time.Duration(daysAgo)*24*time.Hour + time.Duration(10*60+32-(h*60+m))*time.Minute)
}

// aiFeatureCost and aiMemberCost are AISettings' "this month" amounts, in
// millionths of a dollar; aiCells splits them per member and feature.
var aiCells = []struct {
	member, feature string
	micros          int64
	requests        int
}{
	{"daniel@acme.io", "subject_ideas", 610_000, 120},
	{"sofia@acme.io", "subject_ideas", 300_000, 60},
	{"daniel@acme.io", "rewrite", 1_100_000, 80},
	{"sofia@acme.io", "rewrite", 540_000, 40},
	{"sofia@acme.io", "translate", 3_860_000, 40},
	{"daniel@acme.io", "alt_text", 250_000, 20},
	{"sofia@acme.io", "alt_text", 170_000, 14},
	{"daniel@acme.io", "accessibility_fix", 90_000, 10},
	{"sofia@acme.io", "accessibility_fix", 100_000, 10},
	{"marcus@acme.io", "segment_draft", 240_000, 8},
	{"linh@acme.io", "report_summary", 440_000, 8},
	{"marcus@acme.io", "report_summary", 140_000, 2},
}

// aiOutWeight is the share of a feature's cost that is output tokens
// (translations write the most).
var aiOutWeight = map[string]float64{"subject_ideas": 0.45, "rewrite": 0.5, "translate": 0.8, "alt_text": 0.25, "accessibility_fix": 0.3, "segment_draft": 0.33, "report_summary": 0.34}

// aiRequest is one row of the activity log.
type aiRequest struct {
	at              time.Time
	member, feature string
	action          string
	target          string
	targetKind      string
	in, out         int
	outcome, detail string
	fixed           bool
}

func seedAIImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	marcus, linh := s.Users["marcus@acme.io"], s.Users["linh@acme.io"]
	if marcus.ID == uuid.Nil || linh.ID == uuid.Nil {
		return nil
	}
	sealed, err := s.Box.SealString(AIDemoKey, "ai:"+ws.ID.String()+":key")
	if err != nil {
		return err
	}
	connected := s.designDay(9, 14, 2)
	capSet := s.designDay(9, 14, 20)
	ok := true
	ms := int32(1200)
	priceDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	if err := s.Q.AIConnectionSave(ctx, store.AIConnectionSaveParams{
		WorkspaceID: ws.ID, Provider: "anthropic", Model: "claude-sonnet-5-5", ModelName: "Claude Sonnet 5.5", ImageInput: true,
		KeySealed: sealed, KeyAddedBy: &marcus.ID, KeyAddedAt: &connected, ConnectedBy: &marcus.ID, ConnectedAt: connected,
		Status: "ok", LastTestAt: &connected, LastTestOk: &ok, LastTestMs: &ms,
		CapKind: "usd", CapUsd: 25, CapSetBy: &linh.ID, CapSetAt: &capSet, InputPrice: 2, OutputPrice: 10, PricesFrom: &priceDate,
		FeaturesOff: []string{}, Demo: true, UpdatedBy: &linh.ID, UpdatedAt: capSet,
	}); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]any{"key": "added (write-only)", "test": map[string]any{"ok": true, "ms": 1200},
		"summary": "model claude-sonnet-5-5 · key added (write-only) · cap $20.00 a month · all 7 features on"})
	after, _ := json.Marshal(map[string]any{"provider": "anthropic", "model": "claude-sonnet-5-5", "capKind": "usd", "capUsd": 20, "inputPrice": 2, "outputPrice": 10,
		"features": []string{"subject_ideas", "rewrite", "translate", "alt_text", "accessibility_fix", "segment_draft", "report_summary"}, "keySet": true})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
		result, ip, user_agent, request_id, after, metadata) VALUES ($1,$2,$3,'user',$4,$5,'ai.connection_created','ai_assistant',$6,'Anthropic','allowed',$7,'',$8,$9,$10)`,
		ID(), ws.ID, connected, marcus.ID, marcus.Name, ws.ID.String(), netip.MustParseAddr("198.51.100.40"), "req_"+ID().String()[:8], after, meta); err != nil {
		return err
	}
	capBefore, _ := json.Marshal(map[string]any{"capKind": "usd", "capUsd": 20, "capTokens": 0})
	capAfter, _ := json.Marshal(map[string]any{"capKind": "usd", "capUsd": 25, "capTokens": 0})
	capMeta, _ := json.Marshal(map[string]any{"summary": "cap $20.00 a month → $25.00 a month"})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
		result, ip, user_agent, request_id, before, after, metadata) VALUES ($1,$2,$3,'user',$4,$5,'ai.cap_updated','ai_assistant',$6,'Anthropic','allowed',$7,'',$8,$9,$10,$11)`,
		ID(), ws.ID, capSet, linh.ID, linh.Name, ws.ID.String(), netip.MustParseAddr("203.0.113.9"), "req_"+ID().String()[:8], capBefore, capAfter, capMeta); err != nil {
		return err
	}
	rows := s.aiRequests(ws)
	for _, r := range rows {
		u := s.Users[r.member]
		var target *uuid.UUID
		if id, ok := s.Campaigns[r.target]; ok && r.targetKind == "campaign" || ok && r.targetKind == "report" {
			target = &id
		}
		label := r.target
		if r.feature == "alt_text" && r.fixed {
			label = r.target + ", 2 images"
		}
		if r.feature == "rewrite" && r.fixed {
			label = r.target + " subject"
		}
		cost := int64(r.in*2 + r.out*10)
		decided := r.at.Add(40 * time.Second)
		if err := s.Q.AIRequestInsert(ctx, store.AIRequestInsertParams{ID: ID(), WorkspaceID: ws.ID, UserID: &u.ID, Feature: r.feature, Action: r.action,
			TargetKind: r.targetKind, TargetID: target, TargetLabel: label, Provider: "anthropic", Model: "claude-sonnet-5-5", InputTokens: int32(r.in),
			OutputTokens: int32(r.out), CostMicros: cost, Outcome: r.outcome, OutcomeDetail: r.detail, DurationMs: int32(2200 + (r.in+r.out)%4000), CreatedAt: r.at, DecidedAt: &decided}); err != nil {
			return err
		}
	}
	if id, ok := s.Campaigns["October Product Update"]; ok {
		if err := s.Q.AIReportSummarySave(ctx, store.AIReportSummarySaveParams{CampaignID: id, WorkspaceID: ws.ID, Summary: AIOctoberSummary, Model: "claude-sonnet-5-5",
			ModelName: "Claude Sonnet 5.5", KeptBy: &linh.ID, KeptAt: s.aiAt(1, 16, 2)}); err != nil {
			return err
		}
	}
	return nil
}

// AIOctoberSummary is the summary Linh kept on October Product Update.
const AIOctoberSummary = "October Product Update reached 47,548 inboxes, 99.1% of the 47,980 sent. 41.2% opened it as people; another 18.6% were machine opens, which aren't counted as people. 6.3% clicked, and the feature-flags link drew 38% of the clicks. Subject B, “What shipped in October, and what we're retiring”, won the test with 42.6% human opens against 39.8%. Outlook had the highest bounce rate (1.0%). 112 people unsubscribed, most often for “too many emails”."

// aiRequests builds this month's log: the board's six rows, and the rest of
// each member's and feature's amounts spread over the month, so the totals
// are AISettings' to the cent (1,620,000 input and 460,000 output tokens).
func (s *Seeder) aiRequests(ws store.Workspace) []aiRequest {
	fixed := []aiRequest{
		{at: s.aiAt(0, 10, 31), member: "daniel@acme.io", feature: "subject_ideas", target: "Weekly Digest #143", in: 1050, out: 190, outcome: "accepted", detail: "1 of 5"},
		{at: s.aiAt(0, 10, 12), member: "sofia@acme.io", feature: "translate", action: "es", target: "Weekly Digest #143", in: 5200, out: 4680, outcome: "accepted_edits"},
		{at: s.aiAt(0, 9, 58), member: "sofia@acme.io", feature: "alt_text", target: "Weekly Digest #143", in: 2250, out: 160, outcome: "accepted"},
		{at: s.aiAt(0, 9, 40), member: "marcus@acme.io", feature: "segment_draft", target: "New segment", targetKind: "segment", in: 1800, out: 180, outcome: "accepted"},
		{at: s.aiAt(1, 16, 2), member: "linh@acme.io", feature: "report_summary", target: "October Product Update", targetKind: "report", in: 3300, out: 340, outcome: "kept"},
		{at: s.aiAt(1, 11, 20), member: "daniel@acme.io", feature: "rewrite", action: "instruction", target: "Weekly Digest #143", in: 560, out: 50, outcome: "refused"},
	}
	for i := range fixed {
		fixed[i].fixed = true
		if fixed[i].targetKind == "" {
			fixed[i].targetKind = "campaign"
		}
	}
	// The month starts on the 1st in the workspace's timezone; rows before
	// the board's two days go between then and Tue 11:00.
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := s.Now.In(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 30, 0, 0, loc)
	cutoff := s.aiAt(1, 11, 0)
	if !cutoff.After(monthStart) {
		cutoff = s.Now.Add(-time.Minute)
	}
	targets := map[string][]string{
		"subject_ideas":     {"Weekly Digest #143", "Weekly Digest #142", "November Product Update", "Beta Program: October check-in"},
		"rewrite":           {"Weekly Digest #143", "Weekly Digest #142", "November Product Update"},
		"translate":         {"Weekly Digest #142", "Weekly Digest #143", "October Product Update"},
		"alt_text":          {"Weekly Digest #142", "Weekly Digest #143", "November Product Update"},
		"accessibility_fix": {"Weekly Digest #143", "Weekly Digest #142"},
		"segment_draft":     {"New segment", "Engaged Outlook users on paid plans", "Gmail only, clickers"},
		"report_summary":    {"Weekly Digest #141", "Weekly Digest #140", "September Product Update", "Partner webinar"},
	}
	actions := map[string][]string{
		"rewrite":   {"shorten", "friendlier", "shorten", "direct", "formal", "instruction"},
		"translate": {"es", "de", "es", "de"},
	}
	outcomes := map[string][]string{
		"subject_ideas":     {"accepted", "rejected", "accepted", "accepted", "rejected"},
		"rewrite":           {"accepted", "accepted_edits", "rejected", "accepted", "accepted", "rejected", "accepted_edits"},
		"translate":         {"accepted_edits", "accepted", "accepted_edits", "rejected", "accepted"},
		"alt_text":          {"accepted", "accepted", "accepted_edits", "rejected"},
		"accessibility_fix": {"accepted", "accepted", "accepted", "rejected"},
		"segment_draft":     {"accepted", "nothing_drafted", "rejected", "accepted"},
		"report_summary":    {"kept", "discarded", "kept"},
	}
	rows := append([]aiRequest(nil), fixed...)
	var spread []aiRequest
	for ci, c := range aiCells {
		cost := c.micros
		n := c.requests
		for _, f := range fixed {
			if f.member == c.member && f.feature == c.feature {
				cost -= int64(f.in*2 + f.out*10)
				n--
			}
		}
		// Weights 3, 4, … 9, repeating, so request sizes vary.
		weights := make([]int64, n)
		var total int64
		for i := range weights {
			weights[i] = int64(3 + (i*7+ci*3)%7)
			total += weights[i]
		}
		var used int64
		for i := 0; i < n; i++ {
			micros := cost * weights[i] / total
			micros -= micros % 2
			if i == n-1 {
				micros = cost - used
			}
			used += micros
			frac := (float64(i) + 0.5) / float64(n)
			at := monthStart.Add(time.Duration(frac*float64(cutoff.Sub(monthStart))) + time.Duration((i*37+ci*11)%50)*time.Minute)
			if at.After(cutoff) {
				at = cutoff.Add(-time.Duration(i%30) * time.Minute)
			}
			ts := targets[c.feature]
			r := aiRequest{at: at, member: c.member, feature: c.feature, target: ts[(i+ci)%len(ts)], targetKind: "campaign", outcome: outcomes[c.feature][(i+ci)%len(outcomes[c.feature])]}
			if c.feature == "segment_draft" {
				r.targetKind = "segment"
			}
			if c.feature == "report_summary" {
				r.targetKind = "report"
			}
			if as := actions[c.feature]; as != nil {
				r.action = as[(i+ci)%len(as)]
			}
			if c.feature == "subject_ideas" && r.outcome == "accepted" {
				r.detail = fmt.Sprintf("%d of 5", 1+(i%2))
			}
			// in and out are set below, from the cost.
			r.in = int(micros) // cost in micros, for now
			spread = append(spread, r)
		}
	}
	// Output tokens: each request's share by its feature's weight, scaled so
	// all of them make 460,000 with the fixed rows; input tokens take the
	// rest of each cost (2 micros a token).
	var fixedOut int64
	for _, f := range fixed {
		fixedOut += int64(f.out)
	}
	targetOut := int64(460_000) - fixedOut
	var weightSum float64
	for _, r := range spread {
		weightSum += float64(r.in) * aiOutWeight[r.feature]
	}
	var outSum int64
	for i := range spread {
		micros := int64(spread[i].in)
		o := int64(float64(targetOut) * float64(micros) * aiOutWeight[spread[i].feature] / weightSum)
		if o*10 > micros {
			o = micros / 10
		}
		spread[i].out = int(o)
		outSum += o
	}
	// Hand the rounding to the largest requests, one token at a time, while
	// each keeps an even remainder for its input tokens.
	order := make([]int, len(spread))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return spread[order[a]].in > spread[order[b]].in })
	for k := 0; outSum != targetOut && k < 100000; k++ {
		i := order[k%len(order)]
		micros := int64(spread[i].in)
		if outSum < targetOut && int64(spread[i].out+1)*10 <= micros {
			spread[i].out++
			outSum++
		} else if outSum > targetOut && spread[i].out > 0 {
			spread[i].out--
			outSum--
		}
	}
	for i := range spread {
		micros := int64(spread[i].in)
		spread[i].in = int((micros - int64(spread[i].out)*10) / 2)
	}
	rows = append(rows, spread...)
	sort.SliceStable(rows, func(a, b int) bool { return rows[a].at.After(rows[b].at) })
	return rows
}
