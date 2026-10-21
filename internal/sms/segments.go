package sms

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Encoding is how a text travels: the GSM 03.38 7-bit alphabet, or UCS-2 as
// soon as one character is outside it.
type Encoding string

// Encodings.
const (
	GSM7 Encoding = "gsm7"
	UCS2 Encoding = "ucs2"
)

// Segment capacities. A text that fits one segment gets the whole of it;
// longer texts are split into parts that each carry a header, so they hold
// less.
const (
	GSM7Single = 160
	GSM7Part   = 153
	UCS2Single = 70
	UCS2Part   = 67
)

// gsm7Basic is the GSM 03.38 default alphabet: one septet each.
const gsm7Basic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
	"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"

// gsm7Extension is the extension table: an escape plus the character, two
// septets each, never split across segments.
const gsm7Extension = "\f^{}\\[~]|€"

var (
	basicSet = runeSet(gsm7Basic)
	extSet   = runeSet(gsm7Extension)
)

func runeSet(s string) map[rune]bool {
	m := make(map[rune]bool, utf8.RuneCountInString(s))
	for _, r := range s {
		m[r] = true
	}
	return m
}

// InGSM7 reports whether r can be sent in GSM-7 (basic or extension table).
func InGSM7(r rune) bool { return basicSet[r] || extSet[r] }

// septets is what r costs in GSM-7.
func septets(r rune) int {
	if extSet[r] {
		return 2
	}
	return 1
}

// Outside is a character that forces UCS-2.
type Outside struct {
	// Char is the character itself ("’").
	Char string `json:"char"`
	// Kind names it for the app ("curly_apostrophe", "emoji", "other").
	Kind string `json:"kind" enum:"curly_apostrophe,curly_quote,en_dash,em_dash,minus,ellipsis,non_breaking_space,thin_space,zero_width_space,prime,backtick,emoji,other"`
	// Replacement is the GSM-7 lookalike the fix puts in its place; empty
	// when there's none (an emoji). A zero-width space is removed.
	Replacement string `json:"replacement"`
	// Removable is true when the fix removes it (zero-width spaces).
	Removable bool `json:"removable"`
	// Count is how many times it appears.
	Count int `json:"count"`
	// First is the character offset (in code points) of its first use.
	First int `json:"first"`
}

// lookalikes maps characters outside GSM-7 to what the fix replaces them
// with, and their kind.
var lookalikes = map[rune]struct{ kind, with string }{
	0x2018: {"curly_apostrophe", "'"}, // left single quotation mark
	0x2019: {"curly_apostrophe", "'"}, // right single quotation mark
	0x201B: {"curly_apostrophe", "'"},
	0x201A: {"curly_apostrophe", "'"},
	0x201C: {"curly_quote", `"`},
	0x201D: {"curly_quote", `"`},
	0x201E: {"curly_quote", `"`},
	0x201F: {"curly_quote", `"`},
	0x2013: {"en_dash", "-"},
	0x2014: {"em_dash", "-"},
	0x2015: {"em_dash", "-"},
	0x2212: {"minus", "-"},
	0x2026: {"ellipsis", "..."},
	0x00A0: {"non_breaking_space", " "},
	0x2009: {"thin_space", " "},
	0x202F: {"thin_space", " "},
	0x2007: {"thin_space", " "},
	0x200B: {"zero_width_space", ""},
	0xFEFF: {"zero_width_space", ""},
	0x2032: {"prime", "'"},
	0x2033: {"prime", `"`},
	'`':    {"backtick", "'"},
}

// isEmoji is a rough test for pictographs: anything outside the Basic
// Multilingual Plane, plus the common symbol blocks people paste.
func isEmoji(r rune) bool {
	return r > 0xFFFF || (r >= 0x2600 && r <= 0x27BF) || (r >= 0x2B00 && r <= 0x2BFF) || r == 0xFE0F || r == 0x200D
}

// Count is how a text is encoded and split into segments.
type Count struct {
	// Characters counts what a person sees (code points).
	Characters int `json:"characters"`
	// Units are septets in GSM-7 or UTF-16 code units in UCS-2.
	Units    int      `json:"units"`
	Encoding Encoding `json:"encoding" enum:"gsm7,ucs2"`
	Segments int      `json:"segments"`
	// Capacity is what each segment holds: 160 or 153 for GSM-7, 70 or 67
	// for UCS-2.
	Capacity int `json:"capacity"`
	// Fill is the units in each segment, for the counter's bar.
	Fill []int `json:"fill"`
	// Left is what's left in the last segment.
	Left int `json:"left"`
	// Outside lists the characters that force UCS-2, in order of first use.
	Outside []Outside `json:"outside"`
}

// CountText measures a text exactly as it will be sent.
func CountText(s string) Count {
	c := Count{Encoding: GSM7, Fill: []int{}, Outside: []Outside{}}
	idx := map[rune]int{}
	pos := 0
	for _, r := range s {
		c.Characters++
		if !InGSM7(r) {
			if i, ok := idx[r]; ok {
				c.Outside[i].Count++
			} else {
				o := Outside{Char: string(r), Kind: "other", Count: 1, First: pos}
				if l, ok := lookalikes[r]; ok {
					o.Kind, o.Replacement = l.kind, l.with
					o.Removable = l.with == ""
				} else if isEmoji(r) {
					o.Kind = "emoji"
				}
				idx[r] = len(c.Outside)
				c.Outside = append(c.Outside, o)
			}
		}
		pos++
	}
	if len(c.Outside) > 0 {
		c.Encoding = UCS2
	}
	cost := func(r rune) int {
		if c.Encoding == UCS2 {
			if r > 0xFFFF {
				return 2 // a surrogate pair
			}
			return 1
		}
		return septets(r)
	}
	for _, r := range s {
		c.Units += cost(r)
	}
	single, part := GSM7Single, GSM7Part
	if c.Encoding == UCS2 {
		single, part = UCS2Single, UCS2Part
	}
	switch {
	case c.Units == 0:
		c.Capacity = single
		c.Left = single
		return c
	case c.Units <= single:
		c.Segments, c.Capacity = 1, single
		c.Fill = []int{c.Units}
		c.Left = single - c.Units
		return c
	}
	// Parts: an escape sequence or a surrogate pair never straddles two.
	c.Capacity = part
	cur := 0
	for _, r := range s {
		u := cost(r)
		if cur+u > part {
			c.Fill = append(c.Fill, cur)
			cur = 0
		}
		cur += u
	}
	c.Fill = append(c.Fill, cur)
	c.Segments = len(c.Fill)
	c.Left = part - cur
	return c
}

// Segments is the number of segments a text takes.
func Segments(s string) int { return CountText(s).Segments }

// FixEncoding replaces every character outside GSM-7 that has a lookalike
// (curly quotes, long dashes, ellipses, special spaces). Characters without
// one, such as emoji, are kept: removing them is the writer's call.
func FixEncoding(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if l, ok := lookalikes[r]; ok && !InGSM7(r) {
			b.WriteString(l.with)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// UTF16Len is the length of s in UTF-16 code units.
func UTF16Len(s string) int { return len(utf16.Encode([]rune(s))) }
