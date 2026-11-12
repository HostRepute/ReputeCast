package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"net/netip"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/abstats"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracker"
)

// A/B tests of a form (M60 G-25). Variant A is the form itself; B and C
// override its copy (heading, description, button, after-submit message),
// its steps (on or off, their grouping and copy) and its popup triggers.
// Fields, lists, consent and spam settings are the same in every variant.
//
// A visitor is assigned a variant by the session endpoint from a random id
// their browser keeps (FNV-64a of the form and the id, mod 100, against the
// cumulative weights), so the same browser always sees the same variant.
// Only visitors who see the form in its own language take part: variants
// aren't translated, so the others see A and aren't counted, as aren't
// hosted pages, which always show A. Views, submissions and confirmations
// are counted per variant from the day the test started; editing a
// variant, or A's copy, restarts the count from that day. The winner is
// decided by internal/abstats, the rule campaign A/B tests use.

// Test states (abDoc.Status).
const (
	ABRunning = "running"
	ABEnded   = "ended"
)

// Variant keys, in order: A is the form itself.
var variantKeys = []string{"A", "B", "C"}

// abDoc is a form's A/B test (forms.ab).
type abDoc struct {
	Status string `json:"status,omitempty"`
	// StartedOn is the first day counted (YYYY-MM-DD, the workspace's).
	StartedOn  string     `json:"startedOn,omitempty"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	Winner     string     `json:"winner,omitempty"`
	AutoEnd    bool       `json:"autoEnd,omitempty"`
	MinViews   int        `json:"minViews,omitempty"`
	Confidence float64    `json:"confidence,omitempty"`
}

// Running reports whether the form's test is counting.
func (a abDoc) Running() bool { return a.Status == ABRunning }

func (a abDoc) rule() abstats.Rule {
	return abstats.Rule{MinSample: int64(a.MinViews), Confidence: a.Confidence}.Normalized()
}

// variantDoc is what a variant overrides (form_variants.content).
type variantDoc struct {
	Heading         string    `json:"heading"`
	Description     string    `json:"description"`
	Button          string    `json:"button"`
	AfterSubmitText string    `json:"afterSubmitText,omitempty"`
	Steps           []stepDoc `json:"steps,omitempty"`
	PopupDelay      int       `json:"popupDelay"`
	PopupScroll     int       `json:"popupScroll"`
	PopupExit       bool      `json:"popupExit,omitempty"`
}

// variantOfContent is what a form's own content sets that a variant
// could override: variant A.
func variantOfContent(c contentDoc) variantDoc {
	return variantDoc{Heading: c.Heading, Description: c.Description, Button: c.Button, AfterSubmitText: c.AfterSubmitText,
		Steps: c.Steps, PopupDelay: c.PopupDelay, PopupScroll: c.PopupScroll, PopupExit: c.PopupExit}
}

// apply is the form's content with the variant's overrides.
func (v variantDoc) apply(c contentDoc) contentDoc {
	c.Heading, c.Description, c.Button, c.AfterSubmitText = v.Heading, v.Description, v.Button, v.AfterSubmitText
	c.Steps = v.Steps
	c.PopupDelay, c.PopupScroll, c.PopupExit = v.PopupDelay, v.PopupScroll, v.PopupExit
	return c
}

func sameVariant(a, b variantDoc) bool {
	if len(a.Steps) == 0 && len(b.Steps) == 0 {
		a.Steps, b.Steps = nil, nil
	}
	return reflect.DeepEqual(a, b)
}

// SignupFormVariantContent is what a variant changes.
type SignupFormVariantContent struct {
	Heading         string           `json:"heading" maxLength:"120"`
	Description     string           `json:"description" maxLength:"500"`
	Button          string           `json:"button" maxLength:"40"`
	AfterSubmitText string           `json:"afterSubmitText,omitempty" maxLength:"300"`
	Steps           []SignupFormStep `json:"steps,omitempty" maxItems:"4" doc:"Omitted or one step: everything at once"`
	PopupDelay      int              `json:"popupDelay" minimum:"0" maximum:"600"`
	PopupScroll     int              `json:"popupScroll" minimum:"0" maximum:"100"`
	PopupExit       bool             `json:"popupExit,omitempty"`
}

// SignupFormVariantInput is one variant of a test: A (the form; no
// content) and B, C with what they change.
type SignupFormVariantInput struct {
	Key     string                    `json:"key" enum:"A,B,C"`
	Weight  int                       `json:"weight" minimum:"1" maximum:"99" doc:"Percent of visitors; the variants add up to 100"`
	Content *SignupFormVariantContent `json:"content,omitempty" doc:"Required for B and C; ignored for A"`
}

// SignupFormABInput starts or changes a form's A/B test. Changing a
// variant or the split while it runs restarts the count from today.
type SignupFormABInput struct {
	Variants []SignupFormVariantInput `json:"variants" minItems:"2" maxItems:"3"`
	AutoEnd  bool                     `json:"autoEnd" doc:"End the test on its own, keeping the winner, once one is clear"`
	MinViews int                      `json:"minViews" minimum:"100" maximum:"100000" doc:"Views each variant needs before a winner can be called"`
	// Confidence is what a winner needs: 0.9, 0.95 or 0.99.
	Confidence float64 `json:"confidence" minimum:"0.9" maximum:"0.99"`
}

// SignupFormABEndInput ends a test, keeping one variant.
type SignupFormABEndInput struct {
	Keep string `json:"keep" enum:"A,B,C"`
}

// SignupFormABVariant is one variant with its numbers since the test started.
type SignupFormABVariant struct {
	Key     string                    `json:"key" enum:"A,B,C"`
	Weight  int                       `json:"weight"`
	Content *SignupFormVariantContent `json:"content,omitempty" doc:"What B or C changes; omitted for A"`
	Views   int64                     `json:"views"`
	// Submissions per view, in percent (Rate); Confirmed: double opt-in
	// signups confirmed so far.
	Submissions int64   `json:"submissions"`
	Confirmed   int64   `json:"confirmed"`
	Rate        float64 `json:"rate"`
}

// SignupFormABResult is a form's A/B test and where it stands.
type SignupFormABResult struct {
	Status    string     `json:"status" enum:"none,running,ended"`
	StartedOn string     `json:"startedOn,omitempty" format:"date" doc:"The first day counted, in the workspace's time zone"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	// Winner is the variant kept when the test ended.
	Winner           string                `json:"winner,omitempty"`
	AutoEnd          bool                  `json:"autoEnd"`
	MinViews         int                   `json:"minViews"`
	TargetConfidence float64               `json:"targetConfidence" doc:"What a winner needs, 0.95 for 95%"`
	Variants         []SignupFormABVariant `json:"variants"`
	// Outcome (while running): winner, not_significant, too_small (a
	// variant has fewer than MinViews views: FewestViews).
	Outcome     string `json:"outcome,omitempty" enum:"winner,not_significant,too_small"`
	Leader      string `json:"leader,omitempty"`
	RunnerUp    string `json:"runnerUp,omitempty"`
	FewestViews int64  `json:"fewestViews"`
	// Lift is the leader's rate over the runner-up's, in percent (+28);
	// Confidence how sure the lead is, in percent (97).
	Lift        float64 `json:"lift"`
	Confidence  float64 `json:"confidence"`
	DoubleOptIn bool    `json:"doubleOptIn"`
}

func variantContentOut(v variantDoc) *SignupFormVariantContent {
	return &SignupFormVariantContent{Heading: v.Heading, Description: v.Description, Button: v.Button, AfterSubmitText: v.AfterSubmitText,
		Steps: stepsOut(v.Steps), PopupDelay: v.PopupDelay, PopupScroll: v.PopupScroll, PopupExit: v.PopupExit}
}

func parseVariantContent(ctx context.Context, in *SignupFormVariantContent, fields []fieldDoc, loc string) (variantDoc, error) {
	if in == nil {
		return variantDoc{}, api.Invalid(loc, say(ctx, mVariantContent, nil))
	}
	v := variantDoc{Heading: strings.TrimSpace(in.Heading), Description: strings.TrimSpace(in.Description), Button: strings.TrimSpace(in.Button),
		AfterSubmitText: strings.TrimSpace(in.AfterSubmitText), PopupDelay: in.PopupDelay, PopupScroll: in.PopupScroll, PopupExit: in.PopupExit}
	if v.Button == "" {
		return variantDoc{}, api.Invalid(loc+".button", say(ctx, mNameButton, nil))
	}
	steps, err := parseSteps(ctx, in.Steps, fields, loc+".steps")
	if err != nil {
		return variantDoc{}, err
	}
	v.Steps = steps
	return v, nil
}

var visitorIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

// PickVariant assigns a visitor (their browser's random id) a variant:
// FNV-64a of the form and the id, mod 100, against the cumulative weights
// in key order. The same id always gets the same variant while the split
// stays.
func PickVariant(formID uuid.UUID, visitor string, vs []store.FormVariant) string {
	if len(vs) == 0 {
		return ""
	}
	h := fnv.New64a()
	_, _ = h.Write(formID[:])
	_, _ = h.Write([]byte(visitor))
	bucket := int32(h.Sum64() % 100) //nolint:gosec // G115: < 100
	acc := int32(0)
	for _, v := range vs {
		acc += v.Weight
		if bucket < acc {
			return v.Key
		}
	}
	return vs[len(vs)-1].Key
}

// SessionRequest is an embedded form asking whether to show (its session).
type SessionRequest struct {
	IP        *netip.Addr
	UserAgent string
	// VisitorID is the browser's random id; InTest says the form shows in
	// its own language, so the visitor can take part in an A/B test. Peek
	// asks without counting a view (a popup before it opens).
	VisitorID string
	InTest    bool
	Peek      bool
	// TrackerVisitor is the site tracker's id of the browser (its rc_vid
	// cookie, when the tracker runs on the page): a segment rule (M62
	// G-32) looks its contact up.
	TrackerVisitor string
}

// SessionAnswer says whether the form shows and in which variant ("" outside a test).
type SessionAnswer struct {
	Show    bool
	Variant string
}

// Session decides whether an embedded or popup form shows (its schedule
// and countries; the script has checked the rest) and, while an A/B test
// runs, in which variant. A form that shows counts a view.
func (s *Service) Session(ctx context.Context, ws *store.Workspace, c Config, req SessionRequest) SessionAnswer {
	if !c.Open(s.now()) || !c.CountryShown(s.country(req.IP)) || !s.segmentShown(ctx, ws, c, req.TrackerVisitor) {
		return SessionAnswer{}
	}
	variant := ""
	if c.AB.Running() && req.InTest && c.Kind != KindHosted && visitorIDPattern.MatchString(req.VisitorID) {
		vs, err := s.Q.FormVariantsList(ctx, c.ID)
		if err != nil {
			s.log().WarnContext(ctx, "load form variants", "form", c.ID, "err", err)
		} else {
			variant = PickVariant(c.ID, req.VisitorID, vs)
		}
	}
	if !req.Peek {
		s.recordView(ctx, ws, c, req.UserAgent, variant)
	}
	return SessionAnswer{Show: true, Variant: variant}
}

// variantOf is the variant a post says it showed, when the form's test has it.
func variantOf(ctx context.Context, q *store.Queries, c Config, posted string) (string, error) {
	if posted == "" || !c.AB.Running() || !slices.Contains(variantKeys, posted) {
		return "", nil
	}
	vs, err := q.FormVariantsList(ctx, c.ID)
	if err != nil {
		return "", err
	}
	if slices.ContainsFunc(vs, func(v store.FormVariant) bool { return v.Key == posted }) {
		return posted, nil
	}
	return "", nil
}

// RecordStep counts a view of a step after the first (the embed script's
// beacon). The step's session token must be good; crawlers don't count.
func (s *Service) RecordStep(ctx context.Context, ws *store.Workspace, c Config, step int, variant, token, userAgent string) {
	if c.Builtin || step < 2 || step > MaxSteps || isBot(userAgent) || !s.TokenValid(formKey(c), token, s.now()) {
		return
	}
	v, err := variantOf(ctx, s.Q, c, variant)
	if err != nil {
		s.log().WarnContext(ctx, "count form step", "err", err)
		return
	}
	if err := s.Q.FormBumpStep(ctx, store.FormBumpStepParams{FormID: c.ID, Day: dayOf(s.now(), location(ws)), Variant: v, Step: int16(step)}); err != nil { //nolint:gosec // G115: 2–4
		s.log().WarnContext(ctx, "count form step", "err", err)
	}
}

// stepFunnel fills a multi-step form's funnel: step 1 is the form's views,
// later steps their beacons, then the signups.
func (s *Service) stepFunnel(ctx context.Context, formID uuid.UUID, since time.Time, steps int, st *SignupFormStats) error {
	own, err := s.Q.FormOwnStatsSince(ctx, store.FormOwnStatsSinceParams{FormID: formID, Since: since})
	if err != nil {
		return err
	}
	rows, err := s.Q.FormStepViewsSince(ctx, store.FormStepViewsSinceParams{FormID: formID, Since: since})
	if err != nil {
		return err
	}
	byStep := map[int32]int64{}
	for _, r := range rows {
		byStep[r.Step] = r.Views
	}
	st.Steps = append(st.Steps, SignupFormStepStat{Step: 1, Views: own.Views})
	for n := 2; n <= steps; n++ {
		st.Steps = append(st.Steps, SignupFormStepStat{Step: n, Views: byStep[int32(n)]}) //nolint:gosec // G115: 2–4
	}
	st.StepSignups = own.Submissions
	return nil
}

// ---- The test API ----

// ABResult returns a form's A/B test and where it stands.
func (s *Service) ABResult(ctx context.Context, ws *store.Workspace, id string) (SignupFormABResult, error) {
	f, err := s.form(ctx, s.Q, ws, id)
	if err != nil {
		return SignupFormABResult{}, err
	}
	return s.abResultOf(ctx, s.Q, ws, f)
}

func (s *Service) abResultOf(ctx context.Context, q *store.Queries, ws *store.Workspace, f store.Form) (SignupFormABResult, error) {
	c := ConfigOf(f)
	rule := c.AB.rule()
	out := SignupFormABResult{Status: "none", Variants: []SignupFormABVariant{}, MinViews: int(rule.MinSample), TargetConfidence: rule.Confidence, DoubleOptIn: c.DoubleOptIn}
	if c.AB.Status == "" {
		return out, nil
	}
	out.Status, out.StartedOn, out.EndedAt, out.Winner, out.AutoEnd = c.AB.Status, c.AB.StartedOn, c.AB.EndedAt, c.AB.Winner, c.AB.AutoEnd
	if !c.AB.Running() {
		return out, nil
	}
	vs, err := q.FormVariantsList(ctx, f.ID)
	if err != nil {
		return out, err
	}
	since, err := time.Parse(time.DateOnly, c.AB.StartedOn)
	if err != nil {
		since = dayOf(s.now(), location(ws))
	}
	rows, err := q.FormVariantStatsSince(ctx, store.FormVariantStatsSinceParams{FormID: f.ID, Since: since})
	if err != nil {
		return out, err
	}
	stats := map[string]store.FormVariantStatsSinceRow{}
	for _, r := range rows {
		stats[r.Variant] = r
	}
	var decide []abstats.Variant
	out.FewestViews = -1
	for _, v := range vs {
		r := stats[v.Key]
		item := SignupFormABVariant{Key: v.Key, Weight: int(v.Weight), Views: r.Views, Submissions: r.Submissions, Confirmed: r.Confirmed, Rate: ratio(r.Submissions, r.Views)}
		if v.Key != "A" {
			var d variantDoc
			_ = json.Unmarshal(v.Content, &d)
			item.Content = variantContentOut(d)
		}
		out.Variants = append(out.Variants, item)
		decide = append(decide, abstats.Variant{Key: v.Key, Eligible: r.Views, Conversions: r.Submissions})
		if out.FewestViews < 0 || r.Views < out.FewestViews {
			out.FewestViews = r.Views
		}
	}
	if out.FewestViews < 0 {
		out.FewestViews = 0
	}
	res := abstats.Decide(rule, decide)
	out.Outcome, out.Leader, out.RunnerUp = res.Outcome, res.Leader, res.RunnerUp
	out.Confidence = math.Round(res.Confidence) // whole percent
	lr, ok1 := abstats.Rate(stats[res.Leader].Submissions, stats[res.Leader].Views)
	rr, ok2 := abstats.Rate(stats[res.RunnerUp].Submissions, stats[res.RunnerUp].Views)
	if ok1 && ok2 && rr > 0 {
		out.Lift = math.Round((lr/rr - 1) * 100)
	}
	return out, nil
}

// abSnapshot is what the audit log keeps of a test.
func abSnapshot(ab abDoc, vs []store.FormVariant) map[string]any {
	variants := []map[string]any{}
	for _, v := range vs {
		variants = append(variants, map[string]any{"key": v.Key, "weight": v.Weight, "content": v.Content})
	}
	return map[string]any{"status": ab.Status, "startedOn": ab.StartedOn, "autoEnd": ab.AutoEnd, "minViews": ab.MinViews, "confidence": ab.Confidence, "variants": variants}
}

func (s *Service) today(ws *store.Workspace) string {
	return dayOf(s.now(), location(ws)).Format(time.DateOnly)
}

// SaveAB starts a form's A/B test, or changes its variants, split or rule.
func (s *Service) SaveAB(ctx context.Context, ws *store.Workspace, id string, in SignupFormABInput) (SignupFormABResult, error) {
	var out SignupFormABResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		c := ConfigOf(f)
		if c.Kind == KindHosted {
			return api.Err(409, "form.ab_hosted", say(ctx, mABHosted, nil))
		}
		if !slices.Contains([]float64{0.9, 0.95, 0.99}, in.Confidence) {
			return api.Invalid("body.confidence", say(ctx, mABConfidence, nil))
		}
		total := 0
		docs := map[string]variantDoc{}
		for i, v := range in.Variants {
			if v.Key != variantKeys[i] {
				return api.Invalid("body.variants", say(ctx, mABKeys, nil))
			}
			total += v.Weight
			if v.Key == "A" {
				continue
			}
			d, err := parseVariantContent(ctx, v.Content, c.Fields, fmt.Sprintf("body.variants[%d].content", i))
			if err != nil {
				return err
			}
			docs[v.Key] = d
		}
		if total != 100 {
			return api.Invalid("body.variants", say(ctx, mABSplit, nil))
		}
		before, err := q.FormVariantsList(ctx, f.ID)
		if err != nil {
			return err
		}
		changed := len(before) != len(in.Variants)
		for i, v := range in.Variants {
			if changed {
				break
			}
			var prev variantDoc
			_ = json.Unmarshal(before[i].Content, &prev)
			changed = before[i].Key != v.Key || int(before[i].Weight) != v.Weight || (v.Key != "A" && !sameVariant(prev, docs[v.Key]))
		}
		ab := c.AB
		if !ab.Running() {
			ab = abDoc{Status: ABRunning, StartedOn: s.today(ws)}
		} else if changed {
			ab.StartedOn = s.today(ws) // the numbers so far compared other versions
		}
		ab.AutoEnd, ab.MinViews, ab.Confidence = in.AutoEnd, in.MinViews, in.Confidence
		now := s.now()
		keys := make([]string, 0, len(in.Variants))
		for _, v := range in.Variants {
			content := json.RawMessage(`{}`)
			if v.Key != "A" {
				content, _ = json.Marshal(docs[v.Key])
			}
			if err := q.FormVariantUpsert(ctx, store.FormVariantUpsertParams{ID: uuid.Must(uuid.NewV7()), FormID: f.ID, Key: v.Key, Weight: int32(v.Weight), Content: content, At: now}); err != nil { //nolint:gosec // G115: 1–99
				return err
			}
			keys = append(keys, v.Key)
		}
		if err := q.FormVariantsDeleteExcept(ctx, store.FormVariantsDeleteExceptParams{FormID: f.ID, Keys: keys}); err != nil {
			return err
		}
		raw, _ := json.Marshal(ab)
		after, err := q.FormSetAB(ctx, store.FormSetABParams{WorkspaceID: ws.ID, ID: f.ID, Ab: raw, UpdatedBy: actor(ctx), At: now})
		if err != nil {
			return err
		}
		vs, err := q.FormVariantsList(ctx, f.ID)
		if err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.ab_updated", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
			Before: abSnapshot(c.AB, before), After: abSnapshot(ab, vs)}); err != nil {
			return err
		}
		out, err = s.abResultOf(ctx, q, ws, after)
		return err
	})
	if err == nil {
		s.forgetHosts()
	}
	return out, err
}

// EndAB ends a form's A/B test, keeping one variant: its overrides become
// the form's content and the other variants are deleted.
func (s *Service) EndAB(ctx context.Context, ws *store.Workspace, id string, in SignupFormABEndInput) (SignupFormABResult, error) {
	var out SignupFormABResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		after, err := s.endAB(ctx, q, ws, f, in.Keep, nil)
		if err != nil {
			return err
		}
		out, err = s.abResultOf(ctx, q, ws, after)
		return err
	})
	if err == nil {
		s.forgetHosts()
	}
	return out, err
}

func (s *Service) endAB(ctx context.Context, q *store.Queries, ws *store.Workspace, f store.Form, keep string, by *audit.Actor) (store.Form, error) {
	c := ConfigOf(f)
	if !c.AB.Running() {
		return f, api.Err(409, "form.ab_none", say(ctx, mABNone, nil))
	}
	result, err := s.abResultOf(ctx, q, ws, f)
	if err != nil {
		return f, err
	}
	vs, err := q.FormVariantsList(ctx, f.ID)
	if err != nil {
		return f, err
	}
	i := slices.IndexFunc(vs, func(v store.FormVariant) bool { return v.Key == keep })
	if i < 0 {
		return f, api.Invalid("body.keep", say(ctx, mABNoVariant, i18n.Args{"key": keep}))
	}
	var content contentDoc
	_ = json.Unmarshal(f.Content, &content)
	if keep != "A" {
		var d variantDoc
		_ = json.Unmarshal(vs[i].Content, &d)
		content = d.apply(content)
		content.Steps = fitSteps(content.Steps, c.Fields)
	}
	now := s.now()
	ab := c.AB
	ab.Status, ab.EndedAt, ab.Winner = ABEnded, &now, keep
	rawAB, _ := json.Marshal(ab)
	rawContent, _ := json.Marshal(content)
	after, err := q.FormEndAB(ctx, store.FormEndABParams{WorkspaceID: ws.ID, ID: f.ID, Ab: rawAB, Content: rawContent, UpdatedBy: actor(ctx), At: now})
	if err != nil {
		return f, err
	}
	if err := q.FormVariantsDelete(ctx, f.ID); err != nil {
		return f, err
	}
	numbers := []map[string]any{}
	for _, v := range result.Variants {
		numbers = append(numbers, map[string]any{"key": v.Key, "views": v.Views, "submissions": v.Submissions, "confirmed": v.Confirmed, "rate": v.Rate})
	}
	return after, s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.ab_ended", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
		Actor: by, Before: abSnapshot(c.AB, vs), After: map[string]any{"status": ABEnded, "winner": keep, "content": json.RawMessage(rawContent)},
		Metadata: map[string]any{"startedOn": c.AB.StartedOn, "outcome": result.Outcome, "leader": result.Leader, "confidence": result.Confidence, "lift": result.Lift, "variants": numbers}})
}

// keepTestInStep keeps a running test in line with a saved form: variants'
// steps follow its fields, and a change to what A shows restarts the count.
func (s *Service) keepTestInStep(ctx context.Context, q *store.Queries, ws *store.Workspace, before, after store.Form) (store.Form, error) {
	c := ConfigOf(after)
	if !c.AB.Running() {
		return after, nil
	}
	vs, err := q.FormVariantsList(ctx, after.ID)
	if err != nil {
		return after, err
	}
	now := s.now()
	for _, v := range vs {
		if v.Key == "A" {
			continue
		}
		var d variantDoc
		_ = json.Unmarshal(v.Content, &d)
		fit := fitSteps(d.Steps, c.Fields)
		if reflect.DeepEqual(fit, d.Steps) || (len(fit) == 0 && len(d.Steps) == 0) {
			continue
		}
		d.Steps = fit
		raw, _ := json.Marshal(d)
		if err := q.FormVariantUpsert(ctx, store.FormVariantUpsertParams{ID: v.ID, FormID: v.FormID, Key: v.Key, Weight: v.Weight, Content: raw, At: now}); err != nil {
			return after, err
		}
	}
	var was, is contentDoc
	_ = json.Unmarshal(before.Content, &was)
	_ = json.Unmarshal(after.Content, &is)
	if sameVariant(variantOfContent(was), variantOfContent(is)) {
		return after, nil
	}
	ab := c.AB
	ab.StartedOn = s.today(ws)
	raw, _ := json.Marshal(ab)
	return q.FormSetAB(ctx, store.FormSetABParams{WorkspaceID: ws.ID, ID: after.ID, Ab: raw, UpdatedBy: actor(ctx), At: now})
}

// ---- Ending tests on their own ----

// ABAutoEndArgs ends running tests that asked to, once a variant wins.
type ABAutoEndArgs struct{}

// Kind implements river.JobArgs.
func (ABAutoEndArgs) Kind() string { return "forms.ab_auto_end" }

// InsertOpts runs on the maintenance queue, once a day.
func (ABAutoEndArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 23 * time.Hour}}
}

// ABAutoEndWorker runs ABAutoEndArgs.
type ABAutoEndWorker struct {
	river.WorkerDefaults[ABAutoEndArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ABAutoEndWorker) Work(ctx context.Context, _ *river.Job[ABAutoEndArgs]) error {
	return w.S.AutoEndTests(ctx)
}

// AutoEndTests ends every running test with auto-end on whose outcome is a
// winner, keeping the leader, and tells the people who edit forms.
func (s *Service) AutoEndTests(ctx context.Context) error {
	forms, err := s.Q.FormsABAutoEnd(ctx)
	if err != nil {
		return err
	}
	for _, f := range forms {
		if err := s.autoEnd(ctx, f); err != nil {
			s.log().ErrorContext(ctx, "end form test", "form", f.ID, "err", err)
		}
	}
	return nil
}

func (s *Service) autoEnd(ctx context.Context, f store.Form) error {
	ws, err := s.Q.GetWorkspace(ctx, f.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		res, err := s.abResultOf(ctx, q, &ws, f)
		if err != nil || res.Outcome != abstats.Winner {
			return err
		}
		if _, err := s.endAB(ctx, q, &ws, f, res.Leader, &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"}); err != nil {
			return err
		}
		var rate float64
		for _, v := range res.Variants {
			if v.Key == res.Leader {
				rate = v.Rate
			}
		}
		args := i18n.Args{"form": f.Name, "key": res.Leader, "rate": strconv.FormatFloat(rate, 'f', 1, 64), "confidence": strconv.FormatFloat(res.Confidence, 'f', 0, 64)}
		s.log().InfoContext(ctx, "form test ended", slog.String("form", f.ID.String()), slog.String("winner", res.Leader))
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.FormsEdit, notify.Notification{
			Kind: notify.KindFormTestEnded, Severity: "ok",
			TitleText: mABEndedTitle.Keep(args), BodyText: mABEndedBody.Keep(args),
			Link: "/w/" + ws.Slug + "/forms?form=" + f.ID.String(), Data: map[string]any{"formId": f.ID.String(), "winner": res.Leader},
		})
	})
}

func registerABJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ABAutoEndWorker{S: s})
	reg.Every(24*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return ABAutoEndArgs{}, nil }, true)
}

// segmentShown applies a form's segment rule (M62 G-32): the browser's
// contact, as Website tracking on the same site knows it, in the segment's
// materialised members. A visitor nobody knows isn't in any segment.
func (s *Service) segmentShown(ctx context.Context, ws *store.Workspace, c Config, trackerVisitor string) bool {
	seg := c.segmentRule()
	if seg == uuid.Nil {
		return true
	}
	known, member := false, false
	if hash, ok := tracker.VisitorHash(ws.ID, trackerVisitor); ok {
		r, err := s.Q.FormVisitorSegment(ctx, store.FormVisitorSegmentParams{SegmentID: seg, WorkspaceID: ws.ID, VidHash: hash})
		if err != nil {
			s.log().WarnContext(ctx, "form segment rule", "form", c.ID, "err", err)
		} else {
			known, member = r.Known, r.Member
		}
	}
	return c.SegmentShown(known, member)
}
