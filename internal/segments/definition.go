package segments

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Definition is a builder segment: groups combine with AND; inside a group
// the conditions match ALL or ANY.
type Definition struct {
	Groups []Group `json:"groups"`
}

// Group is one box of conditions in the builder.
type Group struct {
	Match      string      `json:"match" enum:"all,any" doc:"all = every condition, any = at least one"`
	Conditions []Condition `json:"conditions"`
}

// Condition is one row of the builder: field, operator and value.
type Condition struct {
	// Field is a key from the catalog: mailbox_provider, email_domain, list,
	// tier, consent, source, tag, country, last_open, opened, clicked,
	// opened_campaign, joined or custom:<key>.
	Field string `json:"field" maxLength:"80"`
	Op    string `json:"op" maxLength:"20"`
	// Value is the compared value (list or campaign id, provider key, text, number, date).
	Value string `json:"value,omitempty" maxLength:"500"`
	// Amount and Unit are used by "in the last N days/weeks".
	Amount int    `json:"amount,omitempty" minimum:"0" maximum:"3650"`
	Unit   string `json:"unit,omitempty" enum:"days,weeks"`
}

// Operator keys and their labels in the builder.
var opLabels = map[string]string{
	"is": "is", "is_not": "is not", "includes": "includes", "excludes": "excludes",
	"in_last": "in the last", "not_in_last": "not in the last", "never": "never",
	"equals": "equals", "not_equals": "not equals", "contains": "contains", "ends_with": "ends with",
	"is_empty": "is empty", "is_not_empty": "is not empty", "gt": "greater than", "lt": "less than",
	"before": "before", "after": "after", "has": "has", "has_not": "doesn't have",
	"is_true": "is true", "is_false": "is false", "opened": "opened", "not_opened": "didn't open",
}

// FieldKind says how a field's value is entered.
type FieldKind string

// Field kinds.
const (
	KindEnum     FieldKind = "enum"     // fixed options
	KindList     FieldKind = "list"     // a list id
	KindCampaign FieldKind = "campaign" // a campaign id
	KindText     FieldKind = "text"
	KindNumber   FieldKind = "number"
	KindDate     FieldKind = "date"
	KindBoolean  FieldKind = "boolean"
	KindPeriod   FieldKind = "period" // in the last N days/weeks
)

// CustomField is the part of a custom field definition the compiler needs.
type CustomField struct {
	Key   string
	Label string
	Type  string // text, number, date, boolean, select
}

var fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// fieldSpec describes a built-in field.
type fieldSpec struct {
	Label string
	Kind  FieldKind
	Ops   []string
}

var builtinFields = map[string]fieldSpec{
	"mailbox_provider": {"Mailbox provider", KindEnum, []string{"is", "is_not"}},
	"email_domain":     {"Email domain", KindText, []string{"is", "is_not", "ends_with"}},
	"list":             {"List membership", KindList, []string{"includes", "excludes"}},
	"tier":             {"Tier", KindEnum, []string{"is", "is_not"}},
	"consent":          {"Consent", KindEnum, []string{"is", "is_not"}},
	"source":           {"Source", KindEnum, []string{"is", "is_not"}},
	"tag":              {"Tag", KindText, []string{"has", "has_not"}},
	"country":          {"Country", KindText, []string{"is", "is_not"}},
	"last_open":        {"Last open", KindPeriod, []string{"in_last", "not_in_last", "never"}},
	"opened":           {"Opened any email", KindPeriod, []string{"in_last", "not_in_last"}},
	"clicked":          {"Clicked any email", KindPeriod, []string{"in_last", "not_in_last"}},
	"opened_campaign":  {"Opened campaign", KindCampaign, []string{"opened", "not_opened"}},
	"joined":           {"Joined", KindPeriod, []string{"in_last", "not_in_last"}},
}

// builtinOrder is the order of fields in the builder's select.
var builtinOrder = []string{
	"mailbox_provider", "email_domain", "list", "tier", "last_open", "opened", "clicked",
	"opened_campaign", "consent", "source", "tag", "country", "joined",
}

// Enum values of the built-in enum fields.
var enumValues = map[string][][2]string{
	"mailbox_provider": {{"gmail", "Gmail"}, {"outlook", "Outlook"}, {"yahoo", "Yahoo"}, {"icloud", "iCloud"}, {"other", "Other"}},
	"tier":             {{"active", "Active"}, {"cooling", "Cooling"}, {"at_risk", "At risk"}, {"sunset", "Sunset"}},
	"consent":          {{"double", "Double opt-in"}, {"single", "Single opt-in"}, {"pending", "Imported, pending"}},
}

func customOps(typ string) []string {
	switch typ {
	case "number":
		return []string{"equals", "not_equals", "gt", "lt", "is_empty", "is_not_empty"}
	case "date":
		return []string{"before", "after", "is_empty", "is_not_empty"}
	case "boolean":
		return []string{"is_true", "is_false"}
	case "select":
		return []string{"equals", "not_equals", "is_empty", "is_not_empty"}
	}
	return []string{"equals", "not_equals", "contains", "is_empty", "is_not_empty"}
}

// ValidationError explains which condition is wrong, for a 422.
type ValidationError struct {
	Location string
	Message  string
}

func (e *ValidationError) Error() string { return e.Message }

// Validate checks every condition against the catalog. custom maps custom
// field keys to their definition; lists holds the workspace's list ids.
func (d Definition) Validate(custom map[string]CustomField, lists map[uuid.UUID]string) error {
	if len(d.Groups) > 20 {
		return &ValidationError{"body.definition.groups", "A segment can have at most 20 groups."}
	}
	for gi, g := range d.Groups {
		loc := fmt.Sprintf("body.definition.groups[%d]", gi)
		if g.Match != "all" && g.Match != "any" {
			return &ValidationError{loc + ".match", "Pick ALL or ANY for each group."}
		}
		if len(g.Conditions) == 0 {
			return &ValidationError{loc, fmt.Sprintf("Group %d has no conditions. Add one or remove the group.", gi+1)}
		}
		if len(g.Conditions) > 50 {
			return &ValidationError{loc, "A group can have at most 50 conditions."}
		}
		for ci, c := range g.Conditions {
			cloc := fmt.Sprintf("%s.conditions[%d]", loc, ci)
			if err := c.validate(custom, lists); err != nil {
				return &ValidationError{cloc, err.Error()}
			}
		}
	}
	return nil
}

func (c Condition) validate(custom map[string]CustomField, lists map[uuid.UUID]string) error {
	ops, kind, label := c.spec(custom)
	if ops == nil {
		return fmt.Errorf("%q is not a field you can filter on.", c.Field)
	}
	if !contains(ops, c.Op) {
		return fmt.Errorf("%s can't use the operator %q.", label, c.Op)
	}
	switch {
	case c.Op == "never" || c.Op == "is_empty" || c.Op == "is_not_empty" || c.Op == "is_true" || c.Op == "is_false":
		return nil
	case kind == KindPeriod:
		if c.Amount < 1 || c.Amount > 3650 {
			return fmt.Errorf("%s needs a number of days or weeks between 1 and 3650.", label)
		}
		if c.Unit != "days" && c.Unit != "weeks" {
			return fmt.Errorf("%s needs days or weeks.", label)
		}
	case kind == KindList:
		id, err := uuid.Parse(c.Value)
		if err != nil {
			return fmt.Errorf("Pick a list for %s.", strings.ToLower(label))
		}
		if _, ok := lists[id]; !ok {
			return fmt.Errorf("That list doesn't exist in this workspace any more.")
		}
	case kind == KindCampaign:
		if _, err := uuid.Parse(c.Value); err != nil {
			return fmt.Errorf("Pick a campaign.")
		}
	case kind == KindEnum && c.Field != "source":
		ok := false
		for _, v := range enumValues[c.Field] {
			if v[0] == c.Value {
				ok = true
			}
		}
		if !ok {
			return fmt.Errorf("Pick a value for %s.", strings.ToLower(label))
		}
	case kind == KindNumber:
		if _, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64); err != nil {
			return fmt.Errorf("%s needs a number.", label)
		}
	case kind == KindDate:
		if !dateRe.MatchString(strings.TrimSpace(c.Value)) {
			return fmt.Errorf("%s needs a date like 2026-10-30.", label)
		}
	default:
		if strings.TrimSpace(c.Value) == "" {
			return fmt.Errorf("Enter a value for %s.", strings.ToLower(label))
		}
	}
	return nil
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// spec returns the operators, kind and label of the condition's field.
func (c Condition) spec(custom map[string]CustomField) ([]string, FieldKind, string) {
	if key, ok := strings.CutPrefix(c.Field, "custom:"); ok {
		f, found := custom[key]
		if !found || !fieldKeyRe.MatchString(key) {
			return nil, "", ""
		}
		kind := KindText
		switch f.Type {
		case "number":
			kind = KindNumber
		case "date":
			kind = KindDate
		case "boolean":
			kind = KindBoolean
		}
		return customOps(f.Type), kind, "Custom field · " + key
	}
	s, ok := builtinFields[c.Field]
	if !ok {
		return nil, "", ""
	}
	return s.Ops, s.Kind, s.Label
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
