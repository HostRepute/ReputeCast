package scoring

import (
	"fmt"
	"strconv"
	"strings"
)

// Field conditions are evaluated in SQL, over the contacts row c, so the
// nightly run can score a whole workspace in one statement and every place
// that needs the fields part (replays, the preview, the profile) uses the
// same expression.

// sqlArgs collects bind values for a dynamic statement.
type sqlArgs struct {
	args []any
}

// add binds v and returns its placeholder.
func (a *sqlArgs) add(v any) string {
	a.args = append(a.args, v)
	return fmt.Sprintf("$%d", len(a.args))
}

// fieldValueSQL is a field's value as text, and whether it is a number.
func fieldValueSQL(field, typ string) (expr string, number bool) {
	switch field {
	case "country", "language", "company":
		return "c." + field, false
	}
	// Keys are checked against ^[a-z][a-z0-9_]*$ when a rule is saved.
	if typ == "number" {
		return "CASE WHEN jsonb_typeof(c.attributes -> '" + field + "') = 'number' THEN (c.attributes ->> '" + field + "')::numeric END", true
	}
	return "c.attributes ->> '" + field + "'", false
}

// conditionSQL renders one field rule as a boolean expression.
func conditionSQL(r ScoringRule, types map[string]string, a *sqlArgs) string {
	if !reFieldKey.MatchString(r.Field) {
		return "false"
	}
	val, number := fieldValueSQL(r.Field, types[r.Field])
	switch r.Op {
	case OpIsSet:
		if number {
			return "(" + val + " IS NOT NULL)"
		}
		return "(coalesce(" + val + ", '') <> '')"
	case OpIsOneOf, OpIsNot:
		var in string
		if number {
			nums := []float64{}
			for _, v := range r.Values {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					nums = append(nums, f)
				}
			}
			in = "coalesce(" + val + " = ANY (" + a.add(nums) + "::numeric[]), false)"
		} else {
			lower := make([]string, 0, len(r.Values))
			for _, v := range r.Values {
				lower = append(lower, strings.ToLower(strings.TrimSpace(v)))
			}
			p := a.add(lower)
			in = "coalesce(lower(" + val + ") = ANY (" + p + "::text[]), false)"
			if r.Field == "language" {
				// "es" matches "es-MX".
				in = "(" + in + " OR coalesce(lower(split_part(" + val + ", '-', 1)) = ANY (" + p + "::text[]), false))"
			}
		}
		if r.Op == OpIsNot {
			return "(NOT " + in + ")"
		}
		return in
	}
	return "false"
}

// fieldRules returns the enabled field conditions of a configuration.
func fieldRules(cfg ScoringConfig) []ScoringRule {
	var out []ScoringRule
	for _, r := range cfg.Rules {
		if r.Enabled && r.Kind == KindField {
			out = append(out, r)
		}
	}
	return out
}

// FieldsPartSQL is the sum of the true field conditions' points for the
// contacts row c, numbering its bind values after the ones in a.
func fieldsPartSQL(cfg ScoringConfig, types map[string]string, a *sqlArgs) string {
	rules := fieldRules(cfg)
	if len(rules) == 0 {
		return "0"
	}
	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		parts = append(parts, fmt.Sprintf("CASE WHEN %s THEN %d ELSE 0 END", conditionSQL(r, types, a), r.Points))
	}
	return "(" + strings.Join(parts, " + ") + ")"
}
