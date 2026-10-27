package demo

import (
	"context"
	"encoding/json"
	"time"
)

// seedTrafficRules gives the sample workspace the Traffic rules board's
// reply rules (review GS-12) and what the rules did in past sends: Yahoo's
// TSS04 paused Yahoo for 2 hours during Weekly Digest #139, Gmail's
// 421-4.7.28 slowed Gmail for 30 minutes during #140, and Outlook's
// complaints paused Outlook during October Product Update.
func seedTrafficRules(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	started := func(name string) (time.Time, bool) {
		id, ok := s.Campaigns[name]
		if !ok {
			return time.Time{}, false
		}
		var at time.Time
		if err := s.Tx.QueryRow(ctx, `SELECT started_at FROM campaigns WHERE id = $1`, id).Scan(&at); err != nil {
			return time.Time{}, false
		}
		return at, true
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO traffic_rules (workspace_id, canary, watch, updated_by, updated_at) VALUES ($1, '{}', '{}', $2, now() - interval '2 days')
		ON CONFLICT (workspace_id) DO NOTHING`, ws.ID, linh.ID); err != nil {
		return err
	}
	type rule struct {
		mailbox, contains, action string
		count, window, duration   int // minutes; duration 0 until someone resumes
		last                      string
	}
	for i, r := range []rule{
		{"yahoo", "TSS04", "pause", 10, 10, 120, "Weekly Digest #139"},
		{"outlook", "S3150", "pause_route", 1, 10, 0, ""},
		{"gmail", "421-4.7.28", "slow", 5, 10, 30, "Weekly Digest #140"},
	} {
		var dur *int
		if r.duration > 0 {
			d := r.duration * 60
			dur = &d
		}
		var last *time.Time
		if at, ok := started(r.last); ok {
			t := at.Add(6 * time.Minute)
			last = &t
		}
		var lastCampaign any
		if id, ok := s.Campaigns[r.last]; ok {
			lastCampaign = id
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO traffic_reply_rules (id, workspace_id, mailbox_provider, contains, min_count, window_seconds, action, duration_seconds,
			enabled, position, last_matched_at, last_campaign_id, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10, $11, $12, now() - interval '40 days')`,
			ID(), ws.ID, r.mailbox, r.contains, r.count, r.window*60, r.action, dur, i, last, lastCampaign, linh.ID); err != nil {
			return err
		}
	}
	incident := func(name, kind, mbp, metric string, value, limit float64, held int64, after, lasted time.Duration, resolution string, detail map[string]any) error {
		at, ok := started(name)
		if !ok {
			return nil
		}
		d, _ := json.Marshal(detail)
		_, err := s.Tx.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, mailbox_provider, metric, value, threshold, held, detail, tripped_at, resolved_at, resolution)
			VALUES ($1, $2, $3, 'mailbox_provider', $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			ID(), s.Campaigns[name], kind, mbp, metric, value, limit, held, d, at.Add(after), at.Add(after+lasted), resolution)
		return err
	}
	if err := incident("Weekly Digest #139", "reply_rule", "yahoo", "reply", 10, 10, 2810, 14*time.Minute, 2*time.Hour, "expired", map[string]any{
		"events": 10, "causes": []map[string]string{{"lead": "Yahoo replied TSS04 10 times in 10 minutes.", "body": "ReputeCast paused Yahoo on every route, for 2 h."}},
	}); err != nil {
		return err
	}
	if err := incident("Weekly Digest #140", "reply_rule", "gmail", "reply", 5, 5, 0, 6*time.Minute, 30*time.Minute, "expired", map[string]any{
		"events": 5, "action": "slowed", "slowedTo": 7,
		"causes": []map[string]string{{"lead": "Gmail replied 421-4.7.28 5 times in 10 minutes.", "body": "ReputeCast slowed Gmail to half rate (7 /s), for 30 min."}},
	}); err != nil {
		return err
	}
	return incident("October Product Update", "threshold", "outlook", "complaint", 0.18, 0.1, 2436, 21*time.Minute, 34*time.Minute, "resumed_with_changes", map[string]any{
		"events": 7, "base": 3980,
	})
}
