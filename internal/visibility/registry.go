package visibility

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Groups of the condition picker, in its order. Later phases add theirs
// (paid tiers get a group of their own; the score joins Engagement).
const (
	GroupLists      = "lists"
	GroupContact    = "contact"
	GroupEngagement = "engagement"
)

// Built-in condition kinds.
const (
	KindSegment  = "segment"
	KindList     = "list"
	KindTag      = "tag"
	KindField    = "field"
	KindLanguage = "language"
	KindTier     = "tier"
)

// Facts is what conditions read about one contact. Contacts are read in
// bulk once per count or send, never one query per contact.
type Facts struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	// Lists the contact is subscribed to, and the segments (of those the
	// rules refer to) they are in, by id.
	Lists    map[string]bool
	Segments map[string]bool
	Tags     map[string]bool
	// Attributes are the custom fields.
	Attributes map[string]any
	// Language is the contact's language code as stored ("es-MX").
	Language string
	Tier     string
	// Extra is what registered kinds load for themselves (Kind.Load), by
	// kind key: the score and band of phase 4.
	Extra map[string]any
}

// Name is how "Show why" names the contact: their name, else the address.
func (f *Facts) Name() string {
	n := strings.TrimSpace(f.FirstName + " " + f.LastName)
	if n == "" {
		return f.Email
	}
	return n
}

// Ref is what a condition refers to: a segment, a list or a custom field
// (by id or key). A condition whose reference is gone is broken: Send
// Readiness blocks until someone fixes it.
type Ref struct {
	Kind string // segment, list, field, tag
	ID   string
}

// Operator is one way a kind compares; NoValue operators ("is empty") take
// no value.
type Operator struct {
	Key     string `json:"key"`
	NoValue bool   `json:"noValue,omitempty"`
}

// Kind is a condition the picker offers. Built-in kinds are registered in
// this package; later phases register theirs (RegisterKind) from their own.
type Kind struct {
	Key   string
	Group string
	// Ops are the operators; field conditions take their custom field's
	// operators (FieldOps) instead.
	Ops []Operator
	// Ref names what the condition refers to (zero Ref for none).
	Ref func(c VisibilityCondition) Ref
	// Match decides the condition for a contact.
	Match func(c VisibilityCondition, f *Facts) bool
	// Load reads what Match needs beyond the base facts, for the given
	// contacts, into Facts.Extra[Key]. Nil for built-in kinds.
	Load func(ctx context.Context, q *store.Queries, workspaceID uuid.UUID, conditions []VisibilityCondition, facts map[uuid.UUID]*Facts) error
	// Describe says the condition in words for badges and summaries
	// ("Beta Program list", "plan is empty").
	Describe func(c VisibilityCondition, n *Names) string
	// Why says why a contact doesn't get a block because of this condition
	// ("Sam Rivera isn't on Beta Program"). matched is the condition's result.
	Why func(c VisibilityCondition, f *Facts, matched bool, n *Names) string
}

var (
	registryMu sync.RWMutex
	registry   = map[string]*Kind{}
	order      []string
)

// RegisterKind adds a condition kind. Later phases call it from their own
// package's init (phase 4's score and band, phase 6's paid tier).
func RegisterKind(k Kind) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[k.Key]; !ok {
		order = append(order, k.Key)
	}
	kk := k
	registry[k.Key] = &kk
}

// Lookup returns a registered kind.
func Lookup(key string) (*Kind, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	k, ok := registry[key]
	return k, ok
}

// Kinds lists the registered kinds in registration order.
func Kinds() []*Kind {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Kind, 0, len(order))
	for _, k := range order {
		out = append(out, registry[k])
	}
	return out
}

// FieldOps are the operators of a custom field by its type: the segment
// builder's.
func FieldOps(typ string) []Operator {
	switch typ {
	case "number":
		return []Operator{{Key: "equals"}, {Key: "not_equals"}, {Key: "gt"}, {Key: "lt"}, {Key: "is_empty", NoValue: true}, {Key: "is_not_empty", NoValue: true}}
	case "date":
		return []Operator{{Key: "before"}, {Key: "after"}, {Key: "is_empty", NoValue: true}, {Key: "is_not_empty", NoValue: true}}
	case "boolean":
		return []Operator{{Key: "is_true", NoValue: true}, {Key: "is_false", NoValue: true}}
	case "select":
		return []Operator{{Key: "equals"}, {Key: "not_equals"}, {Key: "is_empty", NoValue: true}, {Key: "is_not_empty", NoValue: true}}
	}
	return []Operator{{Key: "equals"}, {Key: "not_equals"}, {Key: "contains"}, {Key: "is_empty", NoValue: true}, {Key: "is_not_empty", NoValue: true}}
}

// Tiers are the engagement tiers in the picker's order.
var Tiers = []string{"active", "cooling", "at_risk", "sunset"}

func init() {
	RegisterKind(Kind{Key: KindSegment, Group: GroupLists, Ops: []Operator{{Key: "in"}, {Key: "not_in"}},
		Ref: func(c VisibilityCondition) Ref { return Ref{Kind: "segment", ID: c.Value} },
		Match: func(c VisibilityCondition, f *Facts) bool {
			in := f.Segments[c.Value]
			if c.Op == "not_in" {
				return !in
			}
			return in
		},
		Describe: describeSegment, Why: whySegment})
	RegisterKind(Kind{Key: KindList, Group: GroupLists, Ops: []Operator{{Key: "includes"}, {Key: "excludes"}},
		Ref: func(c VisibilityCondition) Ref { return Ref{Kind: "list", ID: c.Value} },
		Match: func(c VisibilityCondition, f *Facts) bool {
			on := f.Lists[c.Value]
			if c.Op == "excludes" {
				return !on
			}
			return on
		},
		Describe: describeList, Why: whyList})
	RegisterKind(Kind{Key: KindTag, Group: GroupLists, Ops: []Operator{{Key: "has"}, {Key: "has_not"}},
		Ref: func(c VisibilityCondition) Ref { return Ref{Kind: "tag", ID: c.Value} },
		Match: func(c VisibilityCondition, f *Facts) bool {
			has := f.Tags[strings.TrimSpace(c.Value)]
			if c.Op == "has_not" {
				return !has
			}
			return has
		},
		Describe: describeTag, Why: whyTag})
	RegisterKind(Kind{Key: KindField, Group: GroupContact,
		Ref: func(c VisibilityCondition) Ref { return Ref{Kind: "field", ID: c.Field} },
		Match: func(c VisibilityCondition, f *Facts) bool {
			return fieldMatches(c, f.Attributes[c.Field])
		},
		Describe: describeField, Why: whyField})
	RegisterKind(Kind{Key: KindLanguage, Group: GroupContact, Ops: []Operator{{Key: "is"}, {Key: "is_not"}},
		Match: func(c VisibilityCondition, f *Facts) bool {
			is := baseLanguage(f.Language) == baseLanguage(c.Value) && c.Value != ""
			if c.Op == "is_not" {
				return !is
			}
			return is
		},
		Describe: describeLanguage, Why: whyLanguage})
	RegisterKind(Kind{Key: KindTier, Group: GroupEngagement, Ops: []Operator{{Key: "is"}, {Key: "is_not"}},
		Match: func(c VisibilityCondition, f *Facts) bool {
			is := f.Tier == c.Value
			if c.Op == "is_not" {
				return !is
			}
			return is
		},
		Describe: describeTier, Why: whyTier})
}

// baseLanguage is the language part of a code: "es-MX" → "es".
func baseLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.IndexAny(code, "-_"); i > 0 {
		code = code[:i]
	}
	return code
}

// fieldMatches compares a custom field's value as the segment builder does:
// text equality is exact, contains ignores case, numbers and dates compare
// as such and a value of another type never matches.
func fieldMatches(c VisibilityCondition, v any) bool {
	text := fieldText(v)
	switch c.Op {
	case "is_empty":
		return text == ""
	case "is_not_empty":
		return text != ""
	case "is_true":
		b, ok := v.(bool)
		return ok && b
	case "is_false":
		b, ok := v.(bool)
		return !ok || !b
	case "gt", "lt":
		n, ok := number(v)
		want, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		if !ok || err != nil {
			return false
		}
		if c.Op == "gt" {
			return n > want
		}
		return n < want
	case "before", "after":
		d, ok := date(text)
		want, err := time.Parse("2006-01-02", strings.TrimSpace(c.Value))
		if !ok || err != nil {
			return false
		}
		if c.Op == "before" {
			return d.Before(want)
		}
		return d.After(want)
	case "contains":
		return c.Value != "" && strings.Contains(strings.ToLower(text), strings.ToLower(c.Value))
	case "not_equals":
		if n, ok := number(v); ok {
			want, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
			return err != nil || n != want
		}
		return text != c.Value
	}
	// equals
	if n, ok := number(v); ok {
		want, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		return err == nil && n == want
	}
	return text == c.Value && text != ""
}

func fieldText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}

func date(s string) (time.Time, bool) {
	if len(s) < 10 {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", s[:10])
	return d, err == nil
}

// Validate checks a rule against the workspace: known kinds and operators,
// values where the operator needs one. It doesn't check references (a
// deleted segment is a broken condition, reported by Broken).
func Validate(r VisibilityRule, n *Names) error {
	if r.Show != ShowOnly && r.Show != ShowExcept {
		return nil
	}
	for i, c := range r.Conditions {
		k, ok := Lookup(c.Kind)
		if !ok {
			return fmt.Errorf("condition %d: unknown kind %q", i+1, c.Kind)
		}
		ops := k.Ops
		if c.Kind == KindField {
			if c.Field == "" {
				return fmt.Errorf("condition %d: pick the field", i+1)
			}
			ops = FieldOps(n.fieldType(c.Field))
		}
		idx := slices.IndexFunc(ops, func(o Operator) bool { return o.Key == c.Op })
		if idx < 0 {
			return fmt.Errorf("condition %d: %q can't use the operator %q", i+1, c.Kind, c.Op)
		}
		if !ops[idx].NoValue && strings.TrimSpace(c.Value) == "" {
			return fmt.Errorf("condition %d: pick a value", i+1)
		}
	}
	return nil
}
