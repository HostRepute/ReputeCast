package demo

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The Report board's "Growth & revenue from this issue" for the October
// Product Update (review AN-12, AN-13): 38 new subscribers (9 from
// forwards, 12 referrals, 17 from the web post) and its views on the web.
// It labels people who joined after the send and referrals made since,
// rather than adding any, so Grow's and Audience's numbers stay as they
// are; it runs once every seeder has.
const (
	octWebPostJoins = 17
	octForwards     = 9
	octReferrals    = 12
	octWebViews     = 2410
)

// octForwardCombo is the smaller UTM combination (9 joins, seed_growth_sample)
// whose tags the October issue's forwards take.
var octForwardCombo = [3]string{"podcast", "audio", "changelog-542"}

func reportOctoberGrowth(c *reportCampaignSeed) func(context.Context, *Seeder) error {
	return func(ctx context.Context, s *Seeder) error {
		ws := s.WS()
		host := "localhost"
		if s.Cfg != nil {
			if u, err := url.Parse(s.Cfg.HTTP.PublicURL); err == nil && u.Host != "" {
				host = u.Host
			}
		}
		// Joins after the send, labelled without changing what Audience ›
		// Growth counts (its UTM and referrer tables, the forms' sites):
		// subscribe-page joins that came from the issue's archive page (only
		// their page changes), and forwards through a subscribe link tagged
		// as a forward of this issue: the joins of one of the smaller UTM
		// combinations (9 of them) trade tags with joins after the send, so
		// every combination keeps its count.
		ids := func(sql string, args ...any) ([]uuid.UUID, error) {
			rows, err := s.Tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var out []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					return nil, err
				}
				out = append(out, id)
			}
			return out, rows.Err()
		}
		var slug string
		_ = s.Tx.QueryRow(ctx, `SELECT slug FROM archive_issues WHERE campaign_id = $1`, c.id).Scan(&slug)
		if slug != "" {
			web, err := ids(`SELECT contact_id FROM contact_growth
				WHERE workspace_id = $1 AND joined_at >= $2 AND source_kind = 'subscribe_page' ORDER BY joined_at, contact_id LIMIT $3`,
				ws.ID, c.started, octWebPostJoins)
			if err != nil {
				return err
			}
			for _, id := range web {
				if _, err := s.Tx.Exec(ctx, `UPDATE contact_growth SET page = $2 WHERE contact_id = $1`, id, host+"/p/"+ws.Slug+"/archive/"+slug); err != nil {
					return err
				}
			}
			// The rest through a form on a copy of the archive page on the
			// form's own site, which keeps the forms' sites as they are.
			if n := octWebPostJoins - len(web); n > 0 {
				if _, err := s.Tx.Exec(ctx, `UPDATE contact_growth SET page = split_part(page, '/', 1) || '/archive/' || $4
					WHERE contact_id IN (SELECT contact_id FROM contact_growth
						WHERE workspace_id = $1 AND joined_at >= $2 AND source_kind = 'form' AND page <> '' ORDER BY joined_at, contact_id LIMIT $3)`,
					ws.ID, c.started, n, slug); err != nil {
					return err
				}
			}
		}
		combo, err := ids(`SELECT contact_id FROM contact_growth
			WHERE workspace_id = $1 AND utm_source = $2 AND utm_medium = $3 AND utm_campaign = $4 ORDER BY joined_at, contact_id LIMIT $5`,
			ws.ID, octForwardCombo[0], octForwardCombo[1], octForwardCombo[2], octForwards)
		if err != nil {
			return err
		}
		after, err := ids(`SELECT contact_id FROM contact_growth
			WHERE workspace_id = $1 AND joined_at >= $2 AND source_kind IN ('form', 'subscribe_page') AND captured AND page NOT LIKE '%/archive/%'
			  AND NOT (utm_source = $3 AND utm_medium = $4 AND utm_campaign = $5)
			ORDER BY joined_at, contact_id LIMIT $6`,
			ws.ID, c.started.Add(time.Hour), octForwardCombo[0], octForwardCombo[1], octForwardCombo[2], len(combo))
		if err != nil {
			return err
		}
		for i := range min(len(combo), len(after)) {
			if _, err := s.Tx.Exec(ctx, `UPDATE contact_growth r SET utm_source = a.utm_source, utm_medium = a.utm_medium, utm_campaign = a.utm_campaign
				FROM contact_growth a WHERE r.contact_id = $1 AND a.contact_id = $2`, combo[i], after[i]); err != nil {
				return err
			}
			if _, err := s.Tx.Exec(ctx, `UPDATE contact_growth SET utm_source = 'email', utm_medium = 'forward', utm_campaign = $2 WHERE contact_id = $1`,
				after[i], c.id.String()); err != nil {
				return err
			}
		}
		// Referral signups after the send whose referrer last clicked this
		// issue a few hours before: one referral per referrer, whose other
		// referrals aren't credited by that click too, until 12 are.
		refs, err := s.Tx.Query(ctx, `SELECT rf.referrer_id, rf.signed_up_at, ct.email
			FROM referrals rf JOIN contacts ct ON ct.id = rf.referrer_id
			WHERE rf.workspace_id = $1 AND rf.signed_up_at >= $2 AND rf.status <> 'not_counted'
			  AND NOT EXISTS (SELECT 1 FROM referrals o WHERE o.referrer_id = rf.referrer_id AND o.id <> rf.id AND o.status <> 'not_counted'
			      AND o.signed_up_at > rf.signed_up_at - interval '6 hours' AND o.signed_up_at <= rf.signed_up_at + interval '7 days')
			  AND NOT EXISTS (SELECT 1 FROM messages m WHERE m.contact_id = rf.referrer_id AND m.campaign_id = $3)
			ORDER BY rf.signed_up_at, rf.id LIMIT 60`, ws.ID, c.started.Add(6*time.Hour), c.id)
		if err != nil {
			return err
		}
		type ref struct {
			contact uuid.UUID
			at      time.Time
			email   string
		}
		var list []ref
		seen := map[uuid.UUID]bool{}
		for refs.Next() {
			var r ref
			if err := refs.Scan(&r.contact, &r.at, &r.email); err != nil {
				refs.Close()
				return err
			}
			if !seen[r.contact] {
				seen[r.contact] = true
				list = append(list, r)
			}
		}
		refs.Close()
		credited := func() (int64, error) {
			return s.Q.ReportGrowthReferrals(ctx, store.ReportGrowthReferralsParams{CampaignID: &c.id, WorkspaceID: ws.ID, Since: c.started})
		}
		for i, r := range list {
			n, err := credited()
			if err != nil {
				return err
			}
			if n >= octReferrals {
				break
			}
			clicked := r.at.Add(-time.Duration(2+i%4) * time.Hour)
			sent := c.started.Add(2 * time.Hour)
			if clicked.Before(sent) {
				clicked = sent.Add(time.Hour)
			}
			id := ID()
			if _, err := s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, variant, ab_rollout, mailbox_provider,
				status, queued_at, sent_at, delivered_at, opened_at, human_opened_at, clicked_at)
				VALUES ($1, $2, 'broadcast', $3, $4, $5, 'B', true, 'other', 'delivered', $6, $6, $7, $8, $8, $8)`,
				id, ws.ID, c.id, r.contact, r.email, sent, sent.Add(3*time.Second), clicked); err != nil {
				return fmt.Errorf("referrer's message: %w", err)
			}
			// A click that credits more than this referral is taken back.
			if m, err := credited(); err != nil {
				return err
			} else if m > octReferrals {
				if _, err := s.Tx.Exec(ctx, `DELETE FROM messages WHERE id = $1`, id); err != nil {
					return err
				}
			}
		}
		// The issue's web views: its archive page, besides its website post.
		var post int64
		_ = s.Tx.QueryRow(ctx, `SELECT coalesce(sum(v.views), 0) FROM website_view_days v JOIN website_posts p ON p.id = v.page_id
			WHERE v.page_kind = 'post' AND p.campaign_id = $1`, c.id).Scan(&post)
		views := Split(max(octWebViews-post, 0), []float64{30, 18, 11, 8, 6, 5, 4, 3, 3, 2, 2, 2, 1, 1})
		for d, n := range views {
			if n == 0 {
				continue
			}
			day := c.started.AddDate(0, 0, d).UTC().Truncate(24 * time.Hour)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO website_view_days (workspace_id, day, page_kind, page_id, views) VALUES ($1, $2, 'issue', $3, $4)
				ON CONFLICT (workspace_id, day, page_kind, page_id) DO UPDATE SET views = EXCLUDED.views`, ws.ID, day, c.id, n); err != nil {
				return err
			}
		}
		return nil
	}
}
