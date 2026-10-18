package polls_test

// Poll results and exports over the API, against the demo: the NPS poll of
// Weekly Digest #141 (PollResults), its comments, closing and reopening,
// and the CSV exports with their permissions and audit entries.

import (
	"context"
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/polls"
)

const ws = "/workspaces/acme-newsletter"

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func TestPollResults(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	campaign := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #141'`)

	var list polls.PollResultsList
	linh.Do(http.MethodGet, ws+"/campaigns/"+campaign+"/polls", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 {
		t.Fatalf("polls of #141: %d", len(list.Items))
	}
	p := list.Items[0]
	// The board's numbers (PollResults).
	if p.Kind != polls.KindScale || p.Answers != 2412 || p.NonContacts != 9 || p.Changed != 74 || p.Automated.Total != 1377 || p.Automated.Gateways != 1129 || p.Automated.Prefetch != 248 {
		t.Fatalf("results: %+v", p)
	}
	if p.NPS == nil || p.NPS.Score != 26 || p.NPS.Promoters != 1086 || p.NPS.Passives != 868 || p.NPS.Detractors != 458 {
		t.Fatalf("NPS: %+v", p.NPS)
	}
	if p.Percent != 7.7 || !p.CanClose || !p.CanExportAnswers || p.Comments != 318 {
		t.Fatalf("share %v, can close %v, can export %v, comments %d", p.Percent, p.CanClose, p.CanExportAnswers, p.Comments)
	}
	nps := map[string]any{}
	for _, r := range p.Breakdown.Tiers {
		if r.TooFew {
			nps[r.Key] = "too few"
		} else if r.NPS != nil {
			nps[r.Key] = *r.NPS
		}
	}
	if nps["active"] != 31 || nps["cooling"] != 12 || nps["at_risk"] != -8 || nps["sunset"] != "too few" {
		t.Fatalf("by tier: %v", nps)
	}
	for _, r := range p.Breakdown.Languages {
		if r.Key == "de" && (r.Answers != 82 || r.NPS == nil || *r.NPS != 22) {
			t.Fatalf("German: %+v", r)
		}
	}
	if p.OverTime.FirstSixPercent != 61 || p.OverTime.After != 24 {
		t.Fatalf("over time: %+v", p.OverTime)
	}

	// Comments by group; names for people with subscribers:view.
	var cm polls.PollCommentList
	linh.Do(http.MethodGet, ws+"/polls/"+p.ID+"/comments?group=promoters&limit=5", nil).MustStatus(t, 200).JSON(t, &cm)
	if cm.Counts.All != 318 || cm.Counts.Promoters != 151 || cm.Counts.Detractors != 71 || len(cm.Items) != 5 || !cm.Names || cm.Total != 151 {
		t.Fatalf("comments: %+v", cm.Counts)
	}
	for _, c := range cm.Items {
		if c.Answer < 9 || c.Group != "promoter" {
			t.Fatalf("a promoter's comment: %+v", c)
		}
	}
	linh.Do(http.MethodGet, ws+"/polls/"+p.ID+"/comments?search=Postgres+pieces", nil).MustStatus(t, 200).JSON(t, &cm)
	if cm.Total != 1 || cm.Items[0].Contact != "Priya Natarajan" {
		t.Fatalf("search: %+v", cm.Items)
	}

	// Exports: results are aggregate and anyone with reports:view gets them;
	// answers with contacts need subscribers:export. Both are audited.
	res := linh.Do(http.MethodGet, ws+"/polls/"+p.ID+"/results.csv", nil).MustStatus(t, 200)
	cr := csv.NewReader(strings.NewReader(string(res.Body)))
	cr.FieldsPerRecord = -1 // sections of different widths
	rows, err := cr.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	flat := ""
	for _, r := range rows {
		flat += strings.Join(r, ",") + "\n"
	}
	if !strings.Contains(flat, "Answers,2412") || strings.Contains(flat, "@") {
		t.Fatalf("results CSV:\n%s", flat)
	}
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	daniel.Do(http.MethodGet, ws+"/polls/"+p.ID+"/results.csv", nil).MustStatus(t, 200)
	daniel.Do(http.MethodGet, ws+"/polls/"+p.ID+"/answers.csv", nil).MustStatus(t, 403)
	var dp polls.PollResults
	daniel.Do(http.MethodGet, ws+"/polls/"+p.ID, nil).MustStatus(t, 200).JSON(t, &dp)
	if dp.CanExportAnswers || len(dp.ExportHelpers) == 0 {
		t.Fatalf("Daniel can't export answers and is told whom to ask: %v %v", dp.CanExportAnswers, dp.ExportHelpers)
	}
	answers := linh.Do(http.MethodGet, ws+"/polls/"+p.ID+"/answers.csv", nil).MustStatus(t, 200)
	if lines := strings.Count(string(answers.Body), "\n"); lines < 2412 || !strings.Contains(string(answers.Body), "priya.n@gmail.com") {
		t.Fatalf("answers CSV: %d lines", lines)
	}
	for _, a := range []string{"poll.results_exported", "poll.answers_exported"} {
		if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, a, p.ID); n < 1 {
			t.Fatalf("%s isn't audited", a)
		}
	}

	// Closing and reopening are audited; the closed poll says how.
	var closed polls.PollResults
	linh.Do(http.MethodPost, ws+"/polls/"+p.ID+"/close", nil).MustStatus(t, 200).JSON(t, &closed)
	if closed.Status != "closed" || closed.ClosedBy != "Linh Nguyen" || closed.ClosedAsScheduled {
		t.Fatalf("closed: %s by %q", closed.Status, closed.ClosedBy)
	}
	linh.Do(http.MethodPost, ws+"/polls/"+p.ID+"/reopen", nil).MustStatus(t, 200).JSON(t, &closed)
	if closed.Status != "open" {
		t.Fatalf("reopened: %s", closed.Status)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action IN ('poll.closed', 'poll.reopened') AND target_id = $1`, p.ID); n != 2 {
		t.Fatalf("close and reopen audited: %d", n)
	}
	// A Viewer can't close it.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do(http.MethodPost, ws+"/polls/"+p.ID+"/close", nil).MustStatus(t, 403)
}

func TestSurveyManagement(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	// The demo survey, as SurveyBuilder shows it.
	var list polls.SurveyList
	linh.Do(http.MethodGet, ws+"/surveys", nil).MustStatus(t, 200).JSON(t, &list)
	var demo polls.Survey
	for _, s := range list.Items {
		if s.Name == "Beta feedback · October 2026" {
			demo = s
		}
	}
	if demo.ID == "" || demo.Status != "open" || len(demo.Questions) != 6 || demo.Started != 688 || demo.Completed != 571 || len(demo.Linked) == 0 || demo.Linked[0].Via != "poll" {
		t.Fatalf("demo survey: %+v", demo)
	}
	var res polls.SurveyResults
	linh.Do(http.MethodGet, ws+"/surveys/"+demo.ID+"/results", nil).MustStatus(t, 200).JSON(t, &res)
	if res.Started != 688 || res.Completed != 571 || res.PollOnly != 117 || res.CompletionPercent != 83 {
		t.Fatalf("survey results: %+v", res)
	}

	// Daniel (Content Writer) sees it but can't change it.
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	var seen polls.Survey
	daniel.Do(http.MethodGet, ws+"/surveys/"+demo.ID, nil).MustStatus(t, 200).JSON(t, &seen)
	if seen.CanEdit || len(seen.EditHelpers) == 0 {
		t.Fatalf("read-only for Daniel: %v %v", seen.CanEdit, seen.EditHelpers)
	}
	daniel.Do(http.MethodPost, ws+"/surveys/"+demo.ID+"/close", nil).MustStatus(t, 403)
	daniel.Do(http.MethodPost, ws+"/surveys", map[string]any{"name": "Not allowed"}).MustStatus(t, 403)

	// Exports of the survey: aggregate for reports:view, answers audited.
	body := string(linh.Do(http.MethodGet, ws+"/surveys/"+demo.ID+"/results.csv", nil).MustStatus(t, 200).Body)
	if !strings.Contains(body, "Answered question 1 only, in the email") || strings.Contains(body, "@") {
		t.Fatalf("survey results CSV:\n%s", body)
	}
	linh.Do(http.MethodGet, ws+"/surveys/"+demo.ID+"/answers.csv", nil).MustStatus(t, 200)
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action IN ('survey.results_exported', 'survey.answers_exported') AND target_id = $1`, demo.ID); n != 2 {
		t.Fatalf("survey exports audited: %d", n)
	}

	// A slug of another survey is refused; deleting is audited.
	var mine polls.Survey
	linh.Do(http.MethodPost, ws+"/surveys", map[string]any{"name": "Management test survey"}).MustStatus(t, 201).JSON(t, &mine)
	linh.Do(http.MethodPatch, ws+"/surveys/"+mine.ID, map[string]any{"slug": demo.Slug}).MustStatus(t, 422)
	linh.Do(http.MethodDelete, ws+"/surveys/"+mine.ID, nil).MustStatus(t, 204)
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'survey.deleted' AND target_id = $1`, mine.ID); n != 1 {
		t.Fatalf("delete audited: %d", n)
	}
}
