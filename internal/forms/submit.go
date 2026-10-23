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
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/phone"
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
	// Phone is the number as typed, PhoneCountry the picker's country (for
	// numbers without +), SmsConsent the SMS consent box (phase 8).
	Phone        string `json:"phone,omitempty"`
	PhoneCountry string `json:"phoneCountry,omitempty"`
	SmsConsent   bool   `json:"smsConsent,omitempty"`
	// number is Phone read by the checks.
	number phone.Number
	// Language is the language the form was shown in (the embed script's
	// _lang, the hosted page's); empty is the form's own language, which a
	// copied plain HTML form shows. BrowserLanguage is the browser's first
	// language: a new contact gets it as their language.
	Language        string `json:"language,omitempty"`
	BrowserLanguage string `json:"browserLanguage,omitempty"`
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
	// Page is the address of the page with the form (its UTM parameters)
	// and Referrer the page before it (growth sources).
	Page     string `json:"-"`
	Referrer string `json:"-"`
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
	// Language is the language the form was shown in: the result page
	// follows it.
	Language string `json:"-"`
	// Texted is "We've also texted +1 312 ••• 0123 to confirm." when the
	// signup gave SMS consent and the opt-in confirmation went out.
	Texted string `json:"texted,omitempty"`
}

// Messages a visitor sees are built-in labels (internal/pagelabels), in
// the language the form was shown in. They say what happened and what to do.

// localize is the form as it was shown to the visitor: the language the
// submission says, when the form is Live in it, else its fallback.
func (s *Service) localize(ctx context.Context, ws *store.Workspace, set *Settings, c Config, requested string) Config {
	out, _, err := s.LocalizedForm(ctx, ws, set, c, requested)
	if err != nil {
		s.log().WarnContext(ctx, "load form copy", "form", c.ID, "err", err)
		return c
	}
	return out
}

// fieldErrorText is an audience field error in the form's language.
func fieldErrorText(fe *audience.FieldError, l pagelabels.Set) string {
	switch fe.Code {
	case "email":
		return l.Text("error.emailInvalid", nil)
	case "pick_list":
		return l.Text("error.pickList", nil)
	case "list_gone":
		return l.Text("error.listGone", nil)
	case "number", "date", "boolean":
		return l.Text("error."+fe.Code, nil)
	case "select":
		return l.Text("error.select", i18n.Args{"options": strings.Join(fe.Options, ", ")})
	case "too_long":
		return l.Text("error.tooLong", nil)
	}
	return fe.Message
}

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
	c = s.localize(ctx, ws, set, c, sub.Language)
	out := s.submit(ctx, ws, set, c, sub)
	out.Language = c.ShownLanguage()
	return out
}

func (s *Service) submit(ctx context.Context, ws *store.Workspace, set *Settings, c Config, sub Submission) Outcome {
	l := c.Labels
	if c.Status != StatusLive {
		return Outcome{Status: OutcomeClosed, Message: l.Text("error.closed", nil)}
	}
	now := s.now()
	limits := s.rates()
	ip := "unknown"
	if sub.IP != nil {
		ip = sub.IP.String()
	}
	if !limits.allow("ip:"+ws.ID.String()+":"+ip, c.Spam.RateLimitPerIP, now) || !limits.allow("form:"+formKey(c), formSubmitsPerMinute, now) {
		s.recordReject(ctx, ws, c, "rate_limit", sub.Origin)
		return Outcome{Status: OutcomeRateLimited, Message: l.Text("error.rateLimited", nil)}
	}
	if !OriginAllowed(sub.Origin, c.AllowedOrigins, s.OwnOrigins(set)) {
		s.recordReject(ctx, ws, c, "origin", sub.Origin)
		return Outcome{Status: OutcomeRejected, Message: l.Text("error.origin", nil)}
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
	// Phase 8: the phone field and the SMS consent box.
	offer, texts := s.textOffer(ctx, ws, c)
	checkPhone(c, offer, texts, &sub, errs)
	if len(errs) > 0 {
		return Outcome{Status: OutcomeInvalid, Errors: errs, Message: firstMessage(errs)}
	}
	if c.Spam.BlockDisposable {
		if emailaddr.IsDisposable(emailaddr.Domain(sub.Email)) {
			s.recordReject(ctx, ws, c, "disposable", sub.Origin)
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: l.Text("error.emailDisposable", nil)}}
		}
		if emailaddr.IsRole(sub.Email) {
			s.recordReject(ctx, ws, c, "role", sub.Origin)
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: l.Text("error.emailRole", nil)}}
		}
	}
	if suspicious {
		if c.Spam.Turnstile && set.TurnstileSiteKey() != "" {
			id, err := s.holdForChallenge(ctx, ws, c, sub)
			if err != nil {
				s.log().ErrorContext(ctx, "hold submission for turnstile", "err", err)
				return Outcome{Status: OutcomeError, Message: l.Text("error.saveFailed", nil)}
			}
			return Outcome{Status: OutcomeChallenge, ChallengeID: id, Message: l.Text("error.challenge", nil)}
		}
		s.recordReject(ctx, ws, c, "timing", sub.Origin)
		if expired {
			return Outcome{Status: OutcomeInvalid, Message: l.Text("error.expired", nil)}
		}
		return Outcome{Status: OutcomeInvalid, Message: l.Text("error.tooFast", nil)}
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

// check validates the fields and returns the lists to join. Its messages
// are in the form's language.
func (s *Service) check(c Config, sub *Submission) ([]uuid.UUID, map[string]string) {
	l := c.Labels
	errs := map[string]string{}
	email, problem := emailaddr.Check(sub.Email)
	switch {
	case strings.TrimSpace(sub.Email) == "":
		errs[FieldEmail] = l.Text("error.emailMissing", nil)
	case problem != "":
		errs[FieldEmail] = l.Text("error.emailInvalid", nil)
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
			errs[f.Key] = l.Text("error.required", nil)
		}
	}
	for k, v := range sub.Fields {
		if len(v) > 500 {
			errs[k] = l.Text("error.tooLong", nil)
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
		errs[FieldFirstName] = l.Text("error.nameTooLong", nil)
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
			errs["lists"] = l.Text("error.pickList", nil)
		}
	} else {
		lists = slices.Clone(c.Lists)
	}
	if len(lists) == 0 && errs["lists"] == "" {
		errs["lists"] = l.Text("error.noList", nil)
	}
	if c.ConsentText != "" && !sub.Consent {
		errs["consent"] = l.Text("error.consent", nil)
	}
	return lists, errs
}

// accept subscribes the contact and records the submission.
func (s *Service) accept(ctx context.Context, ws *store.Workspace, set *Settings, c Config, sub Submission, lists []uuid.UUID) Outcome {
	addr, refused := s.checkAddress(ctx, ws, c, sub)
	if refused != nil {
		return *refused
	}
	now := s.now()
	lang := c.ShownLanguage()
	ev := audience.PublicEvidence{Source: c.Source(), IP: sub.IP, UserAgent: sub.UserAgent, ConsentText: c.ConsentText, ConsentVersion: c.ConsentVersionLabel(), Language: lang, At: now}
	formName := c.Name
	if c.Builtin {
		formName = ""
	}
	fields := map[string]string{}
	for k, v := range sub.Fields {
		if c.HasField(k) {
			fields[k] = v
		}
	}
	var res audience.SignupResult
	var submissionID uuid.UUID
	// A signup through a referral link always confirms by email.
	doubleOptIn := c.DoubleOptIn || s.forceDouble(ctx, ws, sub)
	confirmTexts := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		res, err = s.Audience.Signup(ctx, tx, ws, audience.SignupInput{
			Email: sub.Email, FirstName: sub.FirstName, LastName: sub.LastName, Company: sub.Company, Fields: fields,
			Lists: lists, DoubleOptIn: doubleOptIn, Evidence: ev, BrowserLanguage: sub.BrowserLanguage, FormName: formName,
		})
		if err != nil || res.Blocked {
			return err
		}
		// Phase 8: the number and SMS consent, with its evidence.
		if confirmTexts, err = s.saveTexts(ctx, tx, ws, c, sub, res.ContactID, lists, now); err != nil {
			return err
		}
		q := store.New(tx)
		// UTM parameters and the referring page go with the consent records.
		if err := growth.SaveTouch(ctx, q, res.ContactID, now, growth.TouchOf(sub.Page, sub.Referrer)); err != nil {
			return err
		}
		var formID *uuid.UUID
		if !c.Builtin {
			id := c.ID
			formID = &id
		}
		if err := s.signupAccepted(ctx, tx, ws, SignupEvent{ContactID: res.ContactID, Email: res.Email, FirstName: res.FirstName, Created: res.Created,
			Lists: lists, Pending: res.Pending, Subscribed: res.Subscribed, Already: res.Already, Page: sub.Page, Referrer: sub.Referrer,
			IP: sub.IP, UserAgent: sub.UserAgent, FormID: formID, At: now}); err != nil {
			return err
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
			ListIds: touched, DoubleOptIn: double, Language: lang, At: now, ConfirmedAt: confirmedAt}); err != nil {
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
				// The form.submitted webhook event fires at the same moment.
				answers := map[string]string{FieldFirstName: sub.FirstName, FieldLastName: sub.LastName, FieldCompany: sub.Company}
				for k, v := range fields {
					answers[k] = v
				}
				if err := emitFormSubmitted(ctx, tx, ws, *formID, res.ContactID, answers, now, now); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		var fe *audience.FieldError
		if errors.As(err, &fe) {
			msg := fieldErrorText(fe, c.Labels)
			return Outcome{Status: OutcomeInvalid, Errors: map[string]string{fe.Field: msg}, Message: msg}
		}
		s.log().ErrorContext(ctx, "form submission", "form", c.ID, "err", err)
		return Outcome{Status: OutcomeError, Message: c.Labels.Text("error.saveFailed", nil)}
	}
	if s.Addresses != nil && res.ContactID != uuid.Nil {
		s.Addresses.SignedUp(ctx, ws, res.ContactID, addr)
	}
	out := Outcome{Status: OutcomeSubscribed, Email: sub.Email, FirstName: res.FirstName, Lists: lists}
	if confirmTexts {
		s.Texts.Confirm(ctx, ws, res.ContactID, lists)
		out.Texted = s.Texts.Texted(lang, sub.number.E164)
	}
	if doubleOptIn || res.NeedsConfirmation() || res.Blocked {
		out.Status = OutcomeConfirm
	}
	if res.NeedsConfirmation() && submissionID != uuid.Nil {
		if err := s.sendConfirmation(ctx, ws, set, submissionID, res.ContactID, false, lang); err != nil && !errors.Is(err, errNoSystemMail) {
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
	// Page and Referrer aren't in Sub's JSON: they come along so a
	// challenged signup keeps its touch and its referral code.
	Page     string `json:"page,omitempty"`
	Referrer string `json:"referrer,omitempty"`
}

const challengeTTL = 30 * time.Minute

func (s *Service) holdForChallenge(ctx context.Context, ws *store.Workspace, c Config, sub Submission) (string, error) {
	h := heldSubmission{Builtin: c.Builtin, Sub: sub, Origin: sub.Origin, UserAgent: sub.UserAgent, Page: sub.Page, Referrer: sub.Referrer}
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
func (s *Service) SolveChallenge(ctx context.Context, ws *store.Workspace, set *Settings, challengeID, answer, requested string, ip *netip.Addr, formOf ChallengeForm) Outcome {
	out, lang := s.solveChallenge(ctx, ws, set, challengeID, answer, requested, ip, formOf)
	out.Language = lang
	return out
}

func (s *Service) solveChallenge(ctx context.Context, ws *store.Workspace, set *Settings, challengeID, answer, requested string, ip *netip.Addr, formOf ChallengeForm) (Outcome, string) {
	// Until the held submission is open, messages follow the language the
	// challenge page is shown in (requested).
	pl := pagelabels.English()
	if pc, err := s.loadPagesCopy(ctx, s.Q, ws, set); err == nil {
		if code := languages.Base(requested); code != "" && pc.Live(code) {
			pl = pc.Labels(code)
		}
	}
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return Outcome{Status: OutcomeInvalid, Message: pl.Text("error.challengeExpired", nil)}, pl.Code()
	}
	row, err := s.Q.FormChallengeTake(ctx, store.FormChallengeTakeParams{ID: id, WorkspaceID: ws.ID, At: s.now()})
	if err != nil {
		return Outcome{Status: OutcomeInvalid, Message: pl.Text("error.challengeExpired", nil)}, pl.Code()
	}
	raw, err := s.Box.Open(row.PayloadEnc, "form_challenge:"+row.ID.String())
	if err != nil {
		return Outcome{Status: OutcomeError, Message: pl.Text("error.saveFailed", nil)}, pl.Code()
	}
	var h heldSubmission
	if err := json.Unmarshal(raw, &h); err != nil {
		return Outcome{Status: OutcomeError, Message: pl.Text("error.saveFailed", nil)}, pl.Code()
	}
	c, err := formOf(ctx, h.FormID, h.Builtin)
	if err != nil {
		return Outcome{Status: OutcomeClosed, Message: pl.Text("error.closed", nil)}, pl.Code()
	}
	c = s.localize(ctx, ws, set, c, h.Sub.Language)
	l := c.Labels
	if c.Status != StatusLive {
		return Outcome{Status: OutcomeClosed, Message: l.Text("error.closed", nil)}, c.ShownLanguage()
	}
	secret, ok := s.turnstileSecret(set)
	if !ok {
		return Outcome{Status: OutcomeError, Message: l.Text("error.saveFailed", nil)}, c.ShownLanguage()
	}
	remote := ""
	if ip != nil {
		remote = ip.String()
	}
	res, err := s.Turnstile.Verify(ctx, secret, answer, remote, row.ID.String())
	if err != nil {
		s.log().WarnContext(ctx, "turnstile verify", "err", err)
		return Outcome{Status: OutcomeError, Message: l.Text("error.verifyUnreachable", nil)}, c.ShownLanguage()
	}
	if !res.Success {
		if res.SecretRejected() {
			if err := s.turnstileFailed(ctx, ws, secretRejected(res.Codes), map[string]any{"codes": res.Codes}); err != nil {
				s.log().WarnContext(ctx, "record turnstile error", "err", err)
			}
		}
		s.recordReject(ctx, ws, c, "turnstile", h.Origin)
		return Outcome{Status: OutcomeRejected, Message: l.Text("error.verifyFailed", nil)}, c.ShownLanguage()
	}
	sub := h.Sub
	if h.IP != "" {
		if a, err := netip.ParseAddr(h.IP); err == nil {
			sub.IP = &a
		}
	}
	sub.UserAgent, sub.Origin = h.UserAgent, h.Origin
	sub.Page, sub.Referrer = h.Page, h.Referrer
	lists, errs := s.check(c, &sub)
	offer, texts := s.textOffer(ctx, ws, c)
	checkPhone(c, offer, texts, &sub, errs)
	if len(errs) > 0 {
		return Outcome{Status: OutcomeInvalid, Errors: errs, Message: firstMessage(errs)}, c.ShownLanguage()
	}
	return s.accept(ctx, ws, set, c, sub, lists), c.ShownLanguage()
}
