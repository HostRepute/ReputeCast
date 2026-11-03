# Translating ReputeCast

ReputeCast's own interface comes in English, German (Deutsch), Spanish
(Español), French (Français) and Vietnamese (Tiếng Việt): the app, the
sign-in and invitation pages, the server's messages to members (errors with
their fix, "who can help", notifications) and the emails ReputeCast sends to
members. This guide is for translators and for developers adding text.

What a workspace sends its contacts (campaigns, transactional email, public
pages and forms) is translated by the workspace itself, as language versions
in the composer. That is not covered here.

## Where the text lives

| What | English source | Translation |
|---|---|---|
| The web app | `web/src/**/messages.ts`, `web/src/**/<screen>.messages.ts` | `messages.<lang>.ts` next to the source (`messages.de.ts`, `messages.vi.ts`…) |
| Server messages and member emails | `internal/<package>/messages/en.json` | `internal/<package>/messages/<lang>.json` |
| The words ReputeCast uses | | [`glossary-de.md`](glossary-de.md), [`glossary-es.md`](glossary-es.md), [`glossary-fr.md`](glossary-fr.md), [`glossary-vi.md`](glossary-vi.md) |

A web source declares its messages with an ID (`namespace.key`):

```ts
export const m = defineMessages('campaigns', {
  newCampaign: 'New campaign',
  blockingIssues: '{count, plural, one {# blocking issue} other {# blocking issues}}',
})
```

and its translation repeats the keys it translates:

```ts
export default translate(m, {
  newCampaign: 'Chiến dịch mới',
  blockingIssues: '{count} lỗi chặn gửi',
})
```

A source that defines several groups (`mb`, `m`, `mp`) gets one
`translate()` per group, spread together:
`export default { ...translate(mb, { … }), ...translate(m, { … }) }`.

A server catalog is a JSON object of `"key": "text"`, or
`"key": { "message": "text", "note": "for translators" }` when the English
needs explaining. Read the notes; they say what an argument holds. A few
keys are made from data, such as `short.<event type>` in
`internal/notify/messages`: a new event type shows its English in the code
until its key is added to `en.json`, so add it there when you add the type.

A key that is missing from a translation shows in English, so a screen is
never blank. Translate the glossary's terms first, then the screens.

## Writing a translation

Messages are [ICU MessageFormat](https://unicode-org.github.io/icu/userguide/format_parse/messages/):

- **Arguments** in braces are filled in by the app: `{name}`, `{count}`,
  `{date}`. Keep them, spelled exactly as in the English; move them to where
  your language puts them: `edited {time} by {name}` → `{name} sửa lúc {time}`.
- **Plurals:** `{count, plural, one {# day} other {# days}}`. Use the plural
  categories of your language (CLDR). German, Spanish and French need `one`
  and `other` (French puts 0 in `one`; their `many` falls back to `other`).
  Vietnamese has only `other`, so `{count} ngày` is enough. `=1 {…}` matches an exact number when the
  wording changes ("một lần").
- **Choices:** `{status, select, drafts {…} sent {…} other {…}}`. Keep the
  values on the left (`drafts`, `sent`, `other`) and translate the text in
  braces.
- **Tags** mark text the app styles or links: `<b>…</b>`, `<link>…</link>`,
  `<mono>…</mono>`, `<k>…</k>`, `<kbd>…</kbd>`. Keep each tag around the
  words that play the same part.
- A message is a whole sentence or label. Don't split one, and don't count
  on two messages being shown side by side in a certain order.
- **Language names** (`{language}`, `{languages}`) come in the label form
  your language uses on its own. Vietnamese writes "tiếng" in lower case
  inside a sentence, and Spanish and French write the whole name in lower
  case there ("Añadir inglés", "Version en anglais"): the app does that
  from the argument's place, so write `{language}`, never "tiếng
  {language}" or "el {language}" (each glossary's Voice section). German
  names are capitalised nouns everywhere.
- An apostrophe is fine as is (`isn't`, `l'été`). To show a brace as text,
  quote it: `'{'`.

**Never translate** workspace content (campaign, list, template, form and
custom role names), people's names, email addresses, permission keys
(`campaigns:send`), audit action names, API error codes, provider names,
DNS records, code samples and command lines. The glossary's "Never
translated" list has the rest.

**Numbers, dates and times** come from the reader's region format, never from
a translation: write `{date}`, not a date. The region formats are separate
from the language: a member can read English with the Vietnamese formats.

## Checking

```bash
cd web
pnpm run i18n:check              # every missing message, by file
pnpm run i18n:check --lang vi    # one language
pnpm run i18n:check --summary    # one line per language
pnpm run i18n:check --hardcoded  # every hard-coded string, with its line
```

It reads the web sources and translations and the server's catalogs, and:

- lists the messages each language misses, grouped by the file that defines
  their English;
- counts the hard-coded text left in the screens: JSX text, and plain
  strings in attributes people read (`title`, `label`, `placeholder`,
  `aria-label`…). It reads in English in every language. A literal that
  isn't text to translate, such as a format, goes in an expression:
  `placeholder={'xxxx-xxxx-xxxx'}`;
- fails on a message that isn't valid ICU, a translated key the English
  doesn't have, and an argument or tag the English doesn't use;
- writes `web/src/i18n/status.gen.ts`, the languages complete in this
  release: nothing missing, and no hard-coded text left. Outside
  development, the language picker offers only those (English always); in
  development it shows every language. Every build runs the check first;
  commit `status.gen.ts` when it changes.

Then look at the screens: switch your own app language in **Account &
security › Language & region**, and read the stress-test screens (Campaigns,
Review & send, Settings › Team) at 1440 and 390 px. With the demo data
running, `node scripts/i18n-layout.ts --langs <code>` opens every screen of
`scripts/qa-map.ts` in English and in the language, and lists what the
language breaks that English doesn't: a page that scrolls sideways, text
that spills out of its box or is cut off, a button or a badge on two
lines. Long words must wrap or
shorten as `docs/dev/CONVENTIONS.md` › "Text expansion" describes, never
spill or get cut off. A server catalog is also checked when its package
loads (syntax, keys and arguments), so a broken translation stops the server
and every Go test at once.

### The pseudo-locale

**Pseudo (+40%)** accents every letter, makes each message about 40% longer
and wraps it in ⟦ ⟧: "⟦Šçĥéðûļé šéñð ţö 31,604 ŕéçîþîéñţš ·········⟧". Text
without brackets didn't go through the translation system; a missing ⟧ is
text that was cut off. It appears in the picker in development, or anywhere
the server runs with `i18n.pseudo_locale` on
(`REPUTECAST_I18N__PSEUDO_LOCALE=true`).

## Adding a language

1. Write the glossary first (`docs/i18n/glossary-<code>.md`), from the
   Vietnamese one's sections.
2. Add the language to `web/src/i18n/languages.ts` (`APP_LANGUAGES`, its name
   in its own language), `web/src/i18n/store.ts` (`AppLocale` and its
   `import.meta.glob`) and `internal/i18n/locale.go` (`appLocales`, `Name`,
   `Tag`), and pick the region format it follows by default
   (`DEFAULT_REGIONS`, `DefaultRegion`). Plural rules come from CLDR on both
   sides. Add the words that join a list (`internal/i18n/lists.go`), and,
   if the language writes language names in another case inside a
   sentence, the rule in `internal/i18n/names.go` and
   `web/src/i18n/names.ts`. The API's language enums (`enum:"en,de,…"`)
   and `languages.BuiltinLabels` take the code too; run `make generate`.
3. A missing region format goes in `internal/i18n/region.go` and
   `web/src/i18n/regions.ts`, with test vectors in
   `internal/i18n/testdata/regions.json` that both sides must pass.
4. Add `messages.<code>.ts` files for the web and `messages/<code>.json` for
   the server packages, and run `pnpm run i18n:check --lang <code>` until it
   lists nothing.
5. Check the fonts: IBM Plex covers Latin, Vietnamese, Cyrillic and Greek;
   another script needs a font subset in `web/src/styles/fonts.css`.

A native speaker reviews a language before it ships in a release.
