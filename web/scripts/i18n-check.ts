/**
 * Lists the messages each language still misses and the text the screens
 * still hard-code, checks every message and translation (ICU syntax,
 * arguments and tags), and writes src/i18n/status.gen.ts: the languages the
 * picker offers outside development. docs/i18n/translating.md explains the
 * workflow.
 *
 *   pnpm run i18n:check              # every missing message, by file
 *   pnpm run i18n:check --summary    # one line per language
 *   pnpm run i18n:check --lang vi    # one language
 *   pnpm run i18n:check --hardcoded  # every hard-coded string, with its line
 *   pnpm run i18n:check --json       # for tools
 *
 * Exits 1 when a message or a translation has an error. Missing
 * translations aren't errors: they read in English until translated.
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { check, collect, statusModule, summary } from './lib/i18n-check.ts'

const args = process.argv.slice(2)
const opt = (name: string) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : undefined
}

const repo = fileURLToPath(new URL('../..', import.meta.url))
const files = collect(repo)
// The Docker image builds the web app from web/ alone: without the server's
// catalogs, the committed status stands.
const whole = files.some((f) => f.path.startsWith('internal/'))
const report = check(files)
const only = opt('lang')
if (only) report.languages = report.languages.filter((l) => l.code === only)

if (args.includes('--json')) {
  const json = {
    messages: report.messages.length,
    complete: report.complete,
    languages: report.languages.map((l) => ({ code: l.code, name: l.name, total: l.total, translated: l.translated, missing: Object.fromEntries(l.missing) })),
    hardCoded: report.hardCoded,
    errors: report.errors,
  }
  console.log(JSON.stringify(json, null, 2))
} else {
  console.log(summary(report, { verbose: !args.includes('--summary'), hardCoded: args.includes('--hardcoded') }))
}

const statusPath = fileURLToPath(new URL('../src/i18n/status.gen.ts', import.meta.url))
const next = statusModule(report)
let current = ''
try {
  current = readFileSync(statusPath, 'utf8')
} catch {
  // Written below.
}
if (!whole) {
  if (!args.includes('--json')) console.log("\nThe server's catalogs (internal/*/messages) aren't here: src/i18n/status.gen.ts stays as committed.")
} else if (current !== next) {
  writeFileSync(statusPath, next)
  if (!args.includes('--json')) console.log('\nUpdated src/i18n/status.gen.ts')
}
process.exit(report.errors.length ? 1 : 0)
