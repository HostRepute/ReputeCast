/**
 * Layout sweep for translations: opens every app screen of scripts/qa-map.ts
 * in English and in each language asked for, and lists what the language
 * breaks that English doesn't (docs/dev/CONVENTIONS.md › "Text expansion"):
 *
 * - overflow-x: the page scrolls sideways;
 * - spill: text runs out of its box (a long word that can't wrap);
 * - clipped: text cut off by `overflow: hidden` without an ellipsis;
 * - button-wraps: a button's label on two lines (rule 4);
 * - badge-wraps: a badge on two lines (rule 5).
 *
 *   node scripts/i18n-layout.ts --langs de,es,fr
 *   node scripts/i18n-layout.ts --langs de --only Campaigns,Main --widths 390
 *   node scripts/i18n-layout.ts --langs fr --chrome /opt/pw-browsers/chromium
 *
 * Needs the API and web dev servers with the demo data (as scripts/a11y.ts).
 * Each demo user's language is set through the API for the run and put back
 * to "default" at the end. Writes report.json to --out.
 */
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium, type Browser, type BrowserContext } from '@playwright/test'
import { entries, type Entry } from './qa-map.ts'
import { signIn } from './lib/demo-auth.ts'
import { resolvePath } from './lib/qa-paths.ts'

const args = process.argv.slice(2)
function opt(name: string, def?: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:5173')!
const langs = (opt('langs') ?? 'de,es,fr').split(',').filter(Boolean)
const only = new Set((opt('only') ?? '').split(',').filter(Boolean))
const widths = (opt('widths') ?? '1440,390').split(',').map(Number)
const out = resolve(opt('out', '../tmp/i18n-layout')!)
const chrome = opt('chrome', process.env.CHROME_PATH)
const parallel = Number(opt('parallel', '3'))

type Issue = { kind: string; where: string; text: string }

// Runs in the page (a string, like the other QA scripts: they compile
// without the DOM types). `where` names an element by its tag, classes and
// two ancestors, so the same element matches across languages.
const layoutCheck = `(() => {
  const vw = document.documentElement.clientWidth
  const name = (el) => el.tagName.toLowerCase() + (typeof el.className === 'string' && el.className.trim() ? '.' + el.className.trim().split(/\\s+/).filter((c) => !/^\\[|:|\\d/.test(c)).slice(0, 3).join('.') : '')
  const where = (el) => [el.parentElement?.parentElement, el.parentElement, el].filter(Boolean).map(name).join(' > ')
  const text = (el) => (el.innerText || el.textContent || '').trim().replace(/\\s+/g, ' ').slice(0, 80)
  // Visually hidden text (sr-only: a 1 px clipped box) isn't laid out for reading.
  const visible = (el) => { const r = el.getBoundingClientRect(); return r.width > 2 && r.height > 2 }
  const issues = []
  if (document.scrollingElement.scrollWidth > vw + 1) issues.push({ kind: 'overflow-x', where: 'page', text: document.scrollingElement.scrollWidth + 'px of ' + vw })
  const ownText = (el) => Array.from(el.childNodes).some((n) => n.nodeType === 3 && /\\p{L}/u.test(n.textContent))
  for (const el of document.querySelectorAll('body *')) {
    if (!ownText(el) || !visible(el) || el.closest('[aria-hidden=true], svg, pre, code, textarea, .cm-editor, [contenteditable=true], iframe')) continue
    const st = getComputedStyle(el)
    if (st.display === 'inline') continue
    const clipX = st.overflowX === 'hidden' || st.overflowX === 'clip'
    const clipY = st.overflowY === 'hidden' || st.overflowY === 'clip'
    const wide = el.scrollWidth > el.clientWidth + 1
    const tall = el.scrollHeight > el.clientHeight + 2
    if (wide && !clipX && st.overflowX === 'visible') issues.push({ kind: 'spill', where: where(el), text: text(el) })
    else if (wide && clipX && st.textOverflow !== 'ellipsis') issues.push({ kind: 'clipped', where: where(el), text: text(el) })
    else if (tall && clipY && st.webkitLineClamp === 'none' && !el.title) issues.push({ kind: 'clipped', where: where(el), text: text(el) })
  }
  // The lines an element's text takes: its text nodes' line boxes, so an
  // icon or a count beside the words doesn't count as a line. A control
  // drawn on two lines (a name over a role) is two lines in English too;
  // the count is part of the kind, so only a change shows.
  const textLines = (el) => {
    const tops = []
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT)
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      if (!/\\S/.test(n.textContent)) continue
      const r = document.createRange()
      r.selectNodeContents(n)
      for (const x of r.getClientRects()) if (x.width > 1) tops.push(x.top)
    }
    const step = parseFloat(getComputedStyle(el).fontSize) * 0.9
    tops.sort((a, b) => a - b)
    let lines = tops.length ? 1 : 0
    for (let i = 1; i < tops.length; i++) if (tops[i] - tops[i - 1] > step) lines++
    return lines
  }
  for (const el of document.querySelectorAll('button, a.btn, [role=button], .badge')) {
    if (!visible(el) || !text(el) || el.closest('[aria-hidden=true]')) continue
    const n = textLines(el)
    if (n >= 2) issues.push({ kind: (el.classList.contains('badge') ? 'badge-wraps:' : 'button-wraps:') + n, where: where(el), text: text(el) })
  }
  return issues
})()`

type Screen = { board: string; user: string; path: string }
type Result = { lang: string; board: string; path: string; width: number; issues: Issue[]; error?: string }

async function setLanguage(ctx: BrowserContext, language: string) {
  const r = await ctx.request.put(`${app}/api/v1/me/language`, { data: { language, regionFormat: 'auto', hourCycle: 'auto', weekStart: 'auto' } })
  if (!r.ok()) throw new Error(`set language ${language}: ${r.status()} ${await r.text()}`)
}

async function contextFor(browser: Browser, user: string, lang: string) {
  const ctx = await browser.newContext({ deviceScaleFactor: 1, reducedMotion: 'reduce' })
  if (user !== 'anon') {
    await signIn(ctx, app, user)
    await setLanguage(ctx, lang === 'en' ? 'en' : lang)
  } else {
    // The sign-in pages' language menu remembers the pick here (web/src/i18n/sync.ts).
    await ctx.addInitScript(`localStorage.setItem('rc.language', ${JSON.stringify(JSON.stringify(lang))})`)
  }
  return ctx
}

async function sweep(browser: Browser, lang: string, screens: Screen[]): Promise<Result[]> {
  const results: Result[] = []
  const contexts = new Map<string, BrowserContext>()
  const queue = screens.flatMap((s) => widths.map((w) => ({ ...s, width: w })))
  const ctxOf = async (user: string) => {
    let c = contexts.get(user)
    if (!c) {
      c = await contextFor(browser, user, lang)
      contexts.set(user, c)
    }
    return c
  }
  // Sign in first, one context per user, so the workers share them.
  for (const user of new Set(screens.map((s) => s.user))) await ctxOf(user)
  const worker = async () => {
    for (let item = queue.shift(); item; item = queue.shift()) {
      const page = await (await ctxOf(item.user)).newPage()
      try {
        await page.setViewportSize({ width: item.width, height: 1000 })
        await page.goto(app + item.path, { waitUntil: 'networkidle', timeout: 60000 })
        await page.waitForFunction(`document.documentElement.lang === ${JSON.stringify(lang)}`, undefined, { timeout: 10000 }).catch(() => undefined)
        await page.waitForTimeout(500)
        const issues = (await page.evaluate(layoutCheck)) as Issue[]
        const shown = await page.evaluate('document.documentElement.lang')
        results.push({ lang, board: item.board, path: item.path, width: item.width, issues, ...(shown !== lang ? { error: `shown in ${String(shown)}` } : {}) })
      } catch (err) {
        results.push({ lang, board: item.board, path: item.path, width: item.width, issues: [], error: String(err).slice(0, 200) })
      } finally {
        await page.close()
      }
    }
  }
  await Promise.all(Array.from({ length: parallel }, worker))
  for (const [user, c] of contexts) {
    if (user !== 'anon') await setLanguage(c, 'default').catch(() => undefined)
    await c.close()
  }
  return results
}

const key = (r: { path: string; width: number }, i: Issue) => `${r.path} @${r.width} ${i.kind} ${i.where}`

async function main() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch(chrome ? { executablePath: chrome, headless: true } : { channel: 'chrome', headless: true })
  const screens: Screen[] = []
  const seen = new Set<string>()
  try {
    // Resolve the paths once, signed in as each user in English.
    const resolvers = new Map<string, BrowserContext>()
    for (const e of entries as Entry[]) {
      if (!e.path || e.manual || (only.size && !only.has(e.board))) continue
      const user = e.user ?? 'linh@acme.io'
      let ctx = resolvers.get(user)
      if (!ctx) {
        ctx = await browser.newContext()
        if (user !== 'anon') await signIn(ctx, app, user).catch(() => undefined)
        resolvers.set(user, ctx)
      }
      const path = await resolvePath(ctx, app, e.path)
      if (!path || seen.has(`${user} ${path}`)) continue
      seen.add(`${user} ${path}`)
      screens.push({ board: e.board, user, path })
    }
    for (const c of resolvers.values()) await c.close()
    console.log(`${screens.length} screens × ${widths.length} widths`)

    const english = await sweep(browser, 'en', screens)
    const baseline = new Set(english.flatMap((r) => r.issues.map((i) => key(r, i))))
    const report: Record<string, Result[]> = { en: english }
    for (const lang of langs) {
      const results = await sweep(browser, lang, screens)
      // Only what English doesn't have.
      for (const r of results) r.issues = r.issues.filter((i) => !baseline.has(key(r, i)))
      report[lang] = results
      const found = results.filter((r) => r.issues.length || r.error)
      console.log(`\n${lang}: ${found.reduce((n, r) => n + r.issues.length, 0)} issue(s) on ${found.length} screen(s)`)
      for (const r of found) {
        if (r.error) console.log(`  error ${r.width} ${r.path}: ${r.error}`)
        for (const i of r.issues) console.log(`  ${i.kind.padEnd(12)} ${r.width} ${r.board} ${r.path}\n      ${i.where} “${i.text}”`)
      }
    }
    writeFileSync(`${out}/report.json`, JSON.stringify(report, null, 2))
    console.log(`\nReport: ${out}/report.json`)
  } finally {
    await browser.close()
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
