package audience

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The contacts table pages with keyset cursors (review 2026-11 AU-1): a
// page starts after (or before) the row a cursor names, so page 400 costs
// what page 1 does. Each order is a list of keys and the contact's id
// breaks ties.

// sortKey is one column of an order.
type sortKey struct {
	expr string
	desc bool
	// nullable keys sort their NULLs last.
	nullable bool
	kind     byte // 't' time, 'i' integer, 's' text
}

// contactSorts are the orders of the contacts table (AU-9), by name.
var contactSorts = map[string][]sortKey{
	"activity":  {{expr: "c.last_activity_at", desc: true, nullable: true, kind: 't'}},
	"score":     {{expr: "cs.score", desc: true, nullable: true, kind: 'i'}, {expr: "c.last_activity_at", desc: true, nullable: true, kind: 't'}},
	"joined":    {{expr: "c.created_at", desc: true, kind: 't'}},
	"last_open": {{expr: "c.last_open_at", desc: true, nullable: true, kind: 't'}},
	"email":     {{expr: "c.email", kind: 's'}},
}

// contactSort names a known order: the default is last activity.
func contactSort(name string) (string, []sortKey) {
	if keys, ok := contactSorts[name]; ok {
		return name, keys
	}
	return "activity", contactSorts["activity"]
}

// orderBy is the ORDER BY of keys, or its reverse, with idExpr breaking ties.
func orderBy(keys []sortKey, idExpr string, reverse bool) string {
	parts := make([]string, 0, len(keys)+1)
	for _, k := range keys {
		desc := k.desc != reverse
		dir := " ASC"
		if desc {
			dir = " DESC"
		}
		if k.nullable {
			// NULLs come last in the order, so first in its reverse.
			if reverse {
				dir += " NULLS FIRST"
			} else {
				dir += " NULLS LAST"
			}
		}
		parts = append(parts, k.expr+dir)
	}
	idDesc := keys[0].desc != reverse
	if idDesc {
		parts = append(parts, idExpr+" DESC")
	} else {
		parts = append(parts, idExpr+" ASC")
	}
	return strings.Join(parts, ", ")
}

// contactCursor is where a page starts: after (or before) a row, in one
// order.
type contactCursor struct {
	Sort   string    `json:"s"`
	Before bool      `json:"b,omitempty"`
	Values []any     `json:"v"`
	ID     uuid.UUID `json:"i"`
}

var errCursor = errors.New("audience: unreadable cursor")

func (c contactCursor) encode() string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

// readCursor decodes a cursor of the order sort.
func readCursor(s, sort string, keys []sortKey) (contactCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return contactCursor{}, errCursor
	}
	var c contactCursor
	if json.Unmarshal(b, &c) != nil || c.Sort != sort || len(c.Values) != len(keys) || c.ID == uuid.Nil {
		return contactCursor{}, errCursor
	}
	for i, k := range keys {
		v := c.Values[i]
		if v == nil {
			if !k.nullable {
				return contactCursor{}, errCursor
			}
			continue
		}
		switch k.kind {
		case 't':
			str, ok := v.(string)
			t, err := time.Parse(time.RFC3339Nano, str)
			if !ok || err != nil {
				return contactCursor{}, errCursor
			}
			c.Values[i] = t
		case 'i':
			n, ok := v.(float64)
			if !ok {
				return contactCursor{}, errCursor
			}
			c.Values[i] = int64(n)
		case 's':
			if _, ok := v.(string); !ok {
				return contactCursor{}, errCursor
			}
		}
	}
	return c, nil
}

// cursorValue is a scanned key as a cursor holds it.
func cursorValue(v any) any {
	switch x := v.(type) {
	case *time.Time:
		if x == nil {
			return nil
		}
		return x.UTC().Format(time.RFC3339Nano)
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case *int32:
		if x == nil {
			return nil
		}
		return int64(*x)
	case *string:
		if x == nil {
			return nil
		}
		return *x
	}
	return v
}

// keyset is the condition for the rows beyond c in its direction, its
// values bound through bind (which returns the placeholder).
func keyset(keys []sortKey, idExpr string, c contactCursor, bind func(any) string) string {
	forward := !c.Before
	var at func(i int) string
	at = func(i int) string {
		if i == len(keys) {
			op := "<"
			if (keys[0].desc) != forward {
				op = ">"
			}
			return fmt.Sprintf("%s %s %s", idExpr, op, bind(c.ID))
		}
		k, v := keys[i], c.Values[i]
		var beyond, equal string
		if v == nil {
			equal = k.expr + " IS NULL"
			if forward {
				beyond = "false" // NULLs are last: only ties follow
			} else {
				beyond = k.expr + " IS NOT NULL"
			}
		} else {
			p := bind(v)
			equal = k.expr + " = " + p
			after := "<"
			if !k.desc {
				after = ">"
			}
			before := map[string]string{"<": ">", ">": "<"}[after]
			if forward {
				beyond = k.expr + " " + after + " " + p
				if k.nullable {
					beyond = "(" + beyond + " OR " + k.expr + " IS NULL)"
				}
			} else {
				beyond = k.expr + " " + before + " " + p
			}
		}
		return "(" + beyond + " OR (" + equal + " AND " + at(i+1) + "))"
	}
	return at(0)
}
