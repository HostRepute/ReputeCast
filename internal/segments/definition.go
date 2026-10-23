package segments

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// SegmentDefinition is a builder segment: groups combine with AND; inside a group
// the conditions match ALL or ANY.
type SegmentDefinition struct {
	Groups []SegmentGroup `json:"groups"`
}

// SegmentGroup is one box of conditions in the builder.
type SegmentGroup struct {
	Match      string             `json:"match" enum:"all,any" doc:"all = every condition, any = at least one"`
	Conditions []SegmentCondition `json:"conditions"`
}

// SegmentCondition is one row of the builder: field, operator and value.
type SegmentCondition struct {
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
	// To ends a range ("Score is between Value and To").
	To string `json:"to,omitempty" maxLength:"20"`
}

// A field's name, an operator and a fixed value read as messages/<locale>.json
// says: field.<key>, op.<key> and option.<field>.<value> (lang.go).

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

// fieldSpec describes a built-in field; its name is field.<key>.
type fieldSpec struct {
	Kind FieldKind
	Ops  []string
}

var builtinFields = map[string]fieldSpec{
	"mailbox_provider": {KindEnum, []string{"is", "is_not"}},
	"email_domain":     {KindText, []string{"is", "is_not", "ends_with"}},
	"list":             {KindList, []string{"includes", "excludes"}},
	"tier":             {KindEnum, []string{"is", "is_not"}},
	"consent":          {KindEnum, []string{"is", "is_not"}},
	"source":           {KindEnum, []string{"is", "is_not"}},
	"tag":              {KindText, []string{"has", "has_not"}},
	"country":          {KindText, []string{"is", "is_not"}},
	"language":         {KindEnum, []string{"is", "is_not"}},
	"last_open":        {KindPeriod, []string{"in_last", "not_in_last", "never"}},
	"opened":           {KindPeriod, []string{"in_last", "not_in_last"}},
	"clicked":          {KindPeriod, []string{"in_last", "not_in_last"}},
	"opened_campaign":  {KindCampaign, []string{"opened", "not_opened"}},
	"joined":           {KindPeriod, []string{"in_last", "not_in_last"}},
}

// builtinOrder is the order of fields in the builder's select.
var builtinOrder = []string{
	"mailbox_provider", "email_domain", "list", "tier", "last_open", "opened", "clicked",
	"opened_campaign", "consent", "source", "tag", "country", "language", "joined",
}

// Enum values of the built-in enum fields: the value and, for a name that
// reads the same in every language, the name; the others read as
// option.<field>.<value> says.
var enumValues = map[string][][2]string{
	"mailbox_provider": {{"gmail", "Gmail"}, {"outlook", "Outlook"}, {"yahoo", "Yahoo"}, {"icloud", "iCloud"}, {"other", ""}},
	"tier":             {{"active", ""}, {"cooling", ""}, {"at_risk", ""}, {"sunset", ""}},
	"consent":          {{"double", ""}, {"single", ""}, {"pending", ""}},
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
		return []string{"equals", "not_equals", "is_one_of", "is_empty", "is_not_empty"}
	}
	return []string{"equals", "not_equals", "is_one_of", "contains", "is_empty", "is_not_empty"}
}

// ValidationError explains which condition is wrong, for a 422. Message
// is the English; Text writes it in a reader's language.
type ValidationError struct {
	Location string
	Message  string
	problem  error
}

func (e *ValidationError) Error() string { return e.Message }

// Text is the problem in a reader's language.
func (e *ValidationError) Text(l i18n.Lang) string {
	if e.problem == nil {
		return e.Message
	}
	return textOf(l, e.problem)
}

// invalid is a ValidationError at a location.
func invalid(location string, p error) *ValidationError {
	return &ValidationError{Location: location, Message: p.Error(), problem: p}
}

// Validate checks every condition against the catalog. custom maps custom
// field keys to their definition; lists holds the workspace's list ids.
func (d SegmentDefinition) Validate(custom map[string]CustomField, lists map[uuid.UUID]string) error {
	if len(d.Groups) > 20 {
		return invalid("body.definition.groups", says(mTooManyGrps))
	}
	for gi, g := range d.Groups {
		loc := fmt.Sprintf("body.definition.groups[%d]", gi)
		if g.Match != "all" && g.Match != "any" {
			return invalid(loc+".match", says(mMatch))
		}
		if len(g.Conditions) == 0 {
			return invalid(loc, sayf(mEmptyGroup, i18n.Args{"n": strconv.Itoa(gi + 1)}))
		}
		if len(g.Conditions) > 50 {
			return invalid(loc, says(mTooManyCond))
		}
		for ci, c := range g.Conditions {
			cloc := fmt.Sprintf("%s.conditions[%d]", loc, ci)
			if err := c.validate(custom, lists); err != nil {
				return invalid(cloc, err)
			}
		}
	}
	return nil
}

func (c SegmentCondition) validate(custom map[string]CustomField, lists map[uuid.UUID]string) error {
	ops, kind := c.spec(custom)
	if ops == nil {
		return sayf(mUnknownFld, i18n.Args{"field": strconv.Quote(c.Field)})
	}
	if !contains(ops, c.Op) {
		return sayField(mUnknownOp, c.Field, false, i18n.Args{"op": strconv.Quote(c.Op)})
	}
	if v, ok := extraValidate[c.Field]; ok {
		return v(c)
	}
	switch {
	case c.Op == "never" || c.Op == "is_empty" || c.Op == "is_not_empty" || c.Op == "is_true" || c.Op == "is_false":
		return nil
	case kind == KindPeriod:
		if c.Amount < 1 || c.Amount > 3650 {
			return sayField(mPeriodRange, c.Field, false, nil)
		}
		if c.Unit != "days" && c.Unit != "weeks" {
			return sayField(mPeriodUnit, c.Field, false, nil)
		}
	case kind == KindList:
		id, err := uuid.Parse(c.Value)
		if err != nil {
			return sayField(mPickList, c.Field, true, nil)
		}
		if _, ok := lists[id]; !ok {
			return says(mListGone)
		}
	case kind == KindCampaign:
		if _, err := uuid.Parse(c.Value); err != nil {
			return says(mPickCamp)
		}
	case c.Field == "language":
		if !validLanguage(c.Value) {
			return sayField(mPickLang, c.Field, true, nil)
		}
	case kind == KindEnum && c.Field != "source":
		ok := false
		for _, v := range enumValues[c.Field] {
			if v[0] == c.Value {
				ok = true
			}
		}
		if !ok {
			return sayField(mPickValue, c.Field, true, nil)
		}
	case kind == KindNumber:
		if _, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64); err != nil {
			return sayField(mNumber, c.Field, false, nil)
		}
	case kind == KindDate:
		if !dateRe.MatchString(strings.TrimSpace(c.Value)) {
			return sayField(mDate, c.Field, false, nil)
		}
	default:
		if strings.TrimSpace(c.Value) == "" {
			return sayField(mEnterValue, c.Field, true, nil)
		}
	}
	return nil
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// spec returns the operators and kind of the condition's field.
func (c SegmentCondition) spec(custom map[string]CustomField) ([]string, FieldKind) {
	if key, ok := strings.CutPrefix(c.Field, "custom:"); ok {
		f, found := custom[key]
		if !found || !fieldKeyRe.MatchString(key) {
			return nil, ""
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
		return customOps(f.Type), kind
	}
	s, ok := builtinFields[c.Field]
	if !ok {
		return nil, ""
	}
	return s.Ops, s.Kind
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
