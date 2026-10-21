/**
 * Renders the demo's product covers (internal/demo/assets/products): the
 * covers the PublicSalesPage and PublicPagesStates boards draw, as 3:4
 * PNGs the demo uploads to the image library. Run it only when a cover
 * changes:
 *
 *   node scripts/demo-covers.ts [file.png …]
 *
 * Uses the installed Google Chrome and the public pages' own IBM Plex Sans
 * files, so nothing is fetched.
 */
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { chromium } from '@playwright/test'

const out = resolve('../internal/demo/assets/products')
const fonts = pathToFileURL(resolve('../internal/publicpages/static/fonts')).href

const line = (top: number) =>
  `<span style="position:absolute;left:12%;top:${top}%;display:flex;align-items:center;gap:5px"><span style="width:11px;height:11px;border:1.5px solid #4338CA;border-radius:2px;box-sizing:border-box"></span><span style="width:78px;height:2px;background:#C9CED6;border-radius:1px"></span></span>`

// Each cover at the board's size; the scale makes it 600×800.
const covers: { file: string; w: number; scale: number; bg: string; border?: string; html: string }[] = [
  {
    file: 'field-guide.png',
    w: 300,
    scale: 2,
    bg: '#0B1220',
    html: `<span style="position:absolute;left:0;right:0;top:58%;height:14%;background:#4338CA"></span>
      <span style="position:absolute;left:0;right:0;top:72%;height:6%;background:#3730A3"></span>
      <span style="position:absolute;left:30px;right:30px;top:33px;display:flex;flex-direction:column;gap:10px">
        <span style="font-size:20px;letter-spacing:.12em;font-weight:700;color:#DDD6FE;text-transform:uppercase">Acme · field guide</span>
        <span style="font-size:32px;font-weight:700;line-height:1.08;color:#fff;letter-spacing:-.01em">The Deliverability Field Guide</span>
      </span>
      <span style="position:absolute;left:30px;bottom:21px;font-size:20px;font-weight:600;color:#DDD6FE">Sofia Rossi</span>`,
  },
  {
    file: 'launch-kit.png',
    w: 150,
    scale: 4,
    bg: '#EDE9FE',
    html: `<span style="position:absolute;left:10%;top:36%;width:34%;height:20%;border-radius:4px;background:#4338CA"></span>
      <span style="position:absolute;left:56%;top:36%;width:34%;height:20%;border-radius:4px;background:#fff"></span>
      <span style="position:absolute;left:10%;top:61%;width:34%;height:20%;border-radius:4px;background:#fff"></span>
      <span style="position:absolute;left:56%;top:61%;width:34%;height:20%;border-radius:4px;background:#3730A3"></span>
      <span style="position:absolute;left:10%;right:10%;top:8%;font-size:17px;font-weight:800;line-height:1.05;color:#3730A3;letter-spacing:-.01em">Newsletter Launch Kit</span>
      <span style="position:absolute;left:10%;bottom:7%;font-size:10px;font-weight:600;color:#3730A3">12 templates · checklist</span>`,
  },
  {
    file: 'warm-up-planner.png',
    w: 150,
    scale: 4,
    bg: '#FFFFFF',
    border: '#DDD6FE',
    html: `<span style="position:absolute;left:10%;right:10%;top:40%;bottom:8%;border:1px solid #DDD6FE;border-radius:3px"></span>
      ${[46, 56, 66, 76, 86].map((t) => `<span style="position:absolute;left:10%;right:10%;top:${t}%;height:1px;background:#DDD6FE"></span>`).join('')}
      <span style="position:absolute;top:40%;bottom:8%;left:34%;width:1px;background:#DDD6FE"></span>
      <span style="position:absolute;top:40%;bottom:8%;left:58%;width:1px;background:#DDD6FE"></span>
      <span style="position:absolute;left:10%;right:10%;top:9%;font-size:17px;font-weight:800;line-height:1.05;color:#3730A3">Warm-up planner</span>
      <span style="position:absolute;left:10%;top:28%;font-size:10px;font-weight:600;color:#5B6472">4 weeks · day by day</span>`,
  },
  {
    file: 'dns-checklist.png',
    w: 150,
    scale: 4,
    bg: '#FFFFFF',
    border: '#DDD6FE',
    html: `<span style="position:absolute;left:0;right:0;top:0;height:30%;background:#4338CA"></span>
      <span style="position:absolute;left:12%;right:10%;top:8%;font-size:17px;font-weight:800;line-height:1.05;color:#fff">DNS records checklist</span>
      ${[42, 53, 64, 75, 86].map(line).join('')}`,
  },
]

async function main() {
  const only = new Set(process.argv.slice(2))
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  for (const c of covers) {
    if (only.size > 0 && !only.has(c.file)) continue
    const page = await browser.newPage({ deviceScaleFactor: c.scale })
    const h = (c.w * 4) / 3
    await page.setContent(`<!doctype html><html><head><style>
      @font-face{font-family:'IBM Plex Sans';font-weight:600;src:url(${fonts}/ibm-plex-sans-latin-600-normal.woff2) format('woff2')}
      @font-face{font-family:'IBM Plex Sans';font-weight:700 800;src:url(${fonts}/ibm-plex-sans-latin-700-normal.woff2) format('woff2')}
      body{margin:0;background:transparent}
      .cov{position:relative;display:block;box-sizing:border-box;width:${c.w}px;height:${h}px;overflow:hidden;background:${c.bg};
        ${c.border ? `border:1px solid ${c.border};` : ''}font-family:'IBM Plex Sans',system-ui,sans-serif}
      </style></head><body><span class="cov">${c.html}</span></body></html>`)
    await page.evaluate('document.fonts.ready')
    await page.waitForTimeout(150)
    await page.locator('.cov').screenshot({ path: `${out}/${c.file}` })
    await page.close()
    console.log(`${out}/${c.file}`)
  }
  await browser.close()
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
