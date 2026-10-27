package campaigns_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mailer"
)

type testMailbox struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *testMailbox) SendSystem(_ context.Context, msg mailer.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return "test", nil
}

func (m *testMailbox) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

const personalizedDoc = `<mjml><mj-body><mj-section><mj-column>` +
	`<mj-text>Dear {{ first_name | default: "there" }} {{ last_name }}, you are on {{ plan | default: "free" }}. Signed, {{ frist_name | default: "friend" }}.</mj-text>` +
	`<mj-text><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text>` +
	`</mj-column></mj-section></mj-body></mjml>`

// CO-5: previews and test sends fill {{ first_name }} and {{ last_name }}
// from the contact's own fields (a full name split again gave "Hi Nguyen"
// to someone with no first name, and "Mary" to Mary Ann), and the counts
// of contacts without a value are real, per field the email uses.
func TestComposerPreviewUsesTheContactsOwnFields(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	box := &testMailbox{}
	env.App.Mail = box
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	list, excluded := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $3, 'CO-5 readers'), ($2, $3, 'CO-5 excluded')`, list, excluded, wsID)
	contact := func(email, first, last, attrs string, lists ...uuid.UUID) string {
		id := uuid.Must(uuid.NewV7())
		exec(`INSERT INTO contacts (id, workspace_id, email, first_name, last_name, attributes) VALUES ($1, $2, $3, $4, $5, $6)`, id, wsID, email, first, last, attrs)
		for _, l := range lists {
			exec(`INSERT INTO list_memberships (list_id, contact_id) VALUES ($1, $2)`, l, id)
		}
		return id.String()
	}
	nguyen := contact("nguyen@example.com", "", "Nguyen", `{"plan":"team"}`, list)
	maryAnn := contact("maryann@example.com", "Mary Ann", "Smith", `{}`, list)
	contact("lee@example.com", "Lee", "", `{"plan":"free"}`, list)
	// Excluded: not counted, though it lacks every field.
	contact("gone@example.com", "", "", `{}`, list, excluded)

	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Personal"}).MustStatus(t, 201).JSON(t, &d)
	c.Do("PATCH", ws+"/campaigns/"+d.ID, map[string]any{"audience": map[string]any{"lists": []string{list.String()}, "excludeLists": []string{excluded.String()}}}).MustStatus(t, 200)
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": `Hi {{ first_name | default: "there" }}`, "mjml": personalizedDoc}).MustStatus(t, 200)

	type token struct {
		Key, Value, Suggestion         string
		Missing, FallbackUsed, Unknown bool
		AudienceMissing                *int64
	}
	type preview struct {
		Subject, HTML    string
		Contact          struct{ FirstName, LastName string }
		Tokens           []token
		Counted          *int64
		CountedListsOnly bool
		MissingFirstName *int64
	}
	var pv preview
	c.Do("POST", ws+"/campaigns/"+d.ID+"/preview", map[string]any{"contactId": nguyen}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Subject != "Hi there" || !strings.Contains(pv.HTML, "Dear there Nguyen,") {
		t.Fatalf("no first name, last name Nguyen: %q / %s", pv.Subject, between(pv.HTML, "Dear", "."))
	}
	if pv.Contact.FirstName != "" || pv.Contact.LastName != "Nguyen" {
		t.Fatalf("contact fields: %+v", pv.Contact)
	}
	tokens := map[string]token{}
	for _, tk := range pv.Tokens {
		tokens[tk.Key] = tk
	}
	if tk := tokens["first_name"]; !tk.Missing || !tk.FallbackUsed || tk.Value != "there" {
		t.Fatalf("first_name: %+v", tk)
	}
	if tk := tokens["last_name"]; tk.Missing || tk.Value != "Nguyen" {
		t.Fatalf("last_name: %+v", tk)
	}
	if tk := tokens["frist_name"]; !tk.Unknown || tk.Suggestion != "first_name" || tk.AudienceMissing != nil {
		t.Fatalf("a mistyped field is unknown even with a fallback: %+v", tk)
	}
	// Counted over the list minus the excluded list: one of three lacks each field.
	if pv.Counted == nil || *pv.Counted != 3 || pv.CountedListsOnly {
		t.Fatalf("counted: %v %v", pv.Counted, pv.CountedListsOnly)
	}
	for _, k := range []string{"first_name", "last_name", "plan"} {
		if n := tokens[k].AudienceMissing; n == nil || *n != 1 {
			t.Fatalf("%s missing for %v contacts, want 1", k, n)
		}
	}
	if pv.MissingFirstName == nil || *pv.MissingFirstName != 1 {
		t.Fatalf("missingFirstName: %v", pv.MissingFirstName)
	}

	// A first name with a space stays whole.
	c.Do("POST", ws+"/campaigns/"+d.ID+"/preview", map[string]any{"contactId": maryAnn}).MustStatus(t, 200).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Dear Mary Ann Smith,") {
		t.Fatalf("Mary Ann: %s", between(pv.HTML, "Dear", "."))
	}

	// A test send personalized as a contact says the same.
	c.Do("POST", ws+"/campaigns/"+d.ID+"/test", map[string]any{"to": []string{"linh@acme.io"}, "contactId": nguyen}).MustStatus(t, 200)
	box.mu.Lock()
	defer box.mu.Unlock()
	if len(box.msgs) != 1 || box.msgs[0].Subject != "[Test] Hi there" || !strings.Contains(box.msgs[0].HTML, "Dear there Nguyen,") {
		t.Fatalf("test send: %d messages", len(box.msgs))
	}
}

func between(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], to)
	if j < 0 {
		return s[i:]
	}
	return s[i : i+j+1]
}
