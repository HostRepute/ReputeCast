package polls

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Who answers through a link.
const (
	AnswererContact    = "contact"
	AnswererNonContact = "non_contact"
	AnswererTest       = "test"
)

// TriggerPollAnswered is the automation trigger "Answered a poll or survey".
const TriggerPollAnswered = "poll_answered"

// MaxComment is the longest comment: "Anything to add?", up to 500
// characters.
const MaxComment = 500

// ErrBadLink is a link that doesn't verify: cut short, changed, or for a
// poll that no longer exists. Nothing is recorded.
var ErrBadLink = errors.New("polls: the link doesn't work")

// Visit is who opened or posted an answer page, for the classifier.
type Visit struct {
	UserAgent string
	IP        netip.Addr
	Method    string
	// Script is true for the page's own POST (JavaScript ran).
	Script bool
}

// AnswerPage is everything an answer page shows.
type AnswerPage struct {
	PollID       uuid.UUID
	WorkspaceID  uuid.UUID
	Config       PollConfig
	Answers      []Answer
	CampaignName string
	// Who: a contact (Name is their first name), a non-contact (archive
	// links) or a test address.
	Answerer string
	Name     string
	Email    string
	// ContactID and MessageID are the link's (the page's own links, such as
	// Email preferences, are signed for the contact).
	ContactID uuid.UUID
	MessageID uuid.UUID
	// SentAt is when the campaign started sending.
	SentAt *time.Time
	// Picked is the answer of the link; Answer the recorded one (nil
	// before this person answered).
	Picked int
	Answer *int
	// Previous is the answer this visit replaced (nil when it didn't).
	Previous *int
	Comment  string
	// Saved is true when this request recorded the answer (or the same
	// answer was already recorded).
	Saved bool
	// Closed polls record nothing; ClosedAt is when they closed.
	Closed   bool
	ClosedAt *time.Time
	ClosesAt *time.Time
	// Refused is a non-contact on a poll that doesn't take their answers.
	Refused bool
	// Automated is a visit the classifier attributed to a machine.
	Automated bool
	// Results are the shares, when the poll shows them after answering.
	Results *PollShares
	// SurveyURL continues to a survey when the poll says so.
	SurveyURL string
}

// PollShares are a poll's results as the answer page shows them.
type PollShares struct {
	Total int64
	Rows  []PollShare
}

// PollShare is one answer's share.
type PollShare struct {
	Key     int
	Label   string
	Count   int64
	Percent int
}

// linkClaims reads a poll link's token: the poll's number, the message and
// contact (a contact's answer), neither (the web archive's), or a test
// recipient.
type linkClaims struct {
	answerer  string
	workspace uuid.UUID
	number    int64
	message   uuid.UUID
	contact   uuid.UUID
	test      uuid.UUID
}

func (s *Service) verify(token string) (linkClaims, bool) {
	if s.Links == nil || token == "" || len(token) > 200 {
		return linkClaims{}, false
	}
	if c, err := s.Links.Verify(token, linktoken.Poll); err == nil {
		out := linkClaims{workspace: c.WorkspaceID, number: int64(c.Link), message: c.MessageID, contact: c.ContactID, answerer: AnswererContact}
		if c.ContactID == uuid.Nil {
			out.answerer = AnswererNonContact
		}
		return out, true
	}
	if c, err := s.Links.Verify(token, linktoken.PollTest); err == nil {
		return linkClaims{workspace: c.WorkspaceID, number: int64(c.Link), test: c.MessageID, answerer: AnswererTest}, true
	}
	return linkClaims{}, false
}

// open loads what a page needs for a link, for the request's workspace.
func (s *Service) open(ctx context.Context, q *store.Queries, wsID uuid.UUID, token string) (linkClaims, store.PollByNumberRow, *AnswerPage, error) {
	lc, ok := s.verify(token)
	if !ok || lc.workspace != wsID {
		return lc, store.PollByNumberRow{}, nil, ErrBadLink
	}
	p, err := q.PollByNumber(ctx, lc.number)
	if err != nil || p.WorkspaceID != wsID {
		return lc, p, nil, ErrBadLink
	}
	c, answers := configOf(p.BlockID, p.Kind, p.Question, p.Answers, p.Settings)
	page := &AnswerPage{PollID: p.ID, WorkspaceID: wsID, Config: c, Answers: answers, CampaignName: p.CampaignName, Answerer: lc.answerer, ClosesAt: p.ClosesAt,
		ContactID: lc.contact, MessageID: lc.message, SentAt: p.CampaignStartedAt}
	now := s.now()
	switch {
	case p.ClosedAt != nil:
		page.Closed, page.ClosedAt = true, p.ClosedAt
	case p.ClosesAt != nil && !now.Before(*p.ClosesAt):
		page.Closed, page.ClosedAt = true, p.ClosesAt
	}
	switch lc.answerer {
	case AnswererContact:
		ct, err := q.PollContact(ctx, lc.contact)
		if err != nil || ct.WorkspaceID != wsID || ct.Erased {
			return lc, p, nil, ErrBadLink
		}
		page.Name, page.Email = ct.FirstName, ct.Email
		cid := lc.contact
		if a, err := q.PollAnswerOfContactRead(ctx, store.PollAnswerOfContactReadParams{PollID: p.ID, ContactID: &cid}); err == nil {
			v := int(a.Answer)
			page.Answer, page.Comment = &v, a.Comment
		}
	case AnswererTest:
		r, err := q.PollTestRecipient(ctx, lc.test)
		if err != nil || r.WorkspaceID != wsID {
			return lc, p, nil, ErrBadLink
		}
		page.Email = r.Email
		if a, err := q.PollAnswerOfTesterRead(ctx, store.PollAnswerOfTesterReadParams{PollID: p.ID, TestEmail: r.Email}); err == nil {
			v := int(a.Answer)
			page.Answer, page.Comment = &v, a.Comment
		}
	case AnswererNonContact:
		page.Refused = c.NonContacts == "refuse"
	}
	return lc, p, page, nil
}

// validAnswer reports whether n is one of the poll's answers.
func validAnswer(answers []Answer, n int) bool {
	return slices.ContainsFunc(answers, func(a Answer) bool { return a.Key == n })
}

// Page shows an answer page without recording anything: what the link
// picked, and the answer already recorded. A machine's visit is counted as
// an automated visit.
func (s *Service) Page(ctx context.Context, wsID uuid.UUID, token string, picked int, v Visit) (*AnswerPage, error) {
	lc, p, page, err := s.open(ctx, s.Q, wsID, token)
	if err != nil {
		return nil, err
	}
	if !validAnswer(page.Answers, picked) {
		return nil, ErrBadLink
	}
	page.Picked = picked
	if lc.answerer != AnswererTest {
		if machine, reason := s.classify(ctx, lc, p.ID, picked, v); machine {
			page.Automated = true
			_ = s.Q.PollAutomatedBump(ctx, store.PollAutomatedBumpParams{PollID: p.ID, Reason: reason, At: s.now()})
		}
	}
	if page.Config.Results && page.Answer != nil {
		page.Results = s.shares(ctx, p.ID, page.Answers)
	}
	return page, nil
}

// Record saves an answer from the page (its script's POST once the page
// was visible for a moment, or the Confirm button without JavaScript). The
// latest answer counts; a changed one replaces it, with the history kept.
func (s *Service) Record(ctx context.Context, wsID uuid.UUID, token string, answer int, v Visit) (*AnswerPage, error) {
	var page *AnswerPage
	var trigger *store.InsertAutomationEventParams
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		lc, p, pg, err := s.open(ctx, q, wsID, token)
		if err != nil {
			return err
		}
		page = pg
		if !validAnswer(page.Answers, answer) {
			return ErrBadLink
		}
		page.Picked = answer
		if page.Closed || page.Refused {
			return nil
		}
		now := s.now()
		if lc.answerer != AnswererTest {
			if machine, reason := s.classify(ctx, lc, p.ID, answer, v); machine {
				page.Automated = true
				return q.PollAutomatedBump(ctx, store.PollAutomatedBumpParams{PollID: p.ID, Reason: reason, At: now})
			}
		}
		client := tracking.ClientName(v.UserAgent)
		var msg *uuid.UUID
		if lc.message != uuid.Nil {
			m := lc.message
			msg = &m
		}
		switch lc.answerer {
		case AnswererNonContact:
			id := uuid.Must(uuid.NewV7())
			if err := q.PollAnswerInsert(ctx, store.PollAnswerInsertParams{ID: id, PollID: p.ID, WorkspaceID: wsID, Kind: AnswererNonContact, Answer: int32(answer), Client: client, At: now}); err != nil {
				return err
			}
			a := answer
			page.Answer, page.Saved = &a, true
			return q.PollAnswerEvent(ctx, store.PollAnswerEventParams{AnswerID: id, PollID: p.ID, Kind: "answered", Answer: i32(answer), At: now})
		case AnswererTest:
			existing, err := q.PollAnswerOfTester(ctx, store.PollAnswerOfTesterParams{PollID: p.ID, TestEmail: page.Email})
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				if err := q.PollAnswerInsert(ctx, store.PollAnswerInsertParams{ID: uuid.Must(uuid.NewV7()), PollID: p.ID, WorkspaceID: wsID, Kind: AnswererTest,
					TestEmail: page.Email, Answer: int32(answer), Client: client, At: now}); err != nil {
					return err
				}
			case err != nil:
				return err
			case int(existing.Answer) != answer:
				prev := int(existing.Answer)
				page.Previous = &prev
				if err := q.PollAnswerChange(ctx, store.PollAnswerChangeParams{ID: existing.ID, Answer: int32(answer), Client: client, At: now}); err != nil {
					return err
				}
			}
			a := answer
			page.Answer, page.Saved = &a, true
			return nil
		}
		// A contact's answer.
		cid := lc.contact
		existing, err := q.PollAnswerOfContact(ctx, store.PollAnswerOfContactParams{PollID: p.ID, ContactID: &cid})
		changed := false
		var prev int
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			id := uuid.Must(uuid.NewV7())
			if err := q.PollAnswerInsert(ctx, store.PollAnswerInsertParams{ID: id, PollID: p.ID, WorkspaceID: wsID, ContactID: &cid, MessageID: msg, Kind: AnswererContact,
				Answer: int32(answer), Client: client, At: now}); err != nil {
				return err
			}
			if err := q.PollAnswerEvent(ctx, store.PollAnswerEventParams{AnswerID: id, PollID: p.ID, Kind: "answered", Answer: i32(answer), At: now}); err != nil {
				return err
			}
		case err != nil:
			return err
		case int(existing.Answer) == answer:
			// The same answer again: nothing changes.
			a := answer
			page.Answer, page.Saved = &a, true
			return nil
		default:
			changed, prev = true, int(existing.Answer)
			if err := q.PollAnswerChange(ctx, store.PollAnswerChangeParams{ID: existing.ID, Answer: int32(answer), MessageID: msg, Client: client, At: now}); err != nil {
				return err
			}
			if err := q.PollAnswerEvent(ctx, store.PollAnswerEventParams{AnswerID: existing.ID, PollID: p.ID, Kind: "changed", Answer: i32(answer), Previous: i32(prev), At: now}); err != nil {
				return err
			}
			page.Previous = &prev
		}
		a := answer
		page.Answer, page.Saved = &a, true
		if err := s.saveOnContact(ctx, q, cid, page.Config, page.Answers, answer, changed, prev, now); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"poll": p.ID.String(), "answer": answer, "changed": changed, "campaign": p.CampaignName})
		trigger = &store.InsertAutomationEventParams{WorkspaceID: wsID, Kind: TriggerPollAnswered, ContactID: cid, Ref: "poll:" + p.ID.String(), Payload: payload, OccurredAt: now}
		if err := q.InsertAutomationEvent(ctx, *trigger); err != nil {
			return err
		}
		// Continue to a survey: the answer fills its first question.
		if page.Config.After == "survey" && page.Config.Survey != "" {
			if u, err := s.continueToSurvey(ctx, q, wsID, page, cid, lc.message, answer, now); err == nil {
				page.SurveyURL = u
			} else {
				s.log().WarnContext(ctx, "continue a poll to its survey", "poll", p.ID, "err", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if page.Config.Results && page.Answer != nil && !page.Automated {
		page.Results = s.shares(ctx, page.PollID, page.Answers)
	}
	return page, nil
}

// Comment saves "Anything to add?" with a recorded answer.
func (s *Service) Comment(ctx context.Context, wsID uuid.UUID, token, text string) (*AnswerPage, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > MaxComment {
		text = string([]rune(text)[:MaxComment])
	}
	var page *AnswerPage
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		lc, p, pg, err := s.open(ctx, q, wsID, token)
		if err != nil {
			return err
		}
		page = pg
		if page.Answer == nil || page.Closed {
			return nil
		}
		var a store.PollAnswer
		switch lc.answerer {
		case AnswererContact:
			cid := lc.contact
			a, err = q.PollAnswerOfContact(ctx, store.PollAnswerOfContactParams{PollID: p.ID, ContactID: &cid})
		case AnswererTest:
			a, err = q.PollAnswerOfTester(ctx, store.PollAnswerOfTesterParams{PollID: p.ID, TestEmail: page.Email})
		default:
			return nil
		}
		if err != nil {
			return err
		}
		now := s.now()
		if err := q.PollAnswerComment(ctx, store.PollAnswerCommentParams{ID: a.ID, Comment: text, At: &now}); err != nil {
			return err
		}
		page.Comment, page.Saved = text, true
		return q.PollAnswerEvent(ctx, store.PollAnswerEventParams{AnswerID: a.ID, PollID: p.ID, Kind: "commented", Answer: i32(int(a.Answer)), At: now})
	})
	if err != nil {
		return nil, err
	}
	if page.Config.Results && page.Answer != nil {
		page.Results = s.shares(ctx, page.PollID, page.Answers)
	}
	return page, nil
}

// saveOnContact sets the answer's field, or its tag (swapping the tag of
// a changed answer).
func (s *Service) saveOnContact(ctx context.Context, q *store.Queries, contact uuid.UUID, c PollConfig, answers []Answer, answer int, changed bool, prev int, now time.Time) error {
	switch c.Save {
	case SaveField:
		if strings.TrimSpace(c.Field) == "" {
			return nil
		}
		var value any = answer
		if c.Kind == KindChoice {
			for _, a := range answers {
				if a.Key == answer {
					value = a.Label
				}
			}
		}
		b, _ := json.Marshal(value)
		return q.PollSetContactField(ctx, store.PollSetContactFieldParams{ID: contact, Field: strings.TrimSpace(c.Field), Value: b, At: now})
	case SaveTag:
		ct, err := q.PollContact(ctx, contact)
		if err != nil {
			return err
		}
		tags := []string{}
		pollTags := map[string]bool{}
		want := ""
		for _, a := range answers {
			if a.Tag == "" {
				continue
			}
			pollTags[a.Tag] = true
			if a.Key == answer {
				want = a.Tag
			}
		}
		for _, t := range ct.Tags {
			if !pollTags[t] {
				tags = append(tags, t)
			}
		}
		if want != "" {
			tags = append(tags, want)
		}
		_ = changed
		_ = prev
		return q.PollSetContactTags(ctx, store.PollSetContactTagsParams{ID: contact, Tags: tags, At: now})
	}
	return nil
}

func i32(n int) *int32 {
	v := int32(n)
	return &v
}

// shares are a poll's results for its answer page.
func (s *Service) shares(ctx context.Context, pollID uuid.UUID, answers []Answer) *PollShares {
	rows, err := s.Q.PollAnswerCounts(ctx, pollID)
	if err != nil {
		return nil
	}
	counts := map[int]int64{}
	out := &PollShares{}
	for _, r := range rows {
		counts[int(r.Answer)] = r.N
		out.Total += r.N
	}
	for _, a := range answers {
		sh := PollShare{Key: a.Key, Label: a.Label, Count: counts[a.Key]}
		if out.Total > 0 {
			sh.Percent = int((float64(sh.Count)/float64(out.Total))*100 + 0.5)
		}
		out.Rows = append(out.Rows, sh)
	}
	return out
}

// Visits a person never makes: a gateway that opens links before the
// reader does, a prefetching proxy. Every answer of one email opened within
// a second is a scanner too.
var gatewayNames = []string{"Microsoft Defender", "Mimecast", "Proofpoint", "Barracuda", "Trend Micro", "Sophos", "FireEye", "Forcepoint", "Symantec", "Cisco Secure Email", "Zscaler", "Check Point"}

type burstKey struct {
	message uuid.UUID
	poll    uuid.UUID
}

type burstMark struct {
	answer int
	at     time.Time
}

var (
	burstMu sync.Mutex
	bursts  = map[burstKey][]burstMark{}
)

// classify decides whether a visit is a machine's, and which kind.
func (s *Service) classify(ctx context.Context, lc linkClaims, pollID uuid.UUID, answer int, v Visit) (bool, string) {
	sig := tracking.Signal{UserAgent: v.UserAgent, IP: v.IP, Method: v.Method, SinceSend: -1}
	if lc.message != uuid.Nil {
		if m, err := s.Q.PollMessage(ctx, lc.message); err == nil && m.SentAt != nil {
			sig.SinceSend = s.now().Sub(*m.SentAt)
		}
	}
	burst := false
	if lc.message != uuid.Nil && !v.Script {
		k := burstKey{lc.message, pollID}
		now := s.now()
		burstMu.Lock()
		marks := bursts[k][:0]
		for _, m := range bursts[k] {
			if now.Sub(m.at) < time.Second {
				marks = append(marks, m)
			}
		}
		for _, m := range marks {
			if m.answer != answer {
				burst = true
			}
		}
		marks = append(marks, burstMark{answer, now})
		if len(bursts) > 50000 {
			bursts = map[burstKey][]burstMark{}
		}
		bursts[k] = marks
		burstMu.Unlock()
	}
	verdict := tracking.ClassifyClick(sig, burst)
	if !verdict.Machine {
		return false, ""
	}
	for _, g := range gatewayNames {
		if strings.Contains(verdict.Reason, g) {
			return true, "gateway"
		}
	}
	return true, "prefetch"
}
