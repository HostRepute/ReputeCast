package referrals

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReferralContactRewards is a reward on the contact profile's Referrals card.
type ReferralContactReward struct {
	Name      string     `json:"name"`
	Milestone int        `json:"milestone"`
	Status    string     `json:"status"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
}

// ReferralContactReferrer is a contact as a referrer: the profile's
// Referrals card.
type ReferralContactReferrer struct {
	ProgramID  string                  `json:"programId"`
	ListName   string                  `json:"listName"`
	LinkID     string                  `json:"linkId"`
	Link       string                  `json:"link"`
	Code       string                  `json:"code"`
	LinkStatus string                  `json:"linkStatus" enum:"active,paused"`
	Counted    int64                   `json:"counted"`
	InHold     int64                   `json:"inHold"`
	Awaiting   int64                   `json:"awaiting"`
	Held       int64                   `json:"held"`
	Visits     int64                   `json:"visits"`
	Signups    int64                   `json:"signups"`
	Milestones []int                   `json:"milestones"`
	Rewards    []ReferralContactReward `json:"rewards"`
}

// ReferralContactReferredBy is how a contact joined through a referral.
type ReferralContactReferredBy struct {
	ProgramID  string     `json:"programId"`
	LinkID     string     `json:"linkId"`
	ReferrerID string     `json:"referrerId"`
	Referrer   string     `json:"referrer"`
	Status     string     `json:"status" enum:"awaiting,hold,review,counted,not_counted"`
	Reason     string     `json:"reason"`
	CountsOn   *time.Time `json:"countsOn,omitempty"`
	CountedAt  *time.Time `json:"countedAt,omitempty"`
}

// ReferralTimelineEvent is a Grow event on a contact's timeline.
type ReferralTimelineEvent struct {
	Kind string    `json:"kind" enum:"joined,counted,milestone,reward_sent,partner_request"`
	At   time.Time `json:"at"`
	// Referrer names who referred them, Counted their count after.
	Referrer   string `json:"referrer"`
	ReferrerID string `json:"referrerId"`
	Counted    int64  `json:"counted"`
	Milestone  int    `json:"milestone"`
	Reward     string `json:"reward"`
	// Partner names the newsletter a partner request went to.
	Partner string `json:"partner"`
}

// ReferralContact is Grow on a contact's profile.
type ReferralContact struct {
	Referrer   *ReferralContactReferrer   `json:"referrer,omitempty"`
	ReferredBy *ReferralContactReferredBy `json:"referredBy,omitempty"`
	Events     []ReferralTimelineEvent    `json:"events"`
}

// PartnerEvents adds the partner requests to a contact's timeline (the
// recommendations package).
type PartnerEvents interface {
	ContactEvents(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) ([]ReferralTimelineEvent, error)
}

// Contact is Grow on a contact's profile: their link and counts as a
// referrer, who referred them, and the timeline's events.
func (s *Service) Contact(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, partners PartnerEvents) (ReferralContact, error) {
	out := ReferralContact{Events: []ReferralTimelineEvent{}}
	var exists bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL)`, contactID, ws.ID).Scan(&exists); err != nil {
		return out, err
	}
	if !exists {
		return out, api.NotFound("Contact")
	}
	links, err := s.Q.ReferralLinksOfContact(ctx, contactID)
	if err != nil {
		return out, err
	}
	pages, _ := s.pages(ctx, ws)
	if len(links) > 0 {
		l := links[0]
		var list string
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, l.ListID).Scan(&list)
		full := pages.Join(l.Code)
		r := &ReferralContactReferrer{ProgramID: l.ProgramID.String(), ListName: list, LinkID: l.ID.String(), Code: l.Code, LinkStatus: l.Status,
			Link: strings.TrimPrefix(strings.TrimPrefix(full, "https://"), "http://"), Visits: l.Visits, Signups: l.Signups,
			Milestones: []int{}, Rewards: []ReferralContactReward{}}
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'counted'), count(*) FILTER (WHERE status = 'hold'),
			count(*) FILTER (WHERE status = 'awaiting'), count(*) FILTER (WHERE status = 'review') FROM referrals WHERE link_id = $1`, l.ID).
			Scan(&r.Counted, &r.InHold, &r.Awaiting, &r.Held); err != nil {
			return out, err
		}
		rows, err := s.Q.ReferralRewardsOfContact(ctx, store.ReferralRewardsOfContactParams{ProgramID: l.ProgramID, ContactID: contactID})
		if err != nil {
			return out, err
		}
		for _, row := range rows {
			rw := row.ReferralReward
			r.Milestones = append(r.Milestones, int(rw.MilestoneCount))
			r.Rewards = append(r.Rewards, ReferralContactReward{Name: rw.RewardName, Milestone: int(rw.MilestoneCount), Status: rw.Status, SentAt: rw.SentAt, UsedAt: rw.UsedAt})
		}
		out.Referrer = r
	}
	var ref struct {
		program, link, referrer *uuid.UUID
		status, reason          string
		countsOn, countedAt     *time.Time
		first, last, email      *string
	}
	err = s.Pool.QueryRow(ctx, `SELECT r.program_id, r.link_id, r.referrer_id, r.status, r.reason, r.hold_until, r.counted_at, c.first_name, c.last_name, c.email
		FROM referrals r LEFT JOIN contacts c ON c.id = r.referrer_id WHERE r.contact_id = $1 AND r.workspace_id = $2 ORDER BY r.signed_up_at LIMIT 1`,
		contactID, ws.ID).Scan(&ref.program, &ref.link, &ref.referrer, &ref.status, &ref.reason, &ref.countsOn, &ref.countedAt, &ref.first, &ref.last, &ref.email)
	if err == nil && ref.program != nil {
		rb := &ReferralContactReferredBy{ProgramID: ref.program.String(), Status: ref.status, Reason: ref.reason, CountsOn: ref.countsOn, CountedAt: ref.countedAt}
		if ref.link != nil {
			rb.LinkID = ref.link.String()
		}
		if ref.referrer != nil {
			rb.ReferrerID = ref.referrer.String()
			rb.Referrer = displayName(deref(ref.first), deref(ref.last), deref(ref.email))
		}
		out.ReferredBy = rb
	}
	evs, err := s.timeline(ctx, ws, contactID)
	if err != nil {
		return out, err
	}
	out.Events = evs
	if partners != nil {
		pe, err := partners.ContactEvents(ctx, ws, contactID)
		if err != nil {
			return out, err
		}
		out.Events = append(out.Events, pe...)
	}
	return out, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// timeline is a contact's Grow events: joining through a link, their
// referral counting (with the referrer's count), milestones and rewards.
func (s *Service) timeline(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) ([]ReferralTimelineEvent, error) {
	rows, err := s.Pool.Query(ctx, `SELECT e.kind, e.occurred_at, e.detail, coalesce(c.first_name, ''), coalesce(c.last_name, ''), coalesce(c.email, ''), coalesce(rl.contact_id::text, '')
		FROM referral_events e
		LEFT JOIN referral_links rl ON rl.id = e.link_id
		LEFT JOIN contacts c ON c.id = rl.contact_id
		WHERE e.workspace_id = $1 AND (e.contact_id = $2 OR (e.kind = 'counted' AND e.detail->>'referred' = $2::text))
		  AND e.kind IN ('signed_up', 'counted', 'milestone_reached', 'reward_sent')
		ORDER BY e.occurred_at DESC LIMIT 50`, ws.ID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReferralTimelineEvent{}
	for rows.Next() {
		var kind, first, last, email, referrerID string
		var at time.Time
		var raw []byte
		if err := rows.Scan(&kind, &at, &raw, &first, &last, &email, &referrerID); err != nil {
			return nil, err
		}
		var d map[string]any
		_ = json.Unmarshal(raw, &d)
		ev := ReferralTimelineEvent{At: at, Referrer: displayName(first, last, email), ReferrerID: referrerID}
		switch kind {
		case EventSignedUp:
			ev.Kind = "joined"
		case EventCounted:
			ev.Kind = "counted"
			if n, ok := d["counted"].(float64); ok {
				ev.Counted = int64(n)
			}
			// On the referrer's own timeline the referrer is themselves.
			if referred, _ := d["referred"].(string); referred != contactID.String() {
				continue
			}
		case EventMilestoneReached:
			ev.Kind = "milestone"
			if n, ok := d["count"].(float64); ok {
				ev.Milestone = int(n)
			}
			ev.Reward, _ = d["reward"].(string)
		case EventRewardSent:
			ev.Kind = "reward_sent"
			if n, ok := d["count"].(float64); ok {
				ev.Milestone = int(n)
			}
			ev.Reward, _ = d["reward"].(string)
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}
