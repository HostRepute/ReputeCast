package api

import (
	"embed"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Server messages follow the reader's app language (connect-and-reach
// brief §3.7): the middleware stores the request's language in the context
// (i18n.LangFrom), and LocalizeProblem, a Huma transformer, writes the
// access problems (missing permissions and who can help, sign-in) in it.
// API clients get English unless they send Accept-Language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "api")

var (
	mDenied     = msgs.M("denied")
	mDeniedAsk  = msgs.M("deniedAsk")
	mEscalation = msgs.M("escalation")
	mKeyExpired = msgs.M("keyExpired")
	mActionThis = msgs.M("action.this")
)

// fixed maps the English of the problems the middleware writes to their messages.
var fixed = map[string]i18n.Msg{}

func init() {
	for _, k := range msgs.Keys() {
		if strings.HasPrefix(k, "fixed.") {
			m := msgs.M(k)
			fixed[m.Source()] = m
		}
	}
}

// RequestLang is how a request reads: the Accept-Language header when it
// names a language ReputeCast has, else the signed-in member's own choice,
// else English with the ReputeCast standard formats.
func RequestLang(acceptLanguage string, u *store.User) i18n.Lang {
	var lang, region string
	if u != nil {
		if u.Language != nil {
			lang = *u.Language
		}
		if u.RegionFormat != nil {
			region = *u.RegionFormat
		}
	}
	if l, ok := i18n.Negotiate(acceptLanguage); ok {
		if mine, ok := i18n.ParseLocale(lang); ok && mine == l {
			return i18n.Resolve(lang, region, "", "")
		}
		return i18n.For(l)
	}
	if lang != "" {
		return i18n.Resolve(lang, region, "", "")
	}
	return i18n.Default
}

func langOf(hctx huma.Context) i18n.Lang {
	if l, ok := i18n.LangFromContext(hctx.Context()); ok {
		return l
	}
	return RequestLang(hctx.Header("Accept-Language"), nil)
}

// DeniedText explains a missing permission in a language: "Sending needs
// the campaigns:send permission. Ask Marcus Chen or Linh Nguyen."
func DeniedText(l i18n.Lang, perm authz.Permission, helpers []string) string {
	action := mActionThis
	if m, err := msgs.Lookup("action." + string(perm)); err == nil {
		action = m
	}
	args := i18n.Args{"action": action.Text(l, nil), "permission": string(perm)}
	if len(helpers) > 0 {
		args["helpers"] = i18n.OrList(l.Locale, helpers)
		return mDeniedAsk.Text(l, args)
	}
	return mDenied.Text(l, args)
}

var (
	escalationRe = regexp.MustCompile(`^You can't grant (.+) because you don't hold it\.$`)
	keyExpiredRe = regexp.MustCompile(`^This API key expired on (.+)\. Rotate it in Settings › Keys & MCP\.$`)
)

// localizedDetail writes an access problem's detail in another language.
func localizedDetail(p *Problem, l i18n.Lang) (string, bool) {
	if p.Code == CodePermissionDenied && p.Permission != "" && p.Detail == authz.DeniedMessage(authz.Permission(p.Permission), p.Helpers) {
		return DeniedText(l, authz.Permission(p.Permission), p.Helpers), true
	}
	if p.Code == CodeEscalation {
		if m := escalationRe.FindStringSubmatch(p.Detail); m != nil {
			return mEscalation.Text(l, i18n.Args{"permissions": m[1]}), true
		}
	}
	if m, ok := fixed[p.Detail]; ok {
		return m.Text(l, nil), true
	}
	if m := keyExpiredRe.FindStringSubmatch(p.Detail); m != nil {
		if t, err := time.Parse("Jan 2, 2006", m[1]); err == nil {
			return mKeyExpired.Text(l, i18n.Args{"date": l.Region.Date(t)}), true
		}
	}
	return "", false
}

// LocalizeProblem is a Huma transformer that writes access problems in the
// request's language. English, and problems it doesn't know yet, pass
// through unchanged.
func LocalizeProblem(hctx huma.Context, _ string, v any) (any, error) {
	p, ok := v.(*Problem)
	if !ok || p == nil {
		return v, nil
	}
	l := langOf(hctx)
	if l.Locale == i18n.English {
		return v, nil
	}
	d, ok := localizedDetail(p, l)
	if !ok {
		return v, nil
	}
	cp := *p
	cp.Detail = d
	return &cp, nil
}
