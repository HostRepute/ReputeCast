// Package editorial projects selected editorial content and validates proposed
// replacements. It has no model client, transport, database or actor identity.
package editorial

import (
	"errors"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	htmlparser "golang.org/x/net/html"
	"golang.org/x/text/unicode/norm"

	"github.com/hostrepute/reputecast/internal/languages"
)

// SharingPolicy identifies the reviewed-editorial-content policy in audit
// receipts. It is not a claim that free text or aggregate data is anonymous.
const SharingPolicy = "reviewed-editorial-v1"

// These detectors reject recognizable private records. They do not establish
// anonymity of arbitrary prose; people must review selected editorial sources
// and acknowledge residual risk before sharing. Confirmation never bypasses
// these checks or authorizes contact-record tools.
var URL = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.-]*://|www\.|mailto:|tel:|data:)[^\s<>"']+`)
var sensitive = regexp.MustCompile(`(?i)(?:[\pL\pN.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\pL\pN.-]+\.[\pL]{2,}|\b(?:sk-|AKIA|ASIA|ghp_|github_pat_)[a-z0-9_-]{12,}|-----BEGIN [^-]*PRIVATE KEY-----|\b(?:api[_ -]?key|password|secret|bearer|access[_ -]?token)\s*[:= ]\s*\S{6,}|\+?\d[\d ().-]{7,}\d)`)
var opaque = regexp.MustCompile(`(?:[\pL\pN_+/=-]{48,}|%[0-9a-fA-F]{2}|\\[ux][0-9a-fA-F]{2}|\b[\da-fA-F]{8}-[\da-fA-F-]{27,}\b)`)
var record = regexp.MustCompile(`(?im)(?:\b(?:name|email|phone|address|customer|recipient|contact|subscriber|password|secret|token)\s*[=:]|\b\d{1,6}\s+[\pL .'-]+\s+(?:street|road|avenue|lane|drive|st|rd|ave)\b|\b(?:dear|hi|hello|chào|cher|chère|hallo|hola)\s+(?-i:[\p{Lu}])[\pL'-]+|\b[\pL\pN._-]+\s*\[at\]\s*[\pL\pN.-]+)`)
var Liquid = regexp.MustCompile(`(?s)\{\{.*?\}\}|\{%.*?%\}`)

// Text withholds recognized private records and replaces visible URLs.
// Detection normalizes encoding, but returned Liquid remains verbatim.
func Text(text string) (string, bool) {
	check := text
	for range 3 {
		check = html.UnescapeString(check)
	}
	check = norm.NFKC.String(check)
	if html.UnescapeString(check) != check || len(URL.FindAllString(check, -1)) != len(URL.FindAllString(text, -1)) {
		return "", false
	}
	for _, r := range check {
		if unicode.Is(unicode.Cf, r) || (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') {
			return "", false
		}
	}
	if sensitive.MatchString(check) || opaque.MatchString(check) || record.MatchString(check) {
		return "", false
	}
	for _, token := range Liquid.FindAllString(text, -1) {
		if URL.MatchString(token) {
			return "", false
		}
	}
	return URL.ReplaceAllString(text, "[link omitted]"), true
}

// Sample reads visible text from supported MJML blocks, not attributes,
// comments, images or compliance footers. It preserves unexpanded tokens.
func Sample(src string) (string, bool) {
	var b strings.Builder
	for _, block := range languages.Blocks(src, "") {
		if block.Label == "footer" {
			continue
		}
		for _, segment := range block.Segments {
			switch segment.Kind {
			case "text", "button", "nav", "table":
			default:
				continue
			}
			prefix := "RC" + uuid.NewString() + "TOKEN"
			var restore []string
			masked := Liquid.ReplaceAllStringFunc(segment.HTML, func(token string) string {
				key := prefix + strconv.Itoa(len(restore)/2) + "END"
				restore = append(restore, key, token)
				return key
			})
			z := htmlparser.NewTokenizer(strings.NewReader(masked))
			var part strings.Builder
			for {
				typ := z.Next()
				if typ == htmlparser.ErrorToken {
					if !errors.Is(z.Err(), io.EOF) {
						return "", false
					}
					break
				}
				if typ == htmlparser.StartTagToken || typ == htmlparser.SelfClosingTagToken {
					name, _ := z.TagName()
					if string(name) == "script" || string(name) == "style" || string(name) == "iframe" || string(name) == "object" {
						return "", false
					}
				}
				if typ == htmlparser.TextToken {
					part.Write(z.Text())
				} else {
					part.WriteByte(' ')
				}
			}
			text, ok := Text(strings.NewReplacer(restore...).Replace(part.String()))
			if !ok {
				return "", false
			}
			b.WriteString(text)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String()), true
}
