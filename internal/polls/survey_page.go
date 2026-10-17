package polls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// Longest text answers: a short answer, and a long one.
const (
	MaxShortAnswer = 200
	MaxLongAnswer  = 2000
)

// SurveyPage is everything a survey page shows.
type SurveyPage struct {
	SurveyID    uuid.UUID
	Number      int64
	Slug        string
	Name        string
	Description string
	Questions   []SurveyQuestion
	Settings    SurveySettings
	ClosesAt    *time.Time
	// Closed surveys take no answers; ClosedAt is when they closed.
	Closed   bool
	ClosedAt *time.Time
	// Who answers: a contact (FirstName, Email), someone who isn't one, or a
	// test address.
	Answerer  string
	FirstName string
	Email     string
	// Refused is someone who isn't a contact, on a survey that doesn't take
	// their answers.
	Refused bool
	// Answers are the response's answers: question key → a choice's key, a
	// multiple choice's keys, a number, or text.
	Answers map[string]any
	// FromPoll is a response whose first question came from the email's
	// poll; CompletedAt an earlier completed response, editable while the
	// survey is open.
	FromPoll    bool
	CompletedAt *time.Time
	// Saved is true when this request saved the answers.
	Saved bool
	// Errors are the questions left empty, in order, with their message.
	Errors []SurveyError
	Token  string
}

// SurveyError is a question that needs an answer.
type SurveyError struct {
	Key     string
	Number  int
	Text    string
	Message string
}

// Error returns a question's message, or "".
func (p *SurveyPage) Error(key string) string {
	for _, e := range p.Errors {
		if e.Key == key {
			return e.Message
		}
	}
	return ""
}

type surveyClaims struct {
	answerer string
	message  uuid.UUID
	contact  uuid.UUID
	test     uuid.UUID
}

// openSurvey loads a survey page for the request's workspace.
func (s *Service) openSurvey(ctx context.Context, q *store.Queries, wsID uuid.UUID, slug, token string) (surveyClaims, *SurveyPage, error) {
	sv, err := q.SurveyBySlug(ctx, store.SurveyBySlugParams{WorkspaceID: wsID, Slug: slug})
	if err != nil || sv.Status == SurveyDraft {
		return surveyClaims{}, nil, ErrBadLink
	}
	qs, st := decodeSurvey(sv.Questions, sv.Settings)
	page := &SurveyPage{SurveyID: sv.ID, Number: sv.Number, Slug: sv.Slug, Name: sv.Name, Description: sv.Description, Questions: qs, Settings: st,
		ClosesAt: sv.ClosesAt, Answers: map[string]any{}, Token: token}
	now := s.now()
	switch {
	case sv.Status == SurveyClosed:
		page.Closed, page.ClosedAt = true, sv.ClosedAt
	case sv.ClosesAt != nil && !now.Before(*sv.ClosesAt):
		page.Closed, page.ClosedAt = true, sv.ClosesAt
	}
	sc := surveyClaims{answerer: AnswererNonContact}
	if token != "" {
		if s.Links == nil || len(token) > 200 {
			return sc, nil, ErrBadLink
		}
		if c, err := s.Links.Verify(token, linktoken.Survey); err == nil && c.WorkspaceID == wsID && int64(c.Link) == sv.Number && c.ContactID != uuid.Nil {
			sc = surveyClaims{answerer: AnswererContact, message: c.MessageID, contact: c.ContactID}
		} else if c, err := s.Links.Verify(token, linktoken.SurveyTest); err == nil && c.WorkspaceID == wsID && int64(c.Link) == sv.Number {
			sc = surveyClaims{answerer: AnswererTest, test: c.MessageID}
		} else {
			return sc, nil, ErrBadLink
		}
	}
	page.Answerer = sc.answerer
	switch sc.answerer {
	case AnswererContact:
		ct, err := q.PollContact(ctx, sc.contact)
		if err != nil || ct.WorkspaceID != wsID || ct.Erased {
			return sc, nil, ErrBadLink
		}
		page.FirstName, page.Email = ct.FirstName, ct.Email
		cid := sc.contact
		if r, err := q.SurveyResponseOfContact(ctx, store.SurveyResponseOfContactParams{SurveyID: sv.ID, ContactID: &cid}); err == nil {
			if r.CompletedAt != nil && !st.OneResponse {
				// Several responses each: every visit starts a new one.
				break
			}
			_ = json.Unmarshal(r.Answers, &page.Answers)
			page.CompletedAt = r.CompletedAt
			page.FromPoll = r.Source == "poll" && r.CompletedAt == nil
		}
	case AnswererTest:
		r, err := q.PollTestRecipient(ctx, sc.test)
		if err != nil || r.WorkspaceID != wsID {
			return sc, nil, ErrBadLink
		}
		page.Email = r.Email
	case AnswererNonContact:
		page.Refused = !st.AcceptNonContacts
	}
	if page.Answers == nil {
		page.Answers = map[string]any{}
	}
	return sc, page, nil
}

// SurveyPageFor shows a survey page; it records nothing.
func (s *Service) SurveyPageFor(ctx context.Context, wsID uuid.UUID, slug, token string) (*SurveyPage, error) {
	_, page, err := s.openSurvey(ctx, s.Q, wsID, slug, token)
	return page, err
}

// requiredMessage is what a required question left empty says.
func requiredMessage(q SurveyQuestion) string {
	if strings.TrimSpace(q.RequiredText) != "" {
		return q.RequiredText
	}
	switch q.Type {
	case QuestionSingle:
		return "Pick one answer."
	case QuestionMulti:
		return "Pick at least one answer."
	case QuestionScale:
		return "Pick a number from 0 to 10."
	case QuestionEmoji:
		return "Pick a face."
	}
	return "Write an answer."
}

// readAnswers reads a survey form's answers: a question's key holds its
// answer (a choice's key, the keys of a multiple choice, a number, text).
func readAnswers(qs []SurveyQuestion, form url.Values) (map[string]any, []SurveyError) {
	answers := map[string]any{}
	var errs []SurveyError
	for i, q := range qs {
		var v any
		switch q.Type {
		case QuestionSingle, QuestionScale, QuestionEmoji:
			raw := strings.TrimSpace(form.Get(q.Key))
			if n, err := strconv.Atoi(raw); err == nil && slices.ContainsFunc(QuestionOptions(q), func(o SurveyOption) bool { return o.Key == n }) {
				v = n
			}
		case QuestionMulti:
			keys := []int{}
			for _, raw := range form[q.Key] {
				n, err := strconv.Atoi(strings.TrimSpace(raw))
				if err == nil && !slices.Contains(keys, n) && slices.ContainsFunc(q.Answers, func(o SurveyOption) bool { return o.Key == n }) {
					keys = append(keys, n)
				}
			}
			if len(keys) > 0 {
				slices.Sort(keys)
				v = keys
			}
		case QuestionShort, QuestionLong:
			text := strings.TrimSpace(strings.ReplaceAll(form.Get(q.Key), "\r\n", "\n"))
			limit := MaxLongAnswer
			if q.Type == QuestionShort {
				limit = MaxShortAnswer
				text = strings.Join(strings.Fields(text), " ")
			}
			if utf8.RuneCountInString(text) > limit {
				text = string([]rune(text)[:limit])
			}
			if text != "" {
				v = text
			}
		}
		if v == nil {
			if q.Required {
				errs = append(errs, SurveyError{Key: q.Key, Number: i + 1, Text: q.Text, Message: requiredMessage(q)})
			}
			continue
		}
		answers[q.Key] = v
	}
	return answers, errs
}

// SubmitSurvey saves a survey page's answers, when the person pressed
// Send: scanners don't. A required question left empty saves nothing.
func (s *Service) SubmitSurvey(ctx context.Context, wsID uuid.UUID, slug, token string, form url.Values) (*SurveyPage, error) {
	var page *SurveyPage
	var event *store.InsertAutomationEventParams
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		sc, pg, err := s.openSurvey(ctx, q, wsID, slug, token)
		if err != nil {
			return err
		}
		page = pg
		if page.Closed || page.Refused {
			return nil
		}
		answers, errs := readAnswers(page.Questions, form)
		if len(errs) > 0 {
			page.Answers, page.Errors = answers, errs
			return nil
		}
		ans, _ := json.Marshal(answers)
		now := s.now()
		var msg *uuid.UUID
		if sc.message != uuid.Nil {
			m := sc.message
			msg = &m
		}
		switch sc.answerer {
		case AnswererNonContact:
			if err := q.SurveyResponseInsert(ctx, store.SurveyResponseInsertParams{ID: uuid.Must(uuid.NewV7()), SurveyID: page.SurveyID, WorkspaceID: wsID,
				Kind: AnswererNonContact, Answers: ans, Source: "page", At: now, CompletedAt: &now}); err != nil {
				return err
			}
		case AnswererTest:
			if err := q.SurveyResponseInsert(ctx, store.SurveyResponseInsertParams{ID: uuid.Must(uuid.NewV7()), SurveyID: page.SurveyID, WorkspaceID: wsID,
				Kind: AnswererTest, TestEmail: page.Email, Answers: ans, Source: "page", At: now, CompletedAt: &now}); err != nil {
				return err
			}
		default:
			cid := sc.contact
			existing, err := q.SurveyResponseOfContact(ctx, store.SurveyResponseOfContactParams{SurveyID: page.SurveyID, ContactID: &cid})
			changed := false
			switch {
			case errors.Is(err, pgx.ErrNoRows) || (err == nil && existing.CompletedAt != nil && !page.Settings.OneResponse):
				if err := q.SurveyResponseInsert(ctx, store.SurveyResponseInsertParams{ID: uuid.Must(uuid.NewV7()), SurveyID: page.SurveyID, WorkspaceID: wsID,
					ContactID: &cid, MessageID: msg, Kind: AnswererContact, Answers: ans, Source: "page", At: now, CompletedAt: &now}); err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				var completed *time.Time
				n := int32(0)
				if existing.CompletedAt == nil {
					completed = &now
				} else {
					changed, n = true, 1
				}
				if err := q.SurveyResponseUpdate(ctx, store.SurveyResponseUpdateParams{ID: existing.ID, Answers: ans, CompletedAt: completed, Changed: n, At: now}); err != nil {
					return err
				}
			}
			if err := s.saveSurveyOnContact(ctx, q, cid, page.Questions, answers, now); err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{"survey": page.SurveyID.String(), "changed": changed, "name": page.Name})
			event = &store.InsertAutomationEventParams{WorkspaceID: wsID, Kind: TriggerPollAnswered, ContactID: cid, Ref: "survey:" + page.SurveyID.String(), Payload: payload, OccurredAt: now}
			if err := q.InsertAutomationEvent(ctx, *event); err != nil {
				return err
			}
		}
		page.Answers, page.Saved = answers, true
		if page.CompletedAt == nil {
			page.CompletedAt = &now
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

// saveSurveyOnContact sets the fields and tags the questions save to.
func (s *Service) saveSurveyOnContact(ctx context.Context, q *store.Queries, contact uuid.UUID, qs []SurveyQuestion, answers map[string]any, now time.Time) error {
	var addTags, dropTags []string
	for _, qn := range qs {
		v, ok := answers[qn.Key]
		switch qn.Save.To {
		case "field":
			field := strings.TrimSpace(qn.Save.Field)
			if field == "" || !ok {
				continue
			}
			value := fieldValue(qn, v)
			b, _ := json.Marshal(value)
			if err := q.PollSetContactField(ctx, store.PollSetContactFieldParams{ID: contact, Field: field, Value: b, At: now}); err != nil {
				return err
			}
		case "tags":
			if qn.Type != QuestionSingle && qn.Type != QuestionMulti {
				continue
			}
			picked := pickedKeys(v)
			for _, o := range qn.Answers {
				tag := strings.TrimSpace(o.Tag)
				if tag == "" {
					continue
				}
				if slices.Contains(picked, o.Key) {
					addTags = append(addTags, tag)
				} else if ok {
					dropTags = append(dropTags, tag)
				}
			}
		}
	}
	if len(addTags) == 0 && len(dropTags) == 0 {
		return nil
	}
	ct, err := q.PollContact(ctx, contact)
	if err != nil {
		return err
	}
	tags := []string{}
	for _, t := range ct.Tags {
		if !slices.Contains(dropTags, t) && !slices.Contains(addTags, t) {
			tags = append(tags, t)
		}
	}
	tags = append(tags, addTags...)
	return q.PollSetContactTags(ctx, store.PollSetContactTagsParams{ID: contact, Tags: tags, At: now})
}

// pickedKeys reads a choice answer: one key, or several.
func pickedKeys(v any) []int {
	switch x := v.(type) {
	case int:
		return []int{x}
	case float64:
		return []int{int(x)}
	case []int:
		return x
	case []any:
		out := []int{}
		for _, e := range x {
			if f, ok := e.(float64); ok {
				out = append(out, int(f))
			}
		}
		return out
	}
	return nil
}

// fieldValue is what a question writes to its field: a choice's label
// (several joined with commas), a number, or the text.
func fieldValue(q SurveyQuestion, v any) any {
	switch q.Type {
	case QuestionSingle, QuestionMulti:
		labels := []string{}
		for _, k := range pickedKeys(v) {
			for _, o := range q.Answers {
				if o.Key == k {
					labels = append(labels, o.Label)
				}
			}
		}
		return strings.Join(labels, ", ")
	case QuestionScale, QuestionEmoji:
		if k := pickedKeys(v); len(k) == 1 {
			return k[0]
		}
	}
	return v
}

// AnswerText is a question's answer in words: "Daily", "Feature flags,
// Audit log exports", "9", or the text.
func AnswerText(q SurveyQuestion, v any) string {
	if v == nil {
		return ""
	}
	switch q.Type {
	case QuestionSingle, QuestionMulti:
		return fmt.Sprint(fieldValue(q, v))
	case QuestionScale:
		if k := pickedKeys(v); len(k) == 1 {
			return strconv.Itoa(k[0])
		}
	case QuestionEmoji:
		if k := pickedKeys(v); len(k) == 1 {
			faces := q.Faces
			if faces != 3 {
				faces = 5
			}
			return fmt.Sprintf("%d of %d, %s", k[0], faces, FaceWord(faces, k[0]))
		}
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
