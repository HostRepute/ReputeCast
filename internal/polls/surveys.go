package polls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Survey question types.
const (
	QuestionSingle = "single"
	QuestionMulti  = "multi"
	QuestionScale  = "scale"
	QuestionEmoji  = "emoji"
	QuestionShort  = "short"
	QuestionLong   = "long"
)

// MaxQuestions is how many questions a survey takes, all on one page.
const MaxQuestions = 20

// Survey statuses.
const (
	SurveyDraft  = "draft"
	SurveyOpen   = "open"
	SurveyClosed = "closed"
)

// SurveyOption is one answer of a choice question.
type SurveyOption struct {
	Key   int    `json:"key" minimum:"1"`
	Label string `json:"label" maxLength:"200"`
	// Tag is added to the contact who picks it, when the question saves tags.
	Tag string `json:"tag,omitempty" maxLength:"64"`
}

// SurveySave says where a question's answer goes on the contact.
type SurveySave struct {
	To    string `json:"to" enum:"none,field,tags"`
	Field string `json:"field,omitempty" maxLength:"64"`
}

// SurveyQuestion is one question of a survey.
type SurveyQuestion struct {
	Key         string         `json:"key" pattern:"^q[0-9]{1,3}$"`
	Type        string         `json:"type" enum:"single,multi,scale,emoji,short,long"`
	Text        string         `json:"text" maxLength:"500"`
	Description string         `json:"description,omitempty" maxLength:"1000"`
	Required    bool           `json:"required"`
	Answers     []SurveyOption `json:"answers"`
	Save        SurveySave     `json:"save"`
	// NPS wording of a 0–10 scale, with its ends' words.
	NPS  bool   `json:"nps,omitempty"`
	Low  string `json:"low,omitempty" maxLength:"60"`
	High string `json:"high,omitempty" maxLength:"60"`
	// Faces of an emoji scale: three or five.
	Faces int `json:"faces,omitempty"`
	// RequiredText is said under a required question left empty ("Pick
	// the one closest to your job."); empty uses the type's sentence.
	RequiredText string `json:"requiredText,omitempty" maxLength:"200"`
}

// SurveyClose is when a survey stops taking answers.
type SurveyClose struct {
	Mode string `json:"mode" enum:"date,manual"`
	// Date and Time are local to the survey's time zone ("2026-10-31", "23:59").
	Date string `json:"date,omitempty"`
	Time string `json:"time,omitempty"`
}

// SurveySettings are a survey's settings.
type SurveySettings struct {
	Close    SurveyClose `json:"close"`
	Timezone string      `json:"timezone,omitempty" maxLength:"64"`
	// OneResponse keeps one response per contact, editable until it closes.
	OneResponse bool `json:"oneResponse"`
	// AcceptNonContacts takes answers from links without a contact's token.
	AcceptNonContacts bool   `json:"acceptNonContacts"`
	Thanks            string `json:"thanks,omitempty" maxLength:"1000"`
}

// SurveyProblem is what keeps a survey from being published.
type SurveyProblem struct {
	Question string `json:"question,omitempty"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// SurveyLink is a campaign that links to a survey.
type SurveyLink struct {
	CampaignID string     `json:"campaignId"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	SentAt     *time.Time `json:"sentAt,omitempty"`
	Recipients *int64     `json:"recipients,omitempty"`
	// Via is how the email links to it: a link, or a poll that continues to it.
	Via string `json:"via" enum:"link,poll"`
}

// Survey is a survey with its counts.
type Survey struct {
	ID          string           `json:"id"`
	Number      int64            `json:"number"`
	Slug        string           `json:"slug"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      string           `json:"status" enum:"draft,open,closed"`
	Questions   []SurveyQuestion `json:"questions"`
	Settings    SurveySettings   `json:"settings"`
	ClosesAt    *time.Time       `json:"closesAt,omitempty"`
	PublishedAt *time.Time       `json:"publishedAt,omitempty"`
	PublishedBy string           `json:"publishedBy,omitempty"`
	ClosedAt    *time.Time       `json:"closedAt,omitempty"`
	ClosedBy    string           `json:"closedBy,omitempty"`
	CreatedBy   string           `json:"createdBy"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	// URL is the public page; Variable the link variable for emails.
	URL      string `json:"url"`
	Variable string `json:"variable"`
	// Started and Completed count responses.
	Started   int64           `json:"started"`
	Completed int64           `json:"completed"`
	Linked    []SurveyLink    `json:"linked"`
	Problems  []SurveyProblem `json:"problems"`
	// CanEdit is the viewer's forms:edit; EditHelpers are members to ask
	// when they lack it.
	CanEdit     bool     `json:"canEdit"`
	EditHelpers []string `json:"editHelpers"`
}

// SurveyList is the workspace's surveys.
type SurveyList struct {
	Items []Survey `json:"items"`
}

func decodeSurvey(questions, settings json.RawMessage) ([]SurveyQuestion, SurveySettings) {
	qs := []SurveyQuestion{}
	_ = json.Unmarshal(questions, &qs)
	var st SurveySettings
	_ = json.Unmarshal(settings, &st)
	for i := range qs {
		if qs[i].Answers == nil {
			qs[i].Answers = []SurveyOption{}
		}
	}
	return qs, st
}

// surveyClosesAt is the closing moment of the settings.
func surveyClosesAt(st SurveySettings, wsTZ string) *time.Time {
	if st.Close.Mode != "date" || st.Close.Date == "" {
		return nil
	}
	tz := st.Timezone
	if tz == "" {
		tz = wsTZ
	}
	clock := st.Close.Time
	if clock == "" {
		clock = "23:59"
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", st.Close.Date+" "+clock, state.Location(tz))
	if err != nil {
		return nil
	}
	return &t
}

// Problems lists what keeps a survey from being published.
func SurveyProblems(qs []SurveyQuestion) []SurveyProblem {
	out := []SurveyProblem{}
	if len(qs) == 0 {
		out = append(out, SurveyProblem{Field: "questions", Message: "Add the first question."})
	}
	if len(qs) > MaxQuestions {
		out = append(out, SurveyProblem{Field: "questions", Message: fmt.Sprintf("A survey takes up to %d questions.", MaxQuestions)})
	}
	for _, q := range qs {
		if strings.TrimSpace(q.Text) == "" {
			out = append(out, SurveyProblem{Question: q.Key, Field: "text", Message: "Write the question."})
		}
		if q.Type == QuestionSingle || q.Type == QuestionMulti {
			n := 0
			for _, a := range q.Answers {
				if strings.TrimSpace(a.Label) != "" {
					n++
				}
			}
			if n < 2 {
				out = append(out, SurveyProblem{Question: q.Key, Field: "answers", Message: "A choice question needs at least two answers."})
			}
		}
		switch q.Save.To {
		case "field":
			if strings.TrimSpace(q.Save.Field) == "" {
				out = append(out, SurveyProblem{Question: q.Key, Field: "save", Message: "Pick the field the answer is saved to."})
			}
		case "tags":
			if q.Type == QuestionSingle || q.Type == QuestionMulti {
				for _, a := range q.Answers {
					if strings.TrimSpace(a.Label) != "" && strings.TrimSpace(a.Tag) == "" {
						out = append(out, SurveyProblem{Question: q.Key, Field: "save", Message: "Give every answer a tag, or don't save the answer."})
						break
					}
				}
			}
		}
	}
	return out
}

// QuestionOptions are the answers a question takes, keys included: a
// scale's 0–10, an emoji scale's faces, a choice's answers.
func QuestionOptions(q SurveyQuestion) []SurveyOption {
	switch q.Type {
	case QuestionScale:
		out := make([]SurveyOption, 0, 11)
		for i := 0; i <= 10; i++ {
			out = append(out, SurveyOption{Key: i, Label: strconv.Itoa(i)})
		}
		return out
	case QuestionEmoji:
		faces := q.Faces
		if faces != 3 {
			faces = 5
		}
		out := make([]SurveyOption, 0, faces)
		for i := 1; i <= faces; i++ {
			out = append(out, SurveyOption{Key: i, Label: fmt.Sprintf("%d of %d, %s", i, faces, FaceWord(faces, i))})
		}
		return out
	}
	return q.Answers
}

func (s *Service) surveyOf(ctx context.Context, ws *store.Workspace, r store.SurveyGetRow) Survey {
	qs, st := decodeSurvey(r.Questions, r.Settings)
	out := Survey{ID: r.ID.String(), Number: r.Number, Slug: r.Slug, Name: r.Name, Description: r.Description, Status: r.Status, Questions: qs, Settings: st,
		ClosesAt: r.ClosesAt, PublishedAt: r.PublishedAt, PublishedBy: r.PublishedByName, ClosedAt: r.ClosedAt, ClosedBy: r.ClosedByName, CreatedBy: r.CreatedByName,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Started: r.Started, Completed: r.Completed, Variable: "{{ " + SurveyVar(r.Number) + " }}",
		Linked: []SurveyLink{}, Problems: SurveyProblems(qs), EditHelpers: []string{}}
	out.URL = s.pageLinks(ctx, ws).Survey(r.Slug)
	if out.Status == SurveyOpen && out.ClosesAt != nil && !s.now().Before(*out.ClosesAt) {
		out.Status = SurveyClosed
		out.ClosedAt = out.ClosesAt
	}
	links, err := s.Q.SurveyLinkedCampaigns(ctx, store.SurveyLinkedCampaignsParams{WorkspaceID: ws.ID, VarPattern: SurveyVar(r.Number) + `([^0-9]|$)`, SurveyID: r.ID.String()})
	if err == nil {
		for _, l := range links {
			via := "link"
			if l.ViaPoll {
				via = "poll"
			}
			out.Linked = append(out.Linked, SurveyLink{CampaignID: l.ID.String(), Name: l.Name, Status: l.Status, SentAt: l.StartedAt, Recipients: l.AudienceCount, Via: via})
		}
	}
	return out
}

// Surveys lists the workspace's surveys, newest first.
func (s *Service) Surveys(ctx context.Context, ws *store.Workspace) (SurveyList, error) {
	rows, err := s.Q.SurveysList(ctx, ws.ID)
	if err != nil {
		return SurveyList{}, err
	}
	out := SurveyList{Items: make([]Survey, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, s.surveyOf(ctx, ws, store.SurveyGetRow{ID: r.ID, WorkspaceID: ws.ID, Number: r.Number, Slug: r.Slug, Name: r.Name, Description: r.Description,
			Status: r.Status, Questions: r.Questions, Settings: r.Settings, ClosesAt: r.ClosesAt, PublishedAt: r.PublishedAt, ClosedAt: r.ClosedAt, CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt, CreatedByName: r.CreatedByName, Started: r.Started, Completed: r.Completed}))
	}
	return out, nil
}

// GetSurvey returns one survey.
func (s *Service) GetSurvey(ctx context.Context, ws *store.Workspace, id uuid.UUID) (Survey, error) {
	r, err := s.Q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return Survey{}, api.MapError(err, "Survey")
	}
	return s.surveyOf(ctx, ws, r), nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a name into a page address: "Beta feedback · October 2026"
// → "beta-feedback-october-2026".
func slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "survey"
	}
	return s
}

// SurveyCreate starts a survey.
type SurveyCreate struct {
	Name string `json:"name,omitempty" maxLength:"200"`
}

// CreateSurvey makes a draft survey with no questions yet.
func (s *Service) CreateSurvey(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SurveyCreate) (Survey, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Untitled survey"
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	st := SurveySettings{Close: SurveyClose{Mode: "manual"}, Timezone: ws.Timezone, OneResponse: true, AcceptNonContacts: false,
		Thanks: "Your answers are saved."}
	sb, _ := json.Marshal(st)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		slug, err := s.freeSlug(ctx, q, ws.ID, id, slugify(name))
		if err != nil {
			return err
		}
		by := userOf(p)
		if err := q.SurveyInsert(ctx, store.SurveyInsertParams{ID: id, WorkspaceID: ws.ID, Slug: slug, Name: name, Questions: []byte(`[]`), Settings: sb, CreatedBy: by, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.created", TargetType: "survey", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "slug": slug}})
	})
	if err != nil {
		return Survey{}, err
	}
	return s.GetSurvey(ctx, ws, id)
}

// freeSlug is slug, or slug-2, slug-3… when the workspace has it already.
func (s *Service) freeSlug(ctx context.Context, q *store.Queries, wsID, id uuid.UUID, slug string) (string, error) {
	try := slug
	for i := 2; i < 100; i++ {
		taken, err := q.SurveySlugTaken(ctx, store.SurveySlugTakenParams{WorkspaceID: wsID, Slug: try, ID: id})
		if err != nil {
			return "", err
		}
		if !taken {
			return try, nil
		}
		try = slug + "-" + strconv.Itoa(i)
	}
	return slug + "-" + uuid.NewString()[:8], nil
}

// SurveyUpdate changes a survey: name, address, description, questions and
// settings. Omitted fields stay.
type SurveyUpdate struct {
	Name        *string          `json:"name,omitempty" minLength:"1" maxLength:"200"`
	Slug        *string          `json:"slug,omitempty" pattern:"^[a-z0-9][a-z0-9-]{0,59}$"`
	Description *string          `json:"description,omitempty" maxLength:"1000"`
	Questions   []SurveyQuestion `json:"questions,omitempty" maxItems:"20"`
	Settings    *SurveySettings  `json:"settings,omitempty"`
}

// UpdateSurvey saves a survey's changes. Settings changes are audited with
// before and after; editing an open survey applies to new responses.
func (s *Service) UpdateSurvey(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SurveyUpdate) (Survey, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Survey")
		}
		qs, st := decodeSurvey(cur.Questions, cur.Settings)
		name, slug, desc := cur.Name, cur.Slug, cur.Description
		before, after := map[string]any{}, map[string]any{}
		if in.Name != nil && strings.TrimSpace(*in.Name) != name {
			before["name"], after["name"] = name, strings.TrimSpace(*in.Name)
			name = strings.TrimSpace(*in.Name)
		}
		if in.Slug != nil && *in.Slug != slug {
			if taken, err := q.SurveySlugTaken(ctx, store.SurveySlugTakenParams{WorkspaceID: ws.ID, Slug: *in.Slug, ID: id}); err != nil {
				return err
			} else if taken {
				return api.Invalid("body.slug", "Another survey of this workspace has that address. Pick another.")
			}
			before["slug"], after["slug"] = slug, *in.Slug
			slug = *in.Slug
		}
		if in.Description != nil && *in.Description != desc {
			before["description"], after["description"] = desc, *in.Description
			desc = *in.Description
		}
		if in.Questions != nil {
			next, err := normalizeQuestions(in.Questions)
			if err != nil {
				return err
			}
			if !sameJSON(qs, next) {
				before["questions"], after["questions"] = len(qs), len(next)
				qs = next
			}
		}
		if in.Settings != nil {
			next := *in.Settings
			if next.Close.Mode != "date" {
				next.Close = SurveyClose{Mode: "manual"}
			}
			if !sameJSON(st, next) {
				before["settings"], after["settings"] = st, next
				st = next
			}
		}
		if len(after) == 0 {
			return nil
		}
		// An open survey's page takes answers: it keeps every question
		// complete, as publishing does.
		if cur.Status == SurveyOpen {
			if probs := SurveyProblems(qs); len(probs) > 0 {
				return api.Err(422, "survey_incomplete", "Publishing waits until every question is complete: "+probs[0].Message).WithMeta("problems", probs)
			}
		}
		qb, _ := json.Marshal(qs)
		sb, _ := json.Marshal(st)
		by := userOf(p)
		if err := q.SurveyUpdate(ctx, store.SurveyUpdateParams{ID: id, Slug: slug, Name: name, Description: desc, Questions: qb, Settings: sb,
			ClosesAt: surveyClosesAt(st, ws.Timezone), UpdatedBy: by, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.updated", TargetType: "survey", TargetID: id.String(), TargetName: name,
			Before: before, After: after})
	})
	if err != nil {
		return Survey{}, err
	}
	return s.GetSurvey(ctx, ws, id)
}

// userOf is the person behind an action, when there is one.
func userOf(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	u := p.UserID
	return &u
}

func sameJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// normalizeQuestions numbers questions and answers and checks their types.
func normalizeQuestions(in []SurveyQuestion) ([]SurveyQuestion, error) {
	if len(in) > MaxQuestions {
		return nil, api.Invalid("body.questions", fmt.Sprintf("A survey takes up to %d questions.", MaxQuestions))
	}
	out := make([]SurveyQuestion, 0, len(in))
	keys := map[string]bool{}
	next := 1
	for _, q := range in {
		if q.Key == "" || keys[q.Key] {
			for keys["q"+strconv.Itoa(next)] {
				next++
			}
			q.Key = "q" + strconv.Itoa(next)
		}
		keys[q.Key] = true
		switch q.Type {
		case QuestionSingle, QuestionMulti, QuestionScale, QuestionEmoji, QuestionShort, QuestionLong:
		default:
			return nil, api.Invalid("body.questions", "Pick a type for every question: single choice, multiple choice, 0–10 scale, emoji scale, short or long text.")
		}
		if q.Type == QuestionSingle || q.Type == QuestionMulti {
			used := map[int]bool{}
			for i := range q.Answers {
				if q.Answers[i].Key <= 0 || used[q.Answers[i].Key] {
					k := 1
					for used[k] {
						k++
					}
					q.Answers[i].Key = k
				}
				used[q.Answers[i].Key] = true
			}
		} else {
			q.Answers = []SurveyOption{}
		}
		if q.Answers == nil {
			q.Answers = []SurveyOption{}
		}
		switch q.Save.To {
		case "field", "tags":
		default:
			q.Save = SurveySave{To: "none"}
		}
		if q.Type == QuestionEmoji && q.Faces != 3 {
			q.Faces = 5
		}
		out = append(out, q)
	}
	return out, nil
}

// PublishSurvey opens a survey's page. Every question must be complete.
func (s *Service) PublishSurvey(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (Survey, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Survey")
		}
		qs, st := decodeSurvey(cur.Questions, cur.Settings)
		if probs := SurveyProblems(qs); len(probs) > 0 {
			return api.Err(422, "survey_incomplete", "Publishing waits until every question is complete: "+probs[0].Message).WithMeta("problems", probs)
		}
		now := s.now()
		closes := surveyClosesAt(st, ws.Timezone)
		if closes != nil && !now.Before(*closes) {
			return api.Invalid("body.settings.close", "The closing date has passed. Pick a later date, or close the survey by hand.")
		}
		by := userOf(p)
		if err := q.SurveySetStatus(ctx, store.SurveySetStatusParams{ID: id, Status: SurveyOpen, PublishedAt: &now, PublishedBy: by, ClosesAt: closes, At: now}); err != nil {
			return err
		}
		meta := map[string]any{"page": s.pageLinks(ctx, ws).Survey(cur.Slug), "questions": len(qs), "oneResponse": st.OneResponse, "nonContacts": st.AcceptNonContacts}
		if closes != nil {
			meta["closes"] = closes
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.published", TargetType: "survey", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": SurveyOpen}, Metadata: meta})
	})
	if err != nil {
		return Survey{}, err
	}
	return s.GetSurvey(ctx, ws, id)
}

// SurveyReopen reopens a closed survey, with a new closing date or none.
type SurveyReopen struct {
	Close *SurveyClose `json:"close,omitempty"`
}

// SetSurveyOpen closes or reopens a survey.
func (s *Service) SetSurveyOpen(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, open bool, in SurveyReopen) (Survey, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Survey")
		}
		now := s.now()
		by := userOf(p)
		_, st := decodeSurvey(cur.Questions, cur.Settings)
		if !open {
			if cur.Status == SurveyDraft {
				return api.Conflict("This survey isn't published yet, so there's nothing to close.")
			}
			if err := q.SurveySetStatus(ctx, store.SurveySetStatusParams{ID: id, Status: SurveyClosed, ClosedAt: &now, ClosedBy: by, ClosesAt: cur.ClosesAt, At: now}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.closed", TargetType: "survey", TargetID: id.String(), TargetName: cur.Name,
				Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": SurveyClosed}})
		}
		if in.Close != nil {
			st.Close = *in.Close
		} else {
			st.Close = SurveyClose{Mode: "manual"}
		}
		closes := surveyClosesAt(st, ws.Timezone)
		if closes != nil && !now.Before(*closes) {
			return api.Invalid("body.close", "Pick a closing date in the future, or close the survey by hand.")
		}
		qb, _ := json.Marshal(st)
		if err := q.SurveyUpdate(ctx, store.SurveyUpdateParams{ID: id, Slug: cur.Slug, Name: cur.Name, Description: cur.Description, Questions: cur.Questions, Settings: qb,
			ClosesAt: closes, UpdatedBy: by, At: now}); err != nil {
			return err
		}
		if err := q.SurveySetStatus(ctx, store.SurveySetStatusParams{ID: id, Status: SurveyOpen, ClosesAt: closes, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.reopened", TargetType: "survey", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": SurveyOpen, "closes": closes}})
	})
	if err != nil {
		return Survey{}, err
	}
	return s.GetSurvey(ctx, ws, id)
}

// DeleteSurvey removes a survey; its page stops answering.
func (s *Service) DeleteSurvey(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Survey")
		}
		now := s.now()
		if err := q.SurveyDelete(ctx, store.SurveyDeleteParams{ID: id, At: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "survey.deleted", TargetType: "survey", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status, "started": cur.Started, "completed": cur.Completed}})
	})
}

// continueToSurvey files a poll's answer as the first answer of the survey
// it continues to, and returns the survey's link for the contact.
func (s *Service) continueToSurvey(ctx context.Context, q *store.Queries, wsID uuid.UUID, page *AnswerPage, contact, message uuid.UUID, answer int, now time.Time) (string, error) {
	sid, err := uuid.Parse(page.Config.Survey)
	if err != nil {
		return "", err
	}
	sv, err := q.SurveyGet(ctx, store.SurveyGetParams{WorkspaceID: wsID, ID: sid})
	if err != nil {
		return "", err
	}
	qs, _ := decodeSurvey(sv.Questions, sv.Settings)
	if len(qs) == 0 {
		return "", errors.New("the survey has no questions")
	}
	first := qs[0]
	value, ok := pollAnswerFor(first, page, answer)
	if !ok {
		return "", errors.New("the survey's first question doesn't take this poll's answers")
	}
	cid := contact
	existing, err := q.SurveyResponseOfContact(ctx, store.SurveyResponseOfContactParams{SurveyID: sid, ContactID: &cid})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		ans, _ := json.Marshal(map[string]any{first.Key: value})
		var msg *uuid.UUID
		if message != uuid.Nil {
			m := message
			msg = &m
		}
		if err := q.SurveyResponseInsert(ctx, store.SurveyResponseInsertParams{ID: uuid.Must(uuid.NewV7()), SurveyID: sid, WorkspaceID: wsID, ContactID: &cid, MessageID: msg,
			Kind: AnswererContact, Answers: ans, Source: "poll", At: now}); err != nil {
			return "", err
		}
	case err != nil:
		return "", err
	case existing.CompletedAt == nil:
		var answers map[string]any
		_ = json.Unmarshal(existing.Answers, &answers)
		if answers == nil {
			answers = map[string]any{}
		}
		answers[first.Key] = value
		ans, _ := json.Marshal(answers)
		if err := q.SurveyResponseUpdate(ctx, store.SurveyResponseUpdateParams{ID: existing.ID, Answers: ans, Changed: 0, At: now}); err != nil {
			return "", err
		}
	}
	ws, err := q.GetWorkspace(ctx, wsID)
	if err != nil {
		return "", err
	}
	return s.pageLinks(ctx, &ws).Survey(sv.Slug) + "?t=" + s.sign(linktoken.Survey, wsID, sv.Number, message, contact) + "&from=poll", nil
}

// pollAnswerFor maps a poll's answer onto a survey question: the same
// kind and the same answers (a choice by its position, a scale by its
// number).
func pollAnswerFor(q SurveyQuestion, page *AnswerPage, answer int) (any, bool) {
	switch page.Config.Kind {
	case KindScale:
		return answer, q.Type == QuestionScale
	case KindEmoji:
		return answer, q.Type == QuestionEmoji
	}
	if q.Type != QuestionSingle && q.Type != QuestionMulti {
		return nil, false
	}
	label := ""
	for _, a := range page.Answers {
		if a.Key == answer {
			label = strings.ToLower(strings.TrimSpace(a.Label))
		}
	}
	for _, o := range q.Answers {
		if strings.ToLower(strings.TrimSpace(o.Label)) == label {
			if q.Type == QuestionMulti {
				return []int{o.Key}, true
			}
			return o.Key, true
		}
	}
	return nil, false
}
