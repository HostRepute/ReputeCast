package earn

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// Who reads Earn's server text, and in which language:
//
//   - members read the app's text (problems, the editors, notifications) in
//     their app language: lang(ctx), and kept messages (Msg.Keep) for what
//     is stored and read later;
//   - readers of the public Earn pages (the upgrade, welcome, gift and
//     cancel pages, the paid parts of the preference center) read the
//     page's language, which internal/publicpages picks and passes as an
//     i18n.Lang (UpgradeRequest.Lang, or the context of the request);
//   - readers of an email (member emails, the paid.* values, the Upgrade
//     block's lines) read the email's language: the language version the
//     send picks, else the workspace's default language.
//
// Dates and prices follow the language's region format. The ReputeCast
// standard format keeps the forms the pages and emails always had ("7 Oct
// 2027", "16 Oct", "Tue 21 Oct").

// notFound is a 404 whose sentence is in the reader's language: what names
// a notFound.* message ("tier": "Tier not found.").
func notFound(ctx context.Context, what string) *api.Problem {
	return problem(ctx, http.StatusNotFound, api.CodeNotFound, "notFound."+what, nil)
}

// mapErr is api.MapError with its not-found sentence in the reader's
// language.
func mapErr(ctx context.Context, err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx, what)
	}
	return api.MapError(err, msgs.M("notFound."+what).Source())
}

// detailGrantNote is the key of a subscription's source detail that keeps
// its grant note as a message (complimentary access an import gave).
const detailGrantNote = "grant_note"

// keptOr writes text kept as a message in a reader's language, as long as
// the English stored beside it is still what the message says (a person
// may have rewritten it since); otherwise, and for rows from before, the
// stored English.
func keptOr(t i18n.Text, l i18n.Lang, english string) string {
	if t.IsZero() || t.In(defaultLang) != english {
		return english
	}
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// sourceKept is a paid member's source as ReputeCast wrote it, as a message
// for its readers' languages, for a row that keeps none beside it (written
// before sources kept one): an import's "Imported from Substack", a gift's
// "Gift from Priya Natarajan". A label that isn't ReputeCast's sentence
// stays as stored (zero Text).
func sourceKept(kind, label, giver string) i18n.Text {
	switch kind {
	case "import":
		for _, src := range importSources {
			if t := msgs.M("import.source").Keep(platformArgs(src)); t.In(defaultLang) == label {
				return t
			}
		}
	case "gift":
		if giver != "" {
			if t := msgs.M("source.gift").Keep(i18n.Args{"giver": giver}); t.In(defaultLang) == label {
				return t
			}
		}
	}
	return i18n.Text{}
}

// standard reports whether a language writes ReputeCast's own formats.
func standard(l i18n.Lang) bool { return l.Region == "" || l.Region == i18n.Standard }

// dateIn is a day with its year: "7 Oct 2027", "7 thg 10, 2027".
func dateIn(l i18n.Lang, t time.Time) string {
	if standard(l) {
		return t.Format("2 Jan 2006")
	}
	return l.Region.Date(t)
}

// dayMonthIn is a day without its year: "16 Oct", "16 thg 10".
func dayMonthIn(l i18n.Lang, t time.Time) string {
	if standard(l) {
		return t.Format("2 Jan")
	}
	return l.Region.DayMonth(t)
}

// shortDateIn is a day with its weekday: "Tue 21 Oct", "Th 3, 21 thg 10".
func shortDateIn(l i18n.Lang, t time.Time) string {
	if standard(l) {
		return t.Format("Mon 2 Jan")
	}
	return l.Region.ShortDate(t)
}

// dateTextIn writes a date in the workspace's time zone without the year
// when it's this year ("16 Oct"), with it otherwise ("7 Oct 2027").
func (s *Service) dateTextIn(l i18n.Lang, ws *store.Workspace, t time.Time) string {
	loc := location(ws)
	t = t.In(loc)
	if t.Year() == s.now().In(loc).Year() {
		return dayMonthIn(l, t)
	}
	return dateIn(l, t)
}

// longDateIn always has the year, in the workspace's time zone.
func longDateIn(l i18n.Lang, ws *store.Workspace, t time.Time) string {
	return dateIn(l, t.In(location(ws)))
}

var (
	mPriceSlash = msgs.M("price.slash")
	mPriceWords = msgs.M("price.words")
	mCurrency   = msgs.M("page.currency")
)

// priceTextIn is a price with its period, as the app writes it: "$8/month".
func priceTextIn(l i18n.Lang, cents int64, currency, period string) string {
	return mPriceSlash.Text(l, i18n.Args{"price": money(cents, currency, true), "period": periodKey(period)})
}

// publicPriceIn is a price as public pages and emails write it: "$8 a month".
func publicPriceIn(l i18n.Lang, cents int64, currency, period string) string {
	return mPriceWords.Text(l, i18n.Args{"price": money(cents, currency, true), "period": periodKey(period)})
}

func periodKey(period string) string {
	if period == "month" || period == "year" {
		return period
	}
	return "none"
}

// currencyNameIn names a currency in a sentence: "US dollars".
func currencyNameIn(l i18n.Lang, code string) string {
	code = strings.ToLower(code)
	return mCurrency.Text(l, i18n.Args{"code": code, "upper": strings.ToUpper(code)})
}

// englishText reports whether a message is written in English for a
// language: the language hasn't translated it. English grammar choices the
// code makes (a tier's name in lower case in "Become an insider") apply
// only then.
func englishText(l i18n.Lang, key string) bool {
	if l.Locale == i18n.English || l.Locale == i18n.Pseudo || l.Locale == "" {
		return true
	}
	return slices.Contains(msgs.Missing(l.Locale), key)
}

// lowerFirst writes a name with a small first letter, as English does
// inside a sentence ("Become a founding member").
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// workspaceLang is the language a workspace writes its emails and pages in
// by default, as its readers read it (pagelabels: English with the
// standard formats, Vietnamese with its own, other languages in their
// region's formats).
func workspaceLang(ctx context.Context, q *store.Queries, wsID uuid.UUID) i18n.Lang {
	return pagelabels.Lang(defaultLanguage(ctx, q, wsID))
}

// defaultLanguage is the code of a workspace's default content language
// ("en" when it never set one).
func defaultLanguage(ctx context.Context, q *store.Queries, wsID uuid.UUID) string {
	rows, err := q.LanguagesList(ctx, wsID)
	if err != nil {
		return "en"
	}
	for _, r := range rows {
		if r.IsDefault && r.RemovedAt == nil {
			return r.Code
		}
	}
	return "en"
}

// localizer turns an email's texts into a send's Localize: the values in
// the language of the version the send picks ("" for the workspace's
// default language).
func (s *Service) localizer(ctx context.Context, ws *store.Workspace, texts func(l i18n.Lang) map[string]any) func(string) map[string]any {
	if texts == nil {
		return nil
	}
	return func(code string) map[string]any {
		if code == "" {
			code = defaultLanguage(ctx, s.Q, ws.ID)
		}
		return texts(pagelabels.Lang(code))
	}
}

// stripeReasons are the failure messages Stripe writes for the usual
// decline codes, with their messages' keys (reason.full.*, reason.short.*).
var stripeReasons = map[string]string{
	"Your card has insufficient funds.":                                        "funds",
	"Your card was declined.":                                                  "declined",
	"Your card has expired.":                                                   "expired",
	"Your card's security code is incorrect.":                                  "cvc",
	"Your card’s security code is incorrect.":                                  "cvc",
	"Your card number is incorrect.":                                           "number",
	"Your card does not support this type of purchase.":                        "unsupported",
	"An error occurred while processing your card. Try again in a little bit.": "processing",
}

// failureSentence is a payment failure as a member email says it: Stripe's
// reason in the email's language when it's one of the usual ones, else as
// Stripe wrote it.
func failureSentence(l i18n.Lang, reason string) string {
	if k, ok := stripeReasons[strings.TrimSpace(reason)]; ok {
		return msgs.M("reason.full."+k).Text(l, nil)
	}
	return reason
}

// failureShort is a payment failure in a few words, for the paid member's
// panel ("insufficient funds"): the usual reasons in the reader's
// language, others as Stripe wrote them.
func failureShort(l i18n.Lang, reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	if k, ok := stripeReasons[reason]; ok {
		return msgs.M("reason.short."+k).Text(l, nil)
	}
	return strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(reason, "."), "Your card has "))
}
