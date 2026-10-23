package earn

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// defaultLang is the language of text written for the team when no
// request says otherwise (notifications are stored in English).
var defaultLang = i18n.Lang{Locale: i18n.English, Region: i18n.Standard}

// symbols are the currency symbols money is written with; other
// currencies use their code ("CHF 8").
var symbols = map[string]string{"usd": "$", "eur": "€", "gbp": "£", "cad": "CA$", "aud": "A$", "jpy": "¥", "inr": "₹", "brl": "R$"}

// zeroDecimal currencies have no minor unit.
var zeroDecimal = map[string]bool{"jpy": true, "krw": true, "vnd": true, "clp": true, "isk": true, "ugx": true, "xof": true, "xaf": true}

// money writes an amount in minor units the design's way: the symbol
// first, thousands separators, a minus sign for negative amounts; whole
// when wholeIfRound and the amount has no cents ($8, $64.00 otherwise).
func money(cents int64, currency string, wholeIfRound bool) string {
	cur := strings.ToLower(currency)
	sign := ""
	if cents < 0 {
		sign, cents = "−", -cents
	}
	sym, ok := symbols[cur]
	if !ok {
		sym = strings.ToUpper(cur) + " "
	}
	if zeroDecimal[cur] {
		return sign + sym + thousands(cents)
	}
	whole, frac := cents/100, cents%100
	if wholeIfRound && frac == 0 {
		return sign + sym + thousands(whole)
	}
	return sign + sym + thousands(whole) + "." + fmt.Sprintf("%02d", frac)
}

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// priceText is a price with its period in English, as audit entries
// keep it: "$8/month".
func priceText(cents int64, currency, period string) string {
	return priceTextIn(defaultLang, cents, currency, period)
}

// publicPrice is a price in English, as public pages and emails write it:
// "$8 a month".
func publicPrice(cents int64, currency, period string) string {
	return publicPriceIn(defaultLang, cents, currency, period)
}

// location is the workspace's time zone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// dateText writes a date in English in the workspace's time zone ("7 Oct
// 2027"), without the year when it's this year ("16 Oct").
func (s *Service) dateText(ws *store.Workspace, t time.Time) string {
	return s.dateTextIn(defaultLang, ws, t)
}

// longDate always has the year ("7 Oct 2027").
func longDate(ws *store.Workspace, t time.Time) string { return longDateIn(defaultLang, ws, t) }

// pageURL is an Earn page's address: on the workspace's own domain when it
// has one (news.acme.io/upgrade), else under /p/{workspace}.
func (s *Service) pageURL(ctx context.Context, ws *store.Workspace, path string) string {
	if s.PageBase != nil {
		if base := s.PageBase(ctx, ws); base != "" {
			return strings.TrimRight(base, "/") + "/" + path
		}
	}
	return strings.TrimRight(s.URLs.Base(), "/") + "/p/" + ws.Slug + "/" + path
}

// pageHost is pageURL without its scheme ("news.acme.io/upgrade").
func (s *Service) pageHost(ctx context.Context, ws *store.Workspace, path string) string {
	u := s.pageURL(ctx, ws, path)
	return strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
}

// article is "an" before a word English spells with a, e, i or o first
// ("an Insider", "a Founding member").
func article(word string) string {
	if word != "" && strings.ContainsRune("aeioAEIO", rune(word[0])) {
		return "an"
	}
	return "a"
}

// PageLang is the Earn pages' language when nothing picks one: English,
// in ReputeCast's own formats.
var PageLang = defaultLang

// Text is one of the Earn pages' built-in words in English.
func Text(key string, args i18n.Args) string { return msgs.M(key).Text(PageLang, args) }

// TextIn is one of the Earn pages' built-in words in a page's language
// (internal/publicpages).
func TextIn(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }
