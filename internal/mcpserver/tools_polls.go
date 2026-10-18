package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/store"
)

// PollReader reads poll and survey results (design expansion, X11).
type PollReader interface {
	CampaignPolls(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (polls.PollResultsList, error)
	Surveys(ctx context.Context, ws *store.Workspace) (polls.SurveyList, error)
	SurveyResultsOf(ctx context.Context, ws *store.Workspace, id uuid.UUID) (polls.SurveyResults, error)
}

type pollResultsIn struct {
	Campaign string `json:"campaign,omitempty" jsonschema:"the campaign's name (exact) or id, for the results of its polls"`
	Survey   string `json:"survey,omitempty" jsonschema:"a survey's name (exact) or id, for its results"`
}

// addPollTools adds get_poll_results: aggregate results only, never who
// answered or what they wrote.
func (s *Service) addPollTools(srv *mcp.Server) {
	tool(s, srv, "get_poll_results", authz.ReportsView, func(ctx context.Context, c *agentCall, in pollResultsIn) (any, entry, error) {
		e := entry{action: "poll.results_read", targetType: "poll", targetName: firstNonEmpty(in.Campaign, in.Survey)}
		if s.Polls == nil {
			return map[string]any{"message": "Polls aren't available on this server."}, e, nil
		}
		if strings.TrimSpace(in.Survey) != "" {
			return s.surveyResults(ctx, c, in.Survey, e)
		}
		if strings.TrimSpace(in.Campaign) == "" {
			return nil, e, errors.New("name a campaign (its polls) or a survey")
		}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetType, e.targetID, e.targetName = "campaign", cp.ID.String(), cp.Name
		list, err := s.Polls.CampaignPolls(ctx, &c.Workspace, c.Principal, cp.ID)
		if err != nil {
			return nil, e, err
		}
		type answer struct {
			Answer  string  `json:"answer"`
			Count   int64   `json:"count"`
			Percent float64 `json:"percent"`
		}
		type row struct {
			Name    string `json:"name"`
			Answers int64  `json:"answers"`
			NPS     *int   `json:"nps,omitempty"`
			TooFew  bool   `json:"too_few_to_compare,omitempty"`
		}
		out := []map[string]any{}
		for _, p := range list.Items {
			answers := []answer{}
			for _, a := range p.Counts {
				answers = append(answers, answer{Answer: a.Label, Count: a.Count, Percent: a.Percent})
			}
			tiers := []row{}
			for _, t := range p.Breakdown.Tiers {
				tiers = append(tiers, row{Name: t.Label, Answers: t.Answers, NPS: t.NPS, TooFew: t.TooFew})
			}
			langs := []row{}
			for _, l := range p.Breakdown.Languages {
				langs = append(langs, row{Name: l.Label, Answers: l.Answers, NPS: l.NPS, TooFew: l.TooFew})
			}
			m := map[string]any{"question": p.Question, "kind": p.Kind, "status": p.Status, "closes_at": p.ClosesAt, "closed_at": p.ClosedAt,
				"answers": p.Answers, "delivered": p.Delivered, "percent_of_delivered": p.Percent, "by_answer": answers, "by_tier": tiers, "by_language": langs,
				"answers_changed": p.Changed, "comments": p.Comments, "non_contacts": p.NonContacts,
				"automated_visits_ignored": map[string]int64{"total": p.Automated.Total, "security_gateways": p.Automated.Gateways, "prefetching_proxies": p.Automated.Prefetch}}
			if p.NPS != nil {
				m["nps"] = map[string]any{"score": p.NPS.Score, "promoters": p.NPS.Promoters, "passives": p.NPS.Passives, "detractors": p.NPS.Detractors}
			}
			out = append(out, m)
		}
		return map[string]any{"campaign": cp.Name, "polls": out,
			"note": "Answers are recorded on the answer page, never on the click; automated visits, answers from people who aren't contacts and test answers aren't counted as answers."}, e, nil
	}, nil)
}

func (s *Service) surveyResults(ctx context.Context, c *agentCall, ref string, e entry) (any, entry, error) {
	list, err := s.Polls.Surveys(ctx, &c.Workspace)
	if err != nil {
		return nil, e, err
	}
	ref = strings.TrimSpace(ref)
	var found *polls.Survey
	for i, sv := range list.Items {
		if sv.ID == ref || strings.EqualFold(sv.Name, ref) {
			found = &list.Items[i]
		}
	}
	if found == nil {
		return nil, e, errors.New("no survey named or with id \"" + ref + "\" in this workspace")
	}
	e.targetType, e.targetID, e.targetName = "survey", found.ID, found.Name
	id, _ := uuid.Parse(found.ID)
	r, err := s.Polls.SurveyResultsOf(ctx, &c.Workspace, id)
	if err != nil {
		return nil, e, err
	}
	qs := []map[string]any{}
	for _, q := range r.Questions {
		m := map[string]any{"number": q.Number, "question": q.Text, "type": q.Type, "answered": q.Answered}
		counts := []map[string]any{}
		for _, a := range q.Counts {
			counts = append(counts, map[string]any{"answer": a.Label, "count": a.Count, "percent": a.Percent})
		}
		if len(counts) > 0 {
			m["by_answer"] = counts
		}
		if q.NPS != nil {
			m["nps"] = q.NPS.Score
		}
		qs = append(qs, m)
	}
	return map[string]any{"survey": found.Name, "status": found.Status, "started": r.Started, "completed": r.Completed,
		"completion_percent": r.CompletionPercent, "answered_question_1_only": r.PollOnly, "questions": qs,
		"note": "Aggregate results only; text answers and who answered aren't shared with agents."}, e, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
