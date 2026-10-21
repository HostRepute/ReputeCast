// Package i18n is ReputeCast's server-side localization: the app languages
// (English, Vietnamese and the pseudo-locale used to check layouts), region
// formats, language negotiation, CLDR plural rules and message catalogs.
//
// # Catalogs
//
// A package keeps its messages in messages/<locale>.json next to its code
// and embeds them:
//
//	//go:embed messages/*.json
//	var messagesFS embed.FS
//	var msgs = i18n.MustLoad(messagesFS, "systemmail")
//	var mInviteSubject = msgs.M("invite.subject") // panics at init when the key is missing
//
// en.json holds the English source; every other file is a translation of
// it. A file is a flat object of keys to ICU MessageFormat strings ("key":
// "text", or "key": {"message": "text", "note": "for translators"}).
//
// Why this format: the web app's messages are ICU MessageFormat too
// (web/src/i18n), so translators learn one syntax for the whole product and
// translation platforms (Weblate, Crowdin, Lokalise) import the files as
// "ICU JSON". Plurals follow CLDR on both sides: golang.org/x/text here,
// Intl.PluralRules in the browser. pnpm run i18n:check lists missing keys
// for the server catalogs as well as the web's.
//
// The formatter implements the part of ICU MessageFormat ReputeCast uses:
// simple and number arguments, plural (with =N, offset and #), select,
// selectordinal and <tags>, with ICU's apostrophe quoting. It needs nothing
// beyond golang.org/x/text, which the module already depends on.
//
// Text the server stores for people who read it later keeps the message's
// ID and arguments as a Text (Msg.Keep), and Text.In writes it in each
// reader's language.
//
// # Languages and regions
//
// A Lang is how one person reads: the app language (Locale) and the region
// format (Region) that writes dates and numbers. Region formats are a
// snapshot of Unicode CLDR kept in this package and mirrored by
// web/src/i18n/regions.ts, so the server and every browser write the same
// date the same way; testdata/regions.json holds the vectors both test.
// "ReputeCast standard" keeps the app's own formats (Tue 14 Oct 2026 ·
// 09:00 · 48,213 · 99.1% · $2.84).
package i18n
