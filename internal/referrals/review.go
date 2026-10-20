package referrals

import (
	"context"
	"net/http"
	"slices"
	"sort"
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

// Review decisions.
const (
	DecisionCount     = "count"
	DecisionDont      = "dont"
	DecisionDontPause = "dont_pause"
)

// ReferralPerson is a referrer as the team sees them.
type ReferralPerson struct {
	ContactID string `json:"contactId"`
	Name      string `json:"name"`
	// Email shows to members with subscribers:view.
	Email      string     `json:"email"`
	FirstName  string     `json:"firstName"`
	Since      *time.Time `json:"since,omitempty" doc:"On the program's list since"`
	Counted    int64      `json:"counted"`
	LinkID     string     `json:"linkId"`
	LinkStatus string     `json:"linkStatus" enum:"active,paused"`
}

// ReferralNetwork is where a signup came from: the address's network, with
// its name, autonomous system and kind when known.
type ReferralNetwork struct {
	CIDR string `json:"cidr"`
	Name string `json:"name"`
	ASN  string `json:"asn"`
	Kind string `json:"kind" enum:"hosting,office,residential,mobile,"`
}

// ReferralEngagement is what a referred subscriber opened or clicked since
// confirming.
type ReferralEngagement struct {
	// Opened are the campaigns opened ("Weekly Digest #142"), Short their
	// short names ("#142").
	Opened []string `json:"opened"`
	Short  []string `json:"short"`
	Clicks int64    `json:"clicks"`
	// JoinedAfter: nothing was sent since they confirmed; the last issue
	// went out before ("#142").
	JoinedAfter string `json:"joinedAfter"`
}

// ReferralHeldItem is a held referral with its evidence.
type ReferralHeldItem struct {
	ID          string             `json:"id"`
	Email       string             `json:"email"`
	SignedUpAt  time.Time          `json:"signedUpAt"`
	ConfirmedAt *time.Time         `json:"confirmedAt,omitempty"`
	IP          string             `json:"ip"`
	Network     ReferralNetwork    `json:"network"`
	Engagement  ReferralEngagement `json:"engagement"`
	HoldEnds    *time.Time         `json:"holdEnds,omitempty"`
	Signals     []string           `json:"signals"`
}

// ReferralSignal is a suspicion rule that held a group, with the facts its
// sentence is written from.
type ReferralSignal struct {
	Kind      string          `json:"kind" enum:"burst,shared_ip,numbered"`
	Count     int             `json:"count"`
	Minutes   int             `json:"minutes"`
	From      *time.Time      `json:"from,omitempty"`
	To        *time.Time      `json:"to,omitempty"`
	Threshold int             `json:"threshold"`
	IP        string          `json:"ip"`
	Network   ReferralNetwork `json:"network"`
	// All: every held referral of the group shares it.
	All   bool   `json:"all"`
	First string `json:"first"`
	Last  string `json:"last"`
}

// ReferralReviewGroup is one referrer's held referrals.
type ReferralReviewGroup struct {
	Referrer ReferralPerson     `json:"referrer"`
	Items    []ReferralHeldItem `json:"items"`
	Signals  []ReferralSignal   `json:"signals"`
	// Suggestion: dont_pause ("Suggested: don't count, and pause the
	// link") or count ("Why it may be fine").
	Suggestion string `json:"suggestion" enum:"dont_pause,count,"`
	// Evidence of a group that may be fine.
	SameDomain     string      `json:"sameDomain"`
	DifferentNames bool        `json:"differentNames"`
	Days           int         `json:"days"`
	Opened         int         `json:"opened"`
	OpenedIn       string      `json:"openedIn"`
	ConfirmedDays  []time.Time `json:"confirmedDays"`
	HoldEnds       []time.Time `json:"holdEnds"`
	Engaged        bool        `json:"engaged"`
}

// ReferralSignalCount is how many referrers a signal holds (the filter).
type ReferralSignalCount struct {
	Kind      string `json:"kind"`
	Referrers int    `json:"referrers"`
}

// ReferralReviewQueue is Referrals › Review.
type ReferralReviewQueue struct {
	Held     int                   `json:"held"`
	Oldest   *time.Time            `json:"oldest,omitempty"`
	HoldDays int                   `json:"holdDays"`
	Groups   []ReferralReviewGroup `json:"groups"`
	Signals  []ReferralSignalCount `json:"signals"`
	// Referrers are the filter's options: name and held count.
	Referrers []ReferralFilterOption `json:"referrers"`
	// ShowAddresses: addresses and IPs show to members with subscribers:view.
	ShowAddresses bool           `json:"showAddresses"`
	Helpers       GrowHelperSets `json:"helpers"`
}

// ReferralFilterOption is a referrer in a filter.
type ReferralFilterOption struct {
	LinkID string `json:"linkId"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

type heldRow struct {
	id, link                 uuid.UUID
	referrer, contact        *uuid.UUID
	email                    string
	signedUp                 time.Time
	confirmed, holdEnds      *time.Time
	heldAt                   *time.Time
	ip                       *string
	cidr, netName, asn, kind string
	signals                  []string
}

// Review is the review queue: held referrals grouped by referrer, with
// their evidence and the signals that held them.
func (s *Service) Review(ctx context.Context, ws *store.Workspace, programID uuid.UUID, signal string, link *uuid.UUID) (ReferralReviewQueue, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralReviewQueue{}, err
	}
	p := cur.ReferralProgram
	viewer := api.PrincipalFrom(ctx)
	out := ReferralReviewQueue{Groups: []ReferralReviewGroup{}, Signals: []ReferralSignalCount{}, Referrers: []ReferralFilterOption{},
		HoldDays: int(p.HoldDays), ShowAddresses: viewer.Can(authz.SubscribersView), Helpers: s.helperSets(ctx, ws.ID)}
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.link_id, r.referrer_id, r.contact_id, coalesce(c.email, ''), r.signed_up_at, r.confirmed_at,
			r.confirmed_at + make_interval(days => $2), r.held_at, host(r.ip), r.ip_network, r.network_name, r.network_asn, r.network_kind, r.signals
		FROM referrals r LEFT JOIN contacts c ON c.id = r.contact_id
		WHERE r.program_id = $1 AND r.status = 'review' AND r.link_id IS NOT NULL
		ORDER BY r.signed_up_at, r.id`, programID, p.HoldDays)
	if err != nil {
		return out, err
	}
	var held []heldRow
	for rows.Next() {
		var h heldRow
		if err := rows.Scan(&h.id, &h.link, &h.referrer, &h.contact, &h.email, &h.signedUp, &h.confirmed, &h.holdEnds, &h.heldAt, &h.ip, &h.cidr,
			&h.netName, &h.asn, &h.kind, &h.signals); err != nil {
			rows.Close()
			return out, err
		}
		held = append(held, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Held = len(held)
	byLink := map[uuid.UUID][]heldRow{}
	var order []uuid.UUID
	for _, h := range held {
		if _, ok := byLink[h.link]; !ok {
			order = append(order, h.link)
		}
		byLink[h.link] = append(byLink[h.link], h)
		if h.heldAt != nil && (out.Oldest == nil || h.heldAt.Before(*out.Oldest)) {
			t := *h.heldAt
			out.Oldest = &t
		}
	}
	signalReferrers := map[string]int{}
	for _, lid := range order {
		items := byLink[lid]
		g, err := s.reviewGroup(ctx, ws, p, lid, items, out.ShowAddresses)
		if err != nil {
			return out, err
		}
		seen := map[string]bool{}
		for _, sg := range g.Signals {
			if !seen[sg.Kind] {
				seen[sg.Kind] = true
				signalReferrers[sg.Kind]++
			}
		}
		out.Referrers = append(out.Referrers, ReferralFilterOption{LinkID: lid.String(), Name: g.Referrer.Name, Count: len(items)})
		if signal != "" && !seen[signal] {
			continue
		}
		if link != nil && *link != lid {
			continue
		}
		out.Groups = append(out.Groups, g)
	}
	for _, k := range []string{SignalBurst, SignalSharedIP, SignalNumbered} {
		if n := signalReferrers[k]; n > 0 {
			out.Signals = append(out.Signals, ReferralSignalCount{Kind: k, Referrers: n})
		}
	}
	return out, nil
}

func (s *Service) person(ctx context.Context, p store.ReferralProgram, linkID uuid.UUID, showAddress bool) (ReferralPerson, error) {
	var out ReferralPerson
	var contact uuid.UUID
	var first, last, email, status string
	var since *time.Time
	err := s.Pool.QueryRow(ctx, `SELECT rl.contact_id, c.first_name, c.last_name, c.email, rl.status, lm.subscribed_at
		FROM referral_links rl JOIN contacts c ON c.id = rl.contact_id
		LEFT JOIN list_memberships lm ON lm.list_id = $2 AND lm.contact_id = rl.contact_id
		WHERE rl.id = $1`, linkID, p.ListID).Scan(&contact, &first, &last, &email, &status, &since)
	if err != nil {
		return out, err
	}
	counted, err := s.Q.ReferralCountedOf(ctx, store.ReferralCountedOfParams{ProgramID: p.ID, LinkID: &linkID})
	if err != nil {
		return out, err
	}
	out = ReferralPerson{ContactID: contact.String(), Name: displayName(first, last, email), FirstName: first, Since: since, Counted: counted,
		LinkID: linkID.String(), LinkStatus: status}
	if showAddress {
		out.Email = email
	}
	if !showAddress && displayName(first, last, "") == "" {
		out.Name = ""
	}
	return out, nil
}

func (s *Service) reviewGroup(ctx context.Context, ws *store.Workspace, p store.ReferralProgram, linkID uuid.UUID, items []heldRow, showAddress bool) (ReferralReviewGroup, error) {
	g := ReferralReviewGroup{Items: []ReferralHeldItem{}, Signals: []ReferralSignal{}, ConfirmedDays: []time.Time{}, HoldEnds: []time.Time{}}
	var err error
	if g.Referrer, err = s.person(ctx, p, linkID, showAddress); err != nil {
		return g, err
	}
	loc := workspaceLocation(ws)
	domains := map[string]bool{}
	locals := map[string]bool{}
	days := map[string]bool{}
	confirmedDays := map[string]time.Time{}
	holdDays := map[string]time.Time{}
	for _, h := range items {
		it := ReferralHeldItem{ID: h.id.String(), SignedUpAt: h.signedUp, ConfirmedAt: h.confirmed, HoldEnds: h.holdEnds, Signals: nonNil(h.signals),
			Network: ReferralNetwork{CIDR: h.cidr, Name: h.netName, ASN: h.asn, Kind: h.kind}}
		if showAddress {
			it.Email = h.email
			if h.ip != nil {
				it.IP = *h.ip
			}
		}
		if h.contact != nil && h.confirmed != nil {
			if it.Engagement, err = s.engagement(ctx, ws, p, *h.contact, *h.confirmed); err != nil {
				return g, err
			}
		}
		if len(it.Engagement.Opened) > 0 || it.Engagement.Clicks > 0 {
			g.Opened++
			if g.OpenedIn == "" && len(it.Engagement.Short) > 0 {
				g.OpenedIn = it.Engagement.Opened[0]
			}
		}
		g.Items = append(g.Items, it)
		domains[strings.ToLower(domainOf(h.email))] = true
		locals[NumberedKey(h.email)+"|"+localOf(h.email)] = true
		days[h.signedUp.In(loc).Format("2006-01-02")] = true
		if h.confirmed != nil {
			d := dayOf(*h.confirmed, loc)
			confirmedDays[d.Format("2006-01-02")] = d
		}
		if h.holdEnds != nil {
			d := dayOf(*h.holdEnds, loc)
			holdDays[d.Format("2006-01-02")] = d
		}
	}
	g.ConfirmedDays = append(g.ConfirmedDays, sortedDays(confirmedDays)...)
	g.HoldEnds = append(g.HoldEnds, sortedDays(holdDays)...)
	g.Days = len(days)
	g.Engaged = g.Opened > 0
	if len(domains) == 1 {
		for d := range domains {
			g.SameDomain = d
		}
	}
	g.DifferentNames = len(locals) == len(items)
	g.Signals = signalsOf(items, p)
	hasBurstOrNumbered := false
	for _, sg := range g.Signals {
		if sg.Kind == SignalBurst || sg.Kind == SignalNumbered {
			hasBurstOrNumbered = true
		}
	}
	switch {
	case hasBurstOrNumbered && !g.Engaged:
		g.Suggestion = DecisionDontPause
	case !hasBurstOrNumbered && g.Engaged && g.DifferentNames:
		g.Suggestion = DecisionCount
	}
	return g, nil
}

func sortedDays(m map[string]time.Time) []time.Time {
	out := make([]time.Time, 0, len(m))
	for _, d := range m {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func domainOf(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return email[i+1:]
	}
	return ""
}

func localOf(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return email[:i]
	}
	return email
}

// signalsOf writes the facts of each signal that held a group.
func signalsOf(items []heldRow, p store.ReferralProgram) []ReferralSignal {
	out := []ReferralSignal{}
	has := func(k string) []heldRow {
		var r []heldRow
		for _, h := range items {
			if slices.Contains(h.signals, k) {
				r = append(r, h)
			}
		}
		return r
	}
	if b := has(SignalBurst); len(b) > 0 {
		sg := ReferralSignal{Kind: SignalBurst, Count: len(b), Threshold: int(p.BurstThreshold), All: len(b) == len(items)}
		for _, h := range b {
			if h.confirmed == nil {
				continue
			}
			if sg.From == nil || h.confirmed.Before(*sg.From) {
				t := *h.confirmed
				sg.From = &t
			}
			if sg.To == nil || h.confirmed.After(*sg.To) {
				t := *h.confirmed
				sg.To = &t
			}
		}
		if sg.From != nil && sg.To != nil {
			sg.Minutes = int(sg.To.Sub(*sg.From).Round(time.Minute) / time.Minute)
		}
		out = append(out, sg)
	}
	if sh := has(SignalSharedIP); len(sh) > 0 {
		counts := map[string]int{}
		var best string
		var bestRow heldRow
		for _, h := range sh {
			if h.ip == nil {
				continue
			}
			counts[*h.ip]++
			if counts[*h.ip] > counts[best] {
				best, bestRow = *h.ip, h
			}
		}
		sg := ReferralSignal{Kind: SignalSharedIP, Count: counts[best], Threshold: int(p.SharedIpThreshold), IP: best, All: counts[best] == len(items),
			Network: ReferralNetwork{CIDR: bestRow.cidr, Name: bestRow.netName, ASN: bestRow.asn, Kind: bestRow.kind}}
		out = append(out, sg)
	}
	if nb := has(SignalNumbered); len(nb) > 0 {
		emails := make([]string, 0, len(nb))
		for _, h := range nb {
			emails = append(emails, h.email)
		}
		sort.Strings(emails)
		sg := ReferralSignal{Kind: SignalNumbered, Count: len(nb), All: len(nb) == len(items)}
		if len(emails) > 0 {
			sg.First, sg.Last = emails[0], emails[len(emails)-1]
		}
		out = append(out, sg)
	}
	return out
}

// engagement is what a referred subscriber opened or clicked since they
// confirmed (human opens only).
func (s *Service) engagement(ctx context.Context, ws *store.Workspace, p store.ReferralProgram, contactID uuid.UUID, since time.Time) (ReferralEngagement, error) {
	out := ReferralEngagement{Opened: []string{}, Short: []string{}}
	rows, err := s.Pool.Query(ctx, `SELECT c.name, m.clicked_at IS NOT NULL FROM messages m JOIN campaigns c ON c.id = m.campaign_id
		WHERE m.contact_id = $1 AND m.queued_at >= $2 AND m.opened_at IS NOT NULL ORDER BY m.queued_at`, contactID, since)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var name string
		var clicked bool
		if err := rows.Scan(&name, &clicked); err != nil {
			rows.Close()
			return out, err
		}
		out.Opened = append(out.Opened, name)
		out.Short = append(out.Short, shortCampaign(name))
		if clicked {
			out.Clicks++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Opened) == 0 {
		var n int
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE contact_id = $1 AND queued_at >= $2`, contactID, since).Scan(&n); err != nil {
			return out, err
		}
		if n == 0 {
			var last string
			err := s.Pool.QueryRow(ctx, `SELECT name FROM campaigns WHERE workspace_id = $1 AND deleted_at IS NULL AND status IN ('sent', 'sending')
				AND started_at < $2 AND audience->'lists' ? $3::text ORDER BY started_at DESC LIMIT 1`, ws.ID, since, p.ListID.String()).Scan(&last)
			if err == nil {
				out.JoinedAfter = shortCampaign(last)
			}
		}
	}
	return out, nil
}

// shortCampaign is "#142" for "Weekly Digest #142", the name otherwise.
func shortCampaign(name string) string {
	if i := strings.LastIndex(name, "#"); i >= 0 {
		return name[i:]
	}
	return name
}

// ReferralDecisionInput decides on held referrals: one referrer's, all of
// them or some.
type ReferralDecisionInput struct {
	IDs      []string `json:"ids" minItems:"1" maxItems:"500"`
	Decision string   `json:"decision" enum:"count,dont,dont_pause"`
	Note     string   `json:"note,omitempty" maxLength:"500"`
}

// ReferralDecisionPreview is what a decision would do, before it's made:
// "Counting these 3 brings Mateo Silva to 5 when their hold ends on 21 Oct.
// The Field Guide goes out then."
type ReferralDecisionPreview struct {
	Decision string     `json:"decision"`
	Count    int        `json:"count"`
	Referrer string     `json:"referrer"`
	Counted  int64      `json:"counted"`
	After    int64      `json:"after"`
	HoldEnds *time.Time `json:"holdEnds,omitempty"`
	// Rewards are the milestones the decision reaches (their names).
	Rewards []string `json:"rewards"`
	// Now: the hold has ended, so they count at once.
	Now bool `json:"now"`
}

// ReferralDecision is a decision made, with how long Undo works.
type ReferralDecision struct {
	ID        string                  `json:"id"`
	Preview   ReferralDecisionPreview `json:"preview"`
	UndoUntil time.Time               `json:"undoUntil"`
}

func (s *Service) heldOf(ctx context.Context, programID uuid.UUID, raw []string) ([]store.Referral, error) {
	var ids []uuid.UUID
	for _, r := range raw {
		id, err := uuid.Parse(r)
		if err != nil {
			return nil, api.Invalid("body.ids", s.text(ctx, msgs.M("review.none"), nil))
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, api.Invalid("body.ids", s.text(ctx, msgs.M("review.none"), nil))
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, workspace_id, program_id, link_id, referrer_id, contact_id, seq, status, reason, left_kind, signals, via, ip,
			ip_network, network_name, network_asn, network_kind, signed_up_at, reminded_at, confirmed_at, hold_until, counted_at, not_counted_at, held_at,
			decided_at, decided_by, decision, decision_note, decision_id, undo_until, before_status, ip_reduced_at, created_at, updated_at
		FROM referrals WHERE program_id = $1 AND id = ANY ($2) ORDER BY signed_up_at`, programID, ids)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[store.Referral])
	if err != nil {
		return nil, err
	}
	if len(out) != len(ids) {
		return nil, api.Err(http.StatusConflict, "not_held", s.text(ctx, msgs.M("review.notHeld"), nil))
	}
	for _, r := range out {
		if r.Status != StatusReview {
			return nil, api.Err(http.StatusConflict, "not_held", s.text(ctx, msgs.M("review.notHeld"), nil))
		}
		if r.LinkID == nil || out[0].LinkID == nil || *r.LinkID != *out[0].LinkID {
			return nil, api.Invalid("body.ids", s.text(ctx, msgs.M("review.oneReferrer"), nil))
		}
	}
	return out, nil
}

// PreviewDecision says what a decision would do.
func (s *Service) PreviewDecision(ctx context.Context, ws *store.Workspace, programID uuid.UUID, in ReferralDecisionInput) (ReferralDecisionPreview, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralDecisionPreview{}, err
	}
	refs, err := s.heldOf(ctx, programID, in.IDs)
	if err != nil {
		return ReferralDecisionPreview{}, err
	}
	return s.preview(ctx, cur.ReferralProgram, refs, in.Decision)
}

func (s *Service) preview(ctx context.Context, p store.ReferralProgram, refs []store.Referral, decision string) (ReferralDecisionPreview, error) {
	link := *refs[0].LinkID
	person, err := s.person(ctx, p, link, true)
	if err != nil {
		return ReferralDecisionPreview{}, err
	}
	out := ReferralDecisionPreview{Decision: decision, Count: len(refs), Referrer: person.Name, Counted: person.Counted, After: person.Counted, Rewards: []string{}}
	if decision != DecisionCount {
		return out, nil
	}
	// Referrals in hold count too when they're due before these.
	now := s.now()
	var latest time.Time
	for _, r := range refs {
		end := now
		if r.ConfirmedAt != nil {
			end = r.ConfirmedAt.Add(time.Duration(p.HoldDays) * 24 * time.Hour)
		}
		if end.After(latest) {
			latest = end
		}
	}
	var inHold int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE link_id = $1 AND status = 'hold' AND hold_until <= $2`, link, latest).Scan(&inHold); err != nil {
		return out, err
	}
	out.After = person.Counted + inHold + int64(len(refs))
	if latest.After(now) {
		t := latest
		out.HoldEnds = &t
	} else {
		out.Now = true
	}
	ms, err := s.Q.ReferralMilestones(ctx, p.ID)
	if err != nil {
		return out, err
	}
	for _, m := range ms {
		if int64(m.Count) > person.Counted && int64(m.Count) <= out.After {
			out.Rewards = append(out.Rewards, m.RewardName)
		}
	}
	return out, nil
}

// Decide decides on held referrals: count them (what is left of the hold
// still applies), don't count them, or don't count them and pause the
// referrer's link. Undo works for 10 seconds, before any reward goes out.
func (s *Service) Decide(ctx context.Context, ws *store.Workspace, programID uuid.UUID, in ReferralDecisionInput) (ReferralDecision, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralDecision{}, err
	}
	p := cur.ReferralProgram
	refs, err := s.heldOf(ctx, programID, in.IDs)
	if err != nil {
		return ReferralDecision{}, err
	}
	pv, err := s.preview(ctx, p, refs, in.Decision)
	if err != nil {
		return ReferralDecision{}, err
	}
	now := s.now()
	undo := now.Add(UndoFor)
	id := uuid.Must(uuid.NewV7())
	by := actorUser(ctx)
	note := strings.TrimSpace(in.Note)
	link := *refs[0].LinkID
	person, _ := s.person(ctx, p, link, true)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		ids := make([]string, 0, len(refs))
		for _, r := range refs {
			ids = append(ids, r.ID.String())
			switch in.Decision {
			case DecisionCount:
				// Counted once the hold ends; never before Undo has passed.
				end := undo
				if r.ConfirmedAt != nil {
					if h := r.ConfirmedAt.Add(time.Duration(p.HoldDays) * 24 * time.Hour); h.After(end) {
						end = h
					}
				}
				if _, err := tx.Exec(ctx, `UPDATE referrals SET status = 'hold', hold_until = $2, decided_at = $3, decided_by = $4, decision = $5,
					decision_note = $6, decision_id = $7, undo_until = $8, before_status = status, updated_at = $3 WHERE id = $1`,
					r.ID, end, now, by, in.Decision, note, id, undo); err != nil {
					return err
				}
			default:
				if _, err := tx.Exec(ctx, `UPDATE referrals SET status = 'not_counted', reason = 'reviewer', not_counted_at = $2, decided_at = $2, decided_by = $3,
					decision = $4, decision_note = $5, decision_id = $6, undo_until = $7, before_status = status, updated_at = $2 WHERE id = $1`,
					r.ID, now, by, in.Decision, note, id, undo); err != nil {
					return err
				}
			}
			rid := r.ID
			if err := writeLog(ctx, q, ws.ID, logEntry{program: programID, referral: &rid, link: r.LinkID, contact: r.ContactID, kind: EventDecided, at: now,
				detail: map[string]any{"decision": in.Decision, "note": note, "decisionId": id.String()}}); err != nil {
				return err
			}
		}
		if in.Decision == DecisionDontPause {
			if err := q.ReferralLinkSetStatus(ctx, store.ReferralLinkSetStatusParams{ID: link, Status: "paused", Now: now, By: by, Note: note}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.link_paused", TargetType: "referral_link", TargetID: link.String(),
				TargetName: person.Name, Reason: note, Metadata: map[string]any{"related": person.ContactID, "via": "review"}}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.review_decided", TargetType: "referral_link", TargetID: link.String(),
			TargetName: person.Name, Reason: note, After: map[string]any{"decision": in.Decision, "referrals": ids},
			Metadata: map[string]any{"summary": decisionSummary(in.Decision, len(ids), person.Name), "related": person.ContactID}})
	})
	if err != nil {
		return ReferralDecision{}, err
	}
	return ReferralDecision{ID: id.String(), Preview: pv, UndoUntil: undo}, nil
}

func decisionSummary(decision string, n int, name string) string {
	switch decision {
	case DecisionCount:
		return "Counted " + formatInt(int64(n)) + " held referrals from " + name
	case DecisionDontPause:
		return "Didn't count " + formatInt(int64(n)) + " held referrals from " + name + " and paused the link"
	}
	return "Didn't count " + formatInt(int64(n)) + " held referrals from " + name
}

// UndoDecision takes a decision back within 10 seconds, before any reward
// went out: the referrals wait for review again and a link the decision
// paused is active again.
func (s *Service) UndoDecision(ctx context.Context, ws *store.Workspace, programID, decisionID uuid.UUID) error {
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := tx.Query(ctx, `SELECT id, link_id, decision, undo_until, status, contact_id FROM referrals
			WHERE program_id = $1 AND decision_id = $2 FOR UPDATE`, programID, decisionID)
		if err != nil {
			return err
		}
		type rr struct {
			id       uuid.UUID
			link     *uuid.UUID
			decision string
			undo     *time.Time
			status   string
			contact  *uuid.UUID
		}
		var list []rr
		for rows.Next() {
			var r rr
			if err := rows.Scan(&r.id, &r.link, &r.decision, &r.undo, &r.status, &r.contact); err != nil {
				rows.Close()
				return err
			}
			list = append(list, r)
		}
		rows.Close()
		if len(list) == 0 {
			return api.NotFound("Decision")
		}
		for _, r := range list {
			if r.undo == nil || now.After(*r.undo) || r.status == StatusCounted {
				return api.Err(http.StatusConflict, "undo_late", s.text(ctx, msgs.M("review.undoLate"), nil))
			}
		}
		for _, r := range list {
			if _, err := tx.Exec(ctx, `UPDATE referrals SET status = 'review', reason = '', hold_until = NULL, not_counted_at = NULL, decided_at = NULL,
				decided_by = NULL, decision = '', decision_note = '', decision_id = NULL, undo_until = NULL, updated_at = $2 WHERE id = $1`, r.id, now); err != nil {
				return err
			}
		}
		link := list[0].link
		if list[0].decision == DecisionDontPause && link != nil {
			if err := q.ReferralLinkSetStatus(ctx, store.ReferralLinkSetStatusParams{ID: *link, Status: "active", Now: now}); err != nil {
				return err
			}
		}
		target := ""
		if link != nil {
			target = link.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.review_undone", TargetType: "referral_link", TargetID: target,
			Metadata: map[string]any{"decision": decisionID.String(), "referrals": len(list)}})
	})
}
