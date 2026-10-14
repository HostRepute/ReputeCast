package forms

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/store"
)

// Submission is what a visitor sent, with the request it came in.
type Submission struct {
	Email     string            `json:"email"`
	FirstName string            `json:"firstName"`
	LastName  string            `json:"lastName"`
	Company   string            `json:"company"`
	Fields    map[string]string `json:"fields"`
	// Lists are the checked lists (ids) when visitors pick lists.
	Lists   []string `json:"lists"`
	Consent bool     `json:"consent"`
	// Honeypot is the invisible field only bots fill in.
	Honeypot string `json:"-"`
	// Token is the timestamp token rendered with the form.
	Token string `json:"-"`
	// Plain is a post of the copied plain HTML form, which has no token.
	Plain bool `json:"-"`
	// Origin is the host the browser says the form was on.
	Origin    string      `json:"-"`
	IP        *netip.Addr `json:"-"`
	UserAgent string      `json:"-"`
}

// Outcome statuses.
const (
	OutcomeConfirm     = "confirm"      // check your inbox (double opt-in)
	OutcomeSubscribed  = "subscribed"   // single opt-in, done
	OutcomeInvalid     = "invalid"      // fix the fields
	OutcomeChallenge   = "challenge"    // prove you're a person (Turnstile)
	OutcomeRejected    = "rejected"     // origin not allowed, spam
	OutcomeRateLimited = "rate_limited" // too many tries
	OutcomeClosed      = "closed"       // form paused or not published
	OutcomeError       = "error"        // couldn't save; nothing was lost
)

// Outcome tells the page what to show after a submission.
type Outcome struct {
	Status string `json:"status"`
	// Errors are messages per field (email, first_name, lists, consent, a custom key).
	Errors map[string]string `json:"errors,omitempty"`
	// Message is the form-level message.
	Message     string `json:"message,omitempty"`
	RedirectURL string `json:"redirect,omitempty"`
	// ChallengeID names the held submission waiting for Turnstile.
	ChallengeID string `json:"challenge,omitempty"`
	// Email and FirstName are echoed on the "check your inbox" page.
	Email     string      `json:"-"`
	FirstName string      `json:"-"`
	Lists     []uuid.UUID `json:"-"`
}

// Messages a visitor sees. They say what happened and what to do.
const (
	msgSaveFailed  = "We couldn't save your subscription just now. Nothing was lost; please try again in a minute."
	msgTooFast     = "That was quick. Wait a moment and send the form again."
	msgRateLimited = "Too many tries from your network. Wait a minute and try again."
	msgClosed      = "This form isn't taking signups right now."
	msgOrigin      = "This form can't be used on this site."
	msgChallenge   = "One more step: confirm you're a person."
	msgExpired     = "This form has expired. Reload the page and send it again."
)

func (s *Service) recordReject(ctx context.Context, ws *store.Workspace, c Config, reason, origin string) {
	if c.Builtin || c.ID == uuid.Nil {
		return
	}
	now := s.now()
	day := dayOf(now, location(ws))
	if err := s.Q.FormRecordRejection(ctx, store.FormRecordRejectionParams{FormID: c.ID, Day: day, Reason: reason, Origin: clipOrigin(origin), At: now}); err != nil {
		s.log().WarnContext(ctx, "record form rejection", "err", err)
	}
	if reason == "origin" {
		return // not spam: counted in the blocked-origin alert only
	}
	if err := s.Q.FormBumpDaily(ctx, store.FormBumpDailyParams{FormID: c.ID, Day: day, Blocked: 1}); err != nil {
		s.log().WarnContext(ctx, "count blocked submission", "err", err)
	}
	_ = s.Q.FormBumpTotals(ctx, store.FormBumpTotalsParams{ID: c.ID, Blocked: 1})
}

func clipOrigin(o string) string {
	if len(o) > 255 {
		return o[:255]
	}
	return o
}

// RecordView counts a view of a form (the hosted page, or an embed asking
// for its token). Crawlers and link previews don't count.
func (s *Service) RecordView(ctx context.Context, ws *store.Workspace, c Config, userAgent string) {
	if c.Builtin || c.ID == uuid.Nil || isBot(userAgent) {
		return
	}
	day := dayOf(s.now(), location(ws))
	if err := s.Q.FormBumpDaily(ctx, store.FormBumpDailyParams{FormID: c.ID, Day: day, Views: 1}); err != nil {
		s.log().WarnContext(ctx, "count form view", "err", err)
		return
	}
	_ = s.Q.FormBumpTotals(ctx, store.FormBumpTotalsParams{ID: c.ID, Views: 1})
}

// Submit runs a public submission through the form's checks and, when it
// passes, subscribes the contact through the audience package. A filled
// honeypot is dropped silently, so bots see a success. Copied plain HTML
// forms can't carry a timestamp token and rely on the origin allowlist. A
// submission that fails only the token checks goes to a Turnstile challenge
// when Turnstile is set up.
func (s *Service) Submit(ctx context.Context, ws *store.Workspace, set *Settings, c Config, sub Submission) Outcome {
	if c.Status != StatusLive {
		return Outcome{Status: OutcomeClosed, Message: msgClosed}
	}
	now := s.now()
	limits := s.rates()
	ip := "unknown"
	if sub.IP != nil {
		ip = sub.IP.String()
	}
	if !limits.allow("ip:"+ws.ID.String()+":"+ip, c.Spam.RateLimitPerIP, now) || !limits.allow("form:"+formKey(c), formSubmitsPerMinute, now) {
		s.recordReject(ctx, ws, c, "rate_limit", sub.Origin)
		return Outcome{Status: OutcomeRateLimited, Message: msgRateLimited}
	}
	if !OriginAllowed(sub.Origin, c.AllowedOrigins, s.OwnOrigins(set)) {
		s.recordReject(ctx, ws, c, "origin", sub.Origin)
		return Outcome{Status: OutcomeRejected, Message: msgOrigin}
	}
	if c.Spam.Honeypot && strings.TrimSpace(sub.Honeypot) != "" {
		s.recordReject(ctx, ws, c, "honeypot", sub.Origin)
		if c.DoubleOptIn {
			return Outcome{Status: OutcomeConfirm}
		}
		return Outcome{Status: OutcomeSubscribed}
	}
	suspicious, expired := false, false
	if !sub.Plain {
		age, ok := s.tokenAge(formKey(c), sub.Token, now)
		switch {
		case !ok:
			suspicious, expired = true, true
		case c.Spam.Honeypot && age < time.Duration(c.Spam.MinSeconds)*time.Second:
			suspicious = true
		}
	}
	lists, errs := s.check(c, &sub)
	if len(errs) > 0 {
		return Outcome{Status: OutcomeInvalid, Errors: errs, Message: firstMessage(errs)}
	}
	if c.Spam.BlockDisposable {
		if emailaddr.IsDisposable(emailaddr.Domain(sub.Email)) {
			s.recordReject(ctx, ws, c, "disposable", sub.Origin)
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: "Use a permanent address: disposable inboxes expire before you could confirm."}}
		}
		if emailaddr.IsRole(sub.Email) {
			s.recordReject(ctx, ws, c, "role", sub.Origin)
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: "Use a personal address. Addresses like info@ reach several people, so they can't give consent."}}
		}
	}
	if suspicious {
		if c.Spam.Turnstile && set.TurnstileSiteKey() != "" {
			id, err := s.holdForChallenge(ctx, ws, c, sub)
			if err != nil {
				s.log().ErrorContext(ctx, "hold submission for turnstile", "err", err)
				return Outcome{Status: OutcomeError, Message: msgSaveFailed}
			}
			return Outcome{Status: OutcomeChallenge, ChallengeID: id, Message: msgChallenge}
		}
		s.recordReject(ctx, ws, c, "timing", sub.Origin)
		if expired {
			return Outcome{Status: OutcomeInvalid, Message: msgExpired}
		}
		return Outcome{Status: OutcomeInvalid, Message: msgTooFast}
	}
	return s.accept(ctx, ws, set, c, sub, lists)
}

func firstMessage(errs map[string]string) string {
	for _, k := range []string{FieldEmail, FieldFirstName, FieldLastName, FieldCompany, "lists", "consent"} {
		if m, ok := errs[k]; ok {
			return m
		}
	}
	for _, m := range errs {
		return m
	}
	return ""
}

// check validates the fields and returns the lists to join.
func (s *Service) check(c Config, sub *Submission) ([]uuid.UUID, map[string]string) {
	errs := map[string]string{}
	email, problem := emailaddr.Check(sub.Email)
	switch {
	case strings.TrimSpace(sub.Email) == "":
		errs[FieldEmail] = "Enter your email address."
	case problem != "":
		errs[FieldEmail] = "That doesn't look like a complete email address."
	}
	sub.Email = email
	for _, f := range c.Fields {
		if f.Key == FieldEmail || !f.Required {
			continue
		}
		var v string
		switch f.Key {
		case FieldFirstName:
			v = sub.FirstName
		case FieldLastName:
			v = sub.LastName
		case FieldCompany:
			v = sub.Company
		default:
			v = sub.Fields[f.Key]
		}
		if strings.TrimSpace(v) == "" {
			errs[f.Key] = "Fill this in."
		}
	}
	for k, v := range sub.Fields {
		if len(v) > 500 {
			errs[k] = "Keep it under 500 characters."
		}
	}
	if !c.HasField(FieldFirstName) {
		sub.FirstName = ""
	}
	if !c.HasField(FieldLastName) {
		sub.LastName = ""
	}
	if !c.HasField(FieldCompany) {
		sub.Company = ""
	}
	if len(sub.FirstName) > 100 || len(sub.LastName) > 100 || len(sub.Company) > 200 {
		errs[FieldFirstName] = "That's longer than a name needs to be."
	}
	var lists []uuid.UUID
	if c.ListChoice && len(c.Lists) > 1 {
		for _, raw := range sub.Lists {
			id, err := uuid.Parse(raw)
			if err == nil && slices.Contains(c.Lists, id) && !slices.Contains(lists, id) {
				lists = append(lists, id)
			}
		}
		if len(lists) == 0 {
			errs["lists"] = "Pick at least one newsletter."
		}
	} else {
		lists = slices.Clone(c.Lists)
	}
	if len(lists) == 0 && errs["lists"] == "" {
		errs["lists"] = "This form has no newsletter to join yet."
	}
	if c.ConsentText != "" && !sub.Consent {
		errs["consent"] = "Tick the box to agree to receive emails."
	}
	return lists, errs
}

// accept subscribes the contact and records the submission.
func (s *Service) accept(ctx context.Context, ws *store.Workspace, set *Settings, c Config, sub Submission, lists []uuid.UUID) Outcome {
	now := s.now()
	ev := audience.PublicEvidence{Source: c.Source(), IP: sub.IP, UserAgent: sub.UserAgent, ConsentText: c.ConsentText, ConsentVersion: c.ConsentVersionLabel(), At: now}
	fields := map[string]string{}
	for k, v := range sub.Fields {
		if c.HasField(k) {
			fields[k] = v
		}
	}
	var res audience.SignupResult
	var submissionID uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		res, err = s.Audience.Signup(ctx, tx, ws, audience.SignupInput{
			Email: sub.Email, FirstName: sub.FirstName, LastName: sub.LastName, Company: sub.Company, Fields: fields,
			Lists: lists, DoubleOptIn: c.DoubleOptIn, Evidence: ev,
		})
		if err != nil || res.Blocked {
			return err
		}
		q := store.New(tx)
		var formID *uuid.UUID
		if !c.Builtin {
			id := c.ID
			formID = &id
		}
		touched := append(slices.Clone(res.Pending), res.Subscribed...)
		if len(touched) == 0 {
			return nil // already subscribed to everything: nothing new to record
		}
		double := res.NeedsConfirmation()
		var confirmedAt *time.Time
		if !double {
			confirmedAt = &now
		}
		submissionID = uuid.Must(uuid.NewV7())
		if err := q.FormSubmissionInsert(ctx, store.FormSubmissionInsertParams{ID: submissionID, WorkspaceID: ws.ID, FormID: formID, ContactID: res.ContactID,
			ListIds: touched, DoubleOptIn: double, At: now, ConfirmedAt: confirmedAt}); err != nil {
			return err
		}
		if formID != nil {
			bump := store.FormBumpDailyParams{FormID: *formID, Day: dayOf(now, location(ws)), Submissions: 1}
			totals := store.FormBumpTotalsParams{ID: *formID, Submissions: 1}
			if !double {
				bump.Confirmed, totals.Confirmed = 1, 1
			}
			if err := q.FormBumpDaily(ctx, bump); err != nil {
				return err
			}
			if err := q.FormBumpTotals(ctx, totals); err != nil {
				return err
			}
			if !double {
				// "Form submitted" automations start now; with double opt-in
				// they start when the contact confirms.
				if err := automations.EmitFormSubmitted(ctx, q, ws.ID, res.ContactID, *formID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		var fe *audience.FieldError
		if errors.As(err, &fe) {
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{fe.Field: fe.Message}, Message: fe.Message}
		}
		s.log().ErrorContext(ctx, "form submission", "form", c.ID, "err", err)
		return Outcome{Status: OutcomeError, Message: msgSaveFailed}
	}
	out := Outcome{Status: OutcomeSubscribed, Email: sub.Email, FirstName: res.FirstName, Lists: lists}
	if c.DoubleOptIn || res.NeedsConfirmation() || res.Blocked {
		out.Status = OutcomeConfirm
	}
	if res.NeedsConfirmation() && submissionID != uuid.Nil {
		if err := s.sendConfirmation(ctx, ws, set, submissionID, res.ContactID, false); err != nil && !errors.Is(err, errNoSystemMail) {
			s.log().WarnContext(ctx, "send confirmation", "err", err)
		}
	}
	if c.AfterSubmit == "redirect" && c.RedirectURL != "" {
		out.RedirectURL = c.RedirectURL
	}
	return out
}

type heldSubmission struct {
	FormID    string     `json:"form"`
	Builtin   bool       `json:"builtin"`
	Sub       Submission `json:"sub"`
	Origin    string     `json:"origin"`
	IP        string     `json:"ip"`
	UserAgent string     `json:"ua"`
}

const challengeTTL = 30 * time.Minute

func (s *Service) holdForChallenge(ctx context.Context, ws *store.Workspace, c Config, sub Submission) (string, error) {
	h := heldSubmission{Builtin: c.Builtin, Sub: sub, Origin: sub.Origin, UserAgent: sub.UserAgent}
	if !c.Builtin {
		h.FormID = c.ID.String()
	}
	if sub.IP != nil {
		h.IP = sub.IP.String()
	}
	raw, _ := json.Marshal(h)
	id := uuid.Must(uuid.NewV7())
	enc, err := s.Box.Seal(raw, "form_challenge:"+id.String())
	if err != nil {
		return "", err
	}
	now := s.now()
	var formID *uuid.UUID
	if !c.Builtin {
		fid := c.ID
		formID = &fid
	}
	if err := s.Q.FormChallengeInsert(ctx, store.FormChallengeInsertParams{ID: id, WorkspaceID: ws.ID, FormID: formID, PayloadEnc: enc, Origin: clipOrigin(sub.Origin), At: now, ExpiresAt: now.Add(challengeTTL)}); err != nil {
		return "", err
	}
	return id.String(), nil
}

// ChallengeForm returns the form a held submission belongs to.
type ChallengeForm func(ctx context.Context, formID string, builtin bool) (Config, error)

// SolveChallenge verifies a Turnstile answer for a held submission and,
// when Cloudflare agrees, completes it.
func (s *Service) SolveChallenge(ctx context.Context, ws *store.Workspace, set *Settings, challengeID, answer string, ip *netip.Addr, formOf ChallengeForm) Outcome {
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return Outcome{Status: OutcomeInvalid, Message: "This check expired. Fill in the form again."}
	}
	row, err := s.Q.FormChallengeTake(ctx, store.FormChallengeTakeParams{ID: id, WorkspaceID: ws.ID, At: s.now()})
	if err != nil {
		return Outcome{Status: OutcomeInvalid, Message: "This check expired. Fill in the form again."}
	}
	raw, err := s.Box.Open(row.PayloadEnc, "form_challenge:"+row.ID.String())
	if err != nil {
		return Outcome{Status: OutcomeError, Message: msgSaveFailed}
	}
	var h heldSubmission
	if err := json.Unmarshal(raw, &h); err != nil {
		return Outcome{Status: OutcomeError, Message: msgSaveFailed}
	}
	c, err := formOf(ctx, h.FormID, h.Builtin)
	if err != nil {
		return Outcome{Status: OutcomeClosed, Message: msgClosed}
	}
	if c.Status != StatusLive {
		return Outcome{Status: OutcomeClosed, Message: msgClosed}
	}
	secret, ok := s.turnstileSecret(set)
	if !ok {
		return Outcome{Status: OutcomeError, Message: msgSaveFailed}
	}
	remote := ""
	if ip != nil {
		remote = ip.String()
	}
	res, err := s.Turnstile.Verify(ctx, secret, answer, remote, row.ID.String())
	if err != nil {
		s.log().WarnContext(ctx, "turnstile verify", "err", err)
		return Outcome{Status: OutcomeError, Message: "We couldn't reach the verification service. Nothing was lost; please try again in a minute."}
	}
	if !res.Success {
		if res.SecretRejected() {
			if err := s.turnstileFailed(ctx, ws, "Cloudflare rejected the secret key ("+strings.Join(res.Codes, ", ")+")", map[string]any{"codes": res.Codes}); err != nil {
				s.log().WarnContext(ctx, "record turnstile error", "err", err)
			}
		}
		s.recordReject(ctx, ws, c, "turnstile", h.Origin)
		return Outcome{Status: OutcomeRejected, Message: "We couldn't confirm you're a person. Try the form again in a minute."}
	}
	sub := h.Sub
	if h.IP != "" {
		if a, err := netip.ParseAddr(h.IP); err == nil {
			sub.IP = &a
		}
	}
	sub.UserAgent, sub.Origin = h.UserAgent, h.Origin
	lists, errs := s.check(c, &sub)
	if len(errs) > 0 {
		return Outcome{Status: OutcomeInvalid, Errors: errs, Message: firstMessage(errs)}
	}
	return s.accept(ctx, ws, set, c, sub, lists)
}
