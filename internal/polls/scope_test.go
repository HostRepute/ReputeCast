package polls_test

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/store"
)

// respondent is an answer's or a response's contact.
type respondent struct {
	answerID, contactID, email, name string
}

// respondents are the two newest answers (or responses) of the query's
// contacts: the first is taken off the list, the second put on it.
func respondents(t *testing.T, env *apptest.Env, list, sql string, args ...any) (off, on respondent) {
	t.Helper()
	ctx := context.Background()
	rows, err := env.App.Pool.Query(ctx, sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	var got []respondent
	for rows.Next() {
		var r respondent
		if err := rows.Scan(&r.answerID, &r.contactID, &r.email, &r.name); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if rows.Err() != nil || len(got) != 2 {
		t.Fatalf("two respondents: %+v %v", got, rows.Err())
	}
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM list_memberships WHERE contact_id = $1 AND list_id = $2`, got[0].contactID, list); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at) VALUES ($1, $2, 'subscribed', 'form:test', now())
		ON CONFLICT (list_id, contact_id) DO NOTHING`, list, got[1].contactID); err != nil {
		t.Fatal(err)
	}
	return got[0], got[1]
}

// emails is the email column of an answers export.
func emails(t *testing.T, body []byte) []string {
	t.Helper()
	recs, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil || len(recs) == 0 || recs[0][0] != "email" {
		t.Fatalf("CSV: %v %q", err, body)
	}
	out := []string{}
	for _, r := range recs[1:] {
		out = append(out, r[0])
	}
	return out
}

// A member limited to some lists exports only the answers of the contacts
// on them, and sees only those contacts named beside comments and
// responses.
func TestScopedMemberSeesOnlyTheirContactsAnswers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, admin uuid.UUID
	var beta string
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id, l.id::text FROM workspaces w JOIN roles r ON r.workspace_id = w.id AND r.key = 'admin'
		JOIN lists l ON l.workspace_id = w.id AND l.name = 'Beta Program' WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &admin, &beta); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(authz.Scope{Lists: []uuid.UUID{uuid.MustParse(beta)}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: admin, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim, linh := env.Client(), env.Client()
	kim.SignInDemo("kim@contractor.dev")
	linh.SignInDemo("linh@acme.io")

	// The NPS poll of Weekly Digest #141 and two of its commenters.
	campaign := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #141'`)
	var list polls.PollResultsList
	linh.Do(http.MethodGet, ws+"/campaigns/"+campaign+"/polls", nil).MustStatus(t, 200).JSON(t, &list)
	poll := list.Items[0].ID
	off, on := respondents(t, env, beta, `SELECT a.id::text, c.id::text, c.email, btrim(c.first_name || ' ' || c.last_name)
		FROM poll_answers a JOIN contacts c ON c.id = a.contact_id
		WHERE a.poll_id = $1 AND a.kind = 'contact' AND a.comment <> '' AND btrim(c.first_name) <> ''
		ORDER BY coalesce(a.commented_at, a.updated_at) DESC, a.id DESC LIMIT 2`, poll)

	mine := emails(t, kim.Do(http.MethodGet, ws+"/polls/"+poll+"/answers.csv", nil).MustStatus(t, 200).Body)
	all := emails(t, linh.Do(http.MethodGet, ws+"/polls/"+poll+"/answers.csv", nil).MustStatus(t, 200).Body)
	inScope := one[int64](t, env, `SELECT count(*) FROM poll_answers a WHERE a.poll_id = $1 AND a.kind <> 'test'
		AND EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = a.contact_id AND m.list_id = $2)`, poll, beta)
	if int64(len(mine)) != inScope || slices.Contains(mine, off.email) || !slices.Contains(mine, on.email) || slices.Contains(mine, "") {
		t.Fatalf("Kim's export: %d rows, want %d, with %s and without %s", len(mine), inScope, on.email, off.email)
	}
	if len(all) <= len(mine) || !slices.Contains(all, off.email) {
		t.Fatalf("Linh's export has every answer: %d rows", len(all))
	}

	// Comments: the answer stays, the name shows only on Beta Program, and
	// a name off it finds nothing.
	type comments struct {
		Total int64
		Items []struct{ ID, Contact, ContactID string }
	}
	named := func(c comments, id string) (string, bool) {
		for _, it := range c.Items {
			if it.ID == id {
				return it.Contact, true
			}
		}
		return "", false
	}
	var kc, lc comments
	kim.Do(http.MethodGet, ws+"/polls/"+poll+"/comments?limit=5", nil).MustStatus(t, 200).JSON(t, &kc)
	if name, ok := named(kc, off.answerID); !ok || name != "" {
		t.Fatalf("Kim sees the comment off her lists without its name: %q %v", name, ok)
	}
	if name, ok := named(kc, on.answerID); !ok || name != on.name {
		t.Fatalf("Kim sees the name on Beta Program: %q %v", name, ok)
	}
	kim.Do(http.MethodGet, ws+"/polls/"+poll+"/comments?search="+url.QueryEscape(off.name), nil).MustStatus(t, 200).JSON(t, &kc)
	if _, ok := named(kc, off.answerID); ok {
		t.Fatalf("Kim's search by a name off her lists finds it: %+v", kc.Items)
	}
	linh.Do(http.MethodGet, ws+"/polls/"+poll+"/comments?search="+url.QueryEscape(off.name), nil).MustStatus(t, 200).JSON(t, &lc)
	if name, ok := named(lc, off.answerID); !ok || name != off.name {
		t.Fatalf("Linh finds the comment by its name: %+v", lc.Items)
	}

	// The demo survey: its responses and their export.
	survey := one[string](t, env, `SELECT id::text FROM surveys WHERE name = 'Beta feedback · October 2026'`)
	off, on = respondents(t, env, beta, `SELECT r.id::text, c.id::text, c.email, btrim(c.first_name || ' ' || c.last_name)
		FROM survey_responses r JOIN contacts c ON c.id = r.contact_id
		WHERE r.survey_id = $1 AND r.kind = 'contact' AND r.completed_at IS NOT NULL AND btrim(c.first_name) <> ''
		ORDER BY r.completed_at DESC, r.id DESC LIMIT 2`, survey)
	var kr, lr polls.SurveyResponseList
	kim.Do(http.MethodGet, ws+"/surveys/"+survey+"/responses?limit=5", nil).MustStatus(t, 200).JSON(t, &kr)
	linh.Do(http.MethodGet, ws+"/surveys/"+survey+"/responses?limit=5", nil).MustStatus(t, 200).JSON(t, &lr)
	who := func(l polls.SurveyResponseList, id string) string {
		for _, it := range l.Items {
			if it.ID == id {
				return it.Contact
			}
		}
		t.Fatalf("response %s isn't on the page", id)
		return ""
	}
	if who(kr, off.answerID) != "" || who(kr, on.answerID) != on.name || who(lr, off.answerID) != off.name {
		t.Fatalf("names beside the responses: Kim %+v, Linh %+v", kr.Items, lr.Items)
	}
	mine = emails(t, kim.Do(http.MethodGet, ws+"/surveys/"+survey+"/answers.csv", nil).MustStatus(t, 200).Body)
	all = emails(t, linh.Do(http.MethodGet, ws+"/surveys/"+survey+"/answers.csv", nil).MustStatus(t, 200).Body)
	inScope = one[int64](t, env, `SELECT count(*) FROM survey_responses r WHERE r.survey_id = $1 AND r.kind <> 'test'
		AND EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = r.contact_id AND m.list_id = $2)`, survey, beta)
	if int64(len(mine)) != inScope || slices.Contains(mine, off.email) || !slices.Contains(mine, on.email) {
		t.Fatalf("Kim's survey export: %d rows, want %d, with %s and without %s", len(mine), inScope, on.email, off.email)
	}
	if !slices.Contains(all, off.email) {
		t.Fatalf("Linh's survey export has every response: %d rows", len(all))
	}
}
