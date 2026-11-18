/**
 * The assistant panel's speed (assistant plan §9, B6), in Chrome against a
 * running, seeded server in development (the demo model answers):
 *
 * - open: from the press on the top bar's Assistant button to a usable
 *   composer (focused and enabled), the CPU four times slower, its chunk
 *   either prefetched by hovering the button or only in the HTTP cache (a
 *   reload since; on touch nothing hovers);
 * - delta to paint: from each streamed delta's arrival (the EventSource
 *   event) until its words are on screen and a frame has painted, over
 *   the demo's "#long" answer (360 distinct words in pieces of three);
 * - INP while streaming: taps and typing during that answer on a mid-range
 *   phone (390 px, touch, the CPU four times slower), from the page's
 *   Event Timing entries.
 *
 *   node scripts/assistant-perf.ts --app http://localhost:8080 [--runs 10]
 *
 * Measure the production build the binary serves (`pnpm build`, then
 * `go build ./cmd/reputecast`), not Vite's dev server.
 */
import { chromium, type BrowserContext, type CDPSession, type Page } from '@playwright/test'
import { signIn } from './lib/demo-auth.ts'

const args = process.argv.slice(2)
function opt(name: string, def: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? (args[i + 1] ?? def) : def
}
const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:8080')
const runs = Number(opt('runs', '10'))
// The phone's device pixel ratio (Pixel 7: 2.625); --only open|stream runs one part.
const dpr = Number(opt('dpr', '3'))
const only = opt('only', '')
const home = `${app}/w/acme-newsletter`

// Before the page's own scripts: each delta's arrival, every interaction's
// duration, and a frame loop that notes when a delta's last word shows.
const observe = `(() => {
  const perf = (window.__asstPerf = { deltas: [], events: [] })
  const ES = window.EventSource
  window.EventSource = class extends ES {
    constructor(...a) {
      super(...a)
      this.addEventListener('delta', (ev) => {
        let text = ''
        try { text = JSON.parse(ev.data).text } catch {}
        const words = text.match(/w\\d{3}/g)
        perf.deltas.push({ at: performance.now(), word: words ? words[words.length - 1] : null, shown: 0 })
      })
    }
  }
  new PerformanceObserver((l) => { for (const e of l.getEntries()) if (e.interactionId) perf.events.push([e.interactionId, e.duration, e.name + ' ' + (e.target ? (e.target.getAttribute('aria-label') || e.target.className || e.target.tagName).toString().slice(0, 40) : ''), e.startTime]) })
    .observe({ type: 'event', durationThreshold: 16, buffered: true })
  const after = new MessageChannel()
  let waiting = []
  after.port1.onmessage = () => { const t = performance.now(); for (const d of waiting) d.shown = t; waiting = [] }
  const frame = () => {
    const open = perf.deltas.filter((d) => d.word && !d.shown && !waiting.includes(d))
    if (open.length) {
      const answer = [...document.querySelectorAll('#assistant article.asst-msg.ai')].pop()
      const text = answer ? answer.textContent : ''
      for (const d of open) if (text.includes(d.word)) waiting.push(d)
      // Once this frame has painted.
      if (waiting.length) after.port2.postMessage(0)
    }
    requestAnimationFrame(frame)
  }
  requestAnimationFrame(frame)
})()`

// Cheap checks, twice a second: Playwright's text locators scan the whole
// page on every frame, which would be measured with the panel.
// (In-page code is a string: the scripts compile without the DOM's types.)
const lastAnswerHas = (text: string) => `!!document.querySelector('#assistant article.asst-msg.ai:last-of-type')?.textContent?.includes(${JSON.stringify(text)})`
// A string predicate polled in the page needs eval, which the app's CSP refuses: poll from here.
async function until(page: Page, expr: string, timeout: number) {
  for (const end = Date.now() + timeout; Date.now() < end; await page.waitForTimeout(500)) if (await page.evaluate(expr)) return
  throw new Error(`timed out waiting for ${expr}`)
}
const done = (page: Page) => page.evaluate(lastAnswerHas('End of the long answer.')) as Promise<boolean>
const streamed = (page: Page) => until(page, lastAnswerHas('End of the long answer.'), 120_000)

async function throttle(page: Page, ctx: BrowserContext, rate: number): Promise<CDPSession> {
  const cdp = await ctx.newCDPSession(page)
  await cdp.send('Emulation.setCPUThrottlingRate', { rate })
  return cdp
}

const pct = (xs: number[], p: number) => {
  const s = [...xs].sort((a, b) => a - b)
  return s.length ? s[Math.min(s.length - 1, Math.ceil((p / 100) * s.length) - 1)]! : NaN
}
const ms = (n: number) => `${Math.round(n)} ms`

/** From the press on the button to a focused, enabled composer. */
async function openOnce(page: Page, ctx: BrowserContext, prefetch: boolean, touch: boolean): Promise<number> {
  await page.goto(home)
  await page.locator('#root h1').first().waitFor({ timeout: 60_000 })
  await page.waitForLoadState('networkidle')
  const button = page.getByRole('button', { name: /^Assistant/ })
  if (prefetch) {
    await button.hover()
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(500)
  }
  const cdp = await throttle(page, ctx, 4)
  // (Not the promise itself: evaluate would wait for it, before the press.)
  await page.evaluate(`void (window.__open = new Promise((done) => {
    let start = 0
    document.addEventListener('pointerdown', () => (start = performance.now()), { capture: true, once: true })
    document.addEventListener('focusin', function on(e) {
      const el = e.target
      if (el instanceof HTMLTextAreaElement && el.closest('#assistant') && !el.disabled) {
        document.removeEventListener('focusin', on)
        done(performance.now() - start)
      }
    })
  }))`)
  if (touch) await button.tap()
  else await button.click()
  const took = (await page.evaluate('window.__open')) as number
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 })
  await page.keyboard.press('Escape')
  return took
}

async function openTimes(label: string, touch: boolean, viewport: { width: number; height: number }) {
  const browser = await chromium.launch({ executablePath: process.env.PW_EXECUTABLE, headless: true })
  const ctx = await browser.newContext({ viewport, deviceScaleFactor: touch ? dpr : 1, isMobile: touch, hasTouch: touch })
  await signIn(ctx, app, 'linh@acme.io')
  const page = await ctx.newPage()
  await openOnce(page, ctx, true, touch) // the HTTP cache has the chunk from here on
  for (const prefetch of touch ? [false] : [true, false]) {
    const all: number[] = []
    for (let i = 0; i < runs; i++) all.push(await openOnce(page, ctx, prefetch, touch))
    console.log(`open, ${label}, ${prefetch ? 'chunk prefetched by hover' : 'chunk in the HTTP cache only'}: median ${ms(pct(all, 50))}, p95 ${ms(pct(all, 95))}, max ${ms(Math.max(...all))} (${runs} runs, CPU 4×)`)
  }
  await browser.close()
}

/** A "#long" answer streams; on a phone, taps and typing go on meanwhile. */
async function stream(label: string, phone: boolean) {
  const browser = await chromium.launch({ executablePath: process.env.PW_EXECUTABLE, headless: true })
  const ctx = await browser.newContext(
    phone ? { viewport: { width: 390, height: 844 }, deviceScaleFactor: dpr, isMobile: true, hasTouch: true } : { viewport: { width: 1440, height: 900 } },
  )
  await signIn(ctx, app, 'linh@acme.io')
  await ctx.addInitScript({ content: observe })
  const page = await ctx.newPage()
  await page.goto(home)
  await page.locator('#root h1').first().waitFor({ timeout: 60_000 })
  await page.evaluate(`localStorage.removeItem('rc.assistant.thread.acme-newsletter')`)
  await page.reload()
  await page.locator('#root h1').first().waitFor({ timeout: 60_000 })
  await page.waitForLoadState('networkidle')
  if (phone) await page.getByRole('button', { name: /^Assistant/ }).tap()
  else await page.getByRole('button', { name: /^Assistant/ }).click()
  const composer = page.getByRole('textbox', { name: 'Message the assistant' })
  await composer.fill('Time the panel #long')
  const cdp = await throttle(page, ctx, phone ? 4 : 1)
  await composer.press('Enter')
  await until(page, lastAnswerHas('w001'), 60_000)
  if (phone) {
    // Interactions while it streams: typing, the tool row, the model menu.
    for (let i = 0; i < 6 && !(await done(page)); i++) {
      await composer.tap()
      await page.keyboard.type('what about', { delay: 60 })
      await page.locator('#assistant .asst-tool').first().tap()
      await page.waitForTimeout(250)
      await page.getByRole('button', { name: /^Model:/ }).tap()
      await page.waitForTimeout(250)
      await page.keyboard.press('Escape')
      await page.waitForTimeout(250)
    }
  }
  await streamed(page)
  await page.waitForTimeout(500)
  const perf = (await page.evaluate('window.__asstPerf')) as {
    deltas: { at: number; word: string | null; shown: number }[]
    events: [number, number, string, number][]
  }
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 })
  const lags = perf.deltas.filter((d) => d.word && d.shown).map((d) => d.shown - d.at)
  const missed = perf.deltas.filter((d) => d.word && !d.shown).length
  console.log(`delta to paint, ${label}: p50 ${ms(pct(lags, 50))}, p95 ${ms(pct(lags, 95))}, max ${ms(Math.max(...lags))} over ${lags.length} deltas${missed ? `, ${missed} never seen` : ''}`)
  if (phone) {
    // Only what happened while the answer streamed.
    const from = perf.deltas[0]?.at ?? 0
    const during = perf.events.filter((e) => e[3] >= from)
    const byId = new Map<number, number>()
    for (const [id, d] of during) byId.set(id, Math.max(byId.get(id) ?? 0, d))
    const durations = [...byId.values()]
    const inp = durations.length >= 50 ? pct(durations, 98) : Math.max(0, ...durations)
    const worst = [...during].sort((a, b) => b[1] - a[1]).slice(0, 3)
    console.log(`INP while streaming, ${label}: ${ms(inp)} over ${durations.length} interactions (worst: ${worst.map((w) => `${w[2]} ${ms(w[1])}`).join('; ') || 'none'})`)
  }
  await browser.close()
}

if (only !== 'stream') {
  await openTimes('desktop 1440 px (docked)', false, { width: 1440, height: 900 })
  await openTimes(`phone 390 px at ${dpr}×, touch (full screen)`, true, { width: 390, height: 844 })
}
if (only !== 'open') {
  await stream('desktop 1440 px, CPU 1×', false)
  await stream(`phone 390 px at ${dpr}×, touch, CPU 4×`, true)
}
