package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func init() { seedSending = seedSendingImpl }

// seedSendingImpl gives the sent campaigns of the sample data the send
// history the live monitor and the report read: who started each send,
// the thresholds and throttles in force, the recipients and when it
// finished, and the evidence behind Weekly Digest #139's Yahoo auto-pause.
// Weekly Digest #142 stays scheduled; `reputecast demo send` sends it
// through the demo transport to reproduce the live boards.
func seedSendingImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	thresholds, _ := json.Marshal(map[string]any{"bounce": 2.0, "complaint": 0.1, "mailbox_complaint": 0.1, "deferral": 5.0, "api_errors": 5.0, "scope": "route"})
	throttles, _ := json.Marshal(map[string]float64{"outlook": 5, "yahoo": 4, "icloud": 3})
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.started_at, c.completed_at, coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = c.id), 0)
		FROM campaigns c WHERE c.workspace_id = $1 AND c.status = 'sent' AND c.origin = '' AND c.started_at IS NOT NULL`, ws.ID)
	if err != nil {
		return err
	}
	type sent struct {
		id            uuid.UUID
		started, done time.Time
		recipients    int64
	}
	var all []sent
	for rows.Next() {
		var c sent
		var done *time.Time
		if err := rows.Scan(&c.id, &c.started, &done, &c.recipients); err != nil {
			rows.Close()
			return err
		}
		c.done = c.started
		if done != nil {
			c.done = *done
		}
		all = append(all, c)
	}
	rows.Close()
	for _, c := range all {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_sends (campaign_id, workspace_id, phase, prepared_at, recipients, started_by, started_by_name, scheduled_by,
			throttles, thresholds, ab, finished_at, created_at, updated_at)
			VALUES ($1, $2, 'finished', $3, $4, $5, $6, $5, $7, $8, $9, $10, $3, $10) ON CONFLICT (campaign_id) DO NOTHING`,
			c.id, ws.ID, c.started, c.recipients, linh.ID, linh.Name, throttles, thresholds, json.RawMessage(`{}`), c.done); err != nil {
			return err
		}
	}
	// Weekly Digest #139: Yahoo deferrals tripped the threshold 38 minutes
	// in; Linh resumed it at 4 /s after 26 minutes.
	if id, ok := s.Campaigns["Weekly Digest #139"]; ok {
		detail, _ := json.Marshal(map[string]any{
			"sent": 2010, "total": 4820, "delivered": 1873, "events": 129, "base": 2010, "throttle": 4,
			"campaign":   map[string]any{"metric": "deferral", "events": 162, "base": 18712, "value": 0.87, "limit": 5.0},
			"secondary":  map[string]any{"metric": "bounce", "events": 14, "base": 2010, "value": 0.7, "limit": 2.0},
			"note":       "Mailbox providers defer mail they want to slow down; sending slower usually clears it within the hour.",
			"continuing": []string{"Gmail", "Outlook", "iCloud", "Other"},
			"causes": []map[string]string{
				{"lead": "Yahoo is slowing the send down.", "body": "129 of 2,010 messages were deferred with 421 4.7.0 TSS04; Yahoo accepts mail again at a lower rate."},
				{"lead": "First Tuesday send at the full rate.", "body": "The digest moved from Brevo two weeks earlier and Yahoo hadn't seen this volume from news.acme.io yet."},
			},
			"recommendations": []string{"Lower the Yahoo rate to 2 /s for the rest of this send.", "Keep Yahoo at 4 /s for the next digests while the domain builds reputation."},
			"options": []map[string]any{
				{"key": "resume", "title": "Resume as is", "body": "Send to all 2,810 remaining Yahoo recipients. The threshold stays armed and will pause again at the next deferral.", "action": "Resume as is"},
				{"key": "resume_with_changes", "title": "Resume with changes (recommended)", "body": "Send to 2,810 at 2 /s.", "recommended": true, "action": "Resume with changes"},
				{"key": "cancel_share", "title": "Cancel the Yahoo share", "body": "Skip the remaining 2,810 Yahoo recipients. The rest of the campaign finishes normally.", "action": "Cancel the Yahoo share"},
			},
			"changes": map[string]any{"mailbox": "yahoo", "importedOnly": false, "exclude": 0, "rate": 2},
		})
		if _, err := s.Tx.Exec(ctx, `UPDATE campaign_incidents SET detail = $2 WHERE campaign_id = $1 AND kind = 'threshold'`, id, detail); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE campaign_sends SET throttles = $2 WHERE campaign_id = $1`, id, json.RawMessage(`{"outlook":5,"yahoo":2,"icloud":3}`)); err != nil {
			return err
		}
	}
	return nil
}
