package importer

import "strings"

// momentTokens maps SendGrid's formatDate tokens (moment.js style) to the
// strftime directives of Liquid's date filter, longest first.
var momentTokens = []struct{ moment, strftime string }{
	{"YYYY", "%Y"}, {"YY", "%y"},
	{"MMMM", "%B"}, {"MMM", "%b"}, {"MM", "%m"}, {"M", "%-m"},
	{"DDDD", "%j"}, {"DDD", "%-j"}, {"DD", "%d"}, {"D", "%-d"},
	{"dddd", "%A"}, {"ddd", "%a"}, {"d", "%w"},
	{"HH", "%H"}, {"H", "%-H"}, {"hh", "%I"}, {"h", "%-I"},
	{"mm", "%M"}, {"m", "%-M"},
	{"ss", "%S"}, {"s", "%-S"},
	{"SSS", "%L"},
	{"A", "%p"}, {"a", "%P"},
	{"ZZ", "%z"}, {"Z", "%:z"},
	{"X", "%s"},
}

// momentToStrftime converts a formatDate format. Tokens without a twin (Do,
// Mo, kk…) are returned so the report can name them; they print as written.
func momentToStrftime(f string) (string, []string) {
	var b strings.Builder
	unknown := []string{}
	for i := 0; i < len(f); {
		c := f[i]
		// [literal text]
		if c == '[' {
			if end := strings.IndexByte(f[i:], ']'); end > 0 {
				b.WriteString(strings.ReplaceAll(f[i+1:i+end], "%", "%%"))
				i += end + 1
				continue
			}
		}
		if c == '%' {
			b.WriteString("%%")
			i++
			continue
		}
		matched := false
		for _, t := range momentTokens {
			if strings.HasPrefix(f[i:], t.moment) {
				// "Do", "Mo": ordinals have no strftime directive.
				if next := i + len(t.moment); next < len(f) && f[next] == 'o' && (t.moment == "D" || t.moment == "M" || t.moment == "d") {
					break
				}
				b.WriteString(t.strftime)
				i += len(t.moment)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			// A run of the same letter is one token.
			j := i + 1
			for j < len(f) && (f[j] == c || f[j] == 'o') {
				j++
			}
			unknown = append(unknown, "`"+f[i:j]+"`")
			b.WriteString(f[i:j])
			i = j
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String(), unknown
}
