package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// summaryTask is a summary's request, as data: the report's numbers by
// key and what they are about (reportask.go's reportContext).
type summaryTask struct {
	LanguageName string
	Report       reportContext
}

// causeRe catches a summary that explains causes or gives advice.
var causeRe = regexp.MustCompile(`(?i)\b(because|due to|caused by|as a result of|suggests? that|likely because|probably|you should|we recommend|recommend(ed)?|consider (sending|changing|moving)|try (sending|changing))\b`)

// AIReportSummaryInput asks for a report's summary.
type AIReportSummaryInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
}

// AIReportSummary is a summary suggestion and its checks.
type AIReportSummary struct {
	RequestID string `json:"requestId,omitempty"`
	// Summary has the report's numbers filled in, in the reader's format.
	Summary string `json:"summary"`
	// NumbersMatch: every number is the report's number for the metric the
	// model named (review AN-6); Unmatched lists what isn't.
	NumbersMatch bool     `json:"numbersMatch"`
	Unmatched    []string `json:"unmatched"`
	// NoCauses: it describes the numbers and doesn't diagnose causes.
	NoCauses bool   `json:"noCauses"`
	Sent     AISent `json:"sent"`
	Refusal  string `json:"refusal,omitempty"`
}

// Summarize asks for a plain-language summary of a report's numbers. The
// model writes each number as its metric's key; the numbers are filled in
// from the report in the reader's format, so each is the metric's own, and
// anything else is listed as unmatched. The A/B test's decision goes with
// its confidence, so a test without a clear winner reads as one.
func (s *Service) Summarize(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIReportSummaryInput) (AIReportSummary, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return AIReportSummary{}, invalid(ctx, "body.campaignId", mCampaignID, nil)
	}
	if !p.Can(authz.ReportsView) {
		return AIReportSummary{}, api.Forbidden(authz.ReportsView, nil)
	}
	r, err := s.Reports.Report(ctx, ws, id)
	if err != nil {
		return AIReportSummary{}, err
	}
	if r.NotSent != nil {
		return AIReportSummary{}, problem(ctx, http.StatusConflict, "ai_not_sent", mNotSent, nil)
	}
	l := i18n.LangFrom(ctx)
	nums := numbersFrom(r)
	t := summaryTask{LanguageName: languageName(string(l.Locale)), Report: contextOf(r, nums)}
	raw, _ := json.Marshal(t.Report)
	res, err := s.run(ctx, ws, p, task{feature: FeatureSummary, targetKind: "report", targetID: &id, targetLabel: r.Campaign.Name, req: summaryPrompt(t)})
	if err != nil {
		return AIReportSummary{}, err
	}
	out := AIReportSummary{RequestID: res.RequestID.String(), Unmatched: []string{}, Sent: res.sent(len(raw)), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Summary string `json:"summary"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	text := strings.TrimSpace(plain(a.Summary))
	if text == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	out.Summary, out.Unmatched = checkedText(text, nums, l.Region, codeNumbers(r)...)
	out.NumbersMatch = len(out.Unmatched) == 0
	out.NoCauses = !causeRe.MatchString(out.Summary)
	return out, nil
}

// codeNumbers are the providers' bounce codes (550 5.7.1): a text may
// quote their numbers.
func codeNumbers(r reports.CampaignReport) []string {
	var out []string
	for _, b := range r.Bounces.Rows {
		for _, c := range b.Codes {
			out = append(out, c.CodeText())
		}
	}
	return out
}

// AIKeptSummaryView is a summary someone kept.
type AIKeptSummaryView struct {
	Summary   string    `json:"summary"`
	Model     string    `json:"model"`
	ModelName string    `json:"modelName"`
	KeptBy    string    `json:"keptBy"`
	KeptAt    time.Time `json:"keptAt"`
	// NumbersMatch: every number in it is still the report's ("Every number
	// matches this report"); results that kept coming in can change them.
	NumbersMatch bool     `json:"numbersMatch"`
	Unmatched    []string `json:"unmatched"`
}

// AIKeptSummary is the report's kept summary, if any. HostRepute says
// whether "Ask HostRepute's assistant" shows.
type AIKeptSummary struct {
	Kept       *AIKeptSummaryView `json:"kept,omitempty"`
	HostRepute bool               `json:"hostRepute"`
}

// KeptSummary returns the summary kept on a campaign's report, checked
// against the report as it is now.
func (s *Service) KeptSummary(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AIKeptSummary, error) {
	out := AIKeptSummary{HostRepute: s.hostReputeConnected(ctx, ws.ID)}
	row, err := s.Q.AIReportSummary(ctx, store.AIReportSummaryParams{CampaignID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	name := row.ModelName
	if name == "" {
		name = row.Model
	}
	view := &AIKeptSummaryView{Summary: row.Summary, Model: row.Model, ModelName: name, KeptBy: row.KeptByName, KeptAt: row.KeptAt, Unmatched: []string{}}
	if r, err := s.Reports.Report(ctx, ws, id); err == nil {
		view.Unmatched = uncheckedNumbers(row.Summary, numbersFrom(r), i18n.LangFrom(ctx).Region, codeNumbers(r)...)
	} else {
		return out, err
	}
	view.NumbersMatch = len(view.Unmatched) == 0
	out.Kept = view
	return out, nil
}

// AIKeepSummaryInput keeps a summary on the report.
type AIKeepSummaryInput struct {
	RequestID string `json:"requestId" format:"uuid"`
	// Summary is the suggestion, or what the person made of it (Edit).
	Summary string `json:"summary" minLength:"1" maxLength:"4000"`
	Edited  bool   `json:"edited,omitempty"`
}

// KeepSummary saves a summary at the top of the report, labelled "Summary
// by AI, kept by …". Its numbers are checked again first, edits included:
// a number the report doesn't have isn't kept (review AN-6). It is a
// content change on the report (audited; the activity log says Kept),
// never shown on read-only links.
func (s *Service) KeepSummary(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIKeepSummaryInput) (AIKeptSummary, error) {
	reqID, err := uuid.Parse(in.RequestID)
	if err != nil {
		return AIKeptSummary{}, invalid(ctx, "body.requestId", mSuggestionID, nil)
	}
	r, err := s.Q.AIRequest(ctx, store.AIRequestParams{ID: reqID, WorkspaceID: ws.ID})
	uid := actorID(p)
	if err != nil || uid == nil || r.UserID == nil || *r.UserID != *uid || r.Feature != FeatureSummary || r.TargetID == nil || *r.TargetID != id {
		return AIKeptSummary{}, problem(ctx, http.StatusConflict, "ai_suggestion_gone", mSuggestionGone, nil)
	}
	rep, err := s.Reports.Report(ctx, ws, id)
	if err != nil {
		return AIKeptSummary{}, err
	}
	text := strings.TrimSpace(in.Summary)
	if bad := uncheckedNumbers(text, numbersFrom(rep), i18n.LangFrom(ctx).Region, codeNumbers(rep)...); len(bad) > 0 {
		return AIKeptSummary{}, problem(ctx, http.StatusUnprocessableEntity, "ai_numbers_unmatched", mNumbersUnmatched,
			map[string]any{"numbers": strings.Join(bad, ", "), "count": len(bad)})
	}
	name := r.Model
	if c, err := s.load(ctx, s.Q, ws.ID); err == nil && c != nil && c.Model == r.Model {
		name = c.modelName()
	}
	before, _ := s.Q.AIReportSummary(ctx, store.AIReportSummaryParams{CampaignID: id, WorkspaceID: ws.ID})
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AIReportSummarySave(ctx, store.AIReportSummarySaveParams{CampaignID: id, WorkspaceID: ws.ID, Summary: text, Model: r.Model,
			ModelName: name, RequestID: &reqID, KeptBy: uid, KeptAt: s.now()}); err != nil {
			return err
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "report_summary.keep", TargetType: "campaign", TargetID: id.String(), TargetName: rep.Campaign.Name,
			After: map[string]any{"summary": text, "model": r.Model, "edited": in.Edited}}
		if before.Summary != "" {
			ev.Before = map[string]any{"summary": before.Summary, "keptBy": before.KeptByName}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return AIKeptSummary{}, err
	}
	detail := ""
	if in.Edited {
		detail = "with edits"
	}
	s.finish(ctx, ws, p, reqID, OutcomeKept, detail)
	return s.KeptSummary(ctx, ws, id)
}

// RemoveSummary takes a kept summary off the report (audited, review AN-6).
func (s *Service) RemoveSummary(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	row, err := s.Q.AIReportSummary(ctx, store.AIReportSummaryParams{CampaignID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return api.NotFound("Summary")
	}
	if err != nil {
		return err
	}
	name := ""
	if c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: id}); err == nil {
		name = c.Name
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.AIReportSummaryDelete(ctx, store.AIReportSummaryDeleteParams{CampaignID: id, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Summary")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "report_summary.remove", TargetType: "campaign", TargetID: id.String(),
			TargetName: name, Before: map[string]any{"summary": row.Summary, "model": row.Model, "keptBy": row.KeptByName, "keptAt": row.KeptAt}})
	})
}
