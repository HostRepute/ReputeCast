package sponsors

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The sponsor pages' formatting (internal/publicpages): prices, dates and
// the slot's schedule as the boards write them, in the page's language.

// Money is a price as the sponsor pages write it: "$750".
func Money(cents int64, currency string) string { return money(cents, currency) }

// IssueDate is an issue's day with its weekday, and its year when it
// isn't now's: "Tue 18 Nov", "Tue 6 Jan 2027".
func IssueDate(l i18n.Lang, t, now time.Time) string {
	if t.Year() == now.Year() {
		return shortDate(l, t)
	}
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("Mon 2 Jan 2006")
	}
	return l.Region.WeekdayDate(t)
}

// DayMonth is a day without its weekday: "13 Oct".
func DayMonth(l i18n.Lang, t time.Time) string {
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("2 Jan")
	}
	return l.Region.DayMonth(t)
}

// Stamp is a moment in the workspace's time zone: "14 Oct, 11:42".
func Stamp(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	t = t.In(location(ws))
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("2 Jan, 15:04")
	}
	return l.Region.DayMonth(t) + ", " + l.Region.Time(t, "")
}

// Every is a slot's schedule in a sentence: "every Tuesday" for a weekly
// slot (the weekday of its issues), "every 3 days" otherwise.
func Every(l i18n.Lang, everyDays int, issues []PageIssue) string {
	if (everyDays == 7 || everyDays == 14) && len(issues) > 0 {
		day := strings.ToLower(issues[0].On.Weekday().String()[:3])
		if everyDays == 7 {
			return msgs.M("kit.everyWeek").Text(l, i18n.Args{"weekday": day})
		}
	}
	return msgs.M("kit.everyDays").Text(l, i18n.Args{"days": everyDays})
}

// CurrencyName names a price's currency in a sentence ("US dollars"), as
// Earn's pages do; its code when Earn isn't there.
func CurrencyName(l i18n.Lang, code string) string {
	return earn.CurrencyNameIn(l, code)
}

// PlacementInSentence is a slot's placement inside a sentence: "after the
// first story" (its first letter lower-cased where the sentence is
// English, as the publisher wrote it otherwise).
func PlacementInSentence(l i18n.Lang, placement string) string {
	p := strings.TrimSpace(placement)
	if p == "" || !EnglishText(l, "req.blockNote") {
		return p
	}
	r, n := utf8.DecodeRuneInString(p)
	if n < len(p) {
		if next, _ := utf8.DecodeRuneInString(p[n:]); unicode.IsUpper(next) {
			return p // an acronym or a name: "TL;DR", "Top"
		}
	}
	return string(unicode.ToLower(r)) + p[n:]
}

// Initials are a company's initials for its sponsor block: "FS".
func Initials(name string) string { return initials(name) }

// ContactOf is where sponsors write about the workspace's sponsorships,
// even while the sponsor page is off.
func (s *Service) ContactOf(ctx context.Context, ws *store.Workspace) string {
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return ""
	}
	return s.contact(ctx, ws, st)
}
