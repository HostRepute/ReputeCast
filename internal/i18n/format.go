package i18n

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/feature/plural"
)

// Args are a message's values by name: strings, numbers, HTML (trusted
// markup for HTML output) or times (written as dates).
type Args map[string]any

// HTML is markup a caller has already escaped, kept as is in HTML output.
type HTML string

// Tags turn a message's <tags> into HTML: the function gets the tag's
// content, already escaped. <b>, <strong>, <i> and <em> work without one;
// any other tag without a function keeps only its content.
type Tags map[string]func(inner string) string

type mode int

const (
	modeText mode = iota
	modeHTML
)

type formatter struct {
	lang   Lang
	text   Locale // the language of the text, for plural rules
	args   Args
	tags   Tags
	mode   mode
	pseudo bool
}

func (f *formatter) nodes(ns []node, pound *float64) string {
	var b strings.Builder
	for _, n := range ns {
		switch v := n.(type) {
		case textNode:
			s := string(v)
			if f.pseudo {
				s = accent(s)
			}
			if f.mode == modeHTML {
				s = html.EscapeString(s)
			}
			b.WriteString(s)
		case argNode:
			b.WriteString(f.arg(v))
		case poundNode:
			if pound != nil {
				b.WriteString(f.number(*pound, ""))
			} else {
				b.WriteString("#")
			}
		case pluralNode:
			val, ok := toFloat(f.args[v.name])
			if !ok {
				b.WriteString(f.nodes(v.options["other"], nil))
				continue
			}
			adj := val - float64(v.offset)
			b.WriteString(f.nodes(f.pluralOption(v, val, adj), &adj))
		case selectNode:
			key := fmt.Sprint(f.args[v.name])
			opt, ok := v.options[key]
			if !ok {
				opt = v.options["other"]
			}
			b.WriteString(f.nodes(opt, pound))
		case tagNode:
			inner := f.nodes(v.children, pound)
			if f.mode == modeText {
				b.WriteString(inner)
				continue
			}
			if fn, ok := f.tags[v.name]; ok && fn != nil {
				b.WriteString(fn(inner))
				continue
			}
			switch v.name {
			case "b", "strong", "i", "em":
				b.WriteString("<" + v.name + ">" + inner + "</" + v.name + ">")
			default:
				b.WriteString(inner)
			}
		}
	}
	return b.String()
}

func (f *formatter) pluralOption(v pluralNode, val, adj float64) []node {
	if opt, ok := v.options["="+trimFloat(val)]; ok {
		return opt
	}
	form := pluralForm(f.text, adj, v.ordinal)
	if opt, ok := v.options[form]; ok {
		return opt
	}
	return v.options["other"]
}

func (f *formatter) arg(v argNode) string {
	val, ok := f.args[v.name]
	if !ok {
		return f.escape("{" + v.name + "}")
	}
	switch v.kind {
	case "number":
		if n, ok := toFloat(val); ok {
			return f.number(n, v.style)
		}
	case "date", "time":
		if t, ok := val.(time.Time); ok {
			if v.kind == "time" {
				return f.escape(f.lang.Region.Time(t, ""))
			}
			return f.escape(f.lang.Region.Date(t))
		}
	}
	switch x := val.(type) {
	case HTML:
		if f.mode == modeHTML {
			return string(x)
		}
		return htmlToText(string(x))
	case string:
		return f.escape(x)
	case time.Time:
		return f.escape(f.lang.Region.Date(x))
	case fmt.Stringer:
		return f.escape(x.String())
	}
	if n, ok := toFloat(val); ok {
		return f.number(n, "")
	}
	return f.escape(fmt.Sprint(val))
}

func (f *formatter) escape(s string) string {
	if f.mode == modeHTML {
		return html.EscapeString(s)
	}
	return s
}

// number writes a number by the region format. Styles: "" and "integer"
// (grouped), "percent" (0.991 → 99.1%), or a skeleton like "::percent .0".
func (f *formatter) number(v float64, style string) string {
	r := f.lang.Region
	digits := fractionDigits(v)
	switch {
	case style == "integer":
		return r.Int(int64(math.Round(v)))
	case style == "percent" || strings.Contains(style, "percent"):
		d := 0
		if i := strings.Index(style, " ."); i >= 0 {
			d = strings.Count(style[i+2:], "0") + strings.Count(style[i+2:], "#")
		}
		return r.Percent(v*100, d)
	}
	return r.Decimal(v, digits)
}

func fractionDigits(v float64) int {
	if v == math.Trunc(v) {
		return 0
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if _, frac, ok := strings.Cut(s, "."); ok {
		if len(frac) > 3 {
			return 3
		}
		return len(frac)
	}
	return 0
}

func trimFloat(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// pluralForm is the CLDR plural category of v in the text's language.
func pluralForm(l Locale, v float64, ordinal bool) string {
	rules := plural.Cardinal
	if ordinal {
		rules = plural.Ordinal
	}
	abs := math.Abs(v)
	i := int(math.Trunc(abs))
	s := strconv.FormatFloat(abs, 'f', -1, 64)
	vDigits, f, w, t := 0, 0, 0, 0
	if _, frac, ok := strings.Cut(s, "."); ok {
		vDigits = len(frac)
		f, _ = strconv.Atoi(frac)
		trimmed := strings.TrimRight(frac, "0")
		w = len(trimmed)
		t, _ = strconv.Atoi("0" + trimmed)
	}
	switch rules.MatchPlural(l.pluralTag(), i%10_000_000, vDigits, w, f, t) {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	}
	return "other"
}

func htmlToText(h string) string {
	var b strings.Builder
	in := false
	for _, c := range h {
		switch {
		case c == '<':
			in = true
		case c == '>':
			in = false
		case !in:
			b.WriteRune(c)
		}
	}
	return html.UnescapeString(b.String())
}
