package mjml

import (
	"errors"
	"io"
	"net/url"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// GeneratedPolicy is the content a generator may preserve, not permission to
// fetch it. Links and images must come from the person's explicit context.
// The caller validates Liquid syntax and the allowed personalization tokens.
type GeneratedPolicy struct {
	Links  []string
	Images []string
	// Fonts, Synced and Logos are populated only for retained brand content,
	// never for untrusted model output.
	Fonts  []string
	Synced []string
	Logos  []string
}

// CheckGenerated checks untrusted MJML before compilation. Compiler success
// alone is not a security check: MJML permits raw HTML, CSS and includes. This
// path supports the composer's ordinary layout, text, image and button blocks;
// dynamic blocks and arbitrary HTML stay in the human editor.
func CheckGenerated(src string, policy GeneratedPolicy) error {
	if len(src) == 0 || len(src) > 200_000 {
		return errors.New("generated email must contain at most 200000 bytes of MJML")
	}
	// The tolerant outline parser skips comments and processing instructions.
	// Reject them before parsing so conditional HTML cannot bypass inspection.
	if strings.Contains(src, "<!") || strings.Contains(src, "<?") {
		return errors.New("generated email contains unsupported embedded markup")
	}
	root, err := Parse(src)
	if err != nil {
		return errors.New("generated email has invalid MJML")
	}
	if strings.TrimSpace(src[:root.Start]) != "" || strings.TrimSpace(src[root.End:]) != "" {
		return errors.New("generated email must contain exactly one MJML document")
	}
	var invalid error
	root.Walk(func(n *Node, _ []int) {
		if invalid != nil {
			return
		}
		switch n.Tag {
		case "mj-font":
			if !slices.Contains(policy.Fonts, n.Attr("href")) {
				invalid = errors.New("generated email contains an unselected font")
				return
			}
		case "mjml", "mj-head", "mj-attributes", "mj-all", "mj-class", "mj-title", "mj-preview", "mj-body",
			"mj-wrapper", "mj-section", "mj-group", "mj-column", "mj-text", "mj-button", "mj-image", "mj-divider", "mj-spacer", "mj-table":
		default:
			invalid = errors.New("generated email contains an unsupported MJML block")
			return
		}
		for _, a := range n.Attrs {
			if n.Tag == "mj-font" && a.Name == "href" {
				invalid = generatedURL(a.Value, policy.Fonts)
				if invalid != nil {
					return
				}
				continue
			}
			if err := generatedAttribute(a.Name, a.Value, policy); err != nil {
				invalid = err
				return
			}
		}
		if n.Inner != "" {
			invalid = generatedHTML(n.Inner, policy)
		}
	})
	return invalid
}

func generatedAttribute(name, value string, policy GeneratedPolicy) error {
	name = strings.ToLower(name)
	// Values are decoded by both parsers. Refuse control characters and CSS
	// escapes rather than trying to interpret a second embedded language.
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return errors.New("generated email contains a control character in an attribute")
		}
	}
	switch name {
	case "rc-logo":
		if !slices.Contains(policy.Logos, value) {
			return errors.New("generated email cannot create a brand logo reference")
		}
		return nil
	case AttrSynced:
		if !slices.Contains(policy.Synced, value) {
			return errors.New("generated email cannot create a synced block reference")
		}
		return nil
	case "href":
		return generatedURL(value, policy.Links)
	case "src":
		return generatedURL(value, policy.Images)
	case "style":
		return generatedStyle(value)
	case "width", "height", "align", "vertical-align", "background-color", "color", "container-background-color",
		"font-family", "font-size", "font-weight", "font-style", "line-height", "letter-spacing", "text-decoration", "text-transform",
		"padding", "padding-top", "padding-right", "padding-bottom", "padding-left", "inner-padding", "border", "border-top", "border-right", "border-bottom", "border-left", "border-radius",
		"css-class", "mj-class", "name", "alt", "title", "role", "lang", "dir", "target", "rel", "colspan", "rowspan", "cellpadding", "cellspacing":
		if strings.ContainsAny(value, `<>"\{};`) || strings.Contains(strings.ToLower(value), "url(") || strings.Contains(strings.ToLower(value), "expression(") {
			return errors.New("generated email contains an unsafe attribute value")
		}
		return nil
	default:
		return errors.New("generated email contains an unsupported attribute")
	}
}

func generatedURL(value string, allowed []string) error {
	if !slices.Contains(allowed, value) {
		return errors.New("generated email contains a link or image outside the selected context")
	}
	// Exact Liquid links are validated and restored by the caller. A URL
	// with embedded Liquid still needs an ordinary HTTP(S) destination.
	if strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}") && strings.Count(value, "{{") == 1 && strings.Count(value, "}}") == 1 {
		return nil
	}
	// Local media is a source binding, never a new model destination. The
	// path is already in the allowlist and the server does not fetch it.
	if strings.HasPrefix(value, "/media/") && !strings.ContainsAny(value, `\<>`) {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return errors.New("generated email contains an unsafe link or image")
	}
	return nil
}

func generatedStyle(value string) error {
	for _, declaration := range strings.Split(value, ";") {
		if strings.TrimSpace(declaration) == "" {
			continue
		}
		name, v, ok := strings.Cut(declaration, ":")
		if !ok || strings.ContainsAny(v, `\<>{}@()`) || strings.Contains(v, "/*") {
			return errors.New("generated email contains unsafe CSS")
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "color", "background-color", "font-family", "font-size", "font-weight", "font-style", "line-height", "letter-spacing", "text-decoration", "text-align",
			"margin", "margin-top", "margin-right", "margin-bottom", "margin-left", "padding", "padding-top", "padding-right", "padding-bottom", "padding-left",
			"border", "border-top", "border-right", "border-bottom", "border-left", "border-radius", "width", "height":
		default:
			return errors.New("generated email contains unsupported CSS")
		}
	}
	return nil
}

func generatedHTML(src string, policy GeneratedPolicy) error {
	z := html.NewTokenizer(strings.NewReader(src))
	for {
		t := z.Next()
		switch t {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				return nil
			}
			return errors.New("generated email has invalid HTML")
		case html.CommentToken, html.DoctypeToken:
			return errors.New("generated email contains unsupported embedded markup")
		case html.StartTagToken, html.SelfClosingTagToken, html.EndTagToken:
			token := z.Token()
			switch token.Data {
			case "a", "b", "strong", "i", "em", "u", "s", "br", "span", "p", "small", "sup", "sub", "div",
				"h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "blockquote", "table", "thead", "tbody", "tfoot", "tr", "td", "th":
			default:
				return errors.New("generated email contains unsupported HTML")
			}
			for _, a := range token.Attr {
				if err := generatedAttribute(a.Key, a.Val, policy); err != nil {
					return err
				}
			}
		}
	}
}
