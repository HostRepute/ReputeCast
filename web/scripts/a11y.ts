/**
 * Accessibility sweep: runs axe-core (WCAG 2.0, 2.1 and 2.2, levels A and AA)
 * on every app screen of scripts/qa-map.ts and lists the violations.
 *
 *   node scripts/a11y.ts                          # every screen at 1440 and 390
 *   node scripts/a11y.ts --only Campaigns,Main    # some boards' screens
 *   node scripts/a11y.ts --widths 834 --out ../tmp/a11y
 *   node scripts/a11y.ts --widths 390 --targets   # also target sizes
 *   node scripts/a11y.ts --widths 390,834,1440 --overflow  # also sideways scrolling
 *   node scripts/a11y.ts --public http://localhost:8080   # public pages' own links go to --app
 *
 * --targets adds the design system's target rule (design/design-system,
 * "Targets"): standalone controls at least 44px on phone widths (767px and
 * below) and 24px elsewhere; links inside running text are exempt, and a
 * checkbox or radio counts its whole label.
 *
 * Needs the API and web dev servers with the demo data. Writes report.json
 * to --out and prints the violations per rule, worst first. States that
 * need data changes (manual entries) are checked by hand.
 */
import { mkdirSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { resolve } from 'node:path'
import { chromium, type BrowserContext } from '@playwright/test'
import { entries } from './qa-map.ts'
import { signIn } from './lib/demo-auth.ts'
import { resolvePath } from './lib/qa-paths.ts'

const args = process.argv.slice(2)
function opt(name: string, def?: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:5173')!
const only = new Set((opt('only') ?? '').split(',').filter(Boolean))
const widths = (opt('widths') ?? '1440,390').split(',').map(Number)
const out = resolve(opt('out', '../tmp/a11y')!)
const workspace = 'acme-newsletter'
const axePath = createRequire(import.meta.url).resolve('axe-core/axe.min.js')
const axeOptions = { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'] }, resultTypes: ['violations'] }
// --public: the origin a server writes into its public pages' links and
// stylesheets when that isn't --app (its public URL names another port);
// requests to it are sent to --app instead.
const publicOrigin = opt('public')
const targets = args.includes('--targets')
const overflow = args.includes('--overflow')
// --overflow: the page itself scrolls sideways (a table or a code block in
// its own scroller is fine). Reported like an axe rule, naming the outermost
// elements that stick out past the viewport.
const overflowCheck = `(() => {
  const vw = document.documentElement.clientWidth
  if (document.scrollingElement.scrollWidth <= vw + 1) return []
  const out = (el) => el.getBoundingClientRect().right > vw + 1
  const nodes = []
  for (const el of document.querySelectorAll('body *')) {
    if (nodes.length >= 5) break
    if (!out(el) || el.getBoundingClientRect().width === 0) continue
    if (el.parentElement && el.parentElement !== document.body && out(el.parentElement)) continue
    const name = (el.id ? '#' + el.id : '') + (typeof el.className === 'string' && el.className ? '.' + el.className.trim().split(/\\s+/).slice(0, 3).join('.') : '')
    nodes.push({ target: [el.tagName.toLowerCase() + name + ' right ' + Math.round(el.getBoundingClientRect().right) + 'px of ' + vw], html: el.outerHTML.slice(0, 200) })
  }
  return [{ id: 'page-overflow-x', impact: 'serious', help: 'The page doesn\\'t scroll sideways at this width', helpUrl: 'design/design-system/README.md', nodes }]
})()`
// Reported like an axe rule. A string, like the axe call below.
const targetCheck = `(() => {
  const min = innerWidth <= 767 ? 44 : 24
  const nodes = []
  for (const el of document.querySelectorAll('main :is(button, a[href], select, textarea, summary, [role=switch], input:not([type=hidden]))')) {
    if (el.closest('[inert], [aria-hidden=true]')) continue
    const st = getComputedStyle(el)
    if (st.visibility === 'hidden' || (el.tagName === 'A' && st.display === 'inline')) continue
    // A link or a link-like button inside a sentence (text runs right
    // beside it) is exempt, like an inline link.
    if (/^(A|BUTTON)$/.test(el.tagName) && st.display.startsWith('inline') && !st.display.includes('flex') && Array.from(el.parentNode.childNodes).some((n) => n.nodeType === 3 && /[\\p{L}\\p{N}]/u.test(n.textContent))) continue
    // A field is hit through the label around it too, and a checkbox or radio
    // through its label wherever it is: either one big enough will do.
    const label = el.tagName === 'INPUT' && (el.closest('label') || (/^(checkbox|radio)$/.test(el.type) && el.id && document.querySelector('label[for="' + CSS.escape(el.id) + '"]')))
    const rects = [el, label].filter(Boolean).map((x) => x.getBoundingClientRect())
    const r = rects[0]
    // Visually hidden controls (a file input behind its button) aren't pointer targets.
    if (r.width <= 1 || r.height <= 1 || rects.some((x) => x.width >= min - 0.5 && x.height >= min - 0.5)) continue
    const name = (el.getAttribute('aria-label') || el.textContent || el.value || el.name || '').trim().replace(/\\s+/g, ' ').slice(0, 40)
    nodes.push({ target: [el.tagName.toLowerCase() + ' "' + name + '" ' + Math.round(r.width) + 'x' + Math.round(r.height)], html: el.outerHTML.slice(0, 200) })
  }
  const out = []
  if (nodes.length) out.push({ id: 'target-size-' + min, impact: 'moderate', help: 'Targets are at least ' + min + 'px', helpUrl: 'design/design-system/README.md', nodes })
  return out
})()`

type Node = { target: string[]; html: string; failureSummary?: string }
type Violation = { id: string; impact: string | null; help: string; helpUrl: string; nodes: Node[] }
type Finding = { path: string; width: number; violations: Violation[] }

async function main() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  const contexts = new Map<string, BrowserContext>()
  const findings: Finding[] = []
  const skipped: string[] = []
  const failedUsers = new Set<string>()
  const seen = new Set<string>()
  try {
    for (const e of entries) {
      if (!e.path || e.manual || (only.size && !only.has(e.board))) continue
      const user = e.user ?? 'linh@acme.io'
      if (failedUsers.has(user)) {
        skipped.push(`${e.board} (can't sign in as ${user})`)
        continue
      }
      let ctx = contexts.get(user)
      if (!ctx) {
        // bypassCSP: the public pages allow no inline script, which is how axe is injected.
        ctx = await browser.newContext({ deviceScaleFactor: 1, reducedMotion: 'reduce', bypassCSP: true })
        if (publicOrigin) await ctx.route(`${publicOrigin}/**`, (route) => route.continue({ url: app + route.request().url().slice(publicOrigin.length) }))
        try {
          if (user !== 'anon') await signIn(ctx, app, user)
        } catch {
          // Demo people without a password (SSO only) can't be signed in here.
          failedUsers.add(user)
          skipped.push(`${e.board} (can't sign in as ${user})`)
          await ctx.close()
          continue
        }
        contexts.set(user, ctx)
      }
      const path = await resolvePath(ctx, app, e.path, workspace)
      if (!path) {
        skipped.push(`${e.board} (${e.path})`)
        continue
      }
      for (const width of e.widths ?? widths) {
        if (!widths.includes(width) || seen.has(`${user} ${path} ${width}`)) continue
        seen.add(`${user} ${path} ${width}`)
        const page = await ctx.newPage()
        try {
          await page.setViewportSize({ width, height: 1000 })
          await page.goto(app + path, { waitUntil: 'networkidle' })
          await page.waitForTimeout(400)
          await page.addScriptTag({ path: axePath })
          // A string, like the other QA scripts: they compile without the DOM types.
          const violations = (await page.evaluate(`axe.run(document, ${JSON.stringify(axeOptions)}).then((r) => r.violations)`)) as Violation[]
          if (targets) violations.push(...((await page.evaluate(targetCheck)) as Violation[]))
          if (overflow) violations.push(...((await page.evaluate(overflowCheck)) as Violation[]))
          findings.push({ path, width, violations })
          console.log(`${String(violations.length).padStart(3)} rule(s)  ${width}  ${path}`)
        } catch (err) {
          console.log(`error  ${width}  ${path}: ${String(err)}`)
        } finally {
          await page.close()
        }
      }
    }
  } finally {
    await browser.close()
  }
  writeFileSync(`${out}/report.json`, JSON.stringify({ findings, skipped }, null, 2))
  // Per rule: how many screens, how many elements, an example.
  const rules = new Map<string, { impact: string | null; help: string; screens: Set<string>; nodes: number; example: string }>()
  for (const f of findings) {
    for (const v of f.violations) {
      const r = rules.get(v.id) ?? { impact: v.impact, help: v.help, screens: new Set<string>(), nodes: 0, example: '' }
      r.screens.add(`${f.path} @${f.width}`)
      r.nodes += v.nodes.length
      r.example ||= `${f.path} @${f.width}: ${v.nodes[0]?.target.join(' ')}`
      rules.set(v.id, r)
    }
  }
  const order = ['critical', 'serious', 'moderate', 'minor']
  console.log('\nViolations by rule:')
  for (const [id, r] of [...rules].sort((a, b) => order.indexOf(a[1].impact ?? 'minor') - order.indexOf(b[1].impact ?? 'minor'))) {
    console.log(`- ${id} (${r.impact}): ${r.help} — ${r.nodes} element(s) on ${r.screens.size} screen(s); e.g. ${r.example}`)
  }
  if (skipped.length) console.log(`\nSkipped: ${skipped.join(', ')}`)
  console.log(`\nReport: ${out}/report.json`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
