/**
 * A screen on a slow phone (docs/deploy/performance.md, "The web app on a
 * slow phone"): Chrome at 390 px with its "Slow 4G" preset and the CPU four
 * times slower, signed in to the demo workspace as a returning member, the
 * cache off. Prints the median of --runs loads per path: requests, first
 * paint, when the screen's heading shows, largest paint, total blocking time
 * and layout shift.
 *
 *   node scripts/phone-perf.ts --app http://localhost:8080
 *   node scripts/phone-perf.ts --app https://news.example.test /w/acme-newsletter/audience
 *
 * Measure the production build the binary serves, through a TLS proxy for
 * HTTP/2 as a deployment does (a self-signed certificate is accepted). A
 * load ends once the heading has shown and the network has been quiet:
 * waiting for the network alone can stop while the slow CPU is still
 * running scripts, before the screen has drawn anything.
 */
import { chromium, type Page } from '@playwright/test'
import { signIn } from './lib/demo-auth.ts'

const args = process.argv.slice(2)
function opt(name: string, def: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? (args[i + 1] ?? def) : def
}
const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:8080')
const runs = Number(opt('runs', '3'))
const flags = new Set(['--app', '--runs'])
const paths = args.filter((a, i) => !a.startsWith('--') && !flags.has(args[i - 1] ?? ''))
if (paths.length === 0) paths.push('/w/acme-newsletter', '/w/acme-newsletter/campaigns', '/w/acme-newsletter/audience')

// Chrome DevTools' "Slow 4G" preset.
const network = { offline: false, latency: 562.5, downloadThroughput: (1.6 * 1024 * 1024 * 0.9) / 8, uploadThroughput: (750 * 1024 * 0.9) / 8 }

interface Load {
  requests: number
  fcp: number
  heading: number
  lcp: number
  tbt: number
  cls: number
}

// Runs in the page before its own scripts: what it painted, shifted and
// blocked on, from its own PerformanceObservers.
const observe = `(() => {
  const seen = (window.__phonePerf = { lcp: 0, cls: 0, long: [] })
  new PerformanceObserver((l) => { for (const e of l.getEntries()) seen.lcp = e.startTime }).observe({ type: 'largest-contentful-paint', buffered: true })
  new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) seen.cls += e.value }).observe({ type: 'layout-shift', buffered: true })
  new PerformanceObserver((l) => { for (const e of l.getEntries()) seen.long.push([e.startTime, e.duration]) }).observe({ type: 'longtask', buffered: true })
})()`

// Long tasks after the first paint count past their first 50 ms.
const collect = `(() => {
  const seen = window.__phonePerf
  const fcp = performance.getEntriesByName('first-contentful-paint')[0]?.startTime ?? 0
  const tbt = seen.long.filter(([start]) => start >= fcp).reduce((sum, [, d]) => sum + Math.max(0, d - 50), 0)
  return { fcp, lcp: seen.lcp, cls: seen.cls, tbt }
})()`

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 3, isMobile: true, hasTouch: true, ignoreHTTPSErrors: true })
await signIn(ctx, app, 'linh@acme.io')
await ctx.addInitScript({ content: observe })

// A returning member: the app has applied their language once, which tells
// it to start a screen's code while the session is checked.
const warm = await ctx.newPage()
await warm.goto(`${app}/w/acme-newsletter`)
await warm.locator('#root h1').first().waitFor()
await warm.close()

async function load(path: string): Promise<Load> {
  const page: Page = await ctx.newPage()
  const cdp = await ctx.newCDPSession(page)
  await cdp.send('Network.enable')
  await cdp.send('Network.setCacheDisabled', { cacheDisabled: true })
  await cdp.send('Network.emulateNetworkConditions', network)
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 })
  let requests = 0
  page.on('request', () => requests++)
  await page.goto(app + path, { waitUntil: 'commit', timeout: 120_000 })
  await page.locator('#root h1').first().waitFor({ timeout: 120_000 })
  const heading = (await page.evaluate('performance.now()')) as number
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(1500)
  await page.waitForLoadState('networkidle')
  const m = (await page.evaluate(collect)) as Omit<Load, 'requests' | 'heading'>
  await page.close()
  return { requests, heading, ...m }
}

const s = (ms: number) => `${(ms / 1000).toFixed(2)} s`
for (const path of paths) {
  const all: Load[] = []
  for (let i = 0; i < runs; i++) all.push(await load(path))
  const med = (k: keyof Load) => all.map((l) => l[k]).sort((a, b) => a - b)[Math.floor(all.length / 2)] ?? 0
  console.log(
    `${path}: ${med('requests')} requests, first paint ${s(med('fcp'))}, heading ${s(med('heading'))}, largest paint ${s(med('lcp'))}, blocking ${Math.round(med('tbt'))} ms, layout shift ${med('cls').toFixed(3)}`,
  )
}
await browser.close()
