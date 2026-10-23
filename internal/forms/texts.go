package forms

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// Texts on signup forms (design expansion phase 8, brief §3.3 and §6): a
// phone field with a country picker, and an SMS consent checkbox with its
// own text, unchecked by default. SMS consent is kept per list with its
// evidence (the number, the list, the exact text and its version, the page,
// the IP address and the time) by internal/texting, which the app wires in
// as Service.Texts. Email and texts stay apart: the email consent box never
// signs anyone up for texts.

// FieldPhone is the phone field.
const FieldPhone = "phone"

func init() {
	builtinFields[FieldPhone] = "Phone"
}

// Texts is the texting side of signup forms. Nil when texting isn't wired:
// forms then offer no phone field and no SMS consent.
type Texts interface {
	// Offer is what a form needs to collect texts in a workspace, in a
	// language (English when texting has no words in it); false when the
	// workspace doesn't text.
	Offer(ctx context.Context, ws *store.Workspace, lang string) (TextOffer, bool)
	// Save records a signup's number on the contact and, when consent is
	// true, SMS consent on the lists with its evidence, in the signup's
	// transaction. It reports whether to text the opt-in confirmation.
	Save(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, number phone.Number, consent bool, lists []uuid.UUID, ev TextEvidence) (bool, error)
	// Confirm texts the opt-in confirmation, once the transaction committed.
	Confirm(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lists []uuid.UUID)
	// Texted is the line "We've also texted +1 312 ••• 0123 to confirm.",
	// for a number, in a language.
	Texted(lang, e164 string) string
	// TextedContact is that line for a contact whose opt-in confirmation
	// went out in the last week (the email confirmation page); empty
	// otherwise.
	TextedContact(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lang string) string
}

// TextOffer is the SMS part of a form, in one language.
type TextOffer struct {
	// Countries are the phone field's picker, the routed ones first;
	// Default is the one it starts on.
	Countries []TextCountry
	Default   string
	// ConsentText is the default SMS consent text: the program, how often,
	// "Msg & data rates may apply", HELP and STOP, the terms and the privacy
	// policy. TermsWords are its words for the terms page, linked to TermsURL.
	ConsentText string
	TermsWords  string
	TermsURL    string
	// The page's words.
	PhoneLabel   string
	CountryLabel string
	Placeholder  string
	InvalidPhone string
	PhoneNeeded  string
}

// TextCountry is a country of the phone field's picker: US, United States, +1.
type TextCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Dial string `json:"dial"`
}

// TextEvidence is what SMS consent is recorded with.
type TextEvidence struct {
	Source         string
	Page           string
	ConsentText    string
	ConsentVersion string
	IP             string
	UserAgent      string
	At             time.Time
}

// SMSConsentText is the form's SMS consent text in its language: its own,
// else the default one.
func (c Config) SMSConsentText(offer TextOffer) string {
	if t := strings.TrimSpace(c.Content.SmsConsentText); t != "" {
		return t
	}
	return offer.ConsentText
}

// offersTexts reports whether the form shows the SMS consent checkbox: it
// needs the phone field, and a workspace that texts.
func (c Config) offersTexts() bool {
	return c.Content.SmsConsent && c.HasField(FieldPhone)
}

// smsConsentOf reads a saved form's SMS consent: the box needs the phone
// field, and the text gets a new version whenever its words change.
func smsConsentOf(ctx context.Context, content *contentDoc, in SignupFormContent, fields []fieldDoc, before []byte) error {
	content.SmsConsent = in.SmsConsent
	content.SmsConsentText = strings.TrimSpace(in.SmsConsentText)
	if content.SmsConsent && !slices.ContainsFunc(fields, func(f fieldDoc) bool { return f.Key == FieldPhone }) {
		return api.Invalid("body.content.smsConsent", say(ctx, mSmsNeedsPhone, nil))
	}
	var prev contentDoc
	_ = json.Unmarshal(before, &prev)
	content.SmsConsentVersion = max(prev.SmsConsentVersion, 1)
	if prev.SmsConsentText != "" && content.SmsConsentText != prev.SmsConsentText {
		content.SmsConsentVersion++
	}
	return nil
}

// textOffer is the workspace's offer in the form's language, when texting
// is wired and the workspace texts.
func (s *Service) textOffer(ctx context.Context, ws *store.Workspace, c Config) (TextOffer, bool) {
	if s.Texts == nil {
		return TextOffer{}, false
	}
	return s.Texts.Offer(ctx, ws, c.ShownLanguage())
}

// saveTexts records the signup's number and SMS consent in its
// transaction; true when the opt-in confirmation is to be texted.
func (s *Service) saveTexts(ctx context.Context, tx pgx.Tx, ws *store.Workspace, c Config, sub Submission, contactID uuid.UUID, lists []uuid.UUID, now time.Time) (bool, error) {
	if s.Texts == nil || sub.number.E164 == "" {
		return false, nil
	}
	ev := TextEvidence{Source: c.Source(), Page: evidencePage(sub.Page, c), UserAgent: sub.UserAgent, At: now}
	if sub.IP != nil {
		ev.IP = sub.IP.String()
	}
	if sub.SmsConsent {
		offer, _ := s.textOffer(ctx, ws, c)
		ev.ConsentText = c.SMSConsentText(offer)
		ev.ConsentVersion = strconv.Itoa(max(c.Content.SmsConsentVersion, 1))
	}
	return s.Texts.Save(ctx, tx, ws, contactID, sub.number, sub.SmsConsent, lists, ev)
}

// evidencePage is where consent was given: the page's address without its
// query ("news.acme.io/summit"), else the form's name.
func evidencePage(page string, c Config) string {
	if u, err := url.Parse(page); err == nil && u.Host != "" {
		return u.Host + strings.TrimRight(u.Path, "/")
	}
	if c.Builtin {
		return "Subscribe page"
	}
	return c.Name
}

// checkPhone reads the phone field. A number given without SMS consent is
// saved too; SMS consent without a number asks for one.
func checkPhone(c Config, offer TextOffer, ok bool, sub *Submission, errs map[string]string) {
	if !ok || !c.HasField(FieldPhone) {
		sub.Phone, sub.SmsConsent, sub.number = "", false, phone.Number{}
		return
	}
	if !c.offersTexts() {
		sub.SmsConsent = false
	}
	raw := strings.TrimSpace(sub.Phone)
	f, _ := c.Field(FieldPhone)
	switch {
	case raw == "" && sub.SmsConsent:
		errs[FieldPhone] = offer.PhoneNeeded
	case raw == "" && f.Required:
		errs[FieldPhone] = c.Labels.Text("error.required", nil)
	case raw != "":
		n, err := phone.Parse(raw, sub.PhoneCountry)
		if err != nil {
			errs[FieldPhone] = offer.InvalidPhone
			return
		}
		sub.number = n
	}
}
