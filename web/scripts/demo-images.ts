/**
 * Renders the placeholder images (the grey, labelled boxes of the composer
 * boards) used by the demo campaigns (internal/demo/assets/composer) and by
 * the built-in starter templates (internal/templates/starters). Run it only
 * when a placeholder changes:
 *
 *   node scripts/demo-images.ts
 *
 * The PNGs are committed and embedded in the binary; they reach a
 * workspace's image library like any uploaded image.
 */
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium } from '@playwright/test'

type Img = { file: string; w: number; h: number; label: string }

const sets: { out: string; images: Img[] }[] = [
  {
    out: resolve('../internal/demo/assets/composer'),
    images: [
      { file: 'acme-logo.png', w: 96, h: 28, label: 'Logo · 96×28' },
      { file: 'q4-roadmap.png', w: 552, h: 180, label: 'Hero image · 552×180 · alt "Q4 roadmap illustration"' },
      { file: 'article-card.png', w: 268, h: 120, label: 'Image · 268×120' },
      { file: 'sale-hero.png', w: 552, h: 220, label: 'sale-hero.png · 552×220 · 212 KB' },
      { file: 'plan-card.png', w: 268, h: 110, label: 'Image · 268×110' },
      { file: 'october-hero.png', w: 552, h: 200, label: 'Hero image · 552×200 · alt "October at Acme"' },
    ],
  },
  {
    out: resolve('../internal/templates/starters'),
    images: [
      { file: 'logo.png', w: 120, h: 32, label: 'Your logo' },
      { file: 'hero.png', w: 552, h: 220, label: 'Hero image · 552×220' },
      { file: 'card.png', w: 268, h: 140, label: 'Image · 268×140' },
      { file: 'product.png', w: 168, h: 120, label: 'Product · 168×120' },
      { file: 'banner.png', w: 600, h: 280, label: 'Banner image · 600×280' },
    ],
  },
]

async function main() {
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  const page = await browser.newPage({ deviceScaleFactor: 1 })
  for (const set of sets) {
    mkdirSync(set.out, { recursive: true })
    for (const img of set.images) {
      await page.setContent(`<!doctype html><html><head>
        <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@500&display=swap" rel="stylesheet">
        <style>body{margin:0;background:#fff}
        .ph{box-sizing:border-box;width:${img.w}px;height:${img.h}px;display:flex;align-items:center;justify-content:center;
        background:#E9ECF0;color:#5B6472;font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:12px;font-weight:500;
        border:1px dashed #C9CED6;white-space:nowrap;overflow:hidden}</style></head>
        <body><div class="ph">${img.label.replace(/&/g, '&amp;').replace(/</g, '&lt;')}</div></body></html>`)
      await page.evaluate('document.fonts.ready')
      await page.waitForTimeout(150)
      await page.locator('.ph').screenshot({ path: `${set.out}/${img.file}`, omitBackground: false })
      console.log(`${set.out}/${img.file}`)
    }
  }
  await browser.close()
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
