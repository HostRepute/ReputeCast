package polls

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Export is a CSV file to download.
type Export struct {
	Filename string
	Body     []byte
}

var nonFile = regexp.MustCompile(`[^a-z0-9]+`)

func fileStem(name, fallback string) string {
	s := strings.Trim(nonFile.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return fallback
	}
	if len(s) > 60 {
		s = strings.TrimRight(s[:60], "-")
	}
	return s
}

// csvSafe neutralizes values a spreadsheet would run as a formula.
func csvSafe(vals ...string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
			v = "'" + v
		}
		out[i] = v
	}
	return out
}

func num(n int64) string { return strconv.FormatInt(n, 10) }

func pct(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" }

func stamp(t *time.Time, loc *time.Location) string {
	if t == nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02 15:04")
}

// ExportPollResults writes a poll's results as CSV: aggregate numbers only,
// audited like the other report exports.
func (s *Service) ExportPollResults(ctx context.Context, ws *store.Workspace, p *authz.Principal, pollID uuid.UUID) (Export, error) {
	r, err := s.PollResultsOf(ctx, ws, p, pollID)
	if err != nil {
		return Export{}, err
	}
	loc := state.Location(ws.Timezone)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := func(vals ...string) { _ = w.Write(csvSafe(vals...)) }
	row("Poll", r.Question)
	row("Campaign", r.Campaign)
	status := "Open"
	if r.Status == "closed" {
		status = "Closed " + stamp(r.ClosedAt, loc)
	} else if r.ClosesAt != nil {
		status = "Open, closes " + stamp(r.ClosesAt, loc)
	}
	row("Status", status)
	row("Answers", num(r.Answers))
	row("Delivered", num(r.Delivered))
	row("Answers of delivered", pct(r.Percent))
	row("Answers from people who aren't contacts", num(r.NonContacts))
	row("Answers changed", num(r.Changed))
	row("Comments", num(r.Comments))
	row("Automated visits ignored", num(r.Automated.Total))
	row("From security gateways", num(r.Automated.Gateways))
	row("From prefetching proxies", num(r.Automated.Prefetch))
	if r.NPS != nil {
		row("Net Promoter Score", strconv.Itoa(r.NPS.Score))
		row("Promoters (9–10)", num(r.NPS.Promoters), pct(r.NPS.PromoterPercent))
		row("Passives (7–8)", num(r.NPS.Passives), pct(r.NPS.PassivePercent))
		row("Detractors (0–6)", num(r.NPS.Detractors), pct(r.NPS.DetractorPercent))
	}
	row()
	row("Answer", "Count", "Share")
	for _, c := range r.Counts {
		label := c.Label
		if c.Emoji != "" {
			label = c.Emoji + " " + label
		}
		row(label, num(c.Count), pct(c.Percent))
	}
	breakdown := func(title string, rows []PollBreakdownRow) {
		if len(rows) == 0 {
			return
		}
		row()
		head := []string{title, "Answers", "Share"}
		if r.NPS != nil {
			head = append(head, "NPS")
		}
		row(head...)
		for _, b := range rows {
			vals := []string{b.Label, num(b.Answers), pct(b.Percent)}
			if r.NPS != nil {
				switch {
				case b.TooFew:
					vals = append(vals, "Too few to compare")
				case b.NPS != nil:
					vals = append(vals, strconv.Itoa(*b.NPS))
				}
			}
			row(vals...)
		}
	}
	breakdown("Engagement tier", r.Breakdown.Tiers)
	breakdown("List", r.Breakdown.Lists)
	breakdown("Language", r.Breakdown.Languages)
	if len(r.OverTime.Cumulative) > 0 {
		row()
		row("Hours after sending", "Answers so far")
		for i, n := range r.OverTime.Cumulative {
			row(strconv.Itoa(i+1), num(n))
		}
	}
	w.Flush()
	out := Export{Filename: fileStem(r.Campaign, "campaign") + "-poll-results.csv", Body: buf.Bytes()}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "poll.results_exported", TargetType: "poll", TargetID: r.ID, TargetName: r.Campaign + " · " + r.Question,
		Metadata: map[string]any{"format": "csv", "answers": r.Answers, "withContacts": false}})
	return out, err
}

// ExportPollAnswers writes every answer with its contact (subscribers:export,
// audited).
func (s *Service) ExportPollAnswers(ctx context.Context, ws *store.Workspace, pollID uuid.UUID) (Export, error) {
	r, err := s.Q.PollByID(ctx, store.PollByIDParams{WorkspaceID: ws.ID, ID: pollID})
	if err != nil {
		return Export{}, api.MapError(err, "Poll")
	}
	_, answers := configOf(r.BlockID, r.Kind, r.Question, r.Answers, r.Settings)
	label := map[int]string{}
	for _, a := range answers {
		label[a.Key] = a.Label
	}
	rows, err := s.Q.PollExportAnswers(ctx, r.ID)
	if err != nil {
		return Export{}, err
	}
	loc := state.Location(ws.Timezone)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := func(vals ...string) { _ = w.Write(csvSafe(vals...)) }
	row("email", "first_name", "last_name", "answer", "answer_label", "first_answer", "changes", "comment", "answered_at", "updated_at", "contact")
	for _, a := range rows {
		who := "contact"
		if a.Kind == AnswererNonContact {
			who = "not a contact"
		} else if a.Email == "" {
			who = "erased"
		}
		row(a.Email, a.FirstName, a.LastName, strconv.Itoa(int(a.Answer)), label[int(a.Answer)], strconv.Itoa(int(a.FirstAnswer)), strconv.Itoa(int(a.Changes)),
			a.Comment, stamp(&a.AnsweredAt, loc), stamp(&a.UpdatedAt, loc), who)
	}
	w.Flush()
	out := Export{Filename: fileStem(r.CampaignName, "campaign") + "-poll-answers.csv", Body: buf.Bytes()}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "poll.answers_exported", TargetType: "poll", TargetID: r.ID.String(), TargetName: r.CampaignName + " · " + r.Question,
		Metadata: map[string]any{"format": "csv", "rows": len(rows), "withContacts": true}})
	return out, err
}

// ExportSurveyResults writes a survey's results as CSV, aggregate only.
func (s *Service) ExportSurveyResults(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (Export, error) {
	sv, err := s.GetSurvey(ctx, ws, id)
	if err != nil {
		return Export{}, err
	}
	res, err := s.SurveyResultsOf(ctx, ws, id)
	if err != nil {
		return Export{}, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := func(vals ...string) { _ = w.Write(csvSafe(vals...)) }
	row("Survey", sv.Name)
	row("Started", num(res.Started))
	row("Completed", num(res.Completed), pct(res.CompletionPercent))
	row("Answered question 1 only, in the email", num(res.PollOnly))
	for _, q := range res.Questions {
		row()
		row(strconv.Itoa(q.Number)+". "+q.Text, q.Type, num(q.Answered)+" answers")
		if q.NPS != nil {
			row("Net Promoter Score", strconv.Itoa(q.NPS.Score))
			row("Promoters (9–10)", num(q.NPS.Promoters), pct(q.NPS.PromoterPercent))
			row("Passives (7–8)", num(q.NPS.Passives), pct(q.NPS.PassivePercent))
			row("Detractors (0–6)", num(q.NPS.Detractors), pct(q.NPS.DetractorPercent))
		}
		for _, c := range q.Counts {
			row(c.Label, num(c.Count), pct(c.Percent))
		}
	}
	w.Flush()
	out := Export{Filename: fileStem(sv.Name, "survey") + "-results.csv", Body: buf.Bytes()}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "survey.results_exported", TargetType: "survey", TargetID: sv.ID, TargetName: sv.Name,
		Metadata: map[string]any{"format": "csv", "completed": res.Completed, "withContacts": false}})
	return out, err
}

// ExportSurveyAnswers writes every response with its contact
// (subscribers:export, audited).
func (s *Service) ExportSurveyAnswers(ctx context.Context, ws *store.Workspace, id uuid.UUID) (Export, error) {
	sv, err := s.GetSurvey(ctx, ws, id)
	if err != nil {
		return Export{}, err
	}
	rows, err := s.Q.SurveyAllResponses(ctx, id)
	if err != nil {
		return Export{}, err
	}
	loc := state.Location(ws.Timezone)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := func(vals ...string) { _ = w.Write(csvSafe(vals...)) }
	head := []string{"email", "first_name", "last_name", "started_at", "completed_at", "source"}
	for _, q := range sv.Questions {
		head = append(head, q.Key+" "+q.Text)
	}
	row(head...)
	for _, r := range rows {
		var answers map[string]any
		_ = json.Unmarshal(r.Answers, &answers)
		vals := []string{r.Email, r.FirstName, r.LastName, stamp(&r.StartedAt, loc), stamp(r.CompletedAt, loc), r.Source}
		if r.Kind == AnswererNonContact {
			vals[0] = "not a contact"
		}
		for _, q := range sv.Questions {
			vals = append(vals, AnswerText(q, answers[q.Key]))
		}
		row(vals...)
	}
	w.Flush()
	out := Export{Filename: fileStem(sv.Name, "survey") + "-answers.csv", Body: buf.Bytes()}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "survey.answers_exported", TargetType: "survey", TargetID: sv.ID, TargetName: sv.Name,
		Metadata: map[string]any{"format": "csv", "rows": len(rows), "withContacts": true}})
	return out, err
}
