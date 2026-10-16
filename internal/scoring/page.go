package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// ScoringBucket is one bar of the spread: contacts with a score from From
// to To (the last bucket is 90–100).
type ScoringBucket struct {
	From     int   `json:"from"`
	To       int   `json:"to"`
	Contacts int64 `json:"contacts"`
}

// ScoringTierRow is one engagement tier split by band.
type ScoringTierRow struct {
	Tier     string `json:"tier" enum:"active,cooling,at_risk,sunset"`
	Contacts int64  `json:"contacts"`
	Hot      int64  `json:"hot"`
	Warm     int64  `json:"warm"`
	Cold     int64  `json:"cold"`
}

// ScoringUse is something that reads the score: an automation, a segment
// or (phase 3) a conditional block.
type ScoringUse struct {
	Kind string `json:"kind" enum:"automation,segment,block"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status is an automation's (live, paused, draft).
	Status string `json:"status,omitempty"`
	By     string `json:"by,omitempty"`
	// Since is when an automation went live.
	Since *time.Time `json:"since,omitempty"`
	// Entered is how many contacts an automation started for.
	Entered int64 `json:"entered"`
	// Direction, Threshold and OnceInDays are an automation's trigger.
	Direction  string `json:"direction,omitempty"`
	Threshold  int    `json:"threshold,omitempty"`
	OnceInDays int    `json:"onceInDays,omitempty"`
	// Contacts is a segment's last count, or a block's recipients.
	Contacts *int64 `json:"contacts,omitempty"`
	// Conditions are a segment's conditions, for its one-line summary.
	Conditions json.RawMessage `json:"conditions,omitempty"`
	// Where says where a block is ("Weekly Digest #143 (draft)"); Of is
	// the total its count is out of.
	Where string `json:"where,omitempty"`
	Of    *int64 `json:"of,omitempty"`
	Href  string `json:"href,omitempty"`
}

// UseFinder lists the things of one kind that read the score. Conditional
// content (phase 3) registers one for its blocks.
type UseFinder func(ctx context.Context, wsID uuid.UUID) ([]ScoringUse, error)

var (
	usesMu      sync.Mutex
	usesFinders []UseFinder
)

// RegisterUses adds a finder to Scoring's "Used by" card.
func RegisterUses(f UseFinder) {
	usesMu.Lock()
	defer usesMu.Unlock()
	usesFinders = append(usesFinders, f)
}

// ScoringRecalcInfo is the latest recalculation.
type ScoringRecalcInfo struct {
	ID               string                `json:"id"`
	Status           string                `json:"status" enum:"running,finished,failed,superseded"`
	Reason           string                `json:"reason" enum:"setup,rules,retry"`
	Total            int64                 `json:"total"`
	Done             int64                 `json:"done"`
	StartAutomations bool                  `json:"startAutomations"`
	StartedAt        time.Time             `json:"startedAt"`
	StartedBy        string                `json:"startedBy"`
	FinishedAt       *time.Time            `json:"finishedAt,omitempty"`
	FailedAt         *time.Time            `json:"failedAt,omitempty"`
	Label            string                `json:"label"`
	Changes          ScoringChanges        `json:"changes"`
	Summary          *ScoringRecalcSummary `json:"summary,omitempty"`
	// RemainingSeconds estimates the time left from the pace so far.
	RemainingSeconds *int `json:"remainingSeconds,omitempty"`
}

// ScoringListOption, ScoringFormOption, ScoringFieldOption and
// ScoringEventOption are what the rule editor offers.
type ScoringListOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ScoringFormOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind" enum:"embedded,popup,hosted"`
}

type ScoringFieldOption struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type"`
	Options []string `json:"options"`
}

type ScoringEventOption struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// ScoringOptions are the rule editor's choices.
type ScoringOptions struct {
	Lists     []ScoringListOption  `json:"lists"`
	Forms     []ScoringFormOption  `json:"forms"`
	Fields    []ScoringFieldOption `json:"fields"`
	APIEvents []ScoringEventOption `json:"apiEvents"`
}

// ScoringPage is Audience › Scoring.
type ScoringPage struct {
	// Configured is false until the workspace saves its first rules.
	Configured     bool          `json:"configured"`
	Version        int           `json:"version"`
	AppliedVersion int           `json:"appliedVersion"`
	Config         ScoringConfig `json:"config"`
	SetUpBy        string        `json:"setUpBy"`
	SetUpAt        *time.Time    `json:"setUpAt,omitempty"`
	// Contacts is how many contacts the spread counts.
	Contacts  int64              `json:"contacts"`
	Spread    []ScoringBucket    `json:"spread"`
	Bands     ScoringBandCounts  `json:"bands"`
	ByTier    []ScoringTierRow   `json:"byTier"`
	UsedBy    []ScoringUse       `json:"usedBy"`
	Recalc    *ScoringRecalcInfo `json:"recalculation,omitempty"`
	Options   ScoringOptions     `json:"options"`
	Suggested []ScoringRule      `json:"suggested"`
}

// spreadOf turns contacts per exact score into the ten buckets and the
// bands under b.
func spreadOf(counts map[int]int64, b ScoringBands) ([]ScoringBucket, ScoringBandCounts, int64) {
	buckets := make([]ScoringBucket, 10)
	for i := range buckets {
		buckets[i] = ScoringBucket{From: i * 10, To: i*10 + 9}
	}
	buckets[9].To = MaxScore
	var bands ScoringBandCounts
	var total int64
	for score, n := range counts {
		total += n
		buckets[min(score/10, 9)].Contacts += n
		switch b.Band(score) {
		case BandHot:
			bands.Hot += n
		case BandWarm:
			bands.Warm += n
		default:
			bands.Cold += n
		}
	}
	return buckets, bands, total
}

func scoreCounts(ctx context.Context, q *store.Queries, wsID uuid.UUID) (map[int]int64, error) {
	rows, err := q.ScoringScoreCounts(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, r := range rows {
		out[int(r.Score)] = r.Contacts
	}
	return out, nil
}

// Page returns everything Audience › Scoring shows.
func (s *Service) Page(ctx context.Context, ws *store.Workspace) (ScoringPage, error) {
	q := s.Q
	st, err := s.load(ctx, q, ws.ID, ws.Timezone)
	if err != nil {
		return ScoringPage{}, err
	}
	out := ScoringPage{
		Configured: st.configured(), Version: int(st.row.Version), AppliedVersion: int(st.row.AppliedVersion),
		Config: st.saved, SetUpAt: st.row.SetUpAt,
		Spread: []ScoringBucket{}, ByTier: []ScoringTierRow{}, UsedBy: []ScoringUse{}, Suggested: []ScoringRule{},
	}
	if st.row.SetUpBy != nil {
		if u, err := q.GetUser(ctx, *st.row.SetUpBy); err == nil {
			out.SetUpBy = u.Name
		}
	}
	if out.Options, err = s.options(ctx, q, ws.ID); err != nil {
		return out, err
	}
	if !out.Configured {
		out.Suggested = s.suggested(ctx, q, ws.ID, out.Options)
		if out.Contacts, err = q.ScoringCountContacts(ctx, ws.ID); err != nil {
			return out, err
		}
		return out, nil
	}
	counts, err := scoreCounts(ctx, q, ws.ID)
	if err != nil {
		return out, err
	}
	// Live scores fall in the bands of the applied configuration.
	out.Spread, out.Bands, out.Contacts = spreadOf(counts, st.applied.Bands)
	tiers, err := q.ScoringBandsByTier(ctx, store.ScoringBandsByTierParams{WorkspaceID: ws.ID, WarmFrom: int32(st.applied.Bands.WarmFrom), HotFrom: int32(st.applied.Bands.HotFrom)})
	if err != nil {
		return out, err
	}
	byTier := map[string]ScoringTierRow{}
	for _, t := range tiers {
		byTier[t.Tier] = ScoringTierRow{Tier: t.Tier, Contacts: t.Contacts, Hot: t.Hot, Warm: t.Warm, Cold: t.Cold}
	}
	for _, tier := range []string{"active", "cooling", "at_risk", "sunset"} {
		row, ok := byTier[tier]
		if !ok {
			row = ScoringTierRow{Tier: tier}
		}
		out.ByTier = append(out.ByTier, row)
	}
	if out.UsedBy, err = s.uses(ctx, q, ws); err != nil {
		return out, err
	}
	if out.Recalc, err = s.recalcInfo(ctx, q, ws.ID); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) options(ctx context.Context, q *store.Queries, wsID uuid.UUID) (ScoringOptions, error) {
	out := ScoringOptions{Lists: []ScoringListOption{}, Forms: []ScoringFormOption{}, Fields: []ScoringFieldOption{}, APIEvents: []ScoringEventOption{}}
	lists, err := q.ScoringListsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, ScoringListOption{ID: l.ID.String(), Name: l.Name})
	}
	forms, err := q.ScoringFormsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, f := range forms {
		out.Forms = append(out.Forms, ScoringFormOption{ID: f.ID.String(), Name: f.Name, Kind: f.Kind})
	}
	fields, err := q.ScoringFieldsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		var opts []string
		_ = json.Unmarshal(f.Options, &opts)
		if opts == nil {
			opts = []string{}
		}
		out.Fields = append(out.Fields, ScoringFieldOption{Key: f.Key, Label: f.Label, Type: f.Type, Options: opts})
	}
	since := s.now().AddDate(0, 0, -30)
	names := map[string]int64{}
	if rows, err := q.ScoringAPIEventNames(ctx, store.ScoringAPIEventNamesParams{WorkspaceID: wsID, Since: since}); err == nil {
		for _, r := range rows {
			names[r.Name] = r.N
		}
	}
	if rows, err := q.ScoringAPIEventNamesFromScores(ctx, store.ScoringAPIEventNamesFromScoresParams{WorkspaceID: wsID, Since: since}); err == nil {
		for _, r := range rows {
			names[r.Name] = max(names[r.Name], r.N)
		}
	}
	for n, c := range names {
		out.APIEvents = append(out.APIEvents, ScoringEventOption{Name: n, Count: c})
	}
	slices.SortFunc(out.APIEvents, func(a, b ScoringEventOption) int {
		if a.Count != b.Count {
			return int(b.Count - a.Count)
		}
		return strings.Compare(a.Name, b.Name)
	})
	if len(out.APIEvents) > 10 {
		out.APIEvents = out.APIEvents[:10]
	}
	return out, nil
}

// suggested fills the brief's eight rules in for this workspace.
func (s *Service) suggested(ctx context.Context, q *store.Queries, wsID uuid.UUID, opts ScoringOptions) []ScoringRule {
	var form uuid.UUID
	for _, f := range opts.Forms {
		if strings.Contains(strings.ToLower(f.Name), "beta") {
			form = uuid.MustParse(f.ID)
			break
		}
	}
	if form == uuid.Nil && len(opts.Forms) > 0 {
		form = uuid.MustParse(opts.Forms[0].ID)
	}
	plan := false
	for _, f := range opts.Fields {
		if f.Key == "plan" {
			plan = true
		}
	}
	rules := SuggestedRules(form, plan)
	domain := ""
	if origins, err := q.ScoringFormOrigins(ctx, wsID); err == nil {
		for _, o := range origins {
			if o != "" && (domain == "" || len(o) < len(domain)) {
				domain = o
			}
		}
	}
	for i := range rules {
		if rules[i].Kind == KindClick && rules[i].Contains == "/pricing" && domain != "" {
			rules[i].Contains = domain + "/pricing"
		}
	}
	return rules
}

func (s *Service) uses(ctx context.Context, q *store.Queries, ws *store.Workspace) ([]ScoringUse, error) {
	out := []ScoringUse{}
	flows, err := q.ScoringLiveTriggers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range flows {
		u := ScoringUse{Kind: "automation", ID: f.ID.String(), Name: f.Name, Status: f.Status, By: f.CreatedByName, Entered: f.Entered,
			Href: "/w/" + ws.Slug + "/automations/" + f.ID.String()}
		if f.LiveSince.Year() > 1 {
			since := f.LiveSince
			u.Since = &since
		}
		var t struct {
			Direction  string `json:"direction"`
			Threshold  int    `json:"threshold"`
			OnceInDays int    `json:"onceInDays"`
		}
		if json.Unmarshal(f.Trigger, &t) == nil {
			u.Direction, u.Threshold, u.OnceInDays = t.Direction, t.Threshold, t.OnceInDays
			if u.Direction == "" {
				u.Direction = "rises"
			}
		}
		out = append(out, u)
	}
	segs, err := q.ScoringSegmentsUsingScore(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sg := range segs {
		var def struct {
			Groups []struct {
				Conditions json.RawMessage `json:"conditions"`
			} `json:"groups"`
		}
		_ = json.Unmarshal(sg.Definition, &def)
		conds := []json.RawMessage{}
		for _, g := range def.Groups {
			var cs []json.RawMessage
			if json.Unmarshal(g.Conditions, &cs) == nil {
				conds = append(conds, cs...)
			}
		}
		out = append(out, ScoringUse{Kind: "segment", ID: sg.ID.String(), Name: sg.Name, By: sg.CreatedByName, Contacts: sg.CachedCount,
			Conditions: marshal(conds), Href: "/w/" + ws.Slug + "/audience/segments/" + sg.ID.String()})
	}
	usesMu.Lock()
	finders := slices.Clone(usesFinders)
	usesMu.Unlock()
	for _, f := range finders {
		more, err := f(ctx, ws.ID)
		if err != nil {
			s.log().WarnContext(ctx, "scoring uses", "err", err)
			continue
		}
		out = append(out, more...)
	}
	return out, nil
}

// recalcInfo returns the latest recalculation while it matters: running,
// stopped, or finished in the last 15 minutes.
func (s *Service) recalcInfo(ctx context.Context, q *store.Queries, wsID uuid.UUID) (*ScoringRecalcInfo, error) {
	rc, err := q.ScoringRecalcLatest(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	if rc.Status == "superseded" || (rc.Status == "finished" && (rc.FinishedAt == nil || now.Sub(*rc.FinishedAt) > 15*time.Minute)) {
		return nil, nil
	}
	info := &ScoringRecalcInfo{ID: rc.ID.String(), Status: rc.Status, Reason: rc.Reason, Total: rc.Total, Done: rc.Done, StartAutomations: rc.StartAutomations,
		StartedAt: rc.StartedAt, StartedBy: rc.StartedByName, FinishedAt: rc.FinishedAt, FailedAt: rc.FailedAt, Label: rc.Label, Changes: ScoringChanges{Rules: []int{}}}
	_ = json.Unmarshal(rc.Changes, &info.Changes)
	if info.Changes.Rules == nil {
		info.Changes.Rules = []int{}
	}
	if rc.Status == "finished" {
		var sum ScoringRecalcSummary
		if json.Unmarshal(rc.Summary, &sum) == nil {
			if sum.Moves == nil {
				sum.Moves = []ScoringBandMove{}
			}
			if sum.Flows == nil {
				sum.Flows = []ScoringFlowImpact{}
			}
			info.Summary = &sum
		}
	}
	if rc.Status == "running" && rc.Done > 0 && rc.Total > rc.Done {
		elapsed := now.Sub(rc.StartedAt).Seconds()
		left := int(elapsed / float64(rc.Done) * float64(rc.Total-rc.Done))
		info.RemainingSeconds = &left
	}
	return info, nil
}
