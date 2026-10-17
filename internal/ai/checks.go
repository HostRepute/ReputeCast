package ai

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// liquidRe finds Liquid tokens and tags as written.
	liquidRe = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)
	// hrefRe finds the addresses of links in HTML.
	hrefRe  = regexp.MustCompile(`(?i)<a\b[^>]*?\bhref\s*=\s*("([^"]*)"|'([^']*)')`)
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`\s+`)
	wordRe  = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}|[^\s]+`)
	// numberRe finds the numbers of a summary: 47,548 · 99.1% · 1.0%.
	numberRe = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?%?`)
)

// tokens returns the Liquid tokens of s, in order.
func tokens(s string) []string { return liquidRe.FindAllString(s, -1) }

// tokenNames are the variables tokens use ("first_name"), for the card.
func tokenNames(ts []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range ts {
		inner := strings.TrimSpace(strings.Trim(strings.TrimSpace(t), "{}%"))
		name := strings.Fields(strings.Split(inner, "|")[0])
		if len(name) == 0 {
			continue
		}
		n := name[0]
		if n == "if" || n == "unless" || n == "for" || n == "elsif" {
			if len(name) > 1 {
				n = strings.Split(name[1], ".")[0]
			}
		}
		if strings.HasPrefix(n, "end") || n == "else" {
			continue
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// links returns the addresses of the links of s, in order.
func links(s string) []string {
	out := []string{}
	for _, m := range hrefRe.FindAllStringSubmatch(s, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		out = append(out, html.UnescapeString(v))
	}
	return out
}

// keepResult says what a check of a suggestion against its source found.
type keepResult struct {
	Tokens         int
	TokensRestored int
	Links          int
	LinksRestored  int
}

// restore puts back the source's tokens and link addresses the model
// changed or dropped: a changed token or address is replaced by the
// source's in order, a dropped token is added at the end, and every
// restoration is counted.
func restore(src, got string) (string, keepResult) {
	r := keepResult{}
	want := tokens(src)
	r.Tokens = len(want)
	if len(want) > 0 {
		have := tokens(got)
		if !sameMultiset(want, have) {
			// Pair the suggestion's tokens with the source's in order.
			i := 0
			got = liquidRe.ReplaceAllStringFunc(got, func(t string) string {
				if i < len(want) {
					w := want[i]
					i++
					if w != t {
						r.TokensRestored++
					}
					return w
				}
				r.TokensRestored++
				return ""
			})
			for ; i < len(want); i++ {
				got = strings.TrimRight(got, " ") + " " + want[i]
				r.TokensRestored++
			}
		}
	}
	wantLinks := links(src)
	r.Links = len(wantLinks)
	if len(wantLinks) > 0 {
		haveLinks := links(got)
		if !sameMultiset(wantLinks, haveLinks) {
			i := 0
			got = hrefRe.ReplaceAllStringFunc(got, func(m string) string {
				if i >= len(wantLinks) {
					return m
				}
				w := wantLinks[i]
				i++
				sub := hrefRe.FindStringSubmatch(m)
				cur := sub[2]
				if cur == "" {
					cur = sub[3]
				}
				if html.UnescapeString(cur) == w {
					return m
				}
				r.LinksRestored++
				return strings.Replace(m, sub[1], `"`+html.EscapeString(w)+`"`, 1)
			})
			if i < len(wantLinks) {
				r.LinksRestored += len(wantLinks) - i
			}
		}
	}
	return got, r
}

func sameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
		if m[x] < 0 {
			return false
		}
	}
	return true
}

// plain turns HTML into readable text, tokens kept.
func plain(s string) string {
	s = strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ", "</p>", " ", "</div>", " ", "</li>", " ", "</h1>", " ", "</h2>", " ", "</h3>", " ").Replace(s)
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

// words counts the words of a text; a token is one word.
func words(s string) int { return len(wordRe.FindAllString(plain(s), -1)) }

// chars counts characters, not bytes.
func chars(s string) int { return utf8.RuneCountInString(s) }

// Questions about sender reputation in an instruction: the assistant
// doesn't look at those; HostRepute's assistant does. A request about the
// words ("make it sound less like spam") is a rewrite like any other.
var (
	reputationTermsRe = regexp.MustCompile(`(?i)\b(block ?list(s|ed|ing)?|black ?list(s|ed|ing)?|deliverability|reputation|inbox placement|sender score|spam ?traps?|dmarc|dkim|spf|postmaster tools|snds)\b`)
	placementRe       = regexp.MustCompile(`(?i)\b(land(s|ed|ing)?|go(es|ing)?|went|end(s|ed|ing)? up|get(s|ting)?|got|sent|delivered|filtered|marked|flagged)\s+(in|into|to|as|in to)\s+(the\s+)?(spam|junk|promotions)\b`)
	whyRe             = regexp.MustCompile(`(?i)\bwhy\b.*\b(spam|junk|bounc\w*|blocked|rejected|promotions)\b`)
)

// asksAboutReputation reports whether an instruction asks about sender
// reputation, blocklists or spam placement.
func asksAboutReputation(instruction string) bool {
	return reputationTermsRe.MatchString(instruction) || placementRe.MatchString(instruction) || whyRe.MatchString(instruction)
}

// imageOfRe catches alt text that starts with "Image of".
var imageOfRe = regexp.MustCompile(`(?i)^\s*(an? )?(image|picture|photo|graphic|screenshot) (of|showing)\b`)

// numbersIn returns the numbers a text mentions, normalized ("47548",
// "99.1%").
func numbersIn(s string) []string {
	out := []string{}
	for _, m := range numberRe.FindAllString(s, -1) {
		n := strings.ReplaceAll(m, ",", "")
		n = strings.TrimSuffix(n, ".")
		if n == "" {
			continue
		}
		out = append(out, n)
	}
	return out
}

// numberForms adds the ways a summary may write a number: the count, a
// percentage with one decimal or none.
func numberForms(set map[string]bool, v float64, percent bool) {
	if percent {
		set[strconv.FormatFloat(v, 'f', 1, 64)+"%"] = true
		set[strconv.FormatFloat(v, 'f', 0, 64)+"%"] = true
		set[strconv.FormatFloat(v, 'f', 2, 64)+"%"] = true
		return
	}
	set[strconv.FormatFloat(v, 'f', 0, 64)] = true
	set[strconv.FormatFloat(v, 'f', 1, 64)] = true
}
