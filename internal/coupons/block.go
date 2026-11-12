package coupons

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/liquid"
)

// The Discount code block is a marker,
//
//	<mj-raw><!-- rc:coupon {"pool":"welcome15","label":"Your code","button":"Shop now","buttonUrl":true} --></mj-raw>
//
// that liquid.ExpandBlocks turns into email-safe table HTML: the label,
// the reader's code in a dashed box (IBM Plex Mono, letter-spacing
// 0.08em, selectable) and an optional button to the address that applies
// it. A reader without a code (the public archive, an unknown pool) gets
// nothing: the block hides itself.

func init() {
	liquid.RegisterBlock("coupon", func(raw []byte) string { return ExpandBlock(parseBlock(raw)) })
}

// CouponBlock is the Discount code block's settings.
type CouponBlock struct {
	Pool      string `json:"pool"`
	Label     string `json:"label"`
	Button    string `json:"button"`
	ButtonURL bool   `json:"buttonUrl"`
}

func parseBlock(raw []byte) CouponBlock {
	b := CouponBlock{Label: "Your code", Button: "Shop now", ButtonURL: true}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &b)
	}
	b.Pool = strings.TrimSpace(b.Pool)
	b.Label = clipRunes(strings.TrimSpace(b.Label), 80)
	b.Button = clipRunes(strings.TrimSpace(b.Button), 40)
	return b
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

var liquidSpan = regexp.MustCompile(`\{\{[\s\S]*?\}\}|\{%[\s\S]*?%\}`)

// escText escapes text for HTML and keeps its Liquid as it is (a label may
// greet the reader: "Your code, {{ first_name }}").
func escText(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range liquidSpan.FindAllStringIndex(s, -1) {
		b.WriteString(html.EscapeString(s[last:m[0]]))
		b.WriteString(s[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(html.EscapeString(s[last:]))
	return b.String()
}

const (
	blockFont = `font-family:'IBM Plex Sans',Arial,sans-serif;`
	blockMono = `font-family:'IBM Plex Mono',Menlo,Consolas,monospace;`
	accent    = "#0F766E"
)

// ExpandBlock is the Discount code block as Liquid: the label, the code in
// a dashed box and the button, centred (ComposerTimedBlocks).
func ExpandBlock(b CouponBlock) string {
	if !slugRe.MatchString(b.Pool) {
		return ""
	}
	v := Root + "." + b.Pool
	var out strings.Builder
	out.WriteString(`{% if ` + v + `.code %}<div class="rc-coupon" style="padding:8px 24px 24px;text-align:center">`)
	if b.Label != "" {
		out.WriteString(`<div style="` + blockFont + `font-size:13px;line-height:1.4;font-weight:600;color:#4B5563;margin:0 0 8px">` + escText(b.Label) + `</div>`)
	}
	out.WriteString(`<div style="display:inline-block;padding:12px 18px;border:2px dashed #6B7280;border-radius:8px;background:#F9FAFB;` + blockMono +
		`font-size:20px;line-height:1.3;font-weight:600;letter-spacing:0.08em;color:#0B1220;-webkit-user-select:all;user-select:all;max-width:100%;box-sizing:border-box;word-break:break-all">{{ ` + v + `.code }}</div>`)
	if b.ButtonURL && b.Button != "" {
		out.WriteString(`{% if ` + v + `.url %}<div style="margin:14px 0 0"><a href="{{ ` + v + `.url }}" style="display:inline-block;padding:9px 14px;border-radius:6px;background:` +
			accent + `;color:#FFFFFF;` + blockFont + `font-size:13px;line-height:1.2;font-weight:600;text-decoration:none">` + escText(b.Button) + `</a></div>{% endif %}`)
	}
	out.WriteString(`</div>{% endif %}`)
	return out.String()
}
