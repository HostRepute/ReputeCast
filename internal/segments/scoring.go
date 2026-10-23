package segments

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Contact scoring and growth sources (audience-insights brief §3.1, §3.4):
// conditions on the score, the score band, how a contact first joined, the
// first touch's UTM parameters and the referring domain. The segment views
// read them from contact_scores and contact_growth (see views.go): the
// score is null while the workspace has no scoring, and so is the band.

// Operators of the score.
var scoreOps = []string{"at_least", "at_most", "between"}

func init() {
	builtinFields["score"] = fieldSpec{KindNumber, scoreOps}
	builtinFields["score_band"] = fieldSpec{KindEnum, []string{"is", "is_not"}}
	builtinFields["growth_source"] = fieldSpec{KindEnum, []string{"is", "is_not"}}
	builtinFields["utm_source"] = fieldSpec{KindText, []string{"is", "is_not", "contains"}}
	builtinFields["utm_medium"] = fieldSpec{KindText, []string{"is", "is_not", "contains"}}
	builtinFields["utm_campaign"] = fieldSpec{KindText, []string{"is", "is_not", "contains"}}
	builtinFields["referring_domain"] = fieldSpec{KindText, []string{"is", "is_not", "ends_with"}}
	enumValues["score_band"] = [][2]string{{"cold", ""}, {"warm", ""}, {"hot", ""}}
	// Score and band read like engagement, after "Clicked any email"; the
	// growth fields follow the source.
	builtinOrder = insertAfter(builtinOrder, "clicked", "score", "score_band")
	builtinOrder = insertAfter(builtinOrder, "source", "growth_source", "utm_source", "utm_medium", "utm_campaign", "referring_domain")
	extraValidate["score"] = validateScore
	extraValidate["growth_source"] = validateGrowthSource
	extraCompile["score"] = compileScore
	extraCompile["score_band"] = compileBand
	extraCompile["growth_source"] = compileGrowthSource
	for _, f := range []string{"utm_source", "utm_medium", "utm_campaign", "referring_domain"} {
		extraCompile[f] = compileGrowthText
	}
}

// RegisterFieldOptions sets how the catalog lists a field's values (the
// growth package lists the sources contacts joined through).
func RegisterFieldOptions(field string, f func(context.Context, *store.Queries, uuid.UUID) ([]SegmentOption, error)) {
	extraOptions[field] = f
}

func insertAfter(order []string, after string, keys ...string) []string {
	i := slices.Index(order, after)
	if i < 0 {
		return append(order, keys...)
	}
	out := append([]string{}, order[:i+1]...)
	out = append(out, keys...)
	return append(out, order[i+1:]...)
}

// The hooks the scoring and growth fields use: validation, SQL and
// catalog options of a field, by key.
var (
	extraValidate = map[string]func(SegmentCondition) error{}
	extraCompile  = map[string]func(*sqlText, SegmentCondition){}
	extraOptions  = map[string]func(context.Context, *store.Queries, uuid.UUID) ([]SegmentOption, error){}
)

func scoreValue(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil && n >= 0 && n <= 100
}

func validateScore(c SegmentCondition) error {
	a, ok := scoreValue(c.Value)
	if !ok {
		return says(mScore)
	}
	if c.Op == "between" {
		b, ok := scoreValue(c.To)
		if !ok {
			return says(mScoreEnds)
		}
		if b < a {
			return says(mRangeSwap)
		}
	}
	return nil
}

func validateGrowthSource(c SegmentCondition) error {
	kind, _, _ := strings.Cut(c.Value, ":")
	if kind == "" || len(c.Value) > 200 {
		return says(mPickGrowth)
	}
	return nil
}

func compileScore(t *sqlText, c SegmentCondition) {
	a, _ := scoreValue(c.Value)
	switch c.Op {
	case "at_most":
		t.raw("c.score <= ")
		t.bind(num(float64(a)))
	case "between":
		b, _ := scoreValue(c.To)
		t.raw("c.score BETWEEN ")
		t.bind(num(float64(a)))
		t.raw(" AND ")
		t.bind(num(float64(b)))
	default:
		t.raw("c.score >= ")
		t.bind(num(float64(a)))
	}
}

func compileBand(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("c.score_band IS DISTINCT FROM ")
	} else {
		t.raw("c.score_band = ")
	}
	t.bind(str(c.Value))
}

func compileGrowthSource(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("c.growth_source IS DISTINCT FROM ")
	} else {
		t.raw("c.growth_source = ")
	}
	t.bind(str(c.Value))
}

func compileGrowthText(t *sqlText, c SegmentCondition) {
	col := "lower(coalesce(c." + c.Field + ", ''))"
	v := strings.ToLower(strings.TrimSpace(c.Value))
	switch c.Op {
	case "is_not":
		t.raw(col + " <> ")
		t.bind(str(v))
	case "contains":
		t.raw(col + " LIKE ")
		t.bind(str("%" + escapeLike(v) + "%"))
	case "ends_with":
		t.raw(col + " LIKE ")
		t.bind(str("%" + escapeLike(v)))
	default:
		t.raw(col + " = ")
		t.bind(str(v))
	}
}

// HasContact reports whether a contact is in a segment.
func (s *Service) HasContact(ctx context.Context, workspaceID, segmentID, contactID uuid.UUID) (bool, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: workspaceID, ID: segmentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if seg.Refresh == "static" || seg.Mode == "sql" {
		ids, err := s.ContactIDs(ctx, workspaceID, segmentID)
		return slices.Contains(ids, contactID), err
	}
	var def SegmentDefinition
	_ = json.Unmarshal(seg.Definition, &def)
	custom, err := s.customFields(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	c := Compile(def, workspaceID, custom)
	q, args := c.Query(2)
	args = append([]any{workspaceID}, args...)
	args = append(args, contactID)
	var in bool
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = 10000"); err != nil {
			return err
		}
		return tx.QueryRow(ctx, withViews()+"SELECT EXISTS (SELECT 1 FROM (\n"+q+"\n) seg WHERE seg.id = $"+strconv.Itoa(len(args))+")", args...).Scan(&in)
	})
	return in, mapSQLError(err)
}

// scoringColumns and scoringJoins add the score, its band and the growth
// source to the contacts view. Contacts without a score row score 0 once
// the workspace has scoring; joins nothing selects are left out by the
// planner (each join is on a unique key).
const scoringColumns = `
           CASE WHEN sb.hot IS NULL THEN NULL ELSE coalesce(sc.score, 0) END AS score,
           CASE WHEN sb.hot IS NULL THEN NULL WHEN coalesce(sc.score, 0) >= sb.hot THEN 'hot'
                WHEN coalesce(sc.score, 0) >= sb.warm THEN 'warm' ELSE 'cold' END AS score_band,
           CASE WHEN g.contact_id IS NULL THEN NULL
                ELSE g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END END AS growth_source,
           g.utm_source, g.utm_medium, g.utm_campaign, g.referrer_domain AS referring_domain`

const scoringJoins = `
    LEFT JOIN public.contact_scores sc ON sc.contact_id = c.id
    LEFT JOIN (SELECT k.workspace_id, (v.config -> 'bands' ->> 'warmFrom')::int AS warm, (v.config -> 'bands' ->> 'hotFrom')::int AS hot
               FROM public.scoring_configs k
               JOIN public.scoring_versions v ON v.workspace_id = k.workspace_id AND v.version = k.applied_version
               WHERE k.applied_version > 0) sb ON sb.workspace_id = c.workspace_id
    LEFT JOIN public.contact_growth g ON g.contact_id = c.id`

// oneOfCondition matches a custom field against comma-separated values,
// ignoring case: "plan is one of team, business" (the "Hot on paid plans"
// segment of the scoring sample).
func oneOfCondition(t *sqlText, text, value string) {
	var vals []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.ToLower(strings.TrimSpace(v)); v != "" && !slices.Contains(vals, v) {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		t.raw("false")
		return
	}
	t.raw("lower(coalesce(" + text + ", '')) IN (")
	for i, v := range vals {
		if i > 0 {
			t.raw(", ")
		}
		t.bind(str(v))
	}
	t.raw(")")
}
