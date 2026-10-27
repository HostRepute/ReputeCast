package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// askTask is a question about a report, as data.
type askTask struct {
	LanguageName string
	Question     string
	Report       reportContext
}

func askPrompt(t askTask) llm.Request {
	report, _ := json.Marshal(t.Report)
	req := userRequest(systemPrompt, textPart(
		fmt.Sprintf("Answer the person's question about this campaign's results in %s, in one to four sentences, from the report's numbers only. If the numbers can't answer it, say so in one sentence.", t.LanguageName),
		numbersRule,
		`Answer {"answer": "…"}.`,
		tag("question", t.Question),
		tag("report", string(report)),
	))
	req.MaxTokens = 500
	req.Task = t
	return req
}

// AIReportQuestionInput asks a question about a report.
type AIReportQuestionInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	Question   string `json:"question" minLength:"1" maxLength:"500"`
}

// AIReportAnswer is the answer to a question about a report: its numbers
// are the report's, filled in, and checked (review AN-14).
type AIReportAnswer struct {
	RequestID string `json:"requestId,omitempty"`
	Answer    string `json:"answer"`
	// NumbersMatch: every number is the report's; Unmatched lists the ones
	// that aren't.
	NumbersMatch bool     `json:"numbersMatch"`
	Unmatched    []string `json:"unmatched"`
	Sent         AISent   `json:"sent"`
	Refusal      string   `json:"refusal,omitempty"`
}

// Ask answers a question about a report with the workspace's own model:
// it sends the report's aggregate numbers and the question, never an
// address.
func (s *Service) Ask(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIReportQuestionInput) (AIReportAnswer, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return AIReportAnswer{}, invalid(ctx, "body.campaignId", mCampaignID, nil)
	}
	if !p.Can(authz.ReportsView) {
		return AIReportAnswer{}, api.Forbidden(authz.ReportsView, nil)
	}
	question := strings.TrimSpace(in.Question)
	if question == "" {
		return AIReportAnswer{}, invalid(ctx, "body.question", mQuestionEmpty, nil)
	}
	r, err := s.Reports.Report(ctx, ws, id)
	if err != nil {
		return AIReportAnswer{}, err
	}
	if r.NotSent != nil {
		return AIReportAnswer{}, problem(ctx, http.StatusConflict, "ai_not_sent", mNotSent, nil)
	}
	l := i18n.LangFrom(ctx)
	nums := numbersFrom(r)
	t := askTask{LanguageName: languageName(string(l.Locale)), Question: question, Report: contextOf(r, nums)}
	raw, _ := json.Marshal(t.Report)
	res, err := s.run(ctx, ws, p, task{feature: FeatureSummary, targetKind: "report", targetID: &id, targetLabel: r.Campaign.Name, req: askPrompt(t)})
	if err != nil {
		return AIReportAnswer{}, err
	}
	out := AIReportAnswer{RequestID: res.RequestID.String(), Unmatched: []string{}, Sent: res.sent(len(raw) + len(question)), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Answer string `json:"answer"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	text := strings.TrimSpace(plain(a.Answer))
	if text == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	out.Answer, out.Unmatched = checkedText(text, nums, l.Region, codeNumbers(r)...)
	out.NumbersMatch = len(out.Unmatched) == 0
	return out, nil
}
