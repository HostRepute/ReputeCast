// The bundle's size budget (review 2026-11 FE-8), checked after `vite
// build` in CI: what the first page load downloads, every lazy chunk, and
// the whole app, gzipped. Raise a budget only on purpose, in the same
// change that needs it.
//
//   node scripts/bundle-budget.ts
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gzipSync } from 'node:zlib'

const KB = 1024
const budget = {
  // index.html's script and the chunks it preloads, with its stylesheets.
  // 390 KB after review 2026-11 FE-9 (433 KB before): a route's search
  // schema and checks load with the app, so they import only small modules
  // (integrations/platforms.ts, pages/templateKeys.ts), never a feature's
  // pages or copy; an agency's menu loads for its staff only. 315 KB in 5
  // scripts since the route definitions and the shared components load as
  // one file each (vite.config.ts, codeSplitting), down from 145 scripts.
  // 305 KB since no language's translations, nor the list of them, load
  // with the app (src/i18n/catalogs).
  initial: 340 * KB,
  // Any one lazy chunk (the email composer's editor is the largest).
  chunk: 400 * KB,
  // Every script of the app. 4.5 MB until German, Spanish and French
  // joined Vietnamese: each language's translations are about 365 KB,
  // downloaded only by those who read the app in it (src/i18n/store.ts).
  // 6 MB since Đợt 2 (M58–M66): its nine milestones' screens and their copy
  // in five languages add about 300 KB (5,742 KB before, 6,044 KB after),
  // nearly all in lazy chunks (the first load grew 3 KB, to 316 KB).
  total: 6 * 1024 * KB,
}

const dist = new URL('../dist/', import.meta.url).pathname
const html = readFileSync(join(dist, 'index.html'), 'utf8')
const gz = (file: string) => gzipSync(readFileSync(join(dist, file))).length
const initialFiles = [...html.matchAll(/(?:src|href)="\/(assets\/[^"]+\.(?:js|css))"/g)].map((m) => m[1]!)
const initial = initialFiles.reduce((n, f) => n + gz(f), 0)
const chunks = readdirSync(join(dist, 'assets'))
  .filter((f) => f.endsWith('.js'))
  .map((f) => ({ f, size: gz(join('assets', f)) }))
const total = chunks.reduce((n, c) => n + c.size, 0)
const kb = (n: number) => `${(n / KB).toFixed(0)} KB`

const problems: string[] = []
if (initial > budget.initial) problems.push(`first load ${kb(initial)} gzipped, budget ${kb(budget.initial)} (${initialFiles.length} files)`)
for (const c of chunks) if (c.size > budget.chunk) problems.push(`${c.f} ${kb(c.size)} gzipped, budget ${kb(budget.chunk)} a chunk`)
if (total > budget.total) problems.push(`all scripts ${kb(total)} gzipped, budget ${kb(budget.total)}`)
const largest = chunks.sort((a, b) => b.size - a.size)[0]
console.log(`Bundle: first load ${kb(initial)} of ${kb(budget.initial)}; largest chunk ${largest?.f} ${kb(largest?.size ?? 0)} of ${kb(budget.chunk)}; all scripts ${kb(total)} of ${kb(budget.total)} (gzipped).`)
if (problems.length) {
  console.error(problems.join('\n'))
  process.exit(1)
}
