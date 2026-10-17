package scoring

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// SegmentMembership reports whether a contact is in a segment (the
// segments domain answers it; scoring can't import it).
type SegmentMembership func(ctx context.Context, ws *store.Workspace, segment, contact uuid.UUID) (bool, error)

// ContactScoreCause is what made the last change: an event, or decay.
type ContactScoreCause struct {
	Kind string `json:"kind" enum:"open,click,form,api_event,soft_bounce,unsubscribe,decay"`
	// EmailName is the campaign or automation of an open, click or bounce.
	EmailName string `json:"emailName,omitempty"`
	// LinkLabel and URL say which link a click was on.
	LinkLabel string `json:"linkLabel,omitempty"`
	URL       string `json:"url,omitempty"`
	FormName  string `json:"formName,omitempty"`
	EventName string `json:"eventName,omitempty"`
	ListName  string `json:"listName,omitempty"`
}

// ContactScoreChange is the last change of a score.
type ContactScoreChange struct {
	At         time.Time         `json:"at"`
	Delta      int               `json:"delta"`
	Before     int               `json:"before"`
	After      int               `json:"after"`
	BeforeBand string            `json:"beforeBand" enum:"cold,warm,hot"`
	AfterBand  string            `json:"afterBand" enum:"cold,warm,hot"`
	Cause      ContactScoreCause `json:"cause"`
}

// ContactScoreRule is one row of the breakdown.
type ContactScoreRule struct {
	Number int         `json:"number" doc:"The rule's number on Scoring"`
	Rule   ScoringRule `json:"rule"`
	// Points is what the rule changed the score by.
	Points int `json:"points"`
	// Count is how many events matched (opens, clicks…); 1 for a true condition.
	Count int `json:"count"`
	// CapReached is the start of the latest period whose cap stopped points.
	CapReached *time.Time `json:"capReached,omitempty"`
	// AtMax is true when the 100 limit stopped some points.
	AtMax bool `json:"atMax"`
	// Latest is the last matching event.
	Latest *time.Time `json:"latest,omitempty"`
	// ZeroEmail and ZeroAt explain a zero: the email of a soft bounce that
	// was delivered after a retry, and when.
	ZeroEmail string     `json:"zeroEmail,omitempty"`
	ZeroAt    *time.Time `json:"zeroAt,omitempty"`
	FormName  string     `json:"formName,omitempty"`
	ListName  string     `json:"listName,omitempty"`
	// Since is when the contact joined (opens counted since then).
}

// ContactScoreDecay is the decay row.
type ContactScoreDecay struct {
	Enabled bool `json:"enabled"`
	// Lost is what decay took (0 or negative).
	Lost int `json:"lost"`
	// QuietWeeks is how many weeks the contact has been without a human
	// open or click.
	QuietWeeks int `json:"quietWeeks"`
	// LastActivity is the last human open or click, and which.
	LastActivity     *time.Time `json:"lastActivity,omitempty"`
	LastActivityKind string     `json:"lastActivityKind,omitempty" doc:"open or click"`
	// LastOpen is the last human open, which the decay row names ("None ·
	// last human open Tue 7 Oct").
	LastOpen *time.Time `json:"lastOpen,omitempty"`
}

// ContactScorePoint is the score at the end of a day (nil before the
// contact joined).
type ContactScorePoint struct {
	Day   string `json:"day" doc:"YYYY-MM-DD in the workspace timezone"`
	Score *int   `json:"score,omitempty"`
}

// ContactScoreUse is a place the score counts for this contact.
type ContactScoreUse struct {
	Kind string `json:"kind" enum:"segment,automation,block"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// In is true when the contact is in the segment, or entered the automation.
	In bool `json:"in"`
	// Threshold and Direction are an automation's trigger; CrossedAt is when
	// the contact last crossed it, LiveSince when the automation went live.
	Threshold int        `json:"threshold,omitempty"`
	Direction string     `json:"direction,omitempty"`
	CrossedAt *time.Time `json:"crossedAt,omitempty"`
	LiveSince *time.Time `json:"liveSince,omitempty"`
	Href      string     `json:"href,omitempty"`
}

// ContactScore is the profile's score card.
type ContactScore struct {
	// Configured is false while the workspace has no rules: no score ("—").
	Configured bool         `json:"configured"`
	Score      int          `json:"score"`
	Band       string       `json:"band" enum:"cold,warm,hot"`
	BandSince  *time.Time   `json:"bandSince,omitempty"`
	Bands      ScoringBands `json:"bands"`
	// Recalculating is true while a recalculation runs: the score shown is
	// the previous one.
	Recalculating bool                `json:"recalculating"`
	LastChange    *ContactScoreChange `json:"lastChange,omitempty"`
	Breakdown     []ContactScoreRule  `json:"breakdown"`
	Decay         ContactScoreDecay   `json:"decay"`
	JoinedAt      time.Time           `json:"joinedAt"`
	Line          []ContactScorePoint `json:"line"`
	Uses          []ContactScoreUse   `json:"uses"`
}

// ContactScore returns a contact's score with its breakdown, replayed
// under the applied rules.
func (s *Service) ContactScore(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, member SegmentMembership) (ContactScore, error) {
	q := s.Q
	c, err := q.ScoringContactBasics(ctx, contactID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (c.WorkspaceID != ws.ID || c.ErasedAt != nil)) {
		return ContactScore{}, api.NotFound("Contact")
	}
	if err != nil {
		return ContactScore{}, err
	}
	st, err := s.load(ctx, q, ws.ID, ws.Timezone)
	if err != nil {
		return ContactScore{}, err
	}
	out := ContactScore{Configured: st.row.AppliedVersion > 0, Bands: st.applied.Bands, JoinedAt: c.CreatedAt,
		Breakdown: []ContactScoreRule{}, Line: []ContactScorePoint{}, Uses: []ContactScoreUse{}, Band: BandCold}
	out.Recalculating = st.row.AppliedVersion != st.row.Version
	if !out.Configured {
		out.Bands = st.saved.Bands
		return out, nil
	}
	now := s.now()
	cfg := st.applied
	eng := NewEngine(cfg, st.loc)
	rows, err := q.ScoringEventsByContact(ctx, store.ScoringEventsByContactParams{ContactID: contactID, Since: now.AddDate(0, 0, -KeepDays)})
	if err != nil {
		return out, err
	}
	events := make([]Event, 0, len(rows))
	for _, r := range rows {
		ev := Event{At: r.OccurredAt, Kind: r.Kind, Ref: r.Ref, Lists: r.Lists, Counted: r.Counted}
		if r.MessageID != nil {
			ev.Message = *r.MessageID
		}
		events = append(events, ev)
	}
	// Field conditions, one by one for the breakdown.
	fr := fieldRules(cfg)
	truth := make([]bool, len(fr))
	fieldsPart := 0
	if len(fr) > 0 {
		a := &sqlArgs{}
		a.add(contactID)
		exprs := make([]string, len(fr))
		for i, r := range fr {
			exprs[i] = conditionSQL(r, st.types, a)
		}
		dest := make([]any, len(fr))
		for i := range dest {
			dest[i] = &truth[i]
		}
		if err := s.Pool.QueryRow(ctx, "SELECT "+strings.Join(exprs, ", ")+" FROM contacts c WHERE c.id = $1", a.args...).Scan(dest...); err != nil {
			return out, err
		}
		for i, r := range fr {
			if truth[i] {
				fieldsPart += r.Points
			}
		}
	}
	// The breakdown replays the events as of the stored score's last
	// computation, so it adds up to the score segments and triggers read
	// (decay steps and field conditions apply at the nightly run).
	asOf := now
	if stored, err := q.ScoringContactScore(ctx, contactID); err == nil && stored.Version == st.row.AppliedVersion && stored.ComputedAt.Before(now) {
		asOf = stored.ComputedAt
	}
	res := eng.Replay(events, asOf, fieldsPart)
	out.Score = res.Score
	out.Band = cfg.Bands.Band(res.Score)
	out.BandSince = BandSince(res, cfg.Bands)
	numbers := ruleIndex(cfg)
	names, err := nameRefs(ctx, q, ws.ID)
	if err != nil {
		return out, err
	}
	// Labels of the emails involved.
	var msgIDs []uuid.UUID
	for _, ev := range res.Events {
		if ev.Message != uuid.Nil && !slices.Contains(msgIDs, ev.Message) {
			msgIDs = append(msgIDs, ev.Message)
		}
	}
	labels := map[uuid.UUID]store.ScoringMessageLabelsRow{}
	var campaigns []uuid.UUID
	if len(msgIDs) > 0 {
		ls, err := q.ScoringMessageLabels(ctx, msgIDs)
		if err != nil {
			return out, err
		}
		for _, l := range ls {
			labels[l.ID] = l
			if l.CampaignID != nil && !slices.Contains(campaigns, *l.CampaignID) {
				campaigns = append(campaigns, *l.CampaignID)
			}
		}
	}
	links := map[string]string{} // campaign|url → label
	if len(campaigns) > 0 {
		ls, err := q.ScoringLinkLabels(ctx, campaigns)
		if err != nil {
			return out, err
		}
		for _, l := range ls {
			links[l.CampaignID.String()+"|"+l.Url] = l.Label
		}
	}
	emailName := func(m uuid.UUID) string { return labels[m].EmailName }
	for _, t := range res.Rules {
		r := ruleByID(cfg, t.RuleID)
		if t.Count == 0 && t.Points == 0 {
			continue // rules that matched nothing get no row
		}
		row := ContactScoreRule{Number: numbers[r.ID], Rule: r, Points: t.Points, Count: t.Count, CapReached: t.CapReached, AtMax: t.AtMax, Latest: t.Latest,
			FormName: names.forms[r.FormID], ListName: names.lists[r.ListID]}
		if t.Zero > 0 && t.Points == 0 && t.LastZero != nil {
			row.ZeroEmail = emailName(t.LastZero.Message)
			at := t.LastZero.At
			row.ZeroAt = &at
		}
		out.Breakdown = append(out.Breakdown, row)
	}
	for i, r := range fr {
		if truth[i] {
			out.Breakdown = append(out.Breakdown, ContactScoreRule{Number: numbers[r.ID], Rule: r, Points: r.Points, Count: 1})
		}
	}
	slices.SortStableFunc(out.Breakdown, func(a, b ContactScoreRule) int { return a.Number - b.Number })
	out.Decay = ContactScoreDecay{Enabled: cfg.Decay.Enabled, Lost: res.DecayLost, LastActivity: res.ActiveAt}
	if res.ActiveAt != nil {
		for i := len(res.Events) - 1; i >= 0; i-- {
			if res.Events[i].At.Equal(*res.ActiveAt) && res.Events[i].activity() {
				out.Decay.LastActivityKind = res.Events[i].Kind
				break
			}
		}
		out.Decay.QuietWeeks = int(asOf.Sub(*res.ActiveAt).Hours() / (24 * 7))
	}
	for i := len(res.Events) - 1; i >= 0; i-- {
		if res.Events[i].Kind == KindOpen {
			at := res.Events[i].At
			out.Decay.LastOpen = &at
			break
		}
	}
	// The last change of the score itself (events at 100 change nothing).
	last := -1
	for i := len(res.Steps) - 1; i >= 0; i-- {
		if Total(res.Steps[i].Before, fieldsPart) != Total(res.Steps[i].After, fieldsPart) {
			last = i
			break
		}
	}
	if last >= 0 {
		step := res.Steps[last]
		b, a := Total(step.Before, fieldsPart), Total(step.After, fieldsPart)
		ch := &ContactScoreChange{At: step.At, Delta: a - b, Before: b, After: a, BeforeBand: cfg.Bands.Band(b), AfterBand: cfg.Bands.Band(a), Cause: ContactScoreCause{Kind: "decay"}}
		if step.Event >= 0 {
			ev := res.Events[step.Event]
			ch.Cause = ContactScoreCause{Kind: ev.Kind}
			switch ev.Kind {
			case KindOpen, KindClick, KindSoftBounce:
				ch.Cause.EmailName = emailName(ev.Message)
				if ev.Kind == KindClick {
					ch.Cause.URL = ev.Ref
					if l := labels[ev.Message]; l.CampaignID != nil {
						ch.Cause.LinkLabel = links[l.CampaignID.String()+"|"+ev.Ref]
					}
				}
			case KindForm:
				ch.Cause.FormName = names.forms[ev.Ref]
			case KindAPIEvent:
				ch.Cause.EventName = ev.Ref
			case KindUnsubscribe:
				ch.Cause.ListName = names.lists[ev.Ref]
			}
		}
		out.LastChange = ch
	}
	out.Line = s.line(eng, events, asOf, fieldsPart, c.CreatedAt, st.loc)
	out.Uses, err = s.contactUses(ctx, q, ws, contactID, res, fieldsPart, member)
	return out, err
}

func ruleByID(cfg ScoringConfig, id string) ScoringRule {
	for _, r := range cfg.Rules {
		if r.ID == id {
			return r
		}
	}
	return ScoringRule{ID: id, Values: []string{}}
}

// line is the score at the end of each of the last 90 days.
func (s *Service) line(eng *Engine, events []Event, now time.Time, fieldsPart int, joined time.Time, loc *time.Location) []ContactScorePoint {
	out := make([]ContactScorePoint, 0, 91)
	l := now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	for d := 90; d >= 0; d-- {
		day := today.AddDate(0, 0, -d)
		end := day.AddDate(0, 0, 1).Add(-time.Second)
		if d == 0 {
			end = now
		}
		p := ContactScorePoint{Day: day.Format("2006-01-02")}
		if !end.Before(joined) {
			score := eng.Replay(events, end, fieldsPart).Score
			p.Score = &score
		}
		out = append(out, p)
	}
	return out
}

// contactUses says where the score counts for this contact: the segments
// that use it and whether the contact is in them, and the automations that
// start on it and whether the contact entered.
func (s *Service) contactUses(ctx context.Context, q *store.Queries, ws *store.Workspace, contactID uuid.UUID, res Result, fieldsPart int, member SegmentMembership) ([]ContactScoreUse, error) {
	out := []ContactScoreUse{}
	segs, err := q.ScoringSegmentsUsingScore(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sg := range segs {
		u := ContactScoreUse{Kind: "segment", ID: sg.ID.String(), Name: sg.Name, Href: "/w/" + ws.Slug + "/audience/segments/" + sg.ID.String()}
		if member != nil {
			in, err := member(ctx, ws, sg.ID, contactID)
			if err != nil {
				s.log().WarnContext(ctx, "segment membership", "segment", sg.ID, "err", err)
			}
			u.In = in
		}
		out = append(out, u)
	}
	flows, err := q.ScoringLiveTriggers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range flows {
		t, ok := triggerOf(f.Trigger)
		if !ok {
			continue
		}
		u := ContactScoreUse{Kind: "automation", ID: f.ID.String(), Name: f.Name, Threshold: t.Threshold, Direction: t.Direction, Href: "/w/" + ws.Slug + "/automations/" + f.ID.String()}
		if f.LiveSince.Year() > 1 {
			since := f.LiveSince
			u.LiveSince = &since
		}
		var entered bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM automation_runs WHERE automation_id = $1 AND contact_id = $2)`, f.ID, contactID).Scan(&entered); err != nil {
			return out, err
		}
		u.In = entered
		// The last time the score crossed the threshold.
		prev := Total(0, fieldsPart)
		for _, st := range res.Steps {
			next := Total(st.After, fieldsPart)
			if t.Crosses(prev, next) {
				at := st.At
				u.CrossedAt = &at
			}
			prev = next
		}
		out = append(out, u)
	}
	return out, nil
}
