/**
 * Accessibility sweep: runs axe-core (WCAG 2.0, 2.1 and 2.2, levels A and AA)
 * on every app screen of scripts/qa-map.ts and lists the violations.
 *
 *   node scripts/a11y.ts                          # every screen at 1440 and 390
 *   node scripts/a11y.ts --only Campaigns,Main    # some boards' screens
 *   node scripts/a11y.ts --widths 834 --out ../tmp/a11y
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

type Node = { target: string[]; html: string; failureSummary?: string }
type Violation = { id: string; impact: string | null; help: string; helpUrl: string; nodes: Node[] }
type Finding = { path: string; width: number; violations: Violation[] }

/** Looks up the ids behind {campaign:Name} and {sso:Name} in a path. */
async function resolvePath(ctx: BrowserContext, path: string) {
  let outPath = path
  for (const m of path.matchAll(/\{(\w+):([^}]+)\}/g)) {
    const [kind, name] = [m[1]!, m[2]!]
    let id: string | undefined
    if (kind === 'campaign') {
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/campaigns`, { params: { q: name, pageSize: 100 } })
      id = ((await r.json()) as { items?: { id: string; name: string }[] }).items?.find((c) => c.name === name)?.id
    } else if (kind === 'sso') {
      const r = await ctx.request.get(`${app}/api/v1/admin/sso`)
      id = ((await r.json()) as { items?: { id: string; name: string }[] }).items?.find((p) => p.name === name)?.id
    } else if (kind === 'setup') {
      // W10: the latest provider setup of an adapter ({setup:ses}).
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/connect/setups`, { params: { adapter: name } })
      id = ((await r.json()) as { items?: { id: string }[] }).items?.[0]?.id
    }
    if (!id) return undefined
    outPath = outPath.replace(m[0], encodeURIComponent(id))
  }
  return outPath
}

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
        ctx = await browser.newContext({ deviceScaleFactor: 1, reducedMotion: 'reduce' })
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
      const path = await resolvePath(ctx, e.path)
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
