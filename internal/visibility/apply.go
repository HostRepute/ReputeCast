package visibility

import (
	"html"
	"sort"
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Hidden is the set of conditional blocks a recipient doesn't get.
type Hidden map[string]bool

// ParseHidden reads messages.hidden_blocks ("a1b2c3,d4e5f6").
func ParseHidden(s string) Hidden {
	out := Hidden{}
	for _, id := range strings.Split(s, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	return out
}

// String writes the set as messages.hidden_blocks stores it: sorted ids,
// comma-separated, "" for none.
func (h Hidden) String() string {
	if len(h) == 0 {
		return ""
	}
	ids := make([]string, 0, len(h))
	for id, on := range h {
		if on {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// Apply cuts the hidden blocks out of compiled HTML and removes every
// marker: the email one recipient gets. HTML without markers comes back as
// is (minus nothing).
func Apply(src string, hidden Hidden) string {
	return apply(src, hidden, nil)
}

// ApplyWhy is Apply for previews: each hidden block becomes a thin line
// saying what was left out and why ("Hidden: Beta corner · Sam Rivera isn't
// on Beta Program"); line(id) returns its text, "" to leave it out.
func ApplyWhy(src string, hidden Hidden, line func(id string) string) string {
	return apply(src, hidden, line)
}

func apply(src string, hidden Hidden, line func(id string) string) string {
	if !strings.Contains(src, "<!--rc:") {
		return src
	}
	var b strings.Builder
	b.Grow(len(src))
	i := 0
	for {
		j := strings.Index(src[i:], "<!--rc:")
		if j < 0 {
			b.WriteString(src[i:])
			break
		}
		j += i
		b.WriteString(src[i:j])
		end := strings.Index(src[j:], "-->")
		if end < 0 {
			b.WriteString(src[j:])
			break
		}
		end += j + 3
		tag := src[j:end]
		id, isIf := markerID(tag)
		if id == "" {
			// Not one of ours ("rc-raw-start" and friends stay).
			b.WriteString(tag)
			i = end
			continue
		}
		if !isIf || !hidden[id] {
			// A shown block's markers go; its content stays.
			i = end
			continue
		}
		// A hidden block: skip to its closing marker.
		closing := mjml.MarkerFi + id + "-->"
		k := strings.Index(src[end:], closing)
		if k < 0 {
			i = end
			continue
		}
		if line != nil {
			if text := line(id); text != "" {
				b.WriteString(whyLine(text))
			}
		}
		i = end + k + len(closing)
	}
	return b.String()
}

// markerID reads "<!--rc:if a1b2c3-->": the id and whether it opens.
func markerID(tag string) (string, bool) {
	switch {
	case strings.HasPrefix(tag, mjml.MarkerIf):
		return strings.TrimSuffix(strings.TrimPrefix(tag, mjml.MarkerIf), "-->"), true
	case strings.HasPrefix(tag, mjml.MarkerFi):
		return strings.TrimSuffix(strings.TrimPrefix(tag, mjml.MarkerFi), "-->"), false
	}
	return "", false
}

// whyLine is the preview's line for a hidden block, in the board's style:
// a dashed info-blue strip with an eye-off icon. It's an email table row so
// it sits where the block was in every layout.
func whyLine(text string) string {
	return `<div class="rc-why" style="margin:6px 24px;padding:6px 12px;border:1px dashed #93C5FD;border-radius:6px;background:#EFF6FF;color:#1E3A8A;` +
		`font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:12px;font-weight:500;line-height:1.5;text-align:left">` +
		`<span aria-hidden="true">⊘ </span>` + html.EscapeString(text) + `</div>`
}

// EmptyLine is the preview's box where an empty version has no content.
func EmptyLine(text string) string {
	return `<div class="rc-why-empty" style="margin:12px 24px;padding:28px 12px;border:2px dashed #FCA5A5;border-radius:6px;background:#FFF5F5;color:#991B1B;` +
		`font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:13px;font-weight:600;text-align:center">` + html.EscapeString(text) + `</div>`
}

// Size is the size of a version's HTML in bytes, without the markers.
func Size(src string, hidden Hidden) int {
	return len(Apply(src, hidden))
}
