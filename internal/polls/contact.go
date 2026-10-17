package polls

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// A contact's answers on their profile: in the activity timeline (poll
// answers and survey responses, as audience.ActivitySource) and as where a
// field's value came from.

// contactEvents are a contact's poll answers and survey responses as
// timeline items, newest first.
func (s *Service) contactEvents(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) ([]audience.ContactActivityItem, error) {
	out := []audience.ContactActivityItem{}
	cid := contactID
	answers, err := s.Q.PollContactAnswers(ctx, store.PollContactAnswersParams{ContactID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	for _, a := range answers {
		cfg, list := configOf("", a.Kind, a.Question, a.Answers, a.Settings)
		label := map[int]string{}
		for _, x := range list {
			label[x.Key] = x.Label
			if x.Emoji != "" && cfg.Kind == KindChoice {
				label[x.Key] = x.Emoji + " " + x.Label
			}
		}
		it := audience.ContactActivityItem{ID: a.ID.String(), Category: "engagement", Type: "poll_answer", At: a.UpdatedAt, Subject: a.CampaignName,
			Question: a.Question, Answer: label[int(a.Answer)], Comment: a.Comment}
		if a.Changes > 0 {
			if prev, err := s.Q.PollAnswerPrevious(ctx, a.ID); err == nil && prev != nil {
				it.Previous = label[int(*prev)]
			}
		}
		out = append(out, it)
	}
	responses, err := s.Q.SurveyContactResponses(ctx, store.SurveyContactResponsesParams{ContactID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	for _, r := range responses {
		qs, st := decodeSurvey(r.Questions, r.Settings)
		var given map[string]any
		_ = json.Unmarshal(r.Answers, &given)
		n := 0
		for _, q := range qs {
			if v, ok := given[q.Key]; ok && v != nil {
				n++
			}
		}
		at := r.UpdatedAt
		it := audience.ContactActivityItem{ID: r.ID.String(), Category: "engagement", Type: "survey_response", At: at, Subject: r.Name,
			Answered: n, Questions: len(qs)}
		if r.CompletedAt == nil {
			it.Type = "survey_started"
		}
		if st.OneResponse && r.ClosesAt != nil && s.now().Before(*r.ClosesAt) && r.SurveyStatus == SurveyOpen {
			it.EditableUntil = r.ClosesAt
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

// ContactActivity is audience.ActivitySource: the answers before the
// cursor, newest first, and how many there are.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	all, err := s.contactEvents(ctx, ws, contactID)
	if err != nil {
		return nil, 0, err
	}
	out := []audience.ContactActivityItem{}
	for _, it := range all {
		if before != nil && !it.At.Before(*before) && (!it.At.Equal(*before) || it.ID >= beforeID) {
			continue
		}
		if len(out) < limit {
			out = append(out, it)
		}
	}
	return out, int64(len(all)), nil
}

// FieldSource says where a contact field's value came from.
type FieldSource struct {
	Field string    `json:"field"`
	Kind  string    `json:"kind" enum:"poll,survey"`
	Name  string    `json:"name"`
	At    time.Time `json:"at"`
}

// FieldSources are the fields of a contact that a poll or a survey set.
type FieldSources struct {
	Items []FieldSource `json:"items"`
}

// ContactFieldSources finds the fields a contact's answers set: the latest
// answer that saved to a field names it.
func (s *Service) ContactFieldSources(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (FieldSources, error) {
	out := FieldSources{Items: []FieldSource{}}
	seen := map[string]bool{}
	cid := contactID
	answers, err := s.Q.PollContactAnswers(ctx, store.PollContactAnswersParams{ContactID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	var all []FieldSource
	for _, a := range answers {
		cfg, _ := configOf("", a.Kind, a.Question, a.Answers, a.Settings)
		if cfg.Save == SaveField && strings.TrimSpace(cfg.Field) != "" {
			all = append(all, FieldSource{Field: strings.TrimSpace(cfg.Field), Kind: "poll", Name: a.CampaignName, At: a.UpdatedAt})
		}
	}
	responses, err := s.Q.SurveyContactResponses(ctx, store.SurveyContactResponsesParams{ContactID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, r := range responses {
		if r.CompletedAt == nil {
			continue
		}
		qs, _ := decodeSurvey(r.Questions, r.Settings)
		var given map[string]any
		_ = json.Unmarshal(r.Answers, &given)
		for _, q := range qs {
			if q.Save.To == "field" && strings.TrimSpace(q.Save.Field) != "" && given[q.Key] != nil {
				all = append(all, FieldSource{Field: strings.TrimSpace(q.Save.Field), Kind: "survey", Name: r.Name, At: r.UpdatedAt})
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].At.After(all[j].At) })
	for _, f := range all {
		if !seen[f.Field] {
			seen[f.Field] = true
			out.Items = append(out.Items, f)
		}
	}
	return out, nil
}
