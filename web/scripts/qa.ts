/**
 * Visual QA sweep: every board in scripts/qa-map.ts next to its app screen,
 * with a pixel-difference score, collected in one HTML report.
 *
 *   node scripts/qa.ts                         # every board
 *   node scripts/qa.ts --only Campaigns,Main   # some boards
 *   node scripts/qa.ts --widths 1440 --out ../tmp/qa
 *
 * Options: --app, --design-url (as in shot.ts), --only, --widths, --out,
 * --ink (brightness below which a pixel counts as ink, default 82%).
 *
 * Needs the API and web dev servers with the demo data, the design board
 * server, and ImageMagick 7 (`magick`). The score (see score()) ranks what
 * to look at first; it doesn't replace looking at the side-by-side images.
 */
import { execFileSync } from 'node:child_process'
import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium, type Browser, type BrowserContext } from '@playwright/test'
import { entries, type Entry } from './qa-map.ts'
import { signIn } from './lib/demo-auth.ts'

const args = process.argv.slice(2)
function opt(name: string, def?: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}

const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:5173')!
const designURL = opt('design-url', process.env.SHOT_DESIGN ?? 'http://localhost:8765')!
const only = new Set((opt('only') ?? '').split(',').filter(Boolean))
const widthsArg = opt('widths')
const out = resolve(opt('out', '../tmp/qa')!)
const inkLevel = opt('ink', '82%')!
const workspace = 'acme-newsletter'

type Result = { board: string; width: number; path?: string; score?: number; side?: string; design: string; app?: string; error?: string }

function magick(...a: string[]) {
  return execFileSync('magick', a, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] })
}

function size(file: string) {
  const [w, h] = magick('identify', '-format', '%w %h', file).trim().split(' ').map(Number)
  return { w: w!, h: h! }
}

/**
 * score compares where the "ink" is (text, icons, lines, filled controls) in
 * the top `top` pixels of the board (`a`) and the app page (`b`): the share
 * of ink in one image with no ink within 2 px in the other. Small shifts don't count; missing, extra or
 * moved content does. The shorter image is padded, so a missing section
 * counts fully. 0 means the same layout; a stub page next to a full board
 * scores near 100.
 */
function score(a: string, b: string, top = 1600) {
  const sa = size(a)
  const sb = size(b)
  // Only the area the board covers counts (a is the board): a longer app
  // page (more rows of real data) isn't a difference, a shorter one is.
  const w = Math.max(sa.w, sb.w)
  const h = Math.min(sa.h, top)
  const tmp = (f: string, kind: string) => `${out}/img/.${kind}-${f.split('/').pop()}`
  const ink = (f: string) => {
    magick(f, '-background', 'white', '-gravity', 'northwest', '-extent', `${w}x${h}`, '-colorspace', 'gray', '-threshold', inkLevel, '-negate', tmp(f, 'ink'))
    magick(tmp(f, 'ink'), '-morphology', 'Dilate', 'Disk:2', tmp(f, 'near'))
  }
  ink(a)
  ink(b)
  const count = (...cmd: string[]) => Number(magick(...cmd, '-format', '%[fx:mean*w*h]', 'info:').trim())
  const only = (x: string, y: string) => count(tmp(x, 'ink'), '(', tmp(y, 'near'), '-negate', ')', '-compose', 'Multiply', '-composite')
  const inkA = count(tmp(a, 'ink'))
  const inkB = count(tmp(b, 'ink'))
  const differing = only(a, b) + only(b, a)
  for (const f of [a, b]) for (const k of ['ink', 'near']) rmSync(tmp(f, k), { force: true })
  return inkA + inkB === 0 ? 0 : (differing / (inkA + inkB)) * 100
}

const resolvers: Record<string, (ctx: BrowserContext, value: string) => Promise<string>> = {
  async campaign(ctx, name) {
    const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/campaigns`, { params: { q: name, pageSize: 100 } })
    const body = (await r.json()) as { items?: { id: string; name: string }[] }
    const hit = body.items?.find((c) => c.name === name)
    if (!hit) throw new Error(`no campaign named ${name}`)
    return hit.id
  },
  async sso(ctx, name) {
    const r = await ctx.request.get(`${app}/api/v1/admin/sso`)
    const body = (await r.json()) as { items?: { id: string; name: string }[] }
    const hit = body.items?.find((p) => p.name === name)
    if (!hit) throw new Error(`no SSO provider named ${name}`)
    return hit.id
  },
  // The latest provider setup of an adapter ({setup:ses}).
  async setup(ctx, adapter) {
    const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/connect/setups`, { params: { adapter } })
    const body = (await r.json()) as { items?: { id: string }[] }
    const hit = body.items?.[0]
    if (!hit) throw new Error(`no ${adapter} setup`)
    return hit.id
  },
  // Ids the demo generates at seed time ({member:daniel@acme.io}, {role:Content Writer},
  // {automation:Welcome series}, {contact:sam.rivera@outlook.com}, {import:draft} or {import:brevo}).
  member: (ctx, email) => lookup(ctx, 'members', 'items', (m) => m.email === email, `no member ${email}`),
  role: (ctx, name) => lookup(ctx, 'roles', 'roles', (r) => r.name === name, `no role named ${name}`),
  automation: (ctx, name) => lookup(ctx, 'automations', 'items', (a) => a.name === name, `no automation named ${name}`),
  contact: (ctx, email) => lookup(ctx, `contacts?q=${encodeURIComponent(email)}`, 'items', (c) => c.email === email, `no contact ${email}`),
  import: (ctx, which) =>
    lookup(ctx, 'imports', 'items', (i) => (which === 'draft' ? i.status === 'draft' : i.source === which && i.status === 'done'), `no ${which} import`),
  // A signup form by its name ({form:Newsletter signup}).
  form: (ctx, name) => lookup(ctx, 'forms', 'forms', (f) => f.name === name, `no form named ${name}`),
  // A recurring digest by its name ({digest:Engineering blog roundup}), and its template's campaign
  // ({digestTemplate:Engineering blog roundup}): the template isn't in the campaigns list.
  digest: (ctx, name) => lookup(ctx, 'digests', 'items', (d) => d.name === name, `no digest named ${name}`),
  async digestTemplate(ctx, name) {
    const id = await lookup(ctx, 'digests', 'items', (d) => d.name === name, `no digest named ${name}`)
    const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/digests/${id}`)
    const body = (await r.json()) as { templateId?: string }
    if (!body.templateId) throw new Error(`the digest ${name} has no template`)
    return body.templateId
  },
  // A survey by its name ({survey:Beta feedback · October 2026}).
  survey: (ctx, name) => lookup(ctx, 'surveys', 'items', (s) => s.name === name, `no survey named ${name}`),
  // A placement test by its name ({placement:Weekly Digest #142 · SES vs Postal}) and a
  // verification job by its label ({verification:Sunset}).
  placement: (ctx, name) => lookup(ctx, 'placement/tests', 'tests', (t) => t.name === name, `no placement test named ${name}`),
  verification: (ctx, label) => lookup(ctx, 'verification/services', 'uses', (u) => u.label === label && u.kind !== 'realtime', `no verification job ${label}`),
  // An offer by its name ({offer:FALL26}) and a sponsor booking by its issue ({booking:141}).
  offer: (ctx, name) => lookup(ctx, 'earn/offers', 'items', (o) => o.name === name, `no offer named ${name}`),
  async booking(ctx, issue) {
    const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/sponsors/overview`, { params: { range: 'all' } })
    type Row = { number: number; booking?: { id: string } }
    const body = (await r.json()) as { past?: Row[]; upcoming?: Row[] }
    const hit = [...(body.past ?? []), ...(body.upcoming ?? [])].find((i) => String(i.number) === issue && i.booking)
    if (!hit?.booking) throw new Error(`no booking of issue ${issue}`)
    return hit.booking.id
  },
}

/** The id of the first item of a workspace list that matches. */
async function lookup(ctx: BrowserContext, list: string, key: string, match: (item: Record<string, unknown>) => boolean, missing: string) {
  const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/${list}`)
  const body = (await r.json()) as Record<string, Record<string, unknown>[] | undefined>
  const hit = body[key]?.find(match)
  if (!hit) throw new Error(missing)
  return String(hit.id)
}

async function resolvePath(ctx: BrowserContext, path: string) {
  let outPath = path
  for (const m of path.matchAll(/\{(\w+):([^}]+)\}/g)) {
    const fn = resolvers[m[1]!]
    if (!fn) throw new Error(`no resolver for {${m[1]}:…}`)
    outPath = outPath.replace(m[0], encodeURIComponent(await fn(ctx, m[2]!)))
  }
  return outPath
}

async function shoot(browser: Browser, url: string, width: number, file: string, e: Entry, ctx?: BrowserContext, wait = 400) {
  const own = !ctx
  const c = ctx ?? (await browser.newContext({ deviceScaleFactor: 1, reducedMotion: 'reduce' }))
  const page = await c.newPage()
  try {
    await page.setViewportSize({ width, height: e.height ?? 1000 })
    await page.goto(url, { waitUntil: 'networkidle' })
    await page.evaluate('document.fonts.ready')
    await page.waitForTimeout(wait)
    await page.screenshot({ path: file, fullPage: !e.viewportOnly })
  } finally {
    await page.close()
    if (own) await c.close()
  }
}

async function main() {
  mkdirSync(`${out}/img`, { recursive: true })
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  const contexts = new Map<string, BrowserContext>()
  const signInErrors = new Map<string, string>()
  const results: Result[] = []
  try {
    for (const e of entries) {
      if (only.size && !only.has(e.board)) continue
      const widths = widthsArg ? widthsArg.split(',').map(Number) : (e.widths ?? [1440, 834, 390])
      let ctx: BrowserContext | undefined
      let path: string | undefined
      let pathError: string | undefined
      if (e.path && !e.manual) {
        const user = e.user ?? 'linh@acme.io'
        ctx = contexts.get(user)
        if (!ctx) {
          ctx = await browser.newContext({ deviceScaleFactor: 1, reducedMotion: 'reduce' })
          try {
            if (user !== 'anon') await signIn(ctx, app, user)
          } catch (err) {
            // Demo people who sign in only through SSO can't be signed in here:
            // their boards are captured design-only, with the reason.
            signInErrors.set(user, String(err).split('\n')[0]!)
          }
          contexts.set(user, ctx)
        }
        try {
          if (signInErrors.has(user)) throw new Error(signInErrors.get(user))
          path = await resolvePath(ctx, e.path)
        } catch (err) {
          pathError = String(err)
        }
      }
      for (const w of widths) {
        const base = `${e.board}-${w}`
        const design = `${out}/img/design-${base}.png`
        const r: Result = { board: e.board, width: w, path, design: `img/design-${base}.png` }
        try {
          await shoot(browser, `${designURL}/${e.board}.dc.html`, w, design, e, undefined, 1200)
          if (pathError) throw new Error(pathError)
          if (path && ctx) {
            const shot = `${out}/img/app-${base}.png`
            await shoot(browser, `${app}${path}`, w, shot, e, ctx)
            r.app = `img/app-${base}.png`
            r.score = score(design, shot)
            const side = `${out}/img/side-${base}.png`
            magick(design, shot, '-background', '#e5e7eb', '-splice', '16x0', '-gravity', 'north', '+append', '-resize', '50%', side)
            r.side = `img/side-${base}.png`
          }
        } catch (err) {
          r.error = String(err).split('\n')[0]
        }
        results.push(r)
        const label = r.error ? `error: ${r.error}` : r.score === undefined ? 'design only' : `${r.score.toFixed(2)}% differs`
        console.log(`${e.board} @${w}: ${label}`)
      }
    }
  } finally {
    for (const c of contexts.values()) await c.close()
    await browser.close()
  }
  results.sort((a, b) => (b.error ? 1e9 : (b.score ?? -1)) - (a.error ? 1e9 : (a.score ?? -1)))
  const rows = results
    .map((r) => {
      const status = r.error ? `<b class=err>${escape(r.error)}</b>` : r.score === undefined ? 'design only' : `${r.score.toFixed(2)}%`
      const img = r.side ? `<a href="${r.side}"><img loading=lazy src="${r.side}" alt=""></a>` : `<a href="${r.design}"><img loading=lazy src="${r.design}" alt=""></a>`
      return `<tr><td><b>${r.board}</b><br>${r.width}px<br><code>${escape(r.path ?? '')}</code></td><td>${status}</td><td>${img}</td></tr>`
    })
    .join('\n')
  writeFileSync(
    `${out}/index.html`,
    `<!doctype html><meta charset=utf-8><title>ReputeCast visual QA</title>
<style>body{font:14px system-ui;margin:24px}table{border-collapse:collapse}td{border-top:1px solid #ddd;padding:8px;vertical-align:top}img{max-width:1100px;border:1px solid #ddd}.err{color:#b91c1c}code{font-size:12px}</style>
<h1>Visual QA · ${new Date().toISOString()}</h1><p>Left: design · right: app. Score: share of ink (text, icons, lines) in one image with nothing within 2 px in the other, over the top 1600 px; 0 = same layout.</p>
<table>${rows}</table>`,
  )
  console.log(`${out}/index.html`)
}

function escape(s: string) {
  return s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]!)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
