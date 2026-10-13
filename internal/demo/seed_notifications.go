package demo

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func init() { seedNotifications = seedNotificationsImpl }

// seedNotificationsImpl gives Linh the six notifications of the bell popover
// (ShellNotifications) and the notification center: three unread (approval
// requested, send auto-paused, provider down) and three read. Links point at
// the screens where she can act; they are resolved once every seeder has
// run, so the approval request and campaign ids exist.
func seedNotificationsImpl(_ context.Context, s *Seeder) error {
	s.After(func(ctx context.Context, s *Seeder) error {
		ws := s.WS()
		linh := s.Users["linh@acme.io"]
		base := "/w/" + ws.Slug
		approval := base + "/approvals"
		var reqID uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT r.id FROM approval_requests r JOIN campaigns c ON c.id = r.campaign_id
			WHERE r.workspace_id = $1 AND c.name = 'November Product Update' ORDER BY r.requested_at DESC LIMIT 1`, ws.ID).Scan(&reqID); err == nil {
			approval = base + "/approvals/" + reqID.String()
		}
		live := base + "/campaigns"
		if id, ok := s.Campaigns["October Product Update"]; ok {
			live = base + "/campaigns/" + id.String() + "/live"
		}
		items := []struct {
			kind, severity, title, summary, body, link string
			at                                         time.Time
			read                                       bool
		}{
			{"approval_requested", "warn", "Approval requested",
				"Daniel Park asked you to approve “November Product Update” · 47,912 recipients",
				"Daniel Park asked for approval: “November Product Update” · 47,912 recipients · rules: audience ≥ 10,000, warming-up provider, author lacks send",
				approval, s.Ago(2 * time.Hour), false},
			{"send_paused", "bad", "Send auto-paused",
				"“October Product Update” paused on the Outlook route: complaint rate 0.18% (limit 0.10%)",
				"“October Product Update” paused on the Outlook route: complaint rate 0.18% (limit 0.10%). Resume, resume with changes, or cancel.",
				live, s.designDay(3, 9, 21), false},
			{"provider_down", "bad", "Provider down",
				"Postal returned connection errors for 3 minutes · Broadcasts waiting for a decision",
				"Postal (mta.acme.io) returned connection errors for 3 minutes. Automations and System failed over; Broadcasts are waiting for a decision.",
				base + "/providers", s.designDay(3, 9, 12), false},
			{"webhooks_silent", "warn", "Webhooks silent",
				"No events from Postmark for 6 hours · check the endpoint",
				"No events from Postmark for 6 hours. Delivery counts may lag; check the endpoint and the Postmark message stream.",
				base + "/providers", s.designDay(4, 14, 5), true},
			{"invite_accepted", "ok", "Invite accepted",
				"Sofia Rossi joined Acme Newsletter as Editor",
				"Sofia Rossi joined Acme Newsletter as Editor through Google Workspace.",
				base + "/settings/team", s.designDay(11, 9, 2), true},
			{"hostrepute_alert", "info", "HostRepute alert",
				"203.0.113.25 listed on one blocklist (SORBS-SPAM) · delisting link available",
				"203.0.113.25 was listed on SORBS-SPAM. Delisting usually takes 24–48 h; the link is in Deliverability.",
				base + "/deliverability", s.designDay(12, 22, 14), true},
		}
		wsID := ws.ID
		for _, it := range items {
			var read *time.Time
			if it.read {
				t := it.at.Add(30 * time.Minute)
				read = &t
			}
			if _, err := s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, read_at, email_status, emailed_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				ID(), linh.ID, &wsID, it.kind, it.severity, it.title, it.summary, it.body, it.link, it.at, read, emailStatus(it.kind), emailedAt(it.kind, it.at)); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// Linh asked for email on these; they went out before the demo starts, so
// the delivery job doesn't send them again.
func emailStatus(kind string) *string {
	switch kind {
	case "webhooks_silent", "invite_accepted":
		return nil
	}
	v := "sent"
	return &v
}

func emailedAt(kind string, at time.Time) *time.Time {
	if emailStatus(kind) == nil {
		return nil
	}
	t := at.Add(time.Minute)
	return &t
}
