/**
 * Visual QA for reputecast.com: the built site next to the Landing board,
 * at each width, cut into slices you can open side by side.
 *
 *   pnpm build && pnpm preview &                     # serves dist on :5289
 *   node scripts/shot.ts                             # 1440, 834, 390, final states
 *   node scripts/shot.ts --widths 390 --slice 1400
 *   node scripts/shot.ts --motion --frames 300,900,1600,2600,4200 --clip 0,0,1440,820
 *   node scripts/shot.ts --motion --scroll 1700 --frames 1000,5000 --clip 570,0,710,1000
 *
 * Options
 *   --site <url>        the site (default http://localhost:5289)
 *   --design-url <url>  design board server (default http://localhost:8765)
 *   --board <name>      board to compare with (default Landing)
 *   --widths <list>     viewport widths (default 1440,834,390)
 *   --slice <px>        slice height for full-page captures (default 1000)
 *   --out <dir>         output directory (default ../tmp/shots/site)
 *   --motion            allow motion (captures use reduced motion otherwise,
 *                       which renders every animation's final state)
 *   --frames <ms list>  with --motion: viewport frames at these times after load
 *   --scroll <y>        scroll before capturing frames
 *   --clip x,y,w,h      capture only this viewport region for frames
 *   --pause-at <ms>     press "Pause motion" at this time (frames mode)
 *   --site-only / --board-only
 *
 * Uses the installed Google Chrome (Playwright's chrome channel).
 */
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium, type Page } from '@playwright/test'

const args = process.argv.slice(2)
const opt = (name: string, def?: string) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}
const flag = (name: string) => args.includes(`--${name}`)

const site = opt('site', 'http://localhost:5289')!
const designUrl = opt('design-url', 'http://localhost:8765')!
const board = opt('board', 'Landing')!
const widths = opt('widths', '1440,834,390')!.split(',').map(Number).filter(Boolean)
const sliceH = Number(opt('slice', '1000'))
const out = resolve(opt('out', '../tmp/shots/site')!)
const motion = flag('motion')
const frames = (opt('frames') ?? '').split(',').filter(Boolean).map(Number)
const scrollY = opt('scroll') ? Number(opt('scroll')) : undefined
const clip = opt('clip')?.split(',').map(Number)
const pauseAt = opt('pause-at') ? Number(opt('pause-at')) : undefined

const targets = [
  ...(flag('board-only') ? [] : [{ name: 'site', url: `${site}/` }]),
  ...(flag('site-only') ? [] : [{ name: 'board', url: `${designUrl}/${board}.dc.html` }]),
]

async function fullPage(page: Page, prefix: string) {
  const height = await page.evaluate(() => document.documentElement.scrollHeight)
  const files: string[] = []
  for (let y = 0, i = 0; y < height; y += sliceH, i++) {
    const file = `${prefix}-${String(i).padStart(2, '0')}.png`
    await page.screenshot({ path: file, fullPage: true, clip: { x: 0, y, width: page.viewportSize()!.width, height: Math.min(sliceH, height - y) } })
    files.push(file)
  }
  return { height, files }
}

async function timed(page: Page, prefix: string, t0: number) {
  const files: string[] = []
  let paused = false
  for (const t of frames) {
    if (pauseAt !== undefined && !paused && t >= pauseAt) {
      await page.waitForTimeout(Math.max(0, pauseAt - (Date.now() - t0)))
      await page.getByRole('button', { name: /pause motion/i }).click()
      paused = true
    }
    await page.waitForTimeout(Math.max(0, t - (Date.now() - t0)))
    if (scrollY !== undefined) await page.evaluate((y) => window.scrollTo(0, y), scrollY)
    const file = `${prefix}-t${String(t).padStart(5, '0')}.png`
    await page.screenshot({ path: file, clip: clip ? { x: clip[0]!, y: clip[1]!, width: clip[2]!, height: clip[3]! } : undefined })
    files.push(file)
  }
  return files
}

async function main() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  try {
    for (const w of widths) {
      for (const target of targets) {
        const ctx = await browser.newContext({ viewport: { width: w, height: 1000 }, deviceScaleFactor: 1, reducedMotion: motion ? 'no-preference' : 'reduce' })
        const page = await ctx.newPage()
        const t0 = Date.now()
        await page.goto(target.url, { waitUntil: frames.length ? 'load' : 'networkidle' })
        await page.evaluate('document.fonts.ready')
        const prefix = `${out}/${target.name}-${w}`
        if (frames.length) {
          for (const f of await timed(page, prefix, t0)) console.log(f)
        } else {
          await page.waitForTimeout(1200)
          const { height, files } = await fullPage(page, prefix)
          console.log(`${target.name} ${w}px: ${height}px tall`)
          for (const f of files) console.log(`  ${f}`)
        }
        await ctx.close()
      }
    }
  } finally {
    await browser.close()
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
