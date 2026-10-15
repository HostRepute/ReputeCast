package transactional

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

var (
	anchorRe  = regexp.MustCompile(`(?is)(<a\b[^>]*?\bhref\s*=\s*)("([^"]*)"|'([^']*)')`)
	bodyEndRe = regexp.MustCompile(`(?i)</body>`)
)

// trackLinks rewrites a message's web links to the click redirect, so
// clicks are counted, and adds the open pixel. Tracking is off by default
// and per template. A link that carries a sensitive value (a reset link)
// is never rewritten: its target would have to be stored to redirect, and
// sensitive values are never stored. Unsubscribe and preference links stay
// as they are. It returns the HTML and the rewritten links by position
// (1-based: links[0] is position 1).
func trackLinks(doc string, opens, clicks bool, sensitive []string, click func(pos int) string, open string) (string, []string) {
	links := []string{}
	if clicks {
		doc = anchorRe.ReplaceAllStringFunc(doc, func(m string) string {
			sub := anchorRe.FindStringSubmatch(m)
			raw := sub[3]
			if sub[4] != "" {
				raw = sub[4]
			}
			target := strings.TrimSpace(html.UnescapeString(raw))
			u, err := url.Parse(target)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return m
			}
			if strings.Contains(u.Path, "/unsubscribe/") || strings.Contains(u.Path, "/preferences/") {
				return m
			}
			for _, v := range sensitive {
				if v != "" && strings.Contains(target, v) {
					return m
				}
			}
			links = append(links, target)
			return sub[1] + `"` + html.EscapeString(click(len(links))) + `"`
		})
	}
	if opens && open != "" {
		pixel := `<img src="` + html.EscapeString(open) + `" width="1" height="1" alt="" style="display:block;width:1px;height:1px;border:0;margin:0;padding:0" />`
		if loc := bodyEndRe.FindStringIndex(doc); loc != nil {
			doc = doc[:loc[0]] + pixel + doc[loc[0]:]
		} else {
			doc += pixel
		}
	}
	return doc, links
}

// sensitiveValues are the string values of a call's sensitive variables.
func sensitiveValues(vars []TransactionalVariable, data map[string]any) []string {
	out := []string{}
	for _, v := range vars {
		if !v.Sensitive {
			continue
		}
		if s, ok := data[v.Name].(string); ok && len(s) >= 4 {
			out = append(out, s, html.EscapeString(s))
		}
	}
	return out
}

// trackingURL moves a tracking URL to the workspace's link tracking
// domain when one is set ("go.news.acme.io").
func trackingURL(u, domain string) string {
	if domain == "" {
		return u
	}
	p, err := url.Parse(u)
	if err != nil {
		return u
	}
	p.Host = domain
	p.Scheme = "https"
	return p.String()
}

// clickToken signs a transactional click (position is 1-based).
func (s *Service) clickToken(ws, msg, contact uuid.UUID, pos int) string {
	return s.Links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: ws, MessageID: msg, ContactID: contact, Link: uint32(pos)})
}
