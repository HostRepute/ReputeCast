package segments

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The builder compiles to one SQL text against the segment views (contacts
// c, list_members lm, events e; see views.go). The same text is shown to
// power users ("Generated SQL", with values inlined) and executed (with
// bind parameters), so what you read is what runs.

// arg is a value bound to the query.
type arg struct {
	v       any
	literal string // how the value reads when inlined
}

// sqlText accumulates SQL with placeholders for bound values.
type sqlText struct {
	parts []string // literal SQL fragments; args go between them
	args  []arg
}

func (t *sqlText) raw(s string) { t.parts = append(t.parts, s) }

func (t *sqlText) bind(a arg) {
	t.args = append(t.args, a)
	t.parts = append(t.parts, "\x00")
}

// render joins the text, numbering placeholders from first (or inlining
// literals when inline is set).
func (t *sqlText) render(inline bool, first int) (string, []any) {
	var sb strings.Builder
	var args []any
	n := 0
	for _, p := range t.parts {
		if p != "\x00" {
			sb.WriteString(p)
			continue
		}
		a := t.args[n]
		n++
		if inline {
			sb.WriteString(a.literal)
			continue
		}
		args = append(args, a.v)
		fmt.Fprintf(&sb, "$%d", first+len(args)-1)
	}
	return sb.String(), args
}

func str(v string) arg { return arg{v: v, literal: quote(v)} }

func uid(v uuid.UUID) arg { return arg{v: v, literal: quote(v.String())} }

func num(v float64) arg {
	return arg{v: v, literal: strconv.FormatFloat(v, 'f', -1, 64)}
}

// quote renders a standard SQL string literal (standard_conforming_strings
// is on, so only the quote needs doubling).
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// escapeLike escapes LIKE wildcards in a value used as a pattern.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// Compiled is a builder segment rendered to SQL.
type Compiled struct {
	text sqlText
}

// Compile turns a validated definition into SQL over the segment views.
// The workspace is always the first bound value.
func Compile(d SegmentDefinition, workspaceID uuid.UUID, custom map[string]CustomField) Compiled {
	var t sqlText
	t.raw("SELECT c.id, c.email\nFROM contacts c\nWHERE c.workspace_id = ")
	t.bind(uid(workspaceID))
	for _, g := range d.Groups {
		if g.Match == "any" && len(g.Conditions) > 1 {
			t.raw("\n  AND (")
			for i, c := range g.Conditions {
				if i > 0 {
					t.raw(" OR ")
				}
				condition(&t, c, custom)
			}
			t.raw(")")
			continue
		}
		for _, c := range g.Conditions {
			t.raw("\n  AND ")
			condition(&t, c, custom)
		}
	}
	t.raw("\n  AND c.suppressed_at IS NULL")
	return Compiled{text: t}
}

// Display is the SQL shown under "Generated SQL", values inlined.
func (c Compiled) Display() string {
	s, _ := c.text.render(true, 1)
	return s + ";"
}

// Query returns the SQL with bind parameters numbered from first.
func (c Compiled) Query(first int) (string, []any) {
	return c.text.render(false, first)
}

const indent = "\n               "

func interval(c SegmentCondition) string {
	unit := c.Unit
	if unit == "" {
		unit = "days"
	}
	n := c.Amount
	if n < 1 {
		n = 1
	}
	return fmt.Sprintf("now() - interval '%d %s'", n, unit)
}

// condition renders one condition. Field keys of custom fields are checked
// against ^[a-z][a-z0-9_]*$ by Validate, so they're safe to inline.
func condition(t *sqlText, c SegmentCondition, custom map[string]CustomField) {
	switch c.Field {
	case "mailbox_provider", "tier", "consent", "source", "country":
		op := " = "
		if c.Op == "is_not" {
			op = " <> "
		}
		t.raw("c." + c.Field + op)
		t.bind(str(c.Value))
	case "email_domain":
		switch c.Op {
		case "ends_with":
			t.raw("split_part(c.email, '@', 2) LIKE ")
			t.bind(str("%" + escapeLike(strings.ToLower(strings.TrimSpace(c.Value)))))
		case "is_not":
			t.raw("split_part(c.email, '@', 2) <> ")
			t.bind(str(strings.ToLower(strings.TrimSpace(c.Value))))
		default:
			t.raw("split_part(c.email, '@', 2) = ")
			t.bind(str(strings.ToLower(strings.TrimSpace(c.Value))))
		}
	case "list":
		id, _ := uuid.Parse(c.Value)
		if c.Op == "excludes" {
			t.raw("NOT ")
		}
		t.raw("EXISTS (SELECT 1 FROM list_members lm WHERE lm.contact_id = c.id" + indent + "AND lm.list_id = ")
		t.bind(uid(id))
		t.raw(" AND lm.status = 'subscribed')")
	case "tag":
		if c.Op == "has_not" {
			t.raw("NOT ")
		}
		t.bind(str(strings.TrimSpace(c.Value)))
		t.raw(" = ANY (c.tags)")
	case "last_open":
		switch c.Op {
		case "never":
			t.raw("c.last_open_at IS NULL")
		case "not_in_last":
			t.raw("(c.last_open_at IS NULL OR c.last_open_at < " + interval(c) + ")")
		default:
			t.raw("c.last_open_at >= " + interval(c))
		}
	case "opened", "clicked":
		if c.Op == "not_in_last" {
			t.raw("NOT ")
		}
		t.raw("EXISTS (SELECT 1 FROM events e WHERE e.contact_id = c.id" + indent)
		if c.Field == "opened" {
			t.raw("AND e.type = 'open' AND e.human = true")
		} else {
			t.raw("AND e.type = 'click'")
		}
		t.raw(indent + "AND e.occurred_at >= " + interval(c) + ")")
	case "opened_campaign":
		id, _ := uuid.Parse(c.Value)
		if c.Op == "not_opened" {
			t.raw("NOT ")
		}
		t.raw("EXISTS (SELECT 1 FROM events e WHERE e.contact_id = c.id" + indent + "AND e.type = 'open' AND e.human = true AND e.campaign_id = ")
		t.bind(uid(id))
		t.raw(")")
	case "joined":
		if c.Op == "not_in_last" {
			t.raw("c.created_at < " + interval(c))
		} else {
			t.raw("c.created_at >= " + interval(c))
		}
	default:
		key := strings.TrimPrefix(c.Field, "custom:")
		customCondition(t, key, custom[key], c)
	}
}

func customCondition(t *sqlText, key string, f CustomField, c SegmentCondition) {
	text := "c.fields->>'" + key + "'"
	jsonv := "c.fields->'" + key + "'"
	switch c.Op {
	case "is_empty":
		t.raw("coalesce(" + text + ", '') = ''")
		return
	case "is_not_empty":
		t.raw("coalesce(" + text + ", '') <> ''")
		return
	case "is_true":
		t.raw(jsonv + " = 'true'::jsonb")
		return
	case "is_false":
		t.raw("coalesce(" + jsonv + ", 'false'::jsonb) = 'false'::jsonb")
		return
	}
	switch f.Type {
	case "number":
		v, _ := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		// A value that isn't a number never matches instead of failing the count.
		expr := "CASE WHEN jsonb_typeof(" + jsonv + ") = 'number' THEN (" + text + ")::numeric END"
		switch c.Op {
		case "gt":
			t.raw(expr + " > ")
		case "lt":
			t.raw(expr + " < ")
		case "not_equals":
			t.raw(expr + " IS DISTINCT FROM ")
		default:
			t.raw(expr + " = ")
		}
		t.bind(num(v))
	case "date":
		expr := "CASE WHEN " + text + " ~ '^\\d{4}-\\d{2}-\\d{2}' THEN left(" + text + ", 10)::date END"
		if c.Op == "before" {
			t.raw(expr + " < ")
		} else {
			t.raw(expr + " > ")
		}
		v := strings.TrimSpace(c.Value)
		d, _ := time.Parse("2006-01-02", v)
		t.bind(arg{v: d, literal: quote(v)})
		t.raw("::date")
	default:
		switch c.Op {
		case "not_equals":
			t.raw(text + " IS DISTINCT FROM ")
			t.bind(str(c.Value))
		case "contains":
			t.raw(text + " ILIKE ")
			t.bind(str("%" + escapeLike(c.Value) + "%"))
		default:
			t.raw(text + " = ")
			t.bind(str(c.Value))
		}
	}
}
