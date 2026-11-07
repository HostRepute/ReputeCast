package polls

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// SurveyQuestionResult is one question's results over the completed
// responses.
type SurveyQuestionResult struct {
	Key      string            `json:"key"`
	Number   int               `json:"number"`
	Type     string            `json:"type"`
	Text     string            `json:"text"`
	Required bool              `json:"required"`
	Answered int64             `json:"answered"`
	Counts   []PollAnswerCount `json:"counts"`
	NPS      *PollNPS          `json:"nps,omitempty"`
	// SavedTo is the field the answers fill, when they do.
	SavedTo string `json:"savedTo,omitempty"`
}

// SurveyResults are a survey's completion and per-question results.
type SurveyResults struct {
	Started           int64   `json:"started"`
	Completed         int64   `json:"completed"`
	CompletionPercent float64 `json:"completionPercent"`
	// PollOnly answered question 1 in the email's poll and stopped there.
	PollOnly int64 `json:"pollOnly"`
	// Recipients of the emails that link to it.
	Recipients  int64                  `json:"recipients"`
	NonContacts int64                  `json:"nonContacts"`
	Questions   []SurveyQuestionResult `json:"questions"`
}

// SurveyResultsOf counts a survey's responses.
func (s *Service) SurveyResultsOf(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SurveyResults, error) {
	sv, err := s.Q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SurveyResults{}, api.MapError(err, "Survey")
	}
	qs, _ := decodeSurvey(sv.Questions, sv.Settings)
	// Counted in SQL, however many responses there are (review 2026-11 AP-2).
	totals, err := s.Q.SurveyResponseTotals(ctx, sv.ID)
	if err != nil {
		return SurveyResults{}, err
	}
	out := SurveyResults{Questions: []SurveyQuestionResult{}, Started: totals.Started, Completed: totals.Completed, NonContacts: totals.NonContacts, PollOnly: totals.PollOnly}
	byKey := make(map[string]int, len(qs))
	counts := make([]map[int]int64, len(qs))
	answered := make([]int64, len(qs))
	for i, q := range qs {
		byKey[q.Key], counts[i] = i, map[int]int64{}
	}
	ans, err := s.Q.SurveyAnsweredCounts(ctx, sv.ID)
	if err != nil {
		return SurveyResults{}, err
	}
	for _, a := range ans {
		if i, ok := byKey[a.QuestionKey]; ok {
			answered[i] = a.N
		}
	}
	picks, err := s.Q.SurveyPickCounts(ctx, sv.ID)
	if err != nil {
		return SurveyResults{}, err
	}
	for _, p := range picks {
		if i, ok := byKey[p.QuestionKey]; ok && qs[i].Type != QuestionShort && qs[i].Type != QuestionLong {
			counts[i][int(p.Pick)] += p.N
		}
	}
	out.CompletionPercent = percent(out.Completed, out.Started)
	for i, q := range qs {
		res := SurveyQuestionResult{Key: q.Key, Number: i + 1, Type: q.Type, Text: q.Text, Required: q.Required, Answered: answered[i], Counts: []PollAnswerCount{}}
		if q.Save.To == "field" {
			res.SavedTo = q.Save.Field
		}
		switch q.Type {
		case QuestionShort, QuestionLong:
		default:
			for _, o := range QuestionOptions(q) {
				res.Counts = append(res.Counts, PollAnswerCount{Key: o.Key, Label: o.Label, Tag: o.Tag, Count: counts[i][o.Key], Percent: percent(counts[i][o.Key], answered[i])})
			}
			if q.Type == QuestionScale {
				res.NPS = npsOf(counts[i])
			}
		}
		out.Questions = append(out.Questions, res)
	}
	links, err := s.Q.SurveyLinkedCampaigns(ctx, store.SurveyLinkedCampaignsParams{WorkspaceID: ws.ID, VarPattern: SurveyVar(sv.Number) + `([^0-9]|$)`, SurveyID: sv.ID.String()})
	if err == nil {
		for _, l := range links {
			if l.StartedAt != nil && l.AudienceCount != nil {
				out.Recipients += *l.AudienceCount
			}
		}
	}
	return out, nil
}

// SurveyResponse is one completed response.
type SurveyResponse struct {
	ID          string     `json:"id"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// Contact is the contact's name for people with subscribers:view.
	Contact    string `json:"contact,omitempty"`
	ContactID  string `json:"contactId,omitempty"`
	NonContact bool   `json:"nonContact"`
	// Answers are the answers in words, one per question ("" unanswered).
	Answers []string `json:"answers"`
}

// SurveyResponseList is a page of completed responses, newest first.
type SurveyResponseList struct {
	Items []SurveyResponse `json:"items"`
	Total int64            `json:"total"`
	Names bool             `json:"names"`
}

// Responses lists a survey's completed responses.
func (s *Service) Responses(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, offset, limit int) (SurveyResponseList, error) {
	sv, err := s.Q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SurveyResponseList{}, api.MapError(err, "Survey")
	}
	qs, _ := decodeSurvey(sv.Questions, sv.Settings)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	names := p.Can(authz.SubscribersView)
	out := SurveyResponseList{Items: []SurveyResponse{}, Total: sv.Completed, Names: names}
	rows, err := s.Q.SurveyResponses(ctx, store.SurveyResponsesParams{SurveyID: sv.ID, ScopeLists: access.ScopeLists(p), Lim: int32(limit), Off: int32(max(offset, 0))})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		var answers map[string]any
		_ = json.Unmarshal(r.Answers, &answers)
		item := SurveyResponse{ID: r.ID.String(), CompletedAt: r.CompletedAt, NonContact: r.Kind == AnswererNonContact, Answers: make([]string, 0, len(qs))}
		// Named only when the contact is on one of the caller's lists, for a
		// member, key or agent limited to some.
		if names && r.ContactID != nil && r.InScope {
			item.Contact = strings.TrimSpace(r.FirstName + " " + r.LastName)
			if item.Contact == "" {
				item.Contact = r.Email
			}
			item.ContactID = r.ContactID.String()
		}
		for _, q := range qs {
			item.Answers = append(item.Answers, AnswerText(q, answers[q.Key]))
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// SurveyTextAnswer is one text answer.
type SurveyTextAnswer struct {
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
	Contact string    `json:"contact,omitempty"`
}

// SurveyTextAnswerList is a page of a text question's answers.
type SurveyTextAnswerList struct {
	Items []SurveyTextAnswer `json:"items"`
	Total int64              `json:"total"`
	Names bool               `json:"names"`
}

// TextAnswers lists a text question's answers, newest first.
func (s *Service) TextAnswers(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, key, search string, offset, limit int) (SurveyTextAnswerList, error) {
	sv, err := s.Q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SurveyTextAnswerList{}, api.MapError(err, "Survey")
	}
	qs, _ := decodeSurvey(sv.Questions, sv.Settings)
	if !slices.ContainsFunc(qs, func(q SurveyQuestion) bool {
		return q.Key == key && (q.Type == QuestionShort || q.Type == QuestionLong)
	}) {
		return SurveyTextAnswerList{}, api.NotFound("Question")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	names := p.Can(authz.SubscribersView)
	out := SurveyTextAnswerList{Items: []SurveyTextAnswer{}, Names: names}
	// Searched and paged in SQL (review 2026-11 AP-2).
	if offset < 0 {
		offset = 0
	}
	needle := strings.TrimSpace(search)
	total, err := s.Q.SurveyTextAnswerCount(ctx, store.SurveyTextAnswerCountParams{SurveyID: sv.ID, QuestionKey: key, Search: needle})
	if err != nil {
		return out, err
	}
	rows, err := s.Q.SurveyTextAnswers(ctx, store.SurveyTextAnswersParams{SurveyID: sv.ID, QuestionKey: key, Search: needle, ScopeLists: access.ScopeLists(p), Lim: int32(limit), Off: int32(offset)})
	if err != nil {
		return out, err
	}
	out.Total = total
	for _, r := range rows {
		a := SurveyTextAnswer{Text: r.Answer}
		if r.CompletedAt != nil {
			a.At = *r.CompletedAt
		}
		if names && r.ContactID != nil && r.InScope {
			a.Contact = strings.TrimSpace(r.FirstName + " " + r.LastName)
			if a.Contact == "" {
				a.Contact = r.Email
			}
		}
		out.Items = append(out.Items, a)
	}
	return out, nil
}
