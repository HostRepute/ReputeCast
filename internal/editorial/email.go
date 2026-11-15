package editorial

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	htmlparser "golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/templates"
)

type Email struct {
	Subject, Preheader, MJML string
}

// EmailContext stays on the instance. Only Source may leave it. Persist the
// context with its immutable proposal rather than rebuilding capabilities
// from model output at acceptance time.
type EmailContext struct {
	Head, Header, Footer string
	Bindings             []Binding
	BindingPrefix        string
	Tokens               []string
	FinalPolicy          mjml.GeneratedPolicy
}

type Binding struct {
	Handle, Value, Kind string
}

type Source struct {
	Source, Subject, Preheader string
	Links, Images, Tokens      []string
}

var (
	ErrSource  = errors.New("unsupported editorial source")
	ErrInput   = errors.New("invalid editorial input")
	ErrPrivacy = errors.New("private editorial text")
	Reserved   = regexp.MustCompile(`(?i)RC_DRAFT_VALUE_|reputecast\.invalid`)
	Marker     = regexp.MustCompile(`<!--\s*rc:[a-zA-Z0-9_-]+\s*-->`)
)

// Project keeps brand markup, URLs and token literals local. It returns
// visible editorial text with capabilities that are opaque to the model.
func (c *EmailContext) Project(source string, links []string, before *Email, maxChars int) (Source, error) {
	out := Source{Links: []string{}, Images: []string{}, Tokens: []string{}}
	var err error
	if len(source) > 200_000 || Reserved.MatchString(source) {
		return out, ErrSource
	}
	if root, err := mjml.Parse(source); err == nil {
		if head := mjml.Head(root); head != nil {
			c.Head = source[head.Start:head.End]
			source = source[:head.Start] + source[head.End:]
		}
	}
	source, c.Footer, err = withoutSection(source, "footer")
	if err != nil {
		return out, ErrSource
	}
	if c.Footer == "" {
		starter, _ := templates.StarterMJML("blank", "")
		_, c.Footer, _ = withoutSection(starter, "footer")
	}
	source, c.Header, err = withoutSection(source, "header")
	if err != nil || strings.Contains(source, "{%") {
		return out, ErrSource
	}
	for _, address := range links {
		if !validURL(address) {
			return out, ErrInput
		}
		c.bind("link", address)
	}
	c.Tokens = Liquid.FindAllString(source, -1)
	masked := Liquid.ReplaceAllStringFunc(source, func(token string) string { return c.bind("token", token) })
	z := htmlparser.NewTokenizer(strings.NewReader(masked))
	for {
		typ := z.Next()
		if typ == htmlparser.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return out, ErrSource
			}
			break
		}
		if typ != htmlparser.StartTagToken && typ != htmlparser.SelfClosingTagToken {
			continue
		}
		for _, a := range z.Token().Attr {
			if a.Key == "href" || a.Key == "src" {
				localImage := a.Key == "src" && strings.HasPrefix(a.Val, "/media/") && validURL("https://local.invalid"+a.Val)
				if !validURL(a.Val) && !localImage && !slices.ContainsFunc(c.Bindings, func(b Binding) bool { return b.Kind == "token" && b.Handle == a.Val }) {
					return out, ErrSource
				}
				kind := "link"
				if a.Key == "src" {
					kind = "image"
				}
				c.bind(kind, a.Val)
			}
		}
	}
	if err := mjml.CheckGenerated(Marker.ReplaceAllString(source, ""), c.sourcePolicy()); err != nil {
		return out, ErrSource
	}
	text, safe := Sample(source)
	if !safe || len([]rune(text)) > maxChars {
		return out, ErrPrivacy
	}
	out.Source = Liquid.ReplaceAllStringFunc(text, func(token string) string { return c.bind("token", token) })
	if before != nil {
		out.Subject, out.Preheader = before.Subject, before.Preheader
		for _, field := range []*string{&out.Subject, &out.Preheader} {
			if strings.Contains(*field, "{%") || Reserved.MatchString(*field) {
				return out, ErrSource
			}
			if *field, safe = Text(*field); !safe {
				return out, ErrPrivacy
			}
			*field = Liquid.ReplaceAllStringFunc(*field, func(token string) string { return c.bind("token", token) })
		}
	}
	for _, b := range c.Bindings {
		switch b.Kind {
		case "link":
			out.Links = append(out.Links, b.Handle)
		case "image":
			out.Images = append(out.Images, b.Handle)
		case "token":
			if _, safe := Text(b.Value); !safe || strings.ContainsAny(b.Value, "<>") {
				return out, ErrPrivacy
			}
			// Tokens used only inside a destination are already carried by
			// its link/image binding. Advertising them as text tokens would
			// ask the model to duplicate a redemption URL in the body.
			if strings.Contains(out.Source+out.Subject+out.Preheader, b.Handle) {
				out.Tokens = append(out.Tokens, b.Handle)
			}
		}
	}
	return out, nil
}

// SavedPolicy snapshots capabilities only after restoration and validation.
func SavedPolicy(source string) mjml.GeneratedPolicy {
	p := mjml.GeneratedPolicy{}
	z := htmlparser.NewTokenizer(strings.NewReader(source))
	for z.Next() != htmlparser.ErrorToken {
		token := z.Token()
		for _, a := range token.Attr {
			switch a.Key {
			case "href":
				if token.Data == "mj-font" {
					p.Fonts = append(p.Fonts, a.Val)
				} else {
					p.Links = append(p.Links, a.Val)
				}
			case "src":
				p.Images = append(p.Images, a.Val)
			case mjml.AttrSynced:
				p.Synced = append(p.Synced, a.Val)
			case "rc-logo":
				p.Logos = append(p.Logos, a.Val)
			}
		}
	}
	return p
}

func (c *EmailContext) bind(kind, value string) string {
	for _, b := range c.Bindings {
		if b.Kind == kind && b.Value == value {
			return b.Handle
		}
	}
	key := fmt.Sprint(len(c.Bindings) + 1)
	if c.BindingPrefix != "" {
		key = c.BindingPrefix + "_" + key
	}
	handle := "RC_DRAFT_VALUE_" + key + "_END"
	if kind != "token" {
		handle = fmt.Sprintf("https://reputecast.invalid/%s/%s/end", kind, key)
	}
	c.Bindings = append(c.Bindings, Binding{Handle: handle, Value: value, Kind: kind})
	return handle
}

func (c EmailContext) sourcePolicy() mjml.GeneratedPolicy {
	p := mjml.GeneratedPolicy{}
	for _, b := range c.Bindings {
		value := b.Value
		for _, token := range c.Bindings {
			if token.Kind == "token" {
				value = strings.ReplaceAll(value, token.Handle, token.Value)
			}
		}
		switch b.Kind {
		case "link":
			p.Links = append(p.Links, value)
		case "image":
			p.Images = append(p.Images, value)
		}
	}
	return p
}

func validURL(address string) bool {
	if len(address) > 4096 || Reserved.MatchString(address) {
		return false
	}
	for _, r := range address {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.IsSpace(r) {
			return false
		}
	}
	u, err := url.Parse(address)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil && !strings.ContainsAny(address, "\r\n\t<>{}\\")
}

func withoutSection(src, kind string) (string, string, error) {
	root, err := mjml.Parse(src)
	if err != nil {
		return "", "", err
	}
	body := mjml.Body(root)
	if body == nil {
		return "", "", ErrSource
	}
	for i, n := range body.Children {
		if mjml.SectionLabel(n, i, len(body.Children)) == kind {
			start := n.Start
			if n.Marker != "" {
				// The marker belongs to the removed section. Leaving it behind
				// would relabel the next section on the next parse (and could
				// hide editorial content as a footer).
				markers := Marker.FindAllStringIndex(src[:start], -1)
				if len(markers) > 0 {
					start = markers[len(markers)-1][0]
				}
			}
			return src[:start] + src[n.End:], src[n.Start:n.End], nil
		}
	}
	return src, "", nil
}

// Restore validates opaque capabilities before substituting their local
// values. It never grants a new URL, attribute, Liquid token or executable
// block. The original compliance and brand sections are restored last.
func (c EmailContext) Restore(email *Email, before *Email) error {
	policy := mjml.GeneratedPolicy{}
	for _, b := range c.Bindings {
		if b.Kind == "link" {
			policy.Links = append(policy.Links, b.Handle)
		}
		if b.Kind == "image" {
			policy.Images = append(policy.Images, b.Handle)
		}
	}
	if err := mjml.CheckGenerated(email.MJML, policy); err != nil {
		return err
	}
	all := email.Subject + "\n" + email.Preheader + "\n" + email.MJML
	if Liquid.MatchString(all) || Liquid.MatchString(html.UnescapeString(all)) {
		return errors.New("invented Liquid")
	}
	used := map[string]bool{}
	z := htmlparser.NewTokenizer(strings.NewReader(email.MJML))
	for z.Next() != htmlparser.ErrorToken {
		for _, a := range z.Token().Attr {
			if strings.Contains(a.Val, "RC_DRAFT_VALUE_") {
				return errors.New("text token moved into an attribute")
			}
			if a.Key == "href" || a.Key == "src" {
				used[a.Val] = true
			}
		}
	}
	for _, b := range c.Bindings {
		if b.Kind != "token" && strings.Contains(email.Subject+email.Preheader, b.Handle) {
			return errors.New("link moved into heading")
		}
		if b.Kind != "token" && !used[b.Handle] {
			return errors.New("missing link or image")
		}
	}
	bindings := slices.Clone(c.Bindings)
	slices.SortStableFunc(bindings, func(a, b Binding) int {
		if a.Kind == "token" && b.Kind != "token" {
			return 1
		}
		if b.Kind == "token" && a.Kind != "token" {
			return -1
		}
		return 0
	})
	for _, b := range bindings {
		email.Subject = strings.ReplaceAll(email.Subject, b.Handle, b.Value)
		email.Preheader = strings.ReplaceAll(email.Preheader, b.Handle, b.Value)
		value := b.Value
		if b.Kind != "token" {
			value = html.EscapeString(value)
		}
		email.MJML = strings.ReplaceAll(email.MJML, b.Handle, value)
	}
	if !sameMultiset(c.Tokens, Liquid.FindAllString(email.MJML, -1)) {
		return errors.New("changed personalization tokens")
	}
	if before != nil {
		if !sameMultiset(Liquid.FindAllString(before.Subject, -1), Liquid.FindAllString(email.Subject, -1)) ||
			!sameMultiset(Liquid.FindAllString(before.Preheader, -1), Liquid.FindAllString(email.Preheader, -1)) {
			return errors.New("personalization moved between fields")
		}
	} else if Liquid.MatchString(email.Subject + email.Preheader) {
		return errors.New("invented heading personalization")
	}
	if Reserved.MatchString(email.Subject + email.Preheader + email.MJML) {
		return errors.New("unknown binding")
	}
	if err := mjml.CheckGenerated(email.MJML, c.sourcePolicy()); err != nil {
		return err
	}
	if strings.Contains(email.Subject+email.Preheader, "\n") || strings.ContainsAny(email.Subject+email.Preheader, "\r\x00<>") {
		return errors.New("invalid subject")
	}
	if text, ok := Sample(email.MJML); !ok || text == "" {
		return errors.New("unsafe text")
	}
	for _, text := range []string{email.Subject, email.Preheader} {
		if _, safe := Text(text); !safe {
			return errors.New("unsafe subject")
		}
	}
	if len(liquid.Validate(email.Subject+email.Preheader+email.MJML)) != 0 {
		return errors.New("invalid Liquid")
	}
	root, err := mjml.Parse(email.MJML)
	if err != nil {
		return err
	}
	body := mjml.Body(root)
	if body == nil || len(body.Children) == 0 {
		return errors.New("empty email")
	}
	last := body.Children[len(body.Children)-1].End
	email.MJML = email.MJML[:last] + "\n<!-- rc:footer -->\n" + c.Footer + email.MJML[last:]
	if c.Header != "" {
		first := body.Children[0].Start
		email.MJML = email.MJML[:first] + "\n<!-- rc:header-logo -->\n" + c.Header + email.MJML[first:]
	}
	if c.Head != "" {
		if head := mjml.Head(root); head != nil {
			email.MJML = email.MJML[:head.Start] + c.Head + email.MJML[head.End:]
		} else {
			at := strings.IndexByte(email.MJML, '>') + 1
			email.MJML = email.MJML[:at] + c.Head + email.MJML[at:]
		}
	}
	email.MJML = mjml.SetHeadMeta(email.MJML, email.Subject, email.Preheader)
	return nil
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
