package sso

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// SSO reads in the reader's language (i18n.LangFrom): its problems, the
// presets' hints, the setup page's notes and the test sign-in's outcome.
// What it stores for later readers keeps its message beside the English:
// a discovery test's and a test sign-in's error (discovery, testRun), why
// a sign-in was turned away (flowError, kept as the audit entry's reason)
// and the audit entries' summaries and reasons (audit.KeepSummary,
// audit.KeepReason). Names (providers, presets, IdP menus and claims),
// scopes, URLs and what an identity provider or a library answered stay
// as they are.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "sso")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// textIn writes a kept message in a language, or english when there is
// none (rows written before it was kept).
func textIn(t i18n.Text, l i18n.Lang, english string) string {
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// In is the preset in a language: its hints, the app the redirect URI goes
// to, how it enforces two-factor and its docs link's label. Product names,
// issuer patterns and scopes stay as they are.
func (p SSOPreset) In(l i18n.Lang) SSOPreset {
	if l.Locale == i18n.English {
		return p
	}
	text := func(field, english string) string {
		if english == "" {
			return ""
		}
		m, err := msgs.Lookup("preset." + p.Key + "." + field)
		if err != nil {
			return english
		}
		return m.Text(l, nil)
	}
	out := p
	out.Name = text("name", p.Name)
	out.ShortName = text("shortName", p.ShortName)
	out.App = text("app", p.App)
	out.DiscoveryHint = text("discoveryHint", p.DiscoveryHint)
	out.GroupsHint = text("groupsHint", p.GroupsHint)
	out.GroupsMissingHint = text("groupsMissingHint", p.GroupsMissingHint)
	out.TrustedTwoFA = text("trustedTwoFactor", p.TrustedTwoFA)
	out.DocsLabel = text("docsLabel", p.DocsLabel)
	return out
}

// presetsIn are the presets in a language, in the setup screen's order.
func presetsIn(l i18n.Lang) []SSOPreset {
	out := make([]SSOPreset, len(Presets))
	for i, p := range Presets {
		out[i] = p.In(l)
	}
	return out
}

// trustedTwoFA is how the provider's preset enforces two-factor when it is
// trusted, kept for later readers.
func trustedTwoFA(preset string) i18n.Text {
	return keep("preset."+PresetFor(preset).Key+".trustedTwoFactor", nil)
}

// via is the select value of where a membership's role comes from: a
// group mapping or the just-in-time default.
func via(group string) string {
	if group != "" {
		return "group"
	}
	return "jit"
}

// yesNo is the select value of a condition.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
