package templates

import (
	"bytes"
	"context"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// The gallery shows each template as its email reads (review 2026-10,
// R4.U4): compiled on the server, a starter in the workspace's brand kit,
// personalized for a reader without a first name so the fallbacks show,
// and with nothing in it that runs, links or takes focus. The web app
// draws it scaled down in a sandboxed frame.

// EmailTemplatePreview is a template as its email reads, for the gallery.
type EmailTemplatePreview struct {
	HTML string `json:"html" doc:"The compiled email with its fallbacks shown and its links, scripts and form controls taken out: a picture of the email, not a page to use"`
}

// Preview compiles a template for the gallery's picture of it.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, id, address string) (EmailTemplatePreview, error) {
	t, err := s.Get(ctx, ws, id, address)
	if err != nil {
		return EmailTemplatePreview{}, err
	}
	out := t.HTML
	if strings.TrimSpace(t.MJML) != "" && (out == "" || t.Builtin) {
		res := mjml.Compile(t.MJML)
		if len(res.Errors) > 0 {
			l := i18n.LangFrom(ctx)
			return EmailTemplatePreview{}, api.Invalid("path.templateId", mPreviewCompile.Text(l, i18n.Args{"error": res.Errors[0].Message}))
		}
		out = res.HTML
	}
	return EmailTemplatePreview{HTML: Picture(out)}, nil
}

// pictureLinks are the per-reader links a picture leaves pointing nowhere.
var pictureLinks = map[string]string{"unsubscribe_url": "#", "preferences_url": "#", "view_in_browser_url": "#"}

// Picture turns an email's HTML into a picture of it: personalization
// shows its fallbacks (a reader without a first name), and links, scripts,
// frames, form controls, event handlers and focus order are taken out.
func Picture(src string) string {
	if rendered, err := liquid.Render(src, liquid.Vars("", "", "", map[string]any{}, pictureLinks)); err == nil {
		src = rendered
	}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return ""
	}
	disarm(doc)
	var b bytes.Buffer
	if err := html.Render(&b, doc); err != nil {
		return ""
	}
	return b.String()
}

// dropped are the elements a picture leaves out, with what they hold.
var dropped = map[atom.Atom]bool{
	atom.Script: true, atom.Noscript: true, atom.Iframe: true, atom.Frame: true, atom.Object: true, atom.Embed: true,
	atom.Form: true, atom.Input: true, atom.Button: true, atom.Select: true, atom.Textarea: true, atom.Audio: true, atom.Video: true,
	// Remote stylesheets (web fonts) and a base URL would reach outside the picture.
	atom.Base: true, atom.Link: true,
}

// disarm takes out what could run, link or take focus.
func disarm(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.ElementNode && dropped[c.DataAtom] {
			n.RemoveChild(c)
		} else {
			disarm(c)
		}
		c = next
	}
	if n.Type != html.ElementNode {
		return
	}
	kept := n.Attr[:0]
	for _, a := range n.Attr {
		k := strings.ToLower(a.Key)
		switch {
		case strings.HasPrefix(k, "on"), k == "tabindex", k == "contenteditable", k == "autofocus", k == "accesskey":
			continue
		case k == "href" && (n.DataAtom == atom.A || n.DataAtom == atom.Area):
			continue
		case n.DataAtom == atom.Meta && k == "http-equiv":
			continue
		}
		kept = append(kept, a)
	}
	n.Attr = kept
}
