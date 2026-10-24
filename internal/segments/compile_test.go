package segments

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var (
	wsID   = uuid.MustParse("0199c0de-7f1a-7c3e-9b2a-1f2e3d4c5b6a")
	digest = uuid.MustParse("0199c0de-0000-7000-8000-000000000001")
	custom = map[string]CustomField{
		"plan":      {Key: "plan", Label: "Plan", Type: "select"},
		"mrr":       {Key: "mrr", Label: "MRR", Type: "number"},
		"trial_end": {Key: "trial_end", Label: "Trial end", Type: "date"},
	}
	lists = map[uuid.UUID]string{digest: "Weekly Digest"}
)

// The board's example: Engaged Outlook users on paid plans.
func boardDefinition() SegmentDefinition {
	return SegmentDefinition{Groups: []SegmentGroup{
		{Match: "all", Conditions: []SegmentCondition{
			{Field: "mailbox_provider", Op: "is", Value: "outlook"},
			{Field: "opened", Op: "in_last", Amount: 30, Unit: "days"},
			{Field: "list", Op: "includes", Value: digest.String()},
		}},
		{Match: "any", Conditions: []SegmentCondition{
			{Field: "custom:plan", Op: "equals", Value: "team"},
			{Field: "custom:plan", Op: "equals", Value: "business"},
		}},
	}}
}

func TestCompileDisplayMatchesBoard(t *testing.T) {
	d := boardDefinition()
	if err := d.Validate(custom, lists); err != nil {
		t.Fatal(err)
	}
	got := Compile(d, wsID, custom).Display()
	want := `SELECT c.id, c.email
FROM contacts c
WHERE c.workspace_id = '0199c0de-7f1a-7c3e-9b2a-1f2e3d4c5b6a'
  AND c.mailbox_provider = 'outlook'
  AND EXISTS (SELECT 1 FROM events e WHERE e.contact_id = c.id
               AND e.type = 'open' AND e.human = true
               AND e.occurred_at >= now() - interval '30 days')
  AND EXISTS (SELECT 1 FROM list_members lm WHERE lm.contact_id = c.id
               AND lm.list_id = '0199c0de-0000-7000-8000-000000000001' AND lm.status = 'subscribed')
  AND (c.fields->>'plan' = 'team' OR c.fields->>'plan' = 'business')
  AND c.suppressed_at IS NULL;`
	if got != want {
		t.Fatalf("display SQL:\n%s\nwant:\n%s", got, want)
	}
}

func TestCompileQueryBindsValues(t *testing.T) {
	d := boardDefinition()
	d.Groups = append(d.Groups, SegmentGroup{Match: "all", Conditions: []SegmentCondition{
		{Field: "tag", Op: "has_not", Value: "churned'; DROP TABLE contacts; --"},
		{Field: "custom:mrr", Op: "gt", Value: "50"},
		{Field: "custom:trial_end", Op: "before", Value: "2026-11-01"},
		{Field: "email_domain", Op: "ends_with", Value: "acme_io%"},
	}})
	if err := d.Validate(custom, lists); err != nil {
		t.Fatal(err)
	}
	q, args := Compile(d, wsID, custom).Query(2)
	if strings.Contains(q, "DROP TABLE") || strings.Contains(q, "'team'") {
		t.Fatalf("values must be bound, not inlined:\n%s", q)
	}
	if !strings.Contains(q, "c.workspace_id = $2") || !strings.Contains(q, "NOT $7 = ANY (c.tags)") || !strings.Contains(q, "< $9::date") {
		t.Fatalf("placeholders:\n%s", q)
	}
	if len(args) != 9 || args[0] != wsID || args[5] != "churned'; DROP TABLE contacts; --" || args[6] != float64(50) {
		t.Fatalf("args: %#v", args)
	}
	if args[8] != `%acme\_io\%` {
		t.Fatalf("LIKE pattern must escape wildcards: %v", args[8])
	}
	disp := Compile(d, wsID, custom).Display()
	if !strings.Contains(disp, `'churned''; DROP TABLE contacts; --'`) {
		t.Fatalf("display must quote safely:\n%s", disp)
	}
}

// Compile never inlines a stored definition's period unit or field key, even
// when the definition skipped Validate: those are the two places it writes
// condition text into the SQL rather than binding it.
func TestCompileUnvalidatedDefinition(t *testing.T) {
	const payload = "days' OR c.id IN (SELECT id FROM public.contacts) OR 'x' = 'x"
	d := SegmentDefinition{Groups: []SegmentGroup{{Match: "any", Conditions: []SegmentCondition{
		{Field: "joined", Op: "in_last", Amount: 1, Unit: payload},
		{Field: "opened", Op: "not_in_last", Amount: 2, Unit: "weeks"},
		{Field: "custom:x' OR '1'='1", Op: "equals", Value: "y"},
		{Field: "x' OR '1'='1", Op: "equals", Value: "y"},
		{Field: "custom:plan", Op: "equals", Value: "team"},
	}}}}
	q, _ := Compile(d, wsID, custom).Query(2)
	for _, bad := range []string{"public.contacts", "'1'='1", "OR 'x'"} {
		if strings.Contains(q, bad) {
			t.Fatalf("inlined %q:\n%s", bad, q)
		}
	}
	for _, want := range []string{"c.created_at >= now() - interval '1 days'", "interval '2 weeks'", " OR false OR false OR ", "c.fields->>'plan' = $"} {
		if !strings.Contains(q, want) {
			t.Fatalf("missing %q:\n%s", want, q)
		}
	}
}

func TestValidate(t *testing.T) {
	bad := []SegmentDefinition{
		{Groups: []SegmentGroup{{Match: "some", Conditions: []SegmentCondition{{Field: "tier", Op: "is", Value: "active"}}}}},
		{Groups: []SegmentGroup{{Match: "all"}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "nope", Op: "is", Value: "x"}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "tier", Op: "gt", Value: "active"}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "tier", Op: "is", Value: "gold"}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "opened", Op: "in_last", Amount: 0, Unit: "days"}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "list", Op: "includes", Value: uuid.NewString()}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "custom:mrr", Op: "gt", Value: "lots"}}}}},
		{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{{Field: "custom:evil'key", Op: "equals", Value: "x"}}}}},
	}
	for i, d := range bad {
		var ve *ValidationError
		if err := d.Validate(custom, lists); !errors.As(err, &ve) {
			t.Errorf("case %d: want a validation error, got %v", i, err)
		}
	}
	if err := (SegmentDefinition{}).Validate(custom, lists); err != nil {
		t.Fatalf("an empty definition matches everyone: %v", err)
	}
}

func TestCheckSQL(t *testing.T) {
	ok := map[string]string{
		"SELECT c.id FROM contacts c;":                               "SELECT c.id FROM contacts c",
		"  with x as (select id from contacts) select id from x ;; ": "with x as (select id from contacts) select id from x",
		"(SELECT id FROM contacts)":                                  "(SELECT id FROM contacts)",
		"SELECT id FROM contacts WHERE email = 'a;b' -- ;\n":         "SELECT id FROM contacts WHERE email = 'a;b' -- ;",
		"SELECT id FROM contacts WHERE note = $$x;y$$":               "SELECT id FROM contacts WHERE note = $$x;y$$",
		"SELECT id /* ; /* nested ; */ */ FROM contacts":             "SELECT id /* ; /* nested ; */ */ FROM contacts",
		`SELECT "id" FROM contacts WHERE email = E'it\'s;'`:          `SELECT "id" FROM contacts WHERE email = E'it\'s;'`,
	}
	for in, want := range ok {
		got, err := CheckSQL(in)
		if err != nil || got != want {
			t.Errorf("CheckSQL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{
		"", "DELETE FROM contacts", "SELECT 1; DROP TABLE contacts", "UPDATE contacts SET tier = 'x'",
		"SELECT 'unterminated", "SELECT /* open", "SET ROLE reputecast; SELECT 1", strings.Repeat("x", MaxSQLLength+1),
	} {
		var se *SQLError
		if _, err := CheckSQL(in); !errors.As(err, &se) {
			t.Errorf("CheckSQL(%.40q): want an SQL error, got %v", in, err)
		}
	}
}
