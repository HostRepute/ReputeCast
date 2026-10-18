package ai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// summaryNumbers are the report's aggregate numbers: what a summary sends.
// There is no address, name or other contact data in a report.
type summaryNumbers struct {
	Campaign        string           `json:"campaign"`
	Sent            int64            `json:"sent"`
	Delivered       int64            `json:"delivered"`
	DeliveredRate   *float64         `json:"deliveredRate,omitempty"`
	HumanOpens      int64            `json:"humanOpens"`
	HumanOpenRate   *float64         `json:"humanOpenRate,omitempty"`
	MachineOpens    int64            `json:"machineOpens"`
	MachineOpenRate *float64         `json:"machineOpenRate,omitempty"`
	Clicks          int64            `json:"clicks"`
	ClickRate       *float64         `json:"clickRate,omitempty"`
	Unsubscribes    int64            `json:"unsubscribes"`
	UnsubscribeRate *float64         `json:"unsubscribeRate,omitempty"`
	Complaints      int64            `json:"complaints"`
	Bounces         int64            `json:"bounces"`
	BounceRate      *float64         `json:"bounceRate,omitempty"`
	TopLinks        []summaryLink    `json:"topLinks"`
	ABTest          *summaryABTest   `json:"abTest,omitempty"`
	Mailboxes       []summaryMailbox `json:"mailboxProviders"`
	Unsubscribe     []summaryReason  `json:"unsubscribeReasons"`
}

type summaryLink struct {
	Label string  `json:"label"`
	Share float64 `json:"shareOfClicks"`
}

type summaryABTest struct {
	Kind     string           `json:"kind"`
	Metric   string           `json:"metric"`
	Winner   string           `json:"winner,omitempty"`
	Variants []summaryVariant `json:"variants"`
}

type summaryVariant struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
	ClickRate     *float64 `json:"clickRate,omitempty"`
	Winner        bool     `json:"winner"`
}

type summaryMailbox struct {
	Name          string   `json:"name"`
	BounceRate    *float64 `json:"bounceRate,omitempty"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
}

type summaryReason struct {
	Reason string  `json:"reason"`
	Count  int64   `json:"count"`
	Share  float64 `json:"share"`
}

// summaryTask is a summary's request, as data.
type summaryTask struct {
	LanguageName string
	Numbers      summaryNumbers
}

func round1(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := math.Round(*p*10) / 10
	return &v
}

// numbersOf picks a report's aggregate numbers.
func numbersOf(r reports.CampaignReport) summaryNumbers {
	h := r.Headline
	n := summaryNumbers{Campaign: r.Campaign.Name, Sent: h.Sent, Delivered: h.Delivered, DeliveredRate: round1(h.DeliveredRate), HumanOpens: h.HumanOpens,
		HumanOpenRate: round1(h.HumanOpenRate), MachineOpens: h.MachineOpens, MachineOpenRate: round1(h.MachineOpenRate), Clicks: h.Clicks, ClickRate: round1(h.ClickRate),
		Unsubscribes: h.Unsubscribes, UnsubscribeRate: round1(h.UnsubscribeRate), Complaints: h.Complaints, Bounces: h.Bounces, BounceRate: round1(h.BounceRate),
		TopLinks: []summaryLink{}, Mailboxes: []summaryMailbox{}, Unsubscribe: []summaryReason{}}
	links := slices.Clone(r.ClickMap.Links)
	slices.SortStableFunc(links, func(a, b reports.ReportLink) int {
		switch {
		case a.Share > b.Share:
			return -1
		case a.Share < b.Share:
			return 1
		}
		return 0
	})
	for i, l := range links {
		if i == 3 {
			break
		}
		label := l.Label
		if label == "" {
			label = hostOf(l.URL)
		}
		n.TopLinks = append(n.TopLinks, summaryLink{Label: label, Share: math.Round(l.Share)})
	}
	if t := r.ABTest; t != nil && t.State == "decided" {
		ab := &summaryABTest{Kind: t.Kind, Metric: t.Metric, Winner: t.Winner, Variants: []summaryVariant{}}
		for _, v := range t.Variants {
			ab.Variants = append(ab.Variants, summaryVariant{Key: v.Key, Label: v.Label, HumanOpenRate: round1(v.HumanOpenRate), ClickRate: round1(v.ClickRate), Winner: v.Winner})
		}
		n.ABTest = ab
	}
	for _, m := range r.Mailboxes {
		n.Mailboxes = append(n.Mailboxes, summaryMailbox{Name: m.Name, BounceRate: round1(m.BounceRate), HumanOpenRate: round1(m.HumanOpenRate)})
	}
	for _, u := range r.Unsubscribes.Reasons {
		if u.Count > 0 {
			n.Unsubscribe = append(n.Unsubscribe, summaryReason{Reason: u.Label, Count: u.Count, Share: math.Round(u.Share)})
		}
	}
	return n
}

// allowedNumbers are the numbers a summary may write.
func allowedNumbers(n summaryNumbers) map[string]bool {
	set := map[string]bool{}
	count := func(v int64) { numberForms(set, float64(v), false) }
	rate := func(p *float64) {
		if p != nil {
			numberForms(set, *p, true)
		}
	}
	for _, v := range []int64{n.Sent, n.Delivered, n.HumanOpens, n.MachineOpens, n.Clicks, n.Unsubscribes, n.Complaints, n.Bounces} {
		count(v)
	}
	for _, p := range []*float64{n.DeliveredRate, n.HumanOpenRate, n.MachineOpenRate, n.ClickRate, n.UnsubscribeRate, n.BounceRate} {
		rate(p)
	}
	for _, l := range n.TopLinks {
		s := l.Share
		rate(&s)
	}
	if n.ABTest != nil {
		for _, v := range n.ABTest.Variants {
			rate(v.HumanOpenRate)
			rate(v.ClickRate)
		}
	}
	for _, m := range n.Mailboxes {
		rate(m.BounceRate)
		rate(m.HumanOpenRate)
	}
	for _, u := range n.Unsubscribe {
		count(u.Count)
		s := u.Share
		rate(&s)
	}
	return set
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
	Summary   string `json:"summary"`
	// NumbersMatch: every number is the report's; Unmatched lists the ones
	// that aren't.
	NumbersMatch bool     `json:"numbersMatch"`
	Unmatched    []string `json:"unmatched"`
	// NoCauses: it describes the numbers and doesn't diagnose causes.
	NoCauses bool   `json:"noCauses"`
	Sent     AISent `json:"sent"`
	Refusal  string `json:"refusal,omitempty"`
}

// Summarize asks for a plain-language summary of a report's numbers.
func (s *Service) Summarize(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIReportSummaryInput) (AIReportSummary, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return AIReportSummary{}, api.Invalid("body.campaignId", "Name the campaign.")
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
	nums := numbersOf(r)
	l := i18n.LangFrom(ctx)
	t := summaryTask{LanguageName: languageName(string(l.Locale)), Numbers: nums}
	raw, _ := json.Marshal(nums)
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
	out.Summary = strings.TrimSpace(plain(a.Summary))
	if out.Summary == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	allowed := allowedNumbers(nums)
	for _, v := range numbersIn(out.Summary) {
		if !allowed[v] && !slices.Contains(out.Unmatched, v) && !isYearOrSmall(v) {
			out.Unmatched = append(out.Unmatched, v)
		}
	}
	out.NumbersMatch = len(out.Unmatched) == 0
	out.NoCauses = !causeRe.MatchString(out.Summary)
	return out, nil
}

// isYearOrSmall leaves out what isn't a result: a year, a variant's
// number, a one-digit count of things ("two links").
func isYearOrSmall(v string) bool {
	if strings.HasSuffix(v, "%") {
		return false
	}
	n := atoi(v)
	return (n >= 1990 && n <= 2100 && len(v) == 4) || (n >= 0 && n < 10 && len(v) == 1)
}

// AIKeptSummaryView is a summary someone kept.
type AIKeptSummaryView struct {
	Summary   string    `json:"summary"`
	Model     string    `json:"model"`
	ModelName string    `json:"modelName"`
	KeptBy    string    `json:"keptBy"`
	KeptAt    time.Time `json:"keptAt"`
}

// AIKeptSummary is the report's kept summary, if any. HostRepute says
// whether "Ask HostRepute's assistant" shows.
type AIKeptSummary struct {
	Kept       *AIKeptSummaryView `json:"kept,omitempty"`
	HostRepute bool               `json:"hostRepute"`
}

// KeptSummary returns the summary kept on a campaign's report.
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
	out.Kept = &AIKeptSummaryView{Summary: row.Summary, Model: row.Model, ModelName: name, KeptBy: row.KeptByName, KeptAt: row.KeptAt}
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
// by AI, kept by …". It is a content change on the report (the activity
// log says Kept), never shown on read-only links.
func (s *Service) KeepSummary(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIKeepSummaryInput) (AIKeptSummary, error) {
	reqID, err := uuid.Parse(in.RequestID)
	if err != nil {
		return AIKeptSummary{}, api.Invalid("body.requestId", "Name the suggestion.")
	}
	r, err := s.Q.AIRequest(ctx, store.AIRequestParams{ID: reqID, WorkspaceID: ws.ID})
	uid := actorID(p)
	if err != nil || uid == nil || r.UserID == nil || *r.UserID != *uid || r.Feature != FeatureSummary || r.TargetID == nil || *r.TargetID != id {
		return AIKeptSummary{}, problem(ctx, http.StatusConflict, "ai_suggestion_gone", mSuggestionGone, nil)
	}
	if _, err := s.Reports.Report(ctx, ws, id); err != nil {
		return AIKeptSummary{}, err
	}
	name := r.Model
	if c, err := s.load(ctx, s.Q, ws.ID); err == nil && c != nil && c.Model == r.Model {
		name = c.modelName()
	}
	if err := s.Q.AIReportSummarySave(ctx, store.AIReportSummarySaveParams{CampaignID: id, WorkspaceID: ws.ID, Summary: strings.TrimSpace(in.Summary), Model: r.Model,
		ModelName: name, RequestID: &reqID, KeptBy: uid, KeptAt: s.now()}); err != nil {
		return AIKeptSummary{}, err
	}
	outcome := OutcomeKept
	detail := ""
	if in.Edited {
		detail = "with edits"
	}
	s.finish(ctx, ws, p, reqID, outcome, detail)
	return s.KeptSummary(ctx, ws, id)
}

// RemoveSummary takes a kept summary off the report.
func (s *Service) RemoveSummary(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	n, err := s.Q.AIReportSummaryDelete(ctx, store.AIReportSummaryDeleteParams{CampaignID: id, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	if n == 0 {
		return api.NotFound("Summary")
	}
	return nil
}
