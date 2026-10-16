package digests

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/liquid"
)

// Feed tokens (brief 3.1): {{ feed.title }}, {{ issue.number }},
// {{ issue.date }}, {{ issue.items.size }}, {{ issue.items.first.title }}
// and issue.items for loops, each item with title, url, date, author,
// image, excerpt and categories.
//
// An issue is the same for every recipient, so its tokens are filled in
// when the issue is built; personalization ({{ first_name }}, the links
// ReputeCast adds per recipient) stays for the send, which renders it per
// recipient as for any campaign. RenderIssue does both halves of that split.

// IssueData is what an issue's tokens read.
type IssueData struct {
	FeedTitle string
	FeedURL   string
	Number    int
	// Date is the send day ("Friday 16 October").
	Date  time.Time
	Items []BlockItem
	Loc   *time.Location
}

// vars are the bindings of the feed tokens. In email HTML the values are
// escaped; in the subject and preheader they are plain text.
func (d IssueData) vars(escape bool) map[string]any {
	esc := func(s string) string {
		if escape {
			return html.EscapeString(s)
		}
		return s
	}
	loc := d.Loc
	if loc == nil {
		loc = time.UTC
	}
	items := make([]map[string]any, 0, len(d.Items))
	for _, it := range d.Items {
		date := ""
		if it.Published != nil {
			date = it.Published.In(loc).Format("Monday 2 January 2006")
		}
		cats := make([]string, 0, len(it.Categories))
		for _, c := range it.Categories {
			cats = append(cats, esc(c))
		}
		items = append(items, map[string]any{
			"title": esc(it.Title), "url": it.URL, "date": date, "author": esc(it.Author), "image": it.Image,
			"excerpt": esc(it.Excerpt), "categories": cats,
		})
	}
	issueDate := ""
	if !d.Date.IsZero() {
		issueDate = d.Date.In(loc).Format("Monday 2 January")
	}
	return map[string]any{
		"feed":  map[string]any{"title": esc(d.FeedTitle), "url": d.FeedURL},
		"issue": map[string]any{"number": d.Number, "date": issueDate, "items": items},
	}
}

var (
	spanRe = regexp.MustCompile(`\{\{-?[\s\S]*?-?\}\}|\{%-?[\s\S]*?-?%\}`)
	// rawRe is a raw or comment block, kept whole.
	rawRe      = regexp.MustCompile(`(?s)\{%-?\s*(raw|comment)\s*-?%\}.*?\{%-?\s*end(raw|comment)\s*-?%\}`)
	stringLit  = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	identRoots = regexp.MustCompile(`(^|[^\w.])([A-Za-z_]\w*)`)
	tagNameRe  = regexp.MustCompile(`^\{%-?\s*(\w+)`)
	forVarRe   = regexp.MustCompile(`^\{%-?\s*for\s+(\w+)\s+in\s`)
	assignRe   = regexp.MustCompile(`^\{%-?\s*(assign|capture)\s+(\w+)`)
)

// blockOpeners are the Liquid tags with an end tag.
var blockOpeners = map[string]string{"if": "endif", "unless": "endunless", "for": "endfor", "case": "endcase", "capture": "endcapture", "tablerow": "endtablerow"}

// middles belong to the block they are in.
var middles = map[string]bool{"else": true, "elsif": true, "when": true}

// roots lists the variable names an expression reads (string literals and
// filter arguments' strings left out).
func roots(expr string) []string {
	expr = stringLit.ReplaceAllString(expr, `""`)
	out := []string{}
	for _, m := range identRoots.FindAllStringSubmatch(expr, -1) {
		out = append(out, m[2])
	}
	return out
}

// keywords of Liquid tags and operators, which aren't variables.
var keywords = map[string]bool{"if": true, "elsif": true, "else": true, "endif": true, "unless": true, "endunless": true, "for": true, "endfor": true,
	"in": true, "case": true, "when": true, "endcase": true, "and": true, "or": true, "not": true, "contains": true, "assign": true, "capture": true,
	"endcapture": true, "limit": true, "offset": true, "reversed": true, "true": true, "false": true, "nil": true, "null": true, "empty": true,
	"blank": true, "tablerow": true, "endtablerow": true, "cols": true, "cycle": true, "break": true, "continue": true, "increment": true, "decrement": true}

// RenderIssue fills in an issue's tokens and leaves every other Liquid
// span as it is, for the send's per-recipient render: a span reads the
// issue when it uses feed, issue, a loop variable of a loop over the
// issue's items, or a variable assigned from them. Block tags go with
// their opening tag, so {% if first_name %}…{{ issue.number }}…{% endif %}
// keeps its condition and gets the number. A condition that mixes both
// (issue.items.size > 1 and first_name) is decided now, without the
// contact. escape writes the values for HTML.
func RenderIssue(src string, d IssueData, escape bool) (string, error) {
	if !strings.Contains(src, "{{") && !strings.Contains(src, "{%") {
		return src, nil
	}
	src = liquid.Prepare(src)
	vars := d.vars(escape)
	kept := map[string]any{}
	keep := func(s string) string {
		k := fmt.Sprintf("__rc_keep_%d", len(kept))
		kept[k] = s
		return "{{ " + k + " }}"
	}
	// Raw and comment blocks stay whole.
	src = rawRe.ReplaceAllStringFunc(src, keep)
	issueNames := map[string]bool{"feed": true, "issue": true}
	type open struct{ issue bool }
	var stack []open
	var b strings.Builder
	last := 0
	for _, loc := range spanRe.FindAllStringIndex(src, -1) {
		span := src[loc[0]:loc[1]]
		b.WriteString(src[last:loc[0]])
		last = loc[1]
		if strings.HasPrefix(span, "{{ __rc_keep_") {
			b.WriteString(span)
			continue
		}
		isTag := strings.HasPrefix(span, "{%")
		name := ""
		if isTag {
			if m := tagNameRe.FindStringSubmatch(span); m != nil {
				name = m[1]
			}
		}
		reads := false
		for _, r := range roots(strings.Trim(span, "{}%- \t\n")) {
			if !keywords[r] && issueNames[r] {
				reads = true
			}
		}
		switch {
		case isTag && strings.HasPrefix(name, "end"):
			issue := false
			if n := len(stack); n > 0 {
				issue = stack[n-1].issue
				stack = stack[:n-1]
			}
			if issue {
				b.WriteString(span)
			} else {
				b.WriteString(keep(span))
			}
		case isTag && middles[name]:
			issue := len(stack) > 0 && stack[len(stack)-1].issue
			if issue {
				b.WriteString(span)
			} else {
				b.WriteString(keep(span))
			}
		case isTag && blockOpeners[name] != "":
			// A loop over the issue's items makes its variable an issue name
			// (and forloop with it).
			if m := forVarRe.FindStringSubmatch(span); m != nil && reads {
				issueNames[m[1]] = true
				issueNames["forloop"] = true
			}
			if m := assignRe.FindStringSubmatch(span); m != nil && m[1] == "capture" {
				// A capture is an issue value when anything inside it reads the
				// issue; it is decided by its opening tag here, conservatively.
				if reads {
					issueNames[m[2]] = true
				}
			}
			stack = append(stack, open{issue: reads})
			if reads {
				b.WriteString(span)
			} else {
				b.WriteString(keep(span))
			}
		case isTag && name == "assign":
			if m := assignRe.FindStringSubmatch(span); m != nil && reads {
				issueNames[m[2]] = true
			}
			if reads {
				b.WriteString(span)
			} else {
				b.WriteString(keep(span))
			}
		default:
			if reads {
				b.WriteString(span)
			} else {
				b.WriteString(keep(span))
			}
		}
	}
	b.WriteString(src[last:])
	for k, v := range kept {
		vars[k] = v
	}
	return liquid.Render(b.String(), vars)
}

// UsesFeedTokens reports whether a subject, preheader or body reads the issue.
func UsesFeedTokens(src string) bool {
	for _, span := range spanRe.FindAllString(liquid.Prepare(src), -1) {
		for _, r := range roots(strings.Trim(span, "{}%- \t\n")) {
			if r == "issue" || r == "feed" {
				return true
			}
		}
	}
	return false
}

// zeroMore catches a subject that reads "…and 0 more" with one item.
var zeroMore = regexp.MustCompile(`(?i)\b(and|plus|\+)\s+0\s+(more|others?|posts?|items?)\b|\b0\s+more\b`)

// SaysZeroMore reports whether a rendered subject counts "0 more".
func SaysZeroMore(subject string) bool { return zeroMore.MatchString(subject) }

// countRe is " and {{ issue.items.size | minus: 1 }} more" in a subject.
var countRe = regexp.MustCompile(`\s+(and|plus)\s+\{\{-?\s*issue\.items\.size\s*\|\s*minus:\s*1\s*-?\}\}\s+(more|others?|posts?)`)

// WrapCount puts the "and N more" of a subject inside
// {% if issue.items.size > 1 %}…{% endif %}, so it only shows with two or
// more items. ok is false when the subject has no such count.
func WrapCount(subject string) (string, bool) {
	loc := countRe.FindStringIndex(subject)
	if loc == nil || strings.Contains(subject, "issue.items.size > 1") {
		return subject, false
	}
	return subject[:loc[0]] + "{% if issue.items.size > 1 %}" + subject[loc[0]:loc[1]] + "{% endif %}" + subject[loc[1]:], true
}
