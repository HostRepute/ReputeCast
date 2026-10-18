package referrals

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Periods of the results.
const (
	Period7    = "7d"
	Period30   = "30d"
	PeriodAll2 = "all"
)

// MinConfirmedForResults: results show after 20 confirmed referrals.
const MinConfirmedForResults = 20

// MinForQuality: below 100 referred subscribers the quality card waits.
const MinForQuality = 100

// ReferralReasonCount is how many referrals weren't counted for a reason.
type ReferralReasonCount struct {
	Reason   string `json:"reason" enum:"self,same_ip,disposable,left,already,never_confirmed,reviewer"`
	LeftKind string `json:"leftKind"`
	Count    int64  `json:"count"`
}

// ReferralFunnel is visits → signups → confirmed → counted, with what
// branches off.
type ReferralFunnel struct {
	Visits      int64                 `json:"visits"`
	Signups     int64                 `json:"signups"`
	Confirmed   int64                 `json:"confirmed"`
	Counted     int64                 `json:"counted"`
	Unconfirmed int64                 `json:"unconfirmed" doc:"Never confirmed, or still waiting to confirm"`
	InHold      int64                 `json:"inHold"`
	HoldEndsBy  *time.Time            `json:"holdEndsBy,omitempty"`
	Held        int64                 `json:"held"`
	NotCounted  int64                 `json:"notCounted"`
	Reasons     []ReferralReasonCount `json:"reasons"`
	// RewardsSent and Referrers (with a counted referral) under Counted.
	RewardsSent int64 `json:"rewardsSent"`
	Referrers   int64 `json:"referrers"`
	Weeks       int   `json:"weeks"`
}

// ReferralBucket is confirmed referrals in a day or week.
type ReferralBucket struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Confirmed int64     `json:"confirmed"`
	// Partial: the bucket is still going ("3 days so far").
	Partial bool `json:"partial"`
	Days    int  `json:"days"`
}

// ReferralChannel is visits and signups by where links were shared.
type ReferralChannel struct {
	Channel string `json:"channel" enum:"copy,x,linkedin,whatsapp,email,other"`
	Visits  int64  `json:"visits"`
	Signups int64  `json:"signups"`
}

// ReferralRewardRow is a milestone's rewards in the results.
type ReferralRewardRow struct {
	MilestoneID string `json:"milestoneId"`
	Count       int    `json:"count"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Reached     int64  `json:"reached"`
	Sent        int64  `json:"sent"`
	Used        int64  `json:"used"`
	Waiting     int64  `json:"waiting"`
	Failed      int64  `json:"failed"`
}

// ReferralQualityLine compares referred subscribers with others.
type ReferralQualityLine struct {
	Referred   float64 `json:"referred"`
	ReferredN  int64   `json:"referredN"`
	ReferredOf int64   `json:"referredOf"`
	Others     float64 `json:"others"`
	OthersN    int64   `json:"othersN"`
	OthersOf   int64   `json:"othersOf"`
}

// ReferralQuality is the quality of referred subscribers.
type ReferralQuality struct {
	Referred int64 `json:"referred"`
	// Enough: 100 or more referred subscribers to compare.
	Enough bool `json:"enough"`
	// Opens on the list's last issue; Campaign names it.
	Campaign     string              `json:"campaign"`
	Opens        ReferralQualityLine `json:"opens"`
	Unsubscribed ReferralQualityLine `json:"unsubscribed"`
	Bounced      ReferralQualityLine `json:"bounced"`
}

// ReferralTopReferrer is a row of the top referrers.
type ReferralTopReferrer struct {
	LinkID       string     `json:"linkId"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Counted      int64      `json:"counted"`
	InHold       int64      `json:"inHold"`
	Held         int64      `json:"held"`
	Milestones   []int      `json:"milestones"`
	Unclaimed    string     `json:"unclaimed"`
	LastReferral *time.Time `json:"lastReferral,omitempty"`
}

// ReferralResults is Referrals › Results.
type ReferralResults struct {
	Program   ReferralProgramDetail `json:"program"`
	Period    string                `json:"period" enum:"7d,30d,all"`
	From      time.Time             `json:"from"`
	To        time.Time             `json:"to"`
	Enough    bool                  `json:"enough" doc:"20 confirmed referrals or more"`
	Funnel    ReferralFunnel        `json:"funnel"`
	Grouping  string                `json:"grouping" enum:"day,week"`
	OverTime  []ReferralBucket      `json:"overTime"`
	Channels  []ReferralChannel     `json:"channels"`
	Top       []ReferralTopReferrer `json:"top"`
	WithThree int64                 `json:"withThree"`
	WithTen   int64                 `json:"withTen"`
	// MilestoneLadder: referrers who reached each milestone.
	Rewards []ReferralRewardRow `json:"rewards"`
	Quality ReferralQuality     `json:"quality"`
}

// Results is a program's results for a period.
func (s *Service) Results(ctx context.Context, ws *store.Workspace, programID uuid.UUID, period, grouping string) (ReferralResults, error) {
	d, err := s.Detail(ctx, ws, programID)
	if err != nil {
		return ReferralResults{}, err
	}
	now := s.now()
	loc := workspaceLocation(ws)
	var from time.Time
	switch period {
	case Period7:
		from = now.AddDate(0, 0, -7)
	case Period30:
		from = now.AddDate(0, 0, -30)
	default:
		period = PeriodAll2
		if d.StartedAt != nil {
			from = *d.StartedAt
		} else {
			cr, _ := s.Q.ReferralProgramByID(ctx, programID)
			from = cr.CreatedAt
		}
	}
	if grouping != "day" {
		grouping = "week"
	}
	out := ReferralResults{Program: d, Period: period, From: from, To: now, Grouping: grouping, OverTime: []ReferralBucket{},
		Channels: []ReferralChannel{}, Top: []ReferralTopReferrer{}, Rewards: []ReferralRewardRow{}}
	f := &out.Funnel
	f.Reasons = []ReferralReasonCount{}
	err = s.Pool.QueryRow(ctx, `SELECT count(*),
			count(*) FILTER (WHERE confirmed_at IS NOT NULL),
			count(*) FILTER (WHERE status = 'counted'),
			count(*) FILTER (WHERE status = 'hold'),
			max(hold_until) FILTER (WHERE status = 'hold'),
			count(*) FILTER (WHERE status = 'review'),
			count(*) FILTER (WHERE status = 'not_counted' AND confirmed_at IS NOT NULL),
			count(DISTINCT link_id) FILTER (WHERE status = 'counted')
		FROM referrals WHERE program_id = $1 AND signed_up_at >= $2`, programID, from).
		Scan(&f.Signups, &f.Confirmed, &f.Counted, &f.InHold, &f.HoldEndsBy, &f.Held, &f.NotCounted, &f.Referrers)
	if err != nil {
		return out, err
	}
	f.Unconfirmed = f.Signups - f.Confirmed
	f.Weeks = int(now.Sub(from).Hours()/(24*7) + 0.5)
	if err := s.Pool.QueryRow(ctx, `SELECT coalesce(sum(visits), 0) FROM referral_visits_daily WHERE program_id = $1 AND day >= $2::date`,
		programID, dayOf(from, loc)).Scan(&f.Visits); err != nil {
		return out, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM referral_rewards WHERE program_id = $1 AND sent_at IS NOT NULL AND reached_at >= $2`,
		programID, from).Scan(&f.RewardsSent); err != nil {
		return out, err
	}
	rrows, err := s.Pool.Query(ctx, `SELECT reason, left_kind, count(*) FROM referrals
		WHERE program_id = $1 AND signed_up_at >= $2 AND status = 'not_counted' AND confirmed_at IS NOT NULL
		GROUP BY 1, 2 ORDER BY 3 DESC, 1`, programID, from)
	if err != nil {
		return out, err
	}
	for rrows.Next() {
		var rc ReferralReasonCount
		if err := rrows.Scan(&rc.Reason, &rc.LeftKind, &rc.Count); err != nil {
			rrows.Close()
			return out, err
		}
		f.Reasons = append(f.Reasons, rc)
	}
	rrows.Close()
	out.Enough = f.Confirmed >= MinConfirmedForResults
	if out.OverTime, err = s.overTime(ctx, programID, from, now, loc, grouping); err != nil {
		return out, err
	}
	crow, err := s.Pool.Query(ctx, `SELECT via, sum(visits), sum(signups) FROM referral_visits_daily
		WHERE program_id = $1 AND day >= $2::date GROUP BY 1`, programID, dayOf(from, loc))
	if err != nil {
		return out, err
	}
	byVia := map[string]ReferralChannel{}
	for crow.Next() {
		var c ReferralChannel
		if err := crow.Scan(&c.Channel, &c.Visits, &c.Signups); err != nil {
			crow.Close()
			return out, err
		}
		byVia[c.Channel] = c
	}
	crow.Close()
	for _, ch := range Channels {
		c := byVia[ch]
		c.Channel = ch
		out.Channels = append(out.Channels, c)
	}
	sortChannels(out.Channels)
	viewer := api.PrincipalFrom(ctx)
	top, err := s.Referrers(ctx, ws, programID, ReferrerParams{PageSize: 7})
	if err != nil {
		return out, err
	}
	for _, r := range top.Items {
		if r.Counted == 0 && r.InHold == 0 && r.Held == 0 {
			continue
		}
		t := ReferralTopReferrer{LinkID: r.LinkID, Name: r.Name, Counted: r.Counted, InHold: r.InHold, Held: r.Held, Milestones: r.Milestones,
			Unclaimed: r.Unclaimed, LastReferral: r.LastReferral}
		if viewer.Can(authz.SubscribersView) {
			t.Email = r.Email
		}
		out.Top = append(out.Top, t)
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE n >= 3), count(*) FILTER (WHERE n >= 10)
		FROM (SELECT link_id, count(*) AS n FROM referrals WHERE program_id = $1 AND status = 'counted' GROUP BY 1) x`, programID).Scan(&out.WithThree, &out.WithTen); err != nil {
		return out, err
	}
	views, err := s.milestoneViews(ctx, ws, programID)
	if err != nil {
		return out, err
	}
	for _, m := range views {
		out.Rewards = append(out.Rewards, ReferralRewardRow{MilestoneID: m.ID, Count: m.Count, Name: m.RewardName, Type: m.Type, Reached: m.Stats.Reached,
			Sent: m.Stats.Sent, Used: m.Stats.Used, Waiting: m.Stats.Waiting, Failed: m.Stats.Failed})
	}
	out.Quality, err = s.quality(ctx, ws, programID, from)
	return out, err
}

func sortChannels(cs []ReferralChannel) {
	// By visits, "Other" last.
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0; j-- {
			a, b := cs[j-1], cs[j]
			if a.Channel == ViaOther || (b.Channel != ViaOther && b.Visits > a.Visits) {
				cs[j-1], cs[j] = b, a
			}
		}
	}
}

// overTime is confirmed referrals per day or per week (weeks start on
// Monday); the last bucket may be partial.
func (s *Service) overTime(ctx context.Context, programID uuid.UUID, from, now time.Time, loc *time.Location, grouping string) ([]ReferralBucket, error) {
	rows, err := s.Pool.Query(ctx, `SELECT (confirmed_at AT TIME ZONE $3)::date, count(*) FROM referrals
		WHERE program_id = $1 AND confirmed_at IS NOT NULL AND confirmed_at >= $2 GROUP BY 1`, programID, from, loc.String())
	if err != nil {
		return nil, err
	}
	perDay := map[string]int64{}
	for rows.Next() {
		var d time.Time
		var n int64
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		perDay[d.Format("2006-01-02")] = n
	}
	rows.Close()
	start := dayOf(from, loc)
	end := dayOf(now, loc)
	out := []ReferralBucket{}
	if grouping == "day" {
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			out = append(out, ReferralBucket{Start: d, End: d, Confirmed: perDay[d.Format("2006-01-02")], Days: 1, Partial: d.Equal(end)})
		}
		return out, nil
	}
	// Weeks from Monday.
	wd := int(start.Weekday())
	if wd == 0 {
		wd = 7
	}
	ws := start.AddDate(0, 0, -(wd - 1))
	for w := ws; !w.After(end); w = w.AddDate(0, 0, 7) {
		b := ReferralBucket{Start: w, End: w.AddDate(0, 0, 6), Days: 7}
		for d := w; d.Before(w.AddDate(0, 0, 7)); d = d.AddDate(0, 0, 1) {
			b.Confirmed += perDay[d.Format("2006-01-02")]
		}
		if b.End.After(end) {
			b.End, b.Partial = end, true
			b.Days = int(end.Sub(w).Hours()/24) + 1
		}
		out = append(out, b)
	}
	return out, nil
}

// quality compares referred subscribers with the whole list: human opens
// on the list's last issue, unsubscribes and bounces since joining.
func (s *Service) quality(ctx context.Context, ws *store.Workspace, programID uuid.UUID, from time.Time) (ReferralQuality, error) {
	var q ReferralQuality
	p, err := s.Q.ReferralProgramByID(ctx, programID)
	if err != nil {
		return q, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE program_id = $1 AND confirmed_at IS NOT NULL AND signed_up_at >= $2`,
		programID, from).Scan(&q.Referred); err != nil {
		return q, err
	}
	q.Enough = q.Referred >= MinForQuality
	// The list's last issue: human opens of referred subscribers who got it
	// against the whole send.
	var campaign uuid.UUID
	err = s.Pool.QueryRow(ctx, `SELECT id, name FROM campaigns WHERE workspace_id = $1 AND deleted_at IS NULL AND status = 'sent'
		AND audience->'lists' ? $2::text AND started_at < $3 ORDER BY started_at DESC LIMIT 1`, ws.ID, p.ListID.String(), s.now().Add(-24*time.Hour)).Scan(&campaign, &q.Campaign)
	if err == nil {
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE m.human_opened_at IS NOT NULL), count(*)
			FROM messages m JOIN referrals r ON r.contact_id = m.contact_id AND r.program_id = $2 AND r.confirmed_at IS NOT NULL
			WHERE m.campaign_id = $1 AND m.status IN ('delivered', 'sent')`, campaign, programID).Scan(&q.Opens.ReferredN, &q.Opens.ReferredOf); err != nil {
			return q, err
		}
		if err := s.Pool.QueryRow(ctx, `SELECT coalesce(sum(opened_human), 0), coalesce(sum(delivered), 0) FROM campaign_counters WHERE campaign_id = $1`,
			campaign).Scan(&q.Opens.OthersN, &q.Opens.OthersOf); err != nil {
			return q, err
		}
		q.Opens.Referred, q.Opens.Others = ratio(q.Opens.ReferredN, q.Opens.ReferredOf), ratio(q.Opens.OthersN, q.Opens.OthersOf)
	}
	// Since joining, against the list's other new subscribers in the same
	// weeks.
	err = s.Pool.QueryRow(ctx, `WITH referred AS (
			SELECT r.contact_id FROM referrals r WHERE r.program_id = $1 AND r.confirmed_at IS NOT NULL AND r.signed_up_at >= $3 AND r.contact_id IS NOT NULL
		), others AS (
			SELECT lm.contact_id FROM list_memberships lm WHERE lm.list_id = $2 AND lm.subscribed_at >= $3
			  AND NOT EXISTS (SELECT 1 FROM referrals r WHERE r.program_id = $1 AND r.contact_id = lm.contact_id)
			  AND NOT EXISTS (SELECT 1 FROM contact_growth g WHERE g.contact_id = lm.contact_id AND g.source_kind = 'partner')
		)
		SELECT
		  (SELECT count(*) FROM referred x JOIN list_memberships lm ON lm.contact_id = x.contact_id AND lm.list_id = $2 WHERE lm.status = 'unsubscribed'),
		  (SELECT count(*) FROM referred),
		  (SELECT count(*) FROM others x JOIN list_memberships lm ON lm.contact_id = x.contact_id AND lm.list_id = $2 WHERE lm.status = 'unsubscribed'),
		  (SELECT count(*) FROM others),
		  (SELECT count(*) FROM referred x JOIN contacts c ON c.id = x.contact_id JOIN suppressions s ON s.workspace_id = c.workspace_id AND s.email = c.email AND s.reason = 'hard_bounce'),
		  (SELECT count(*) FROM others x JOIN contacts c ON c.id = x.contact_id JOIN suppressions s ON s.workspace_id = c.workspace_id AND s.email = c.email AND s.reason = 'hard_bounce')`,
		programID, p.ListID, from).Scan(&q.Unsubscribed.ReferredN, &q.Unsubscribed.ReferredOf, &q.Unsubscribed.OthersN, &q.Unsubscribed.OthersOf,
		&q.Bounced.ReferredN, &q.Bounced.OthersN)
	if err != nil {
		return q, err
	}
	q.Bounced.ReferredOf, q.Bounced.OthersOf = q.Unsubscribed.ReferredOf, q.Unsubscribed.OthersOf
	q.Unsubscribed.Referred, q.Unsubscribed.Others = ratio(q.Unsubscribed.ReferredN, q.Unsubscribed.ReferredOf), ratio(q.Unsubscribed.OthersN, q.Unsubscribed.OthersOf)
	q.Bounced.Referred, q.Bounced.Others = ratio(q.Bounced.ReferredN, q.Bounced.ReferredOf), ratio(q.Bounced.OthersN, q.Bounced.OthersOf)
	return q, nil
}

func ratio(n, of int64) float64 {
	if of == 0 {
		return 0
	}
	return float64(n) / float64(of) * 100
}
