package texting

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/collate"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// Texts on signup forms (brief §3.3, §6): the phone field's country picker,
// the SMS consent box's default text, SMS consent with its evidence, and
// the opt-in confirmation text with the "We've also texted…" line on the
// pages that follow. The forms package calls these through forms.Texts.

// SignupTexts is texting's side of signup forms (forms.Texts).
type SignupTexts struct{ S *Service }

var _ forms.Texts = SignupTexts{}

var (
	mSignupPhone        = msgs.M("signup.phone")
	mSignupCountry      = msgs.M("signup.country")
	mSignupPlaceholder  = msgs.M("signup.placeholder")
	mSignupConsent      = msgs.M("signup.consent")
	mSignupTermsWords   = msgs.M("signup.termsWords")
	mSignupInvalidPhone = msgs.M("signup.invalidPhone")
	mSignupPhoneNeeded  = msgs.M("signup.phoneNeeded")
	mSignupTexted       = msgs.M("signup.texted")
)

// Offer implements forms.Texts: nothing when the workspace has no SMS
// provider. The picker lists the routed countries first and, when every
// other country has a route too, the rest by name.
func (t SignupTexts) Offer(ctx context.Context, ws *store.Workspace, code string) (forms.TextOffer, bool) {
	s := t.S
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil || len(g.providers) == 0 {
		return forms.TextOffer{}, false
	}
	lang := pagelabels.Lang(code)
	brand := strings.TrimSuffix(strings.TrimSpace(g.rules.BrandPrefix), ":")
	if brand == "" {
		brand = ws.Name
	}
	out := forms.TextOffer{
		ConsentText: mSignupConsent.Text(lang, i18n.Args{"brand": brand}), TermsWords: mSignupTermsWords.Text(lang, nil), TermsURL: s.TermsURL(ctx, ws),
		PhoneLabel: mSignupPhone.Text(lang, nil), CountryLabel: mSignupCountry.Text(lang, nil), Placeholder: mSignupPlaceholder.Text(lang, nil),
		InvalidPhone: mSignupInvalidPhone.Text(lang, nil), PhoneNeeded: mSignupPhoneNeeded.Text(lang, nil),
	}
	routed := []string{}
	for _, r := range g.routes {
		for _, c := range r.Countries {
			if !slices.Contains(routed, c) {
				routed = append(routed, c)
			}
		}
	}
	out.Countries = pickerCountries(lang, routed, g.rules.OtherCountries == "route" || len(routed) == 0)
	if len(out.Countries) > 0 {
		out.Default = out.Countries[0].Code
	}
	return out, true
}

// pickerCountries are the phone field's countries: the routed ones first,
// then, with every other, the rest by name. Names are in the page's
// language when ReputeCast has it (countryNameIn), sorted as it sorts them.
func pickerCountries(lang i18n.Lang, routed []string, others bool) []forms.TextCountry {
	out := []forms.TextCountry{}
	add := func(c string) {
		if cc := phone.CallingCode(c); cc > 0 {
			out = append(out, forms.TextCountry{Code: c, Name: countryNameIn(lang, c), Dial: "+" + itoa(cc)})
		}
	}
	for _, c := range routed {
		add(c)
	}
	if !others {
		return out
	}
	rest := []string{}
	for _, c := range phone.Countries() {
		if !slices.Contains(routed, c) {
			rest = append(rest, c)
		}
	}
	compare := strings.Compare
	if localNames(lang) {
		compare = collate.New(lang.Locale.Tag()).CompareString
	}
	slices.SortFunc(rest, func(a, b string) int { return compare(countryNameIn(lang, a), countryNameIn(lang, b)) })
	for _, c := range rest {
		add(c)
	}
	return out
}

// Save implements forms.Texts: the number goes on the contact (the latest
// one typed wins), and SMS consent on each list with its evidence.
func (t SignupTexts) Save(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, n phone.Number, consent bool, lists []uuid.UUID,
	ev forms.TextEvidence) (bool, error) {
	s := t.S
	q := store.New(tx)
	at := ev.At
	if at.IsZero() {
		at = s.now()
	}
	if err := q.ChanSetPhone(ctx, store.ChanSetPhoneParams{Phone: n.E164, PhoneCountry: n.Country, PhoneLineType: string(n.Type), PhoneLineSource: "number_plan",
		At: &at, ID: contactID, WorkspaceID: ws.ID}); err != nil {
		return false, err
	}
	if !consent {
		return false, nil
	}
	confirm := false
	for _, l := range lists {
		c, err := s.SetConsent(ctx, q, ws, contactID, n.E164, l, "subscribed", Evidence{Source: ev.Source, Page: ev.Page, ConsentText: ev.ConsentText,
			ConsentVersion: ev.ConsentVersion, IP: ev.IP, UserAgent: ev.UserAgent, At: at})
		if err != nil {
			return false, err
		}
		confirm = confirm || c
	}
	return confirm, nil
}

// Confirm implements forms.Texts: one opt-in confirmation for the signup,
// on its first list.
func (t SignupTexts) Confirm(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lists []uuid.UUID) {
	if len(lists) > 0 {
		t.S.SendConfirmation(ctx, ws, contactID, lists[0])
	}
}

// Texted implements forms.Texts.
func (SignupTexts) Texted(code, e164 string) string {
	return mSignupTexted.Text(pagelabels.Lang(code), i18n.Args{"phone": maskPhone(e164)})
}

// TextedContact implements forms.Texts: the contact's opt-in confirmation
// of the last week, queued or sent.
func (t SignupTexts) TextedContact(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, code string) string {
	var recipient string
	err := t.S.Pool.QueryRow(ctx, `SELECT recipient FROM channel_messages WHERE workspace_id = $1 AND contact_id = $2 AND kind = 'confirmation'
		AND created_at > $3 AND status NOT IN ('failed','undelivered','cancelled','skipped') ORDER BY created_at DESC LIMIT 1`,
		ws.ID, contactID, t.S.now().Add(-7*24*time.Hour)).Scan(&recipient)
	if err != nil || recipient == "" {
		return ""
	}
	return t.Texted(code, recipient)
}
