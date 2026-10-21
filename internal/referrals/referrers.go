package referrals

import (
	"context"
	"encoding/csv"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReferralReferrerRow is one row of Referrals › Referrers.
type ReferralReferrerRow struct {
	LinkID     string `json:"linkId"`
	ContactID  string `json:"contactId"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Counted    int64  `json:"counted"`
	InHold     int64  `json:"inHold"`
	Held       int64  `json:"held"`
	NotCounted int64  `json:"notCounted"`
	// Milestones reached (their counts).
	Milestones []int `json:"milestones"`
	// Rewards sent, and a coupon not claimed yet.
	RewardsSent  int64      `json:"rewardsSent"`
	Unclaimed    string     `json:"unclaimed"`
	LinkStatus   string     `json:"linkStatus" enum:"active,paused"`
	LastReferral *time.Time `json:"lastReferral,omitempty"`
}

// ReferralReferrerList is a page of referrers.
type ReferralReferrerList struct {
	Items []ReferralReferrerRow `json:"items"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	// WithCounted is how many referrers have a counted referral.
	WithCounted int64 `json:"withCounted"`
	// Filters' counts.
	HeldReferrers   int64                  `json:"heldReferrers"`
	PausedLinks     int64                  `json:"pausedLinks"`
	MilestoneCounts []ReferralMilestoneCnt `json:"milestoneCounts"`
	ShowAddresses   bool                   `json:"showAddresses"`
	Helpers         GrowHelperSets         `json:"helpers"`
}

// ReferralMilestoneCnt is how many referrers reached a milestone.
type ReferralMilestoneCnt struct {
	Count     int   `json:"count"`
	Referrers int64 `json:"referrers"`
}

// ReferrerParams filters the referrers.
type ReferrerParams struct {
	Query     string
	Milestone int // 0 any, -1 none yet, n reached n
	Held      bool
	Paused    bool
	Page      int
	PageSize  int
}

// referrersSQL lists referrers: every link with a referral, or paused.
const referrersSQL = `WITH agg AS (
    SELECT r.link_id,
           count(*) FILTER (WHERE r.status = 'counted') AS counted,
           count(*) FILTER (WHERE r.status = 'hold') AS in_hold,
           count(*) FILTER (WHERE r.status = 'review') AS held,
           count(*) FILTER (WHERE r.status = 'not_counted') AS not_counted,
           max(r.signed_up_at) AS last_referral
    FROM referrals r WHERE r.program_id = $1 AND r.link_id IS NOT NULL GROUP BY r.link_id
)
SELECT rl.id, rl.contact_id, c.first_name, c.last_name, c.email, coalesce(a.counted, 0), coalesce(a.in_hold, 0), coalesce(a.held, 0),
       coalesce(a.not_counted, 0), rl.status, a.last_referral,
       (SELECT coalesce(array_agg(rw.milestone_count ORDER BY rw.milestone_count), '{}') FROM referral_rewards rw WHERE rw.program_id = $1 AND rw.contact_id = rl.contact_id)::int[],
       (SELECT count(*) FROM referral_rewards rw WHERE rw.program_id = $1 AND rw.contact_id = rl.contact_id AND rw.sent_at IS NOT NULL),
       coalesce((SELECT rw.reward_name FROM referral_rewards rw WHERE rw.program_id = $1 AND rw.contact_id = rl.contact_id AND rw.reward_type = 'coupon'
                 AND rw.status = 'sent' AND rw.used_at IS NULL ORDER BY rw.milestone_count LIMIT 1), ''),
       count(*) OVER ()
FROM referral_links rl
JOIN contacts c ON c.id = rl.contact_id
LEFT JOIN agg a ON a.link_id = rl.id
WHERE rl.program_id = $1 AND (a.link_id IS NOT NULL OR rl.status = 'paused')
  AND ($2 = '' OR lower(c.email) LIKE '%' || lower($2) || '%' OR lower(c.first_name || ' ' || c.last_name) LIKE '%' || lower($2) || '%')
  AND (NOT $3::boolean OR coalesce(a.held, 0) > 0)
  AND (NOT $4::boolean OR rl.status = 'paused')
  AND ($5::int = 0
       OR ($5::int < 0 AND NOT EXISTS (SELECT 1 FROM referral_rewards rw WHERE rw.program_id = $1 AND rw.contact_id = rl.contact_id))
       OR ($5::int > 0 AND EXISTS (SELECT 1 FROM referral_rewards rw WHERE rw.program_id = $1 AND rw.contact_id = rl.contact_id AND rw.milestone_count = $5::int)))
ORDER BY coalesce(a.counted, 0) DESC, coalesce(a.in_hold, 0) DESC, a.last_referral DESC NULLS LAST, rl.id
LIMIT $6 OFFSET $7`

// Referrers is Referrals › Referrers: everyone whose link brought a
// signup, by counted referrals.
func (s *Service) Referrers(ctx context.Context, ws *store.Workspace, programID uuid.UUID, p ReferrerParams) (ReferralReferrerList, error) {
	if _, err := s.program(ctx, ws, programID); err != nil {
		return ReferralReferrerList{}, err
	}
	if p.PageSize <= 0 {
		p.PageSize = 25
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	viewer := api.PrincipalFrom(ctx)
	out := ReferralReferrerList{Items: []ReferralReferrerRow{}, Page: p.Page, MilestoneCounts: []ReferralMilestoneCnt{},
		ShowAddresses: viewer.Can(authz.SubscribersView), Helpers: s.helperSets(ctx, ws.ID)}
	rows, err := s.Pool.Query(ctx, referrersSQL, programID, strings.TrimSpace(p.Query), p.Held, p.Paused, p.Milestone, p.PageSize, (p.Page-1)*p.PageSize)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var r ReferralReferrerRow
		var link, contact uuid.UUID
		var first, last, email string
		var ms []int32
		if err := rows.Scan(&link, &contact, &first, &last, &email, &r.Counted, &r.InHold, &r.Held, &r.NotCounted, &r.LinkStatus, &r.LastReferral,
			&ms, &r.RewardsSent, &r.Unclaimed, &out.Total); err != nil {
			rows.Close()
			return out, err
		}
		r.LinkID, r.ContactID = link.String(), contact.String()
		r.Name = displayName(first, last, email)
		if out.ShowAddresses {
			r.Email = email
		} else if displayName(first, last, "") == "" {
			r.Name = ""
		}
		r.Milestones = make([]int, 0, len(ms))
		for _, m := range ms {
			r.Milestones = append(r.Milestones, int(m))
		}
		out.Items = append(out.Items, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	err = s.Pool.QueryRow(ctx, `SELECT
		(SELECT count(DISTINCT link_id) FROM referrals WHERE program_id = $1 AND status = 'counted'),
		(SELECT count(DISTINCT link_id) FROM referrals WHERE program_id = $1 AND status = 'review'),
		(SELECT count(*) FROM referral_links WHERE program_id = $1 AND status = 'paused')`, programID).Scan(&out.WithCounted, &out.HeldReferrers, &out.PausedLinks)
	if err != nil {
		return out, err
	}
	mrows, err := s.Pool.Query(ctx, `SELECT milestone_count, count(*) FROM referral_rewards WHERE program_id = $1 GROUP BY 1 ORDER BY 1`, programID)
	if err != nil {
		return out, err
	}
	for mrows.Next() {
		var c ReferralMilestoneCnt
		var cnt int32
		if err := mrows.Scan(&cnt, &c.Referrers); err != nil {
			mrows.Close()
			return out, err
		}
		c.Count = int(cnt)
		out.MilestoneCounts = append(out.MilestoneCounts, c)
	}
	mrows.Close()
	return out, mrows.Err()
}

// ReferralTrackPoint is a milestone on a referrer's track.
type ReferralTrackPoint struct {
	Count     int        `json:"count"`
	Reward    string     `json:"reward"`
	ReachedAt *time.Time `json:"reachedAt,omitempty"`
}

// ReferralTrack is a referrer's progress: counted, counting soon (in hold),
// waiting for confirmation, and the milestones.
type ReferralTrack struct {
	Counted   int64                `json:"counted"`
	InHold    int64                `json:"inHold"`
	Awaiting  int64                `json:"awaiting"`
	Held      int64                `json:"held"`
	Points    []ReferralTrackPoint `json:"points"`
	NextHold  *time.Time           `json:"nextHold,omitempty" doc:"When the next referral in hold counts"`
	NextCount int                  `json:"nextCount"`
	NextName  string               `json:"nextName"`
	ToNext    int64                `json:"toNext"`
	// NextBySeq is the referral whose hold brings the next milestone.
	NextBySeq int `json:"nextBySeq"`
}

// ReferralItem is one of a referrer's referrals.
type ReferralItem struct {
	ID          string     `json:"id"`
	Number      int        `json:"number"`
	Email       string     `json:"email"`
	Status      string     `json:"status" enum:"awaiting,hold,review,counted,not_counted"`
	Reason      string     `json:"reason"`
	LeftKind    string     `json:"leftKind"`
	SignedUpAt  time.Time  `json:"signedUpAt"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	CountsOn    *time.Time `json:"countsOn,omitempty"`
	CountedAt   *time.Time `json:"countedAt,omitempty"`
	RemindedAt  *time.Time `json:"remindedAt,omitempty"`
}

// ReferralRewardItem is a reward a referrer received.
type ReferralRewardItem struct {
	ID        string `json:"id"`
	Milestone int    `json:"milestone"`
	Name      string `json:"name"`
	Type      string `json:"type" enum:"download,codes,manual,coupon,webhook"`
	Status    string `json:"status" enum:"pending,sent,waiting_codes,to_fulfill,fulfilled,failed,expired"`
	// Used: downloaded, claimed or fulfilled.
	ReachedAt time.Time  `json:"reachedAt"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	FileName  string     `json:"fileName"`
	FileType  string     `json:"fileType"`
	FileSize  int64      `json:"fileSize"`
	HasClaim  bool       `json:"hasClaim"`
	Note      string     `json:"note"`
	Error     string     `json:"error"`
	// Code is shown to members with grow:review.
	Code string `json:"code,omitempty"`
}

// ReferralReferrerPanel is the referrer panel (it has its own link).
type ReferralReferrerPanel struct {
	Referrer ReferralPerson `json:"referrer"`
	// Link and its numbers.
	Link        string     `json:"link"`
	Code        string     `json:"code"`
	LinkCreated time.Time  `json:"linkCreated"`
	Visits      int64      `json:"visits"`
	Signups     int64      `json:"signups"`
	PausedAt    *time.Time `json:"pausedAt,omitempty"`
	PausedBy    string     `json:"pausedBy"`
	PauseNote   string     `json:"pauseNote"`
	// PauseReason summarizes the review decision that paused it.
	PausedNotCounted int64    `json:"pausedNotCounted"`
	PausedSignals    []string `json:"pausedSignals"`
	ShowName         bool     `json:"showName"`
	ShowLeaderboard  bool     `json:"showLeaderboard"`
	// FormsHost is where ?ref= also works ("acme.io forms").
	Track     ReferralTrack        `json:"track"`
	Referrals []ReferralItem       `json:"referrals"`
	Rewards   []ReferralRewardItem `json:"rewards"`
	// ShowAddresses: the referred addresses show to members with
	// subscribers:view.
	ShowAddresses bool           `json:"showAddresses"`
	Helpers       GrowHelperSets `json:"helpers"`
}

// Referrer is a referrer's panel.
func (s *Service) Referrer(ctx context.Context, ws *store.Workspace, programID, linkID uuid.UUID) (ReferralReferrerPanel, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralReferrerPanel{}, err
	}
	p := cur.ReferralProgram
	l, err := s.Q.ReferralLinkGet(ctx, store.ReferralLinkGetParams{WorkspaceID: ws.ID, ID: linkID})
	if err != nil || l.ProgramID != programID {
		return ReferralReferrerPanel{}, api.NotFound("Referrer")
	}
	viewer := api.PrincipalFrom(ctx)
	out := ReferralReferrerPanel{Code: l.Code, LinkCreated: l.CreatedAt, Visits: l.Visits, Signups: l.Signups, PausedAt: l.PausedAt, PausedBy: l.PausedByName,
		PauseNote: l.PauseNote, PausedSignals: []string{}, ShowName: l.ShowName, ShowLeaderboard: l.ShowLeaderboard, Referrals: []ReferralItem{},
		Rewards: []ReferralRewardItem{}, ShowAddresses: viewer.Can(authz.SubscribersView), Helpers: s.helperSets(ctx, ws.ID)}
	if out.Referrer, err = s.person(ctx, p, linkID, out.ShowAddresses); err != nil {
		return out, err
	}
	if links, err := s.pages(ctx, ws); err == nil {
		out.Link = strings.TrimPrefix(strings.TrimPrefix(links.Join(l.Code), "https://"), "http://")
	}
	if out.Referrals, err = s.referralItems(ctx, linkID, out.ShowAddresses); err != nil {
		return out, err
	}
	if out.Track, err = s.track(ctx, p, linkID, out.Referrals); err != nil {
		return out, err
	}
	if l.Status == "paused" {
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE link_id = $1 AND status = 'not_counted' AND decision = 'dont_pause'`, linkID).Scan(&out.PausedNotCounted); err != nil {
			return out, err
		}
		var signals []string
		if err := s.Pool.QueryRow(ctx, `SELECT coalesce(array_agg(DISTINCT s), '{}') FROM referrals, unnest(signals) s WHERE link_id = $1 AND decision = 'dont_pause'`, linkID).Scan(&signals); err != nil {
			return out, err
		}
		out.PausedSignals = signals
	}
	out.Rewards, err = s.rewardItems(ctx, programID, l.ContactID, viewer.Can(authz.GrowReview))
	return out, err
}

func (s *Service) referralItems(ctx context.Context, linkID uuid.UUID, showAddress bool) ([]ReferralItem, error) {
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.seq, coalesce(c.email, ''), r.status, r.reason, r.left_kind, r.signed_up_at, r.confirmed_at, r.hold_until,
			r.counted_at, (SELECT max(fs.reminded_at) FROM form_submissions fs WHERE fs.contact_id = r.contact_id)
		FROM referrals r LEFT JOIN contacts c ON c.id = r.contact_id
		WHERE r.link_id = $1 ORDER BY r.seq`, linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReferralItem{}
	for rows.Next() {
		var it ReferralItem
		var id uuid.UUID
		var seq int32
		if err := rows.Scan(&id, &seq, &it.Email, &it.Status, &it.Reason, &it.LeftKind, &it.SignedUpAt, &it.ConfirmedAt, &it.CountsOn, &it.CountedAt, &it.RemindedAt); err != nil {
			return nil, err
		}
		it.ID, it.Number = id.String(), int(seq)
		if !showAddress {
			it.Email = ""
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// track is a referrer's progress toward the milestones.
func (s *Service) track(ctx context.Context, p store.ReferralProgram, linkID uuid.UUID, items []ReferralItem) (ReferralTrack, error) {
	t := ReferralTrack{Points: []ReferralTrackPoint{}}
	for _, it := range items {
		switch it.Status {
		case StatusCounted:
			t.Counted++
		case StatusHold:
			t.InHold++
			if it.CountsOn != nil && (t.NextHold == nil || it.CountsOn.Before(*t.NextHold)) {
				d := *it.CountsOn
				t.NextHold = &d
			}
		case StatusAwaiting:
			t.Awaiting++
		case StatusReview:
			t.Held++
		}
	}
	ms, err := s.Q.ReferralMilestones(ctx, p.ID)
	if err != nil {
		return t, err
	}
	var contact uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT contact_id FROM referral_links WHERE id = $1`, linkID).Scan(&contact); err != nil {
		return t, err
	}
	reached := map[int32]time.Time{}
	rows, err := s.Pool.Query(ctx, `SELECT milestone_count, reached_at FROM referral_rewards WHERE program_id = $1 AND contact_id = $2`, p.ID, contact)
	if err != nil {
		return t, err
	}
	for rows.Next() {
		var c int32
		var at time.Time
		if err := rows.Scan(&c, &at); err != nil {
			rows.Close()
			return t, err
		}
		reached[c] = at
	}
	rows.Close()
	for _, m := range ms {
		pt := ReferralTrackPoint{Count: int(m.Count), Reward: m.RewardName}
		if at, ok := reached[m.Count]; ok {
			a := at
			pt.ReachedAt = &a
		}
		t.Points = append(t.Points, pt)
		if t.NextCount == 0 && int64(m.Count) > t.Counted {
			t.NextCount, t.NextName = int(m.Count), m.RewardName
			t.ToNext = int64(m.Count) - t.Counted
		}
	}
	// The referral in hold that brings the next milestone, if it stays.
	if t.NextCount > 0 && t.ToNext <= t.InHold {
		n := int64(0)
		for _, it := range items {
			if it.Status == StatusHold {
				n++
				if n == t.ToNext {
					t.NextBySeq = it.Number
					if it.CountsOn != nil {
						d := *it.CountsOn
						t.NextHold = &d
					}
				}
			}
		}
	}
	return t, nil
}

func (s *Service) rewardItems(ctx context.Context, programID, contactID uuid.UUID, showCodes bool) ([]ReferralRewardItem, error) {
	rows, err := s.Q.ReferralRewardsOfContact(ctx, store.ReferralRewardsOfContactParams{ProgramID: programID, ContactID: contactID})
	if err != nil {
		return nil, err
	}
	out := []ReferralRewardItem{}
	for _, row := range rows {
		r := row.ReferralReward
		it := ReferralRewardItem{ID: r.ID.String(), Milestone: int(r.MilestoneCount), Name: r.RewardName, Type: r.RewardType, Status: r.Status,
			ReachedAt: r.ReachedAt, SentAt: r.SentAt, UsedAt: r.UsedAt, ExpiresAt: r.ExpiresAt, FileName: row.FileName, FileType: row.FileType,
			FileSize: row.FileSize, HasClaim: r.ClaimUrl != "", Note: r.FulfillNote, Error: r.LastError}
		if showCodes {
			it.Code, _ = s.rewardCode(ctx, r)
		}
		out = append(out, it)
	}
	return out, nil
}

// SetLinkStatus pauses or resumes a referrer's link. A paused link leads to
// the normal subscribe page, which doesn't say why.
func (s *Service) SetLinkStatus(ctx context.Context, ws *store.Workspace, programID, linkID uuid.UUID, status, note string) error {
	l, err := s.Q.ReferralLinkGet(ctx, store.ReferralLinkGetParams{WorkspaceID: ws.ID, ID: linkID})
	if err != nil || l.ProgramID != programID {
		return api.NotFound("Referrer")
	}
	if l.Status == status {
		key := "link.active"
		if status == "paused" {
			key = "link.paused"
		}
		return api.Err(http.StatusConflict, "link_"+status, s.text(ctx, msgs.M(key), nil))
	}
	name := displayName(l.ContactFirstName, l.ContactLastName, l.ContactEmail)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.ReferralLinkSetStatus(ctx, store.ReferralLinkSetStatusParams{ID: linkID, Status: status, Now: now, By: actorUser(ctx), Note: strings.TrimSpace(note)}); err != nil {
			return err
		}
		kind, action := EventLinkResumed, "referral.link_resumed"
		if status == "paused" {
			kind, action = EventLinkPaused, "referral.link_paused"
		}
		contact := l.ContactID
		if err := writeLog(ctx, q, ws.ID, logEntry{program: programID, link: &linkID, contact: &contact, kind: kind, at: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "referral_link", TargetID: linkID.String(),
			TargetName: name, Reason: strings.TrimSpace(note), Metadata: map[string]any{"related": l.ContactID.String()}})
	})
}

// ExportReferrers writes the referrers list as CSV (subscribers:export,
// audited).
func (s *Service) ExportReferrers(ctx context.Context, ws *store.Workspace, programID uuid.UUID) ([]byte, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"name", "email", "counted", "in_hold", "held", "not_counted", "milestones", "rewards_sent", "link", "link_status", "last_referral"})
	page := 1
	links, _ := s.pages(ctx, ws)
	total := 0
	for {
		list, err := s.Referrers(ctx, ws, programID, ReferrerParams{Page: page, PageSize: 500})
		if err != nil {
			return nil, err
		}
		for _, r := range list.Items {
			ms := make([]string, 0, len(r.Milestones))
			for _, m := range r.Milestones {
				ms = append(ms, formatInt(int64(m)))
			}
			last := ""
			if r.LastReferral != nil {
				last = r.LastReferral.UTC().Format(time.RFC3339)
			}
			var code string
			_ = s.Pool.QueryRow(ctx, `SELECT code FROM referral_links WHERE id = $1`, r.LinkID).Scan(&code)
			_ = w.Write([]string{r.Name, r.Email, formatInt(r.Counted), formatInt(r.InHold), formatInt(r.Held), formatInt(r.NotCounted),
				strings.Join(ms, " "), formatInt(r.RewardsSent), links.Join(code), r.LinkStatus, last})
		}
		total += len(list.Items)
		if len(list.Items) < 500 {
			break
		}
		page++
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "referral.referrers_exported", TargetType: "referral_program",
		TargetID: programID.String(), TargetName: cur.ListName, Metadata: map[string]any{"summary": formatInt(int64(total)) + " referrers exported"}})
	return []byte(b.String()), err
}

// ExportToFulfill writes the "To fulfill" queue as CSV, with the shipping
// addresses (subscribers:export, audited).
func (s *Service) ExportToFulfill(ctx context.Context, ws *store.Workspace, programID uuid.UUID) ([]byte, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return nil, err
	}
	rows, err := s.Q.ReferralRewardsToFulfill(ctx, programID)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"referrer", "email", "milestone", "reward", "reached", "name", "address", "city", "country"})
	for _, r := range rows {
		rec := []string{displayName(r.FirstName, r.LastName, r.Email), r.Email, formatInt(int64(r.MilestoneCount)), r.RewardName, r.ReachedAt.UTC().Format(time.RFC3339), "", "", "", ""}
		if a := s.openAddress(r.ID, r.Address); a != nil {
			rec[5], rec[6], rec[7], rec[8] = a.Name, a.Street, a.City, a.Country
		}
		_ = w.Write(rec)
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "referral.to_fulfill_exported", TargetType: "referral_program",
		TargetID: programID.String(), TargetName: cur.ListName, Metadata: map[string]any{"summary": formatInt(int64(len(rows))) + " rewards to fulfill exported"}})
	return []byte(b.String()), err
}
