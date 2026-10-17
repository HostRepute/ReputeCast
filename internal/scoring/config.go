package scoring

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Rule kinds: the events a rule counts, and field conditions.
const (
	KindOpen        = "open"        // human open of any email, or of a list's emails
	KindClick       = "click"       // click on any link, or a link whose address contains some text
	KindForm        = "form"        // submitted a chosen form
	KindAPIEvent    = "api_event"   // an API event by name
	KindSoftBounce  = "soft_bounce" // not delivered after the retries
	KindUnsubscribe = "unsubscribe" // unsubscribed from a list (any, or one)
	KindField       = "field"       // a field condition, counted while it is true
)

// Cap kinds. "once" counts the first matching event of the look-back;
// once_* count one event per calendar day, week (from Monday) or month in
// the workspace timezone; points_* add at most N points per such period.
const (
	CapNone        = "none"
	CapOnce        = "once"
	CapOnceDay     = "once_day"
	CapOnceWeek    = "once_week"
	CapOnceMonth   = "once_month"
	CapPointsDay   = "points_day"
	CapPointsWeek  = "points_week"
	CapPointsMonth = "points_month"
	CapWhileTrue   = "while_true" // field conditions
)

// Field condition operators.
const (
	OpIsOneOf = "is_one_of"
	OpIsNot   = "is_not"
	OpIsSet   = "is_set"
)

// Bands, in order. Their names are fixed so segments and triggers read the
// same everywhere; only the two thresholds move.
const (
	BandCold = "cold"
	BandWarm = "warm"
	BandHot  = "hot"
)

// MaxScore is the highest score. LookbackDays is how far back events count.
const (
	MaxScore     = 100
	LookbackDays = 180
	// KeepDays is how long score events are kept: the 180 days that count
	// plus 90 so the profile can draw the last 90 days of the score.
	KeepDays = LookbackDays + 90
	MaxRules = 50
)

// ScoringRuleCap limits what a rule adds.
type ScoringRuleCap struct {
	Kind string `json:"kind" enum:"none,once,once_day,once_week,once_month,points_day,points_week,points_month,while_true"`
	// Points is the most a points_* cap adds in its period.
	Points int `json:"points,omitempty" minimum:"0" maximum:"1000"`
}

// ScoringRule is one rule: what counts, the points it gives (or takes) and its cap.
type ScoringRule struct {
	ID   string `json:"id" maxLength:"40" doc:"Kept across versions; new rules get one when saved"`
	Kind string `json:"kind" enum:"open,click,form,api_event,soft_bounce,unsubscribe,field"`
	// ListID limits opens to a list's emails, or unsubscribes to one list.
	ListID string `json:"listId,omitempty" maxLength:"40"`
	// Contains limits clicks to links whose address contains this text.
	Contains string `json:"contains,omitempty" maxLength:"200"`
	FormID   string `json:"formId,omitempty" maxLength:"40"`
	Event    string `json:"event,omitempty" maxLength:"100"`
	// Field, Op and Values make a field condition: a custom field's key, or
	// country, language or company.
	Field   string         `json:"field,omitempty" maxLength:"64"`
	Op      string         `json:"op,omitempty" enum:"is_one_of,is_not,is_set"`
	Values  []string       `json:"values,omitempty" maxItems:"20"`
	Points  int            `json:"points" minimum:"-50" maximum:"50"`
	Cap     ScoringRuleCap `json:"cap"`
	Enabled bool           `json:"enabled"`
}

// ScoringDecay takes a share of the score each week after a stretch
// without a human open or click.
type ScoringDecay struct {
	Enabled    bool `json:"enabled"`
	AfterWeeks int  `json:"afterWeeks" minimum:"1" maximum:"52"`
	Percent    int  `json:"percent" minimum:"1" maximum:"100"`
}

// ScoringBands are the two thresholds: Warm from, Hot from.
type ScoringBands struct {
	WarmFrom int `json:"warmFrom" minimum:"1" maximum:"99"`
	HotFrom  int `json:"hotFrom" minimum:"2" maximum:"100"`
}

// ScoringConfig is a workspace's whole scoring set-up, saved as one version.
type ScoringConfig struct {
	Rules []ScoringRule `json:"rules" maxItems:"50"`
	Decay ScoringDecay  `json:"decay"`
	Bands ScoringBands  `json:"bands"`
}

// DefaultDecay and DefaultBands are the brief's defaults.
var (
	DefaultDecay = ScoringDecay{Enabled: true, AfterWeeks: 4, Percent: 10}
	DefaultBands = ScoringBands{WarmFrom: 20, HotFrom: 50}
)

// Band names a score's band under the thresholds.
func (b ScoringBands) Band(score int) string {
	switch {
	case score >= b.HotFrom:
		return BandHot
	case score >= b.WarmFrom:
		return BandWarm
	}
	return BandCold
}

// ParseConfig reads a stored version.
func ParseConfig(raw []byte) (ScoringConfig, error) {
	var c ScoringConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	if c.Rules == nil {
		c.Rules = []ScoringRule{}
	}
	if c.Bands.WarmFrom == 0 && c.Bands.HotFrom == 0 {
		c.Bands = DefaultBands
	}
	if c.Decay.AfterWeeks == 0 {
		c.Decay.AfterWeeks = DefaultDecay.AfterWeeks
	}
	if c.Decay.Percent == 0 {
		c.Decay.Percent = DefaultDecay.Percent
	}
	for i := range c.Rules {
		if c.Rules[i].Values == nil {
			c.Rules[i].Values = []string{}
		}
	}
	return c, nil
}

// Clone copies a configuration, so a draft can change without touching it.
func (c ScoringConfig) Clone() ScoringConfig {
	out := c
	out.Rules = make([]ScoringRule, len(c.Rules))
	for i, r := range c.Rules {
		r.Values = slices.Clone(r.Values)
		if r.Values == nil {
			r.Values = []string{}
		}
		out.Rules[i] = r
	}
	return out
}

// IsEvent reports whether a rule counts events (not a field condition).
func (r ScoringRule) IsEvent() bool { return r.Kind != KindField }

var (
	reEventName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,99}$`)
	reFieldKey  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
)

// BuiltinFields are the contact fields a condition can use besides custom fields.
var BuiltinFields = []string{"country", "language", "company"}

// FieldError is a rule that can't be saved, with where and why. Message
// is a key of the scoring catalog with its arguments, so the API can say
// it in the reader's language.
type FieldError struct {
	Location string
	Key      string
	Args     map[string]any
}

func (e *FieldError) Error() string { return e.Location + ": " + e.Key }

// Refs are what a configuration may point at in its workspace.
type Refs struct {
	Lists  map[uuid.UUID]bool
	Forms  map[uuid.UUID]bool
	Fields map[string]string // custom field key → type
}

// Normalize fills ids and defaults and trims text; it doesn't validate.
func (c *ScoringConfig) Normalize() {
	if c.Rules == nil {
		c.Rules = []ScoringRule{}
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		if _, err := uuid.Parse(r.ID); err != nil {
			r.ID = uuid.Must(uuid.NewV7()).String()
		}
		r.Contains = strings.TrimSpace(r.Contains)
		r.Event = strings.TrimSpace(r.Event)
		r.Field = strings.TrimSpace(r.Field)
		vals := []string{}
		for _, v := range r.Values {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(vals, v) {
				vals = append(vals, v)
			}
		}
		r.Values = vals
		if r.Cap.Kind == "" {
			r.Cap.Kind = CapNone
		}
		if r.Kind == KindField {
			r.Cap = ScoringRuleCap{Kind: CapWhileTrue}
		}
		if !strings.HasPrefix(r.Cap.Kind, "points_") {
			r.Cap.Points = 0
		}
		switch r.Kind {
		case KindOpen, KindUnsubscribe:
			r.Contains, r.FormID, r.Event, r.Field, r.Op, r.Values = "", "", "", "", "", []string{}
		case KindClick:
			r.ListID, r.FormID, r.Event, r.Field, r.Op, r.Values = "", "", "", "", "", []string{}
		case KindForm:
			r.ListID, r.Contains, r.Event, r.Field, r.Op, r.Values = "", "", "", "", "", []string{}
		case KindAPIEvent:
			r.ListID, r.Contains, r.FormID, r.Field, r.Op, r.Values = "", "", "", "", "", []string{}
		case KindSoftBounce:
			r.ListID, r.Contains, r.FormID, r.Event, r.Field, r.Op, r.Values = "", "", "", "", "", "", []string{}
		case KindField:
			r.ListID, r.Contains, r.FormID, r.Event = "", "", "", ""
			if r.Op == OpIsSet {
				r.Values = []string{}
			}
		}
	}
}

// Validate checks a normalized configuration against its workspace.
func (c ScoringConfig) Validate(refs Refs) error {
	if len(c.Rules) > MaxRules {
		return &FieldError{"body.config.rules", "error.tooManyRules", map[string]any{"max": MaxRules}}
	}
	seen := map[string]bool{}
	for i, r := range c.Rules {
		loc := fmt.Sprintf("body.config.rules[%d]", i)
		if seen[r.ID] {
			return &FieldError{loc + ".id", "error.duplicateRule", nil}
		}
		seen[r.ID] = true
		if err := r.validate(loc, refs); err != nil {
			return err
		}
	}
	d := c.Decay
	if d.AfterWeeks < 1 || d.AfterWeeks > 52 {
		return &FieldError{"body.config.decay.afterWeeks", "error.decayWeeks", nil}
	}
	if d.Percent < 1 || d.Percent > 100 {
		return &FieldError{"body.config.decay.percent", "error.decayPercent", nil}
	}
	b := c.Bands
	if b.WarmFrom < 1 || b.WarmFrom > 99 {
		return &FieldError{"body.config.bands.warmFrom", "error.warmFrom", nil}
	}
	if b.HotFrom <= b.WarmFrom || b.HotFrom > MaxScore {
		return &FieldError{"body.config.bands.hotFrom", "error.hotFrom", map[string]any{"warm": b.WarmFrom}}
	}
	return nil
}

func (r ScoringRule) validate(loc string, refs Refs) error {
	if r.Points == 0 || r.Points < -50 || r.Points > 50 {
		return &FieldError{loc + ".points", "error.points", nil}
	}
	switch r.Kind {
	case KindOpen, KindUnsubscribe:
		if r.ListID != "" {
			id, err := uuid.Parse(r.ListID)
			if err != nil || (refs.Lists != nil && !refs.Lists[id]) {
				return &FieldError{loc + ".listId", "error.list", nil}
			}
		}
	case KindClick:
		if len(r.Contains) > 200 {
			return &FieldError{loc + ".contains", "error.contains", nil}
		}
	case KindForm:
		id, err := uuid.Parse(r.FormID)
		if err != nil {
			return &FieldError{loc + ".formId", "error.formMissing", nil}
		}
		if refs.Forms != nil && !refs.Forms[id] {
			return &FieldError{loc + ".formId", "error.formGone", nil}
		}
	case KindAPIEvent:
		if !reEventName.MatchString(r.Event) {
			return &FieldError{loc + ".event", "error.event", nil}
		}
	case KindSoftBounce:
	case KindField:
		if !reFieldKey.MatchString(r.Field) {
			return &FieldError{loc + ".field", "error.field", nil}
		}
		if !slices.Contains(BuiltinFields, r.Field) {
			if _, ok := refs.Fields[r.Field]; refs.Fields != nil && !ok {
				return &FieldError{loc + ".field", "error.fieldGone", map[string]any{"field": r.Field}}
			}
		}
		switch r.Op {
		case OpIsOneOf, OpIsNot:
			if len(r.Values) == 0 {
				return &FieldError{loc + ".values", "error.values", map[string]any{"field": r.Field}}
			}
		case OpIsSet:
		default:
			return &FieldError{loc + ".op", "error.op", nil}
		}
		return nil
	default:
		return &FieldError{loc + ".kind", "error.kind", nil}
	}
	switch r.Cap.Kind {
	case CapNone, CapOnce, CapOnceDay, CapOnceWeek, CapOnceMonth:
	case CapPointsDay, CapPointsWeek, CapPointsMonth:
		pts := r.Points
		if pts < 0 {
			pts = -pts
		}
		if r.Cap.Points < pts {
			return &FieldError{loc + ".cap.points", "error.capBelowPoints", map[string]any{
				"cap": r.Cap.Points, "period": strings.TrimPrefix(r.Cap.Kind, "points_"), "points": pts, "kind": r.Kind}}
		}
	default:
		return &FieldError{loc + ".cap.kind", "error.cap", nil}
	}
	return nil
}

// SuggestedRules are the eight rules the empty state offers (brief §4).
// The form and the field are filled in for the workspace when they exist:
// the first form whose name says "beta", and a select field called plan.
func SuggestedRules(betaForm uuid.UUID, plan bool) []ScoringRule {
	id := func() string { return uuid.Must(uuid.NewV7()).String() }
	out := []ScoringRule{
		{ID: id(), Kind: KindOpen, Points: 2, Cap: ScoringRuleCap{Kind: CapPointsWeek, Points: 6}, Enabled: true},
		{ID: id(), Kind: KindClick, Points: 5, Cap: ScoringRuleCap{Kind: CapPointsWeek, Points: 20}, Enabled: true},
		{ID: id(), Kind: KindClick, Contains: "/pricing", Points: 15, Cap: ScoringRuleCap{Kind: CapOnceMonth}, Enabled: true},
	}
	if betaForm != uuid.Nil {
		out = append(out, ScoringRule{ID: id(), Kind: KindForm, FormID: betaForm.String(), Points: 20, Cap: ScoringRuleCap{Kind: CapOnce}, Enabled: true})
	}
	out = append(out, ScoringRule{ID: id(), Kind: KindAPIEvent, Event: "trial_started", Points: 25, Cap: ScoringRuleCap{Kind: CapOnce}, Enabled: true})
	if plan {
		out = append(out, ScoringRule{ID: id(), Kind: KindField, Field: "plan", Op: OpIsOneOf, Values: []string{"team", "business"}, Points: 10, Cap: ScoringRuleCap{Kind: CapWhileTrue}, Enabled: true})
	}
	out = append(out,
		ScoringRule{ID: id(), Kind: KindSoftBounce, Points: -5, Cap: ScoringRuleCap{Kind: CapNone}, Enabled: true},
		ScoringRule{ID: id(), Kind: KindUnsubscribe, Points: -20, Cap: ScoringRuleCap{Kind: CapNone}, Enabled: true},
	)
	for i := range out {
		if out[i].Values == nil {
			out[i].Values = []string{}
		}
	}
	return out
}

// RulesChanged reports whether two configurations count differently
// (anything but the bands): then every contact needs a recalculation.
func RulesChanged(a, b ScoringConfig) bool {
	if a.Decay != b.Decay || len(a.Rules) != len(b.Rules) {
		return true
	}
	for i := range a.Rules {
		if !sameRule(a.Rules[i], b.Rules[i]) {
			return true
		}
	}
	return false
}

func sameRule(a, b ScoringRule) bool {
	return a.ID == b.ID && a.Kind == b.Kind && a.ListID == b.ListID && a.Contains == b.Contains && a.FormID == b.FormID &&
		a.Event == b.Event && a.Field == b.Field && a.Op == b.Op && slices.Equal(a.Values, b.Values) &&
		a.Points == b.Points && a.Cap == b.Cap && a.Enabled == b.Enabled
}

// sameMatch reports whether two rules count the same events: only their
// points, cap or switch may differ.
func sameMatch(a, b ScoringRule) bool {
	return a.Kind == b.Kind && a.ListID == b.ListID && a.Contains == b.Contains && a.FormID == b.FormID && a.Event == b.Event &&
		a.Field == b.Field && a.Op == b.Op && slices.Equal(a.Values, b.Values)
}
