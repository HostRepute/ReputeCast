/**
 * Visual QA: screenshots of app screens next to the matching design boards.
 *
 *   node scripts/shot.ts --path /w/acme-newsletter/campaigns --design Campaigns
 *   node scripts/shot.ts --path /w/acme-newsletter/campaigns --widths 1440,834,390 --out ../tmp/shots
 *   node scripts/shot.ts --design SettingsTeam            # design only
 *   node scripts/shot.ts --path /sign-in --anon            # no sign-in
 *
 * Options
 *   --app <url>        web app origin (default http://localhost:5173, or $SHOT_APP)
 *   --design-url <url> design board server (default http://localhost:8765, or $SHOT_DESIGN)
 *   --path <path>      app path to capture (repeatable)
 *   --design <Board>   design board name without .dc.html (repeatable)
 *   --widths <list>    viewport widths (default 1440,834,390)
 *   --height <px>      viewport height (default 1000)
 *   --out <dir>        output directory (default ../tmp/shots)
 *   --user <email>     demo user to sign in as (default linh@acme.io)
 *   --anon             don't sign in
 *   --viewport-only    capture the viewport instead of the full page
 *   --wait <ms>        extra wait after load (default 400)
 *
 * Signs in through the API with the demo password and the demo TOTP secret
 * (see internal/demo). Uses the installed Google Chrome, so no browser
 * download is needed.
 */
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium, type BrowserContext } from '@playwright/test'
import { signIn as demoSignIn } from './lib/demo-auth.ts'

const args = process.argv.slice(2)
function opt(name: string, def?: string) {
  const i = args.indexOf(`--${name}`)
  return i >= 0 ? args[i + 1] : def
}
function many(name: string) {
  const out: string[] = []
  args.forEach((a, i) => {
    if (a === `--${name}` && args[i + 1]) out.push(args[i + 1]!)
  })
  return out
}
const flag = (name: string) => args.includes(`--${name}`)

const app = opt('app', process.env.SHOT_APP ?? 'http://localhost:5173')!
const designURL = opt('design-url', process.env.SHOT_DESIGN ?? 'http://localhost:8765')!
const paths = many('path')
const designs = many('design')
const widths = (opt('widths', '1440,834,390') ?? '').split(',').map(Number).filter(Boolean)
const height = Number(opt('height', '1000'))
const out = resolve(opt('out', '../tmp/shots')!)
const user = opt('user', 'linh@acme.io')!
const extraWait = Number(opt('wait', '400'))
const fullPage = !flag('viewport-only')

// The shared sign-in waits for the next code when this half-minute's was
// already used: the server accepts each authenticator code once.
async function signIn(ctx: BrowserContext) {
  await demoSignIn(ctx, app, user)
}

function slug(s: string) {
  return s.replace(/^\/+/, '').replace(/[^a-zA-Z0-9]+/g, '-').replace(/-+$/, '') || 'root'
}

async function main() {
  if (!paths.length && !designs.length) {
    console.error('Nothing to capture: pass --path and/or --design')
    process.exit(2)
  }
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  const files: string[] = []
  try {
    if (paths.length) {
      const ctx = await browser.newContext({ viewport: { width: widths[0] ?? 1440, height }, deviceScaleFactor: 1, reducedMotion: 'reduce' })
      if (!flag('anon')) await signIn(ctx)
      const page = await ctx.newPage()
      for (const p of paths) {
        for (const w of widths) {
          await page.setViewportSize({ width: w, height })
          await page.goto(`${app}${p}`, { waitUntil: 'networkidle' })
          await page.evaluate("document.fonts.ready")
          await page.waitForTimeout(extraWait)
          const file = `${out}/app-${slug(p)}-${w}.png`
          await page.screenshot({ path: file, fullPage })
          files.push(file)
        }
      }
      await ctx.close()
    }
    if (designs.length) {
      const ctx = await browser.newContext({ viewport: { width: widths[0] ?? 1440, height }, deviceScaleFactor: 1, reducedMotion: 'reduce' })
      const page = await ctx.newPage()
      for (const d of designs) {
        for (const w of widths) {
          await page.setViewportSize({ width: w, height })
          await page.goto(`${designURL}/${d}.dc.html`, { waitUntil: 'networkidle' })
          await page.evaluate("document.fonts.ready")
          await page.waitForTimeout(1200)
          const file = `${out}/design-${d}-${w}.png`
          await page.screenshot({ path: file, fullPage })
          files.push(file)
        }
      }
      await ctx.close()
    }
  } finally {
    await browser.close()
  }
  for (const f of files) console.log(f)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
