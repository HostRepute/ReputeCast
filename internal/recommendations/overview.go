package recommendations

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// Grow › Overview (GrowOverview, GrowStates): the last 30 days' numbers,
// what waits for a person, and the list's new subscribers per week by
// source. The program card and the recommendations card read their own
// endpoints, so one can fail while the other shows.

// OverviewDays is the overview's period: the last 30 days, today included.
const OverviewDays = 30

// GrowOverview is Grow › Overview.
type GrowOverview struct {
	From      time.Time `json:"from" doc:"The period's first day (workspace time zone)"`
	To        time.Time `json:"to" doc:"Today"`
	UpdatedAt time.Time `json:"updatedAt"`
	// The list the numbers are about: the referral program's, else the
	// partner profile's. Empty without either.
	ListID   string `json:"listId"`
	ListName string `json:"listName"`
	// The referral program (empty without one), its status and start.
	ProgramID     string     `json:"programId"`
	ProgramStatus string     `json:"programStatus" enum:"on,off,"`
	ProgramStart  *time.Time `json:"programStart,omitempty"`
	// Recommends: newsletters this workspace recommends; RecommendedBy:
	// partners that recommend it.
	Recommends    int                      `json:"recommends"`
	RecommendedBy int                      `json:"recommendedBy"`
	Tiles         GrowOverviewTiles        `json:"tiles"`
	NeedsYou      []GrowNeed               `json:"needsYou"`
	Weeks         []GrowWeek               `json:"weeks"`
	Helpers       referrals.GrowHelperSets `json:"helpers"`
}

// GrowOverviewTiles are the numbers at the top.
type GrowOverviewTiles struct {
	// FromGrow: referred and partner subscribers who confirmed, of the
	// list's NewSubscribers (imports left out).
	FromGrow       int64 `json:"fromGrow"`
	NewSubscribers int64 `json:"newSubscribers"`
	// Referral conversion: Confirmed of Visits.
	Visits     int64   `json:"visits"`
	Confirmed  int64   `json:"confirmed"`
	Conversion float64 `json:"conversion" doc:"Percent"`
	// PartnerSignups confirmed through Partners.
	PartnerSignups int64    `json:"partnerSignups"`
	Partners       []string `json:"partners"`
	// RewardsSent, by reward.
	RewardsSent int64             `json:"rewardsSent"`
	Rewards     []GrowRewardCount `json:"rewards"`
}

// GrowRewardCount is how many of a reward went out.
type GrowRewardCount struct {
	Milestone int    `json:"milestone"`
	Name      string `json:"name"`
	Count     int64  `json:"count"`
}

// GrowNamedCount is a person and a count ("Alex Moreau (6)").
type GrowNamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// GrowNeed is an item of "Needs you".
type GrowNeed struct {
	Kind string `json:"kind" enum:"held,partner_invalid,partner_unreachable,unclaimed,codes_out,codes_low,webhook_failing,to_fulfill,stripe_missing,file_missing"`
	// At orders the list: the newest problem first.
	At    time.Time `json:"at"`
	Count int64     `json:"count"`
	// Held: the referrers and how many each; unclaimed: the referrers.
	People []GrowNamedCount `json:"people"`
	// Oldest held referral, or since when a partner link is broken.
	Since *time.Time `json:"since,omitempty"`
	// Partner problems: the newsletter and why.
	Partner string `json:"partner"`
	Reason  string `json:"reason" enum:"signature_rejected,ended,unknown_link,"`
	// Reward problems: the reward and its milestone; codes left; when
	// unclaimed codes expire.
	Reward      string      `json:"reward"`
	Milestone   int         `json:"milestone"`
	MilestoneID string      `json:"milestoneId"`
	CodesLeft   int64       `json:"codesLeft"`
	Expires     []time.Time `json:"expires"`
}

// GrowWeek is the list's new confirmed subscribers in a week, by source.
type GrowWeek struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Days      int       `json:"days"`
	Partial   bool      `json:"partial" doc:"The week is still going"`
	Other     int64     `json:"other" doc:"Forms and other sources (imports left out)"`
	Referrals int64     `json:"referrals"`
	Partners  int64     `json:"partners"`
}

// RegisterOverview adds GET /grow/overview.
func RegisterOverview(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "get-grow-overview", Path: "/workspaces/{workspace}/grow/overview",
		Summary: "Grow's overview: the last 30 days' numbers, what needs a person, and new subscribers per week by source",
		Tags:    []string{"Grow"}, Access: api.Workspace(authz.GrowView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[GrowOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}

// Overview computes Grow › Overview.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (GrowOverview, error) {
	now := s.now()
	today := dayOf(now, ws)
	from := today.AddDate(0, 0, -(OverviewDays - 1))
	out := GrowOverview{From: from, To: today, UpdatedAt: now, NeedsYou: []GrowNeed{}, Weeks: []GrowWeek{},
		Tiles: GrowOverviewTiles{Partners: []string{}, Rewards: []GrowRewardCount{}},
		Helpers: referrals.GrowHelperSets{
			Edit:    referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowEdit),
			Review:  referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowReview),
			Connect: referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowConnect),
		}}
	loc := location(ws)
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)

	// The program (the first: one per list, and the overview is about one).
	var program uuid.UUID
	var listID uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT p.id, p.status, p.first_on_at, p.list_id, l.name FROM referral_programs p JOIN lists l ON l.id = p.list_id
		WHERE p.workspace_id = $1 AND p.deleted_at IS NULL ORDER BY (p.status = 'on') DESC, p.first_on_at NULLS LAST, p.created_at LIMIT 1`, ws.ID).
		Scan(&program, &out.ProgramStatus, &out.ProgramStart, &listID, &out.ListName)
	if err != nil && !isNoRows(err) {
		return out, err
	}
	if program != uuid.Nil {
		out.ProgramID = program.String()
	} else {
		// The partner profile's list, the one partner readers join.
		var name string
		if err := s.Pool.QueryRow(ctx, `SELECT l.id, l.name FROM recommendation_settings rs JOIN lists l ON l.id = rs.profile_list_id WHERE rs.workspace_id = $1`, ws.ID).
			Scan(&listID, &name); err == nil {
			out.ListName = name
		} else if !isNoRows(err) {
			return out, err
		}
	}
	if listID != uuid.Nil {
		out.ListID = listID.String()
	}
	if err := s.Pool.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM recommendations WHERE workspace_id = $1 AND removed_at IS NULL),
			(SELECT count(*) FROM partner_links WHERE workspace_id = $1 AND direction = 'in' AND status <> 'ended')`, ws.ID).
		Scan(&out.Recommends, &out.RecommendedBy); err != nil {
		return out, err
	}
	if err := s.overviewTiles(ctx, ws, &out, program, listID, start); err != nil {
		return out, err
	}
	if listID != uuid.Nil {
		if out.Weeks, err = s.overviewWeeks(ctx, ws, program, listID, from, today); err != nil {
			return out, err
		}
	}
	if out.NeedsYou, err = s.needsYou(ctx, ws, program); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) overviewTiles(ctx context.Context, ws *store.Workspace, out *GrowOverview, program, listID uuid.UUID, start time.Time) error {
	t := &out.Tiles
	if program != uuid.Nil {
		if err := s.Pool.QueryRow(ctx, `SELECT
				(SELECT coalesce(sum(visits), 0) FROM referral_visits_daily WHERE program_id = $1 AND day >= $3::date),
				(SELECT count(*) FROM referrals WHERE program_id = $1 AND confirmed_at >= $2),
				(SELECT count(*) FROM referral_rewards WHERE program_id = $1 AND sent_at >= $2)`,
			program, start, out.From).Scan(&t.Visits, &t.Confirmed, &t.RewardsSent); err != nil {
			return err
		}
		if t.Visits > 0 {
			t.Conversion = float64(t.Confirmed) / float64(t.Visits) * 100
		}
		rows, err := s.Pool.Query(ctx, `SELECT milestone_count, reward_name, count(*) FROM referral_rewards WHERE program_id = $1 AND sent_at >= $2
			GROUP BY 1, 2 ORDER BY 1`, program, start)
		if err != nil {
			return err
		}
		for rows.Next() {
			var r GrowRewardCount
			if err := rows.Scan(&r.Milestone, &r.Name, &r.Count); err != nil {
				rows.Close()
				return err
			}
			t.Rewards = append(t.Rewards, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	if listID == uuid.Nil {
		return nil
	}
	// Partner signups that confirmed, on the list, and through whom.
	rows, err := s.Pool.Query(ctx, `SELECT coalesce(nullif(pl.label, ''), pl.remote_name), count(*) FROM partner_requests_in r JOIN partner_links pl ON pl.id = r.link_id
		WHERE r.workspace_id = $1 AND pl.list_id = $2 AND r.confirmed_at >= $3 GROUP BY 1 ORDER BY 2 DESC, 1`, ws.ID, listID, start)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		var n int64
		if err := rows.Scan(&name, &n); err != nil {
			rows.Close()
			return err
		}
		t.PartnerSignups += n
		t.Partners = append(t.Partners, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	t.FromGrow = t.Confirmed + t.PartnerSignups
	other, err := s.otherJoins(ctx, ws, program, listID, start, nil)
	if err != nil {
		return err
	}
	t.NewSubscribers = other + t.FromGrow
	return nil
}

// otherJoins counts the list's new confirmed subscribers who came neither
// through a referral nor a partner, imports left out; with days, per day
// (workspace time) instead.
func (s *Service) otherJoins(ctx context.Context, ws *store.Workspace, program, listID uuid.UUID, start time.Time, days map[string]int64) (int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT (lm.subscribed_at AT TIME ZONE $4)::date, count(*) FROM list_memberships lm
		WHERE lm.list_id = $1 AND lm.subscribed_at >= $2 AND lm.status <> 'pending' AND lm.source NOT LIKE 'import%'
		  AND NOT EXISTS (SELECT 1 FROM referrals r WHERE r.program_id = $3 AND r.contact_id = lm.contact_id AND r.confirmed_at IS NOT NULL)
		  AND NOT EXISTS (SELECT 1 FROM partner_requests_in pr JOIN partner_links pl ON pl.id = pr.link_id
		                  WHERE pr.contact_id = lm.contact_id AND pl.list_id = $1 AND pr.confirmed_at IS NOT NULL)
		GROUP BY 1`, listID, start, program, location(ws).String())
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var total int64
	for rows.Next() {
		var d time.Time
		var n int64
		if err := rows.Scan(&d, &n); err != nil {
			return 0, err
		}
		total += n
		if days != nil {
			days[d.Format("2006-01-02")] += n
		}
	}
	return total, rows.Err()
}

// overviewWeeks: weeks from Monday covering the period; the first and the
// last may be partial.
func (s *Service) overviewWeeks(ctx context.Context, ws *store.Workspace, program, listID uuid.UUID, from, today time.Time) ([]GrowWeek, error) {
	loc := location(ws)
	wd := int(from.Weekday())
	if wd == 0 {
		wd = 7
	}
	first := from.AddDate(0, 0, -(wd - 1))
	start := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
	other := map[string]int64{}
	if _, err := s.otherJoins(ctx, ws, program, listID, start, other); err != nil {
		return nil, err
	}
	perDay := func(q string, args ...any) (map[string]int64, error) {
		out := map[string]int64{}
		rows, err := s.Pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var d time.Time
			var n int64
			if err := rows.Scan(&d, &n); err != nil {
				return nil, err
			}
			out[d.Format("2006-01-02")] = n
		}
		return out, rows.Err()
	}
	refs := map[string]int64{}
	if program != uuid.Nil {
		var err error
		if refs, err = perDay(`SELECT (confirmed_at AT TIME ZONE $3)::date, count(*) FROM referrals WHERE program_id = $1 AND confirmed_at >= $2 GROUP BY 1`,
			program, start, loc.String()); err != nil {
			return nil, err
		}
	}
	partners, err := perDay(`SELECT (r.confirmed_at AT TIME ZONE $4)::date, count(*) FROM partner_requests_in r JOIN partner_links pl ON pl.id = r.link_id
		WHERE r.workspace_id = $1 AND pl.list_id = $2 AND r.confirmed_at >= $3 GROUP BY 1`, ws.ID, listID, start, loc.String())
	if err != nil {
		return nil, err
	}
	out := []GrowWeek{}
	for w := first; !w.After(today); w = w.AddDate(0, 0, 7) {
		g := GrowWeek{Start: w, End: w.AddDate(0, 0, 6), Days: 7}
		for d := w; d.Before(w.AddDate(0, 0, 7)) && !d.After(today); d = d.AddDate(0, 0, 1) {
			k := d.Format("2006-01-02")
			g.Other += other[k]
			g.Referrals += refs[k]
			g.Partners += partners[k]
		}
		// The week with today in it is still going.
		if !g.End.Before(today) {
			g.End, g.Partial = today, true
			g.Days = int(today.Sub(w).Hours()/24) + 1
		}
		out = append(out, g)
	}
	return out, nil
}

// needsYou lists what waits for a person: held referrals, broken partner
// links, unclaimed coupons and milestones whose rewards can't go out.
func (s *Service) needsYou(ctx context.Context, ws *store.Workspace, program uuid.UUID) ([]GrowNeed, error) {
	out := []GrowNeed{}
	if program != uuid.Nil {
		// Held referrals, by referrer.
		rows, err := s.Pool.Query(ctx, `SELECT coalesce(nullif(trim(c.first_name || ' ' || c.last_name), ''), c.email, ''), count(*), min(r.held_at), max(r.held_at)
			FROM referrals r LEFT JOIN contacts c ON c.id = r.referrer_id
			WHERE r.program_id = $1 AND r.status = 'review' GROUP BY 1 ORDER BY 2 DESC, 1`, program)
		if err != nil {
			return nil, err
		}
		held := GrowNeed{Kind: "held", People: []GrowNamedCount{}, Expires: []time.Time{}}
		for rows.Next() {
			var p GrowNamedCount
			var oldest, newest *time.Time
			if err := rows.Scan(&p.Name, &p.Count, &oldest, &newest); err != nil {
				rows.Close()
				return nil, err
			}
			held.Count += p.Count
			held.People = append(held.People, p)
			if oldest != nil && (held.Since == nil || oldest.Before(*held.Since)) {
				o := *oldest
				held.Since = &o
			}
			if newest != nil && newest.After(held.At) {
				held.At = *newest
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if held.Count > 0 {
			out = append(out, held)
		}
		// Coupons nobody claimed yet.
		rows, err = s.Pool.Query(ctx, `SELECT rw.reward_name, rw.milestone_count, coalesce(rw.milestone_id::text, ''), coalesce(nullif(trim(c.first_name || ' ' || c.last_name), ''), c.email), rw.expires_at, rw.sent_at
			FROM referral_rewards rw JOIN contacts c ON c.id = rw.contact_id
			WHERE rw.program_id = $1 AND rw.reward_type = 'coupon' AND rw.status = 'sent' AND rw.used_at IS NULL AND (rw.expires_at IS NULL OR rw.expires_at > $2)
			ORDER BY rw.milestone_count, rw.expires_at NULLS LAST, rw.sent_at`, program, s.now())
		if err != nil {
			return nil, err
		}
		unclaimed := map[int]*GrowNeed{}
		order := []int{}
		for rows.Next() {
			var name, milestoneID, person string
			var milestone int
			var expires, sent *time.Time
			if err := rows.Scan(&name, &milestone, &milestoneID, &person, &expires, &sent); err != nil {
				rows.Close()
				return nil, err
			}
			n := unclaimed[milestone]
			if n == nil {
				n = &GrowNeed{Kind: "unclaimed", Reward: name, Milestone: milestone, MilestoneID: milestoneID, People: []GrowNamedCount{}, Expires: []time.Time{}}
				unclaimed[milestone] = n
				order = append(order, milestone)
			}
			n.Count++
			n.People = append(n.People, GrowNamedCount{Name: person, Count: 1})
			if expires != nil {
				n.Expires = append(n.Expires, *expires)
			}
			if sent != nil && sent.After(n.At) {
				n.At = *sent
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for _, m := range order {
			out = append(out, *unclaimed[m])
		}
		// Milestones whose rewards can't go out, and rewards to send by hand.
		if s.Referrals != nil && s.Referrals() != nil {
			list, err := s.Referrals().Milestones(ctx, ws, program)
			if err != nil {
				return nil, err
			}
			for _, m := range list.Items {
				if n, ok := milestoneNeed(m, s.now()); ok {
					out = append(out, n)
				}
				if m.Stats.ToFulfill > 0 {
					n := GrowNeed{Kind: "to_fulfill", Count: m.Stats.ToFulfill, Reward: m.RewardName, Milestone: m.Count, MilestoneID: m.ID, People: []GrowNamedCount{}, Expires: []time.Time{}}
					for _, f := range list.ToFulfill {
						if f.Milestone == m.Count && f.ReachedAt.After(n.At) {
							n.At = f.ReachedAt
						}
					}
					out = append(out, n)
				}
			}
		}
	}
	// Partner links that are broken.
	rows, err := s.Pool.Query(ctx, `SELECT remote_name, status, invalid_since, invalid_reason, failing_since FROM partner_links
		WHERE workspace_id = $1 AND direction = 'out' AND (status = 'invalid' OR (status = 'connected' AND failing_since IS NOT NULL))
		ORDER BY coalesce(invalid_since, failing_since) DESC`, ws.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name, status, reason string
		var invalidSince, failingSince *time.Time
		if err := rows.Scan(&name, &status, &invalidSince, &reason, &failingSince); err != nil {
			rows.Close()
			return nil, err
		}
		n := GrowNeed{Kind: "partner_unreachable", Partner: name, Since: failingSince, People: []GrowNamedCount{}, Expires: []time.Time{}}
		if status == "invalid" {
			n.Kind, n.Since, n.Reason = "partner_invalid", invalidSince, reason
		}
		if n.Since != nil {
			n.At = *n.Since
		}
		out = append(out, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}

// milestoneNeed is a milestone's reward problem, if it has one.
func milestoneNeed(m referrals.ReferralMilestoneView, now time.Time) (GrowNeed, bool) {
	n := GrowNeed{Kind: m.Health.Kind, Reward: m.RewardName, Milestone: m.Count, MilestoneID: m.ID, People: []GrowNamedCount{}, Expires: []time.Time{}, At: now,
		Count: m.Health.Waiting, CodesLeft: m.Health.CodesLeft}
	switch m.Health.Kind {
	case "codes_out", "codes_low", "webhook_failing", "stripe_missing", "file_missing":
		if m.WebhookCalledAt != nil && m.Health.Kind == "webhook_failing" {
			n.At = *m.WebhookCalledAt
		}
		return n, true
	}
	return n, false
}

// location is the workspace's time zone (UTC when it has none).
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if l, err := time.LoadLocation(ws.Timezone); err == nil {
			return l
		}
	}
	return time.UTC
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
