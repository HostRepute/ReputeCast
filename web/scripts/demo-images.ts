/**
 * Renders the placeholder images (the grey, labelled boxes of the composer
 * boards) used by the demo campaigns (internal/demo/assets/composer) and by
 * the built-in starter templates (internal/templates/starters). Run it only
 * when a placeholder changes:
 *
 *   node scripts/demo-images.ts [file.png …]
 *
 * Name files to render only those (the others stay byte for byte).
 *
 * The PNGs are committed and embedded in the binary; they reach a
 * workspace's image library like any uploaded image.
 */
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { chromium } from '@playwright/test'

/**
 * `transparent` renders a PNG with an alpha channel (logos, so the dark-mode
 * check sees a transparent logo). `font`, `size` and `border` (px) override
 * the label's IBM Plex Sans 12px and the 1px dashed border. `social` draws
 * a social icon instead: a dark circle with white initials.
 */
type Img = { file: string; w: number; h: number; label: string; transparent?: boolean; font?: string; size?: number; border?: number; social?: boolean }

const sets: { out: string; images: Img[] }[] = [
  {
    out: resolve('../internal/demo/assets/composer'),
    images: [
      { file: 'acme-logo.png', w: 96, h: 28, label: 'Logo · 96×28', transparent: true },
      { file: 'q4-roadmap.png', w: 552, h: 180, label: 'Hero image · 552×180 · alt "Q4 roadmap illustration"' },
      { file: 'article-card.png', w: 268, h: 120, label: 'Image · 268×120' },
      { file: 'sale-hero.png', w: 552, h: 220, label: 'sale-hero.png · 552×220 · 212 KB' },
      { file: 'plan-card.png', w: 268, h: 110, label: 'Image · 268×110' },
      // The October Product Update as the Report board's click map draws it
      // at 340px (× 1.765): the hero labelled in the email's font, blank cards.
      { file: 'october-logo.png', w: 81, h: 28, label: 'Logo', font: "Georgia, 'Times New Roman', serif", size: 19.4, border: 2 },
      { file: 'october-hero.png', w: 564, h: 127, label: 'Hero image', font: "Georgia, 'Times New Roman', serif", size: 19.4, border: 2 },
      { file: 'october-card.png', w: 275, h: 81, label: '', border: 2 },
      // Weekly Digest #143 (accessibility boards): the hero without alt
      // text and the social links' icons.
      { file: 'scheduled-rollouts.png', w: 552, h: 200, label: 'scheduled-rollouts.png · 552×200 · no alt text' },
      { file: 'social-x.png', w: 28, h: 28, label: 'X', transparent: true, social: true },
      { file: 'social-linkedin.png', w: 28, h: 28, label: 'in', transparent: true, social: true },
      { file: 'social-github.png', w: 28, h: 28, label: 'GH', transparent: true, social: true },
    ],
  },
  {
    out: resolve('../internal/templates/starters'),
    images: [
      { file: 'logo.png', w: 120, h: 32, label: 'Your logo', transparent: true },
      { file: 'hero.png', w: 552, h: 220, label: 'Hero image · 552×220' },
      { file: 'card.png', w: 268, h: 140, label: 'Image · 268×140' },
      { file: 'product.png', w: 168, h: 120, label: 'Product · 168×120' },
      { file: 'banner.png', w: 600, h: 280, label: 'Banner image · 600×280' },
    ],
  },
]

async function main() {
  const only = new Set(process.argv.slice(2))
  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  const page = await browser.newPage({ deviceScaleFactor: 1 })
  for (const set of sets) {
    mkdirSync(set.out, { recursive: true })
    for (const img of set.images) {
      if (only.size > 0 && !only.has(img.file)) continue
      const look = img.social
        ? `background:#374151;color:#fff;font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:11px;font-weight:700;border-radius:50%`
        : `background:${img.transparent ? 'rgba(233,236,240,.92)' : '#E9ECF0'};color:#5B6472;font-family:${img.font ?? "'IBM Plex Sans',system-ui,sans-serif"};font-size:${img.size ?? 12}px;font-weight:500;
        border:${img.border ?? 1}px dashed #C9CED6`
      await page.setContent(`<!doctype html><html><head>
        <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@500;700&display=swap" rel="stylesheet">
        <style>body{margin:0;background:${img.transparent ? 'transparent' : '#fff'}}
        .ph{box-sizing:border-box;width:${img.w}px;height:${img.h}px;display:flex;align-items:center;justify-content:center;
        ${look};white-space:nowrap;overflow:hidden}</style></head>
        <body><div class="ph">${img.label.replace(/&/g, '&amp;').replace(/</g, '&lt;')}</div></body></html>`)
      await page.evaluate('document.fonts.ready')
      await page.waitForTimeout(150)
      await page.locator('.ph').screenshot({ path: `${set.out}/${img.file}`, omitBackground: !!img.transparent })
      console.log(`${set.out}/${img.file}`)
    }
  }
  await browser.close()
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
