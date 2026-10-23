package api

import (
	"embed"
	"regexp"
	"strings"

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
	mEscalation = msgs.M("escalation")
	mKeyExpired = msgs.M("keyExpired")

	// Problems the middleware writes in the request's language.
	mNotForThisKey         = msgs.M("problem.notForThisKey")
	mIntegrationOnly       = msgs.M("problem.integrationOnly")
	mWordPressOnly         = msgs.M("problem.wordpressOnly")
	mKeyOnly               = msgs.M("problem.keyOnly")
	mIntegrationsOff       = msgs.M("problem.integrationsOff")
	mIntegrationKeyRevoked = msgs.M("problem.integrationKeyRevoked")
	mIntegrationKeyExpired = msgs.M("problem.integrationKeyExpired")
	mOwnerLeft             = msgs.M("problem.ownerLeft")
)

// fixed maps the English of problems written without the request's
// language to their messages: the middleware's own (fixed.*), and every
// package's "Campaign not found." (notFound.*, one message per kind of
// thing), which NotFound and MapError write without a context.
var fixed = map[string]i18n.Msg{}

func init() {
	for _, k := range msgs.Keys() {
		if strings.HasPrefix(k, "fixed.") || strings.HasPrefix(k, "notFound.") {
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
// the campaigns:send permission. Ask Marcus Chen or Linh Nguyen." Its
// words are internal/authz's (authz.DeniedText).
func DeniedText(l i18n.Lang, perm authz.Permission, helpers []string) string {
	return authz.DeniedText(l, perm, helpers)
}

// RoleName writes a role's name in a language: the built-in roles (Owner,
// Admin, Editor, Viewer, and an agency's client roles) are translated;
// custom roles, and a built-in role stored under another name, keep the
// name they were given.
func RoleName(l i18n.Lang, key, name string) string {
	b, ok := authz.Builtin(key)
	if !ok || (name != "" && name != b.Name) {
		return name
	}
	m, err := msgs.Lookup("role." + key)
	if err != nil {
		return b.Name
	}
	return m.Text(l, nil)
}

var escalationRe = regexp.MustCompile(`^You can't grant (.+) because you don't hold it\.$`)

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
