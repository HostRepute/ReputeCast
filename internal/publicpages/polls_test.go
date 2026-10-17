package publicpages_test

// Poll answer pages and survey pages (design expansion, X11): opening a
// link records nothing, the page records the answer (its script, or
// Confirm without JavaScript), machines never answer, a changed answer
// replaces the old one, people who aren't contacts are counted apart or
// refused, and a closed poll records nothing.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// pollTester is a contact of the test's own, on Weekly Digest.
func pollTester(t *testing.T, env *apptest.Env, ids demoIDs, email, first string) uuid.UUID {
	t.Helper()
	var out struct{ ID string }
	owner(env).Do(http.MethodPost, wsAPI+"/contacts", map[string]any{"email": email, "firstName": first, "lists": []string{ids.digest.String()}, "consent": "single"}).
		MustStatus(t, 201).JSON(t, &out)
	return uuid.MustParse(out.ID)
}

func pollLink(env *apptest.Env, ws, contact, message uuid.UUID, number int64) string {
	return env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Poll, WorkspaceID: ws, ContactID: contact, MessageID: message, Link: uint32(number)})
}

func answerOf(t *testing.T, env *apptest.Env, poll, contact uuid.UUID) (answer, changes int, comment string) {
	t.Helper()
	err := env.App.Pool.QueryRow(context.Background(), `SELECT answer, changes, comment FROM poll_answers WHERE poll_id = $1 AND contact_id = $2`, poll, contact).Scan(&answer, &changes, &comment)
	if err != nil {
		return -1, 0, ""
	}
	return answer, changes, comment
}

func TestPollAnswerPage(t *testing.T) {
	env, ids := demo(t)
	// Weekly Digest #141's NPS poll (save to nps_score, non-contacts counted).
	poll := one[uuid.UUID](t, env, `SELECT id FROM polls WHERE workspace_id = $1 AND question LIKE 'How likely are you%'`, ids.ws)
	number := one[int64](t, env, `SELECT number FROM polls WHERE id = $1`, poll)
	pat := pollTester(t, env, ids, "pat.poll@example.com", "Pat")
	tok := pollLink(env, ids.ws, pat, uuid.Nil, number)
	base := "/p/" + wsSlug + "/poll/" + tok

	t.Run("opening the link records nothing", func(t *testing.T) {
		r := get(t, env, base+"/9")
		must(t, r, 200, "How likely are you to recommend the Weekly Digest to a colleague?", "You picked 9 in the email. Nothing is saved until you confirm it.", "Confirm my answer")
		if a, _, _ := answerOf(t, env, poll, pat); a != -1 {
			t.Fatalf("a GET recorded %d", a)
		}
	})

	t.Run("the page's script records the answer", func(t *testing.T) {
		r := do(t, env, http.MethodPost, base+"/9", url.Values{}, hdr{"X-RC-Script": "1"})
		must(t, r, 200, "Thanks, Pat. Your answer is saved.")
		if a, n, _ := answerOf(t, env, poll, pat); a != 9 || n != 0 {
			t.Fatalf("answer %d, changes %d", a, n)
		}
		// The answer fills nps_score, and starts "Answered a poll or survey".
		if v := one[string](t, env, `SELECT attributes->>'nps_score' FROM contacts WHERE id = $1`, pat); v != "9" {
			t.Fatalf("nps_score %q", v)
		}
		var payload json.RawMessage
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT payload FROM automation_events WHERE contact_id = $1 AND kind = 'poll_answered' AND ref = $2`, pat, "poll:"+poll.String()).Scan(&payload); err != nil {
			t.Fatalf("no trigger event: %v", err)
		}
		if !strings.Contains(string(payload), `"answer": 9`) && !strings.Contains(string(payload), `"answer":9`) {
			t.Fatalf("payload %s", payload)
		}
	})

	t.Run("without JavaScript, Confirm records a changed answer", func(t *testing.T) {
		r := do(t, env, http.MethodPost, base+"/9", url.Values{"answer": {"7"}}, nil)
		must(t, r, 200, "Changed from 9 to 7. Your new answer is saved; the latest one counts.")
		if a, n, _ := answerOf(t, env, poll, pat); a != 7 || n != 1 {
			t.Fatalf("answer %d, changes %d", a, n)
		}
		if n := count(t, env, `SELECT count(*) FROM poll_answer_events e JOIN poll_answers a ON a.id = e.answer_id WHERE a.contact_id = $1 AND e.kind = 'changed' AND e.previous = 9 AND e.answer = 7`, pat); n != 1 {
			t.Fatalf("history: %d changed events", n)
		}
		if n := count(t, env, `SELECT count(*) FROM automation_events WHERE contact_id = $1 AND kind = 'poll_answered' AND payload->>'changed' = 'true'`, pat); n != 1 {
			t.Fatalf("the changed answer's event: %d", n)
		}
		// The same answer again changes nothing.
		do(t, env, http.MethodPost, base+"/7", url.Values{}, hdr{"X-RC-Script": "1"})
		if _, n, _ := answerOf(t, env, poll, pat); n != 1 {
			t.Fatalf("the same answer counted as a change: %d", n)
		}
	})

	t.Run("a comment", func(t *testing.T) {
		r := do(t, env, http.MethodPost, base+"/7/comment", url.Values{"comment": {"More charts, please."}}, nil)
		if r.Status != 200 && r.Status != 303 {
			t.Fatalf("comment: %d %s", r.Status, clip(r.Body))
		}
		if _, _, c := answerOf(t, env, poll, pat); c != "More charts, please." {
			t.Fatalf("comment %q", c)
		}
	})

	t.Run("machines never answer", func(t *testing.T) {
		before := count(t, env, `SELECT coalesce(sum(n), 0)::int FROM poll_automated_visits WHERE poll_id = $1`, poll)
		sam := pollTester(t, env, ids, "sam.scanned@example.com", "Sam")
		stok := "/p/" + wsSlug + "/poll/" + pollLink(env, ids.ws, sam, uuid.Nil, number)
		// A gateway's scanner opens and posts; a link preview opens.
		do(t, env, http.MethodGet, stok+"/10", nil, hdr{"User-Agent": "Mimecast-LinkScanner/2.1"})
		do(t, env, http.MethodPost, stok+"/10", url.Values{}, hdr{"User-Agent": "Mimecast-LinkScanner/2.1"})
		do(t, env, http.MethodGet, stok+"/0", nil, hdr{"User-Agent": "Mozilla/5.0 (compatible; LinkExpanding/1.0)"})
		if a, _, _ := answerOf(t, env, poll, sam); a != -1 {
			t.Fatalf("a scanner answered %d", a)
		}
		gateways := count(t, env, `SELECT coalesce(sum(n), 0)::int FROM poll_automated_visits WHERE poll_id = $1 AND reason = 'gateway'`, poll)
		after := count(t, env, `SELECT coalesce(sum(n), 0)::int FROM poll_automated_visits WHERE poll_id = $1`, poll)
		if after != before+3 || gateways < 1129+2 {
			t.Fatalf("automated visits %d → %d (gateways %d)", before, after, gateways)
		}
		// Every answer of one email within a second, without the page's
		// script: a scanner following each link.
		msg := one[uuid.UUID](t, env, `SELECT id FROM messages WHERE campaign_id = (SELECT campaign_id FROM polls WHERE id = $1) LIMIT 1`, poll)
		btok := "/p/" + wsSlug + "/poll/" + pollLink(env, ids.ws, sam, msg, number)
		do(t, env, http.MethodPost, btok+"/3", url.Values{}, nil)
		do(t, env, http.MethodPost, btok+"/4", url.Values{}, nil)
		if a, _, _ := answerOf(t, env, poll, sam); a != 3 {
			t.Fatalf("the burst's second answer replaced the first: %d", a)
		}
	})

	t.Run("people who aren't contacts", func(t *testing.T) {
		before := count(t, env, `SELECT count(*) FROM poll_answers WHERE poll_id = $1 AND kind = 'non_contact'`, poll)
		anon := "/p/" + wsSlug + "/poll/" + pollLink(env, ids.ws, uuid.Nil, uuid.Nil, number)
		r := do(t, env, http.MethodPost, anon+"/8", url.Values{}, hdr{"X-RC-Script": "1"})
		must(t, r, 200, "Thanks. Your answer is counted, but it isn't linked to you.")
		if n := count(t, env, `SELECT count(*) FROM poll_answers WHERE poll_id = $1 AND kind = 'non_contact'`, poll); n != before+1 {
			t.Fatalf("non-contact answers %d → %d", before, n)
		}
		// The beta poll doesn't take them.
		beta := one[int64](t, env, `SELECT number FROM polls WHERE workspace_id = $1 AND question = 'How often do you use feature flags?'`, ids.ws)
		r = do(t, env, http.MethodPost, "/p/"+wsSlug+"/poll/"+pollLink(env, ids.ws, uuid.Nil, uuid.Nil, beta)+"/1", url.Values{}, hdr{"X-RC-Script": "1"})
		must(t, r, 200, "This poll is for subscribers.")
	})

	t.Run("a broken link", func(t *testing.T) {
		bad := tok[:len(tok)-2] + "xx"
		r := get(t, env, "/p/"+wsSlug+"/poll/"+bad+"/9")
		must(t, r, 404, "This link doesn't work anymore")
		r = get(t, env, base+"/42")
		must(t, r, 404, "This link doesn't work anymore")
	})

	t.Run("a closed poll records nothing", func(t *testing.T) {
		owner(env).Do(http.MethodPost, wsAPI+"/polls/"+poll.String()+"/close", nil).MustStatus(t, 200)
		r := do(t, env, http.MethodPost, base+"/10", url.Values{}, hdr{"X-RC-Script": "1"})
		must(t, r, 200, "This poll closed on")
		if a, _, _ := answerOf(t, env, poll, pat); a != 7 {
			t.Fatalf("a closed poll recorded %d", a)
		}
		if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'poll.closed' AND target_id = $1`, poll.String()); n != 1 {
			t.Fatalf("closing isn't audited: %d", n)
		}
	})
}

func TestSurveyPage(t *testing.T) {
	env, ids := demo(t)
	c := owner(env)
	// A survey of the test's own: two required questions and a long text.
	var sv struct {
		ID, Slug, Status string
		Number           int64
		Problems         []struct{ Message string }
	}
	c.Do(http.MethodPost, wsAPI+"/surveys", map[string]any{"name": "Page test survey"}).MustStatus(t, 201).JSON(t, &sv)
	c.Do(http.MethodPost, wsAPI+"/surveys/"+sv.ID+"/publish", nil).MustStatus(t, 422)
	questions := []map[string]any{
		{"key": "q1", "type": "single", "text": "How often do you deploy?", "required": true, "answers": []map[string]any{{"key": 1, "label": "Daily"}}, "save": map[string]any{"to": "none"}},
		{"key": "q2", "type": "scale", "text": "How likely are you to recommend us?", "required": true, "answers": []map[string]any{}, "save": map[string]any{"to": "none"}, "nps": true},
		{"key": "q3", "type": "long", "text": "Anything else?", "required": false, "answers": []map[string]any{}, "save": map[string]any{"to": "none"}},
	}
	c.Do(http.MethodPatch, wsAPI+"/surveys/"+sv.ID, map[string]any{"questions": questions, "slug": "page-test-survey"}).MustStatus(t, 200).JSON(t, &sv)
	if len(sv.Problems) != 1 || sv.Problems[0].Message != "A choice question needs at least two answers." {
		t.Fatalf("problems: %+v", sv.Problems)
	}
	c.Do(http.MethodPost, wsAPI+"/surveys/"+sv.ID+"/publish", nil).MustStatus(t, 422)
	questions[0]["answers"] = []map[string]any{{"key": 1, "label": "Daily"}, {"key": 2, "label": "Weekly"}}
	c.Do(http.MethodPatch, wsAPI+"/surveys/"+sv.ID, map[string]any{"questions": questions}).MustStatus(t, 200)
	c.Do(http.MethodPost, wsAPI+"/surveys/"+sv.ID+"/publish", nil).MustStatus(t, 200).JSON(t, &sv)
	if sv.Status != "open" {
		t.Fatalf("published: %+v", sv)
	}
	// An open survey keeps every question complete.
	broken := []map[string]any{questions[0], questions[1], {"key": "q3", "type": "long", "text": "", "required": false, "answers": []map[string]any{}, "save": map[string]any{"to": "none"}}}
	c.Do(http.MethodPatch, wsAPI+"/surveys/"+sv.ID, map[string]any{"questions": broken}).MustStatus(t, 422)

	pat := pollTester(t, env, ids, "pat.survey@example.com", "Pat")
	tok := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Survey, WorkspaceID: ids.ws, ContactID: pat, Link: uint32(sv.Number)})
	page := "/p/" + wsSlug + "/s/" + sv.Slug + "?t=" + url.QueryEscape(tok)

	r := get(t, env, page)
	must(t, r, 200, "Page test survey", "How often do you deploy?", "Anything else?")
	// Without a contact's link: not accepted.
	must(t, get(t, env, "/p/"+wsSlug+"/s/"+sv.Slug), 200, "subscribers")

	// A required question left empty: the summary and the message under it.
	r = do(t, env, http.MethodPost, page, url.Values{"q1": {"2"}}, nil)
	if r.Status != http.StatusUnprocessableEntity || !strings.Contains(r.Text(), "1 question needs an answer") {
		t.Fatalf("missing answer: %d %s", r.Status, clip(r.Text()))
	}
	if n := count(t, env, `SELECT count(*) FROM survey_responses WHERE survey_id = $1 AND completed_at IS NOT NULL`, sv.ID); n != 0 {
		t.Fatalf("an incomplete response completed: %d", n)
	}
	r = do(t, env, http.MethodPost, page, url.Values{"q1": {"2"}, "q2": {"9"}, "q3": {"Ship it."}}, nil)
	must(t, r, 200, "Your answers are saved")
	var results struct {
		Started, Completed int64
		Questions          []struct {
			Answered int64
			NPS      *struct{ Score int }
		}
	}
	c.Do(http.MethodGet, wsAPI+"/surveys/"+sv.ID+"/results", nil).MustStatus(t, 200).JSON(t, &results)
	if results.Completed != 1 || results.Questions[1].NPS == nil || results.Questions[1].NPS.Score != 100 || results.Questions[2].Answered != 1 {
		t.Fatalf("results: %+v", results)
	}
	// Changing the response keeps one per contact.
	do(t, env, http.MethodPost, page, url.Values{"q1": {"1"}, "q2": {"3"}}, nil)
	c.Do(http.MethodGet, wsAPI+"/surveys/"+sv.ID+"/results", nil).MustStatus(t, 200).JSON(t, &results)
	if results.Completed != 1 || results.Questions[1].NPS.Score != -100 {
		t.Fatalf("after changing: %+v", results)
	}
	// Closed: the page takes no answers.
	c.Do(http.MethodPost, wsAPI+"/surveys/"+sv.ID+"/close", nil).MustStatus(t, 200)
	r = do(t, env, http.MethodPost, page, url.Values{"q1": {"2"}, "q2": {"10"}}, nil)
	if !strings.Contains(r.Text(), "This survey closed") {
		t.Fatalf("closed: %d %s", r.Status, clip(r.Text()))
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('survey.created','survey.updated','survey.published','survey.closed')`, sv.ID); n < 4 {
		t.Fatalf("survey changes are audited: %d", n)
	}
}
