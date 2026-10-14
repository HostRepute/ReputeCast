/**
 * Generates the committed image assets from the brand files and the design:
 *
 *   public/favicon.svg, favicon.ico (16/32/48), apple-touch-icon.png,
 *   icon-192.png, icon-512.png, icon-maskable-512.png   from brand/logo
 *   public/og.png                 1200 x 630 Open Graph / Twitter card
 *   ../docs/assets/readme-banner.png   1280 x 640, the ReadmeBanner board
 *   ../docs/assets/readme-banner.svg   the same banner with every glyph
 *                                      outlined, so it renders identically
 *                                      without IBM Plex installed
 *
 *   node scripts/assets.ts
 *
 * The banner is rendered from scripts/assets/banner.html (a copy of the
 * board) in Chrome with the self-hosted IBM Plex files. For the SVG, Chrome
 * measures where every character sits, and opentype.js turns the same font
 * files into glyph outlines at those positions.
 */
import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { chromium, type Browser, type Page } from '@playwright/test'
import opentype, { type Font, type Path } from 'opentype.js'

const site = resolve(import.meta.dirname, '..')
const repo = resolve(site, '..')
const pub = join(site, 'public')
const docsAssets = join(repo, 'docs', 'assets')
const fontDir = (family: 'sans' | 'mono') => join(site, 'node_modules', '@fontsource', `ibm-plex-${family}`, 'files')
const WEIGHTS = [400, 500, 600, 700] as const

const BANNER_LABEL =
  'ReputeCast: the newsletter platform that protects your inbox placement. Open source and self-hosted; checks deliverability before you send and guards your reputation while you send.'

function fontFaces(): string {
  return WEIGHTS.map((w) => {
    const file = join(fontDir('sans'), `ibm-plex-sans-latin-${w}-normal.woff2`)
    const data = readFileSync(file).toString('base64')
    return `@font-face{font-family:'IBM Plex Sans';font-weight:${w};font-style:normal;src:url(data:font/woff2;base64,${data}) format('woff2')}`
  }).join('\n')
}

function bannerHtml(w: number, h: number, foot: string, padX = 64): string {
  return readFileSync(join(site, 'scripts', 'assets', 'banner.html'), 'utf8')
    .replaceAll('{{FONTS}}', fontFaces())
    .replaceAll('{{W}}', String(w))
    .replaceAll('{{H}}', String(h))
    .replaceAll('{{FOOT}}', foot)
    .replaceAll('{{PADX}}', String(padX))
}

async function load(page: Page, html: string) {
  await page.setContent(html, { waitUntil: 'load' })
  await page.evaluate(() => document.fonts.ready)
}

type Border = { w: number; color: string }
type Box = { kind: 'box'; x: number; y: number; w: number; h: number; bg: string; radius: number; borders: [Border, Border, Border, Border]; opacity: number; shadow: string }
type Svg = { kind: 'svg'; x: number; y: number; w: number; h: number; markup: string }
type Glyphs = { kind: 'text'; weight: number; size: number; color: string; chars: { ch: string; x: number; top: number }[] }
type Item = Box | Svg | Glyphs

/** Walks the rendered banner in paint order and reports boxes, the logo and every character's position. */
async function scene(page: Page): Promise<{ items: Item[]; ascent: number }> {
  return page.evaluate(() => {
    const root = document.getElementById('banner')!
    const origin = root.getBoundingClientRect()
    const items: Item[] = []
    const visible = (c: string) => c && c !== 'transparent' && !/rgba\(.*,\s*0\)$/.test(c)
    const walk = (el: Element) => {
      const r = el.getBoundingClientRect()
      const s = getComputedStyle(el)
      if (el instanceof SVGSVGElement) {
        items.push({ kind: 'svg', x: r.left - origin.left, y: r.top - origin.top, w: r.width, h: r.height, markup: el.innerHTML })
        return
      }
      const sides = (['Top', 'Right', 'Bottom', 'Left'] as const).map((side) => ({
        w: s.getPropertyValue(`border-${side.toLowerCase()}-style`) === 'none' ? 0 : parseFloat(s.getPropertyValue(`border-${side.toLowerCase()}-width`)),
        color: s.getPropertyValue(`border-${side.toLowerCase()}-color`),
      })) as [Border, Border, Border, Border]
      if (visible(s.backgroundColor) || sides.some((b) => b.w > 0) || s.boxShadow !== 'none') {
        items.push({
          kind: 'box',
          x: r.left - origin.left,
          y: r.top - origin.top,
          w: r.width,
          h: r.height,
          bg: s.backgroundColor,
          radius: Math.min(
            s.borderTopLeftRadius.endsWith('%') ? (parseFloat(s.borderTopLeftRadius) / 100) * Math.min(r.width, r.height) : parseFloat(s.borderTopLeftRadius) || 0,
            r.width / 2,
            r.height / 2,
          ),
          borders: sides,
          opacity: parseFloat(s.opacity),
          shadow: s.boxShadow,
        })
      }
      for (const node of el.childNodes) {
        if (node.nodeType === Node.TEXT_NODE) {
          const text = node.textContent ?? ''
          const chars: { ch: string; x: number; top: number }[] = []
          const range = document.createRange()
          for (let i = 0; i < text.length; i++) {
            const ch = text[i]!
            if (/\s/.test(ch)) continue
            range.setStart(node, i)
            range.setEnd(node, i + 1)
            const cr = range.getBoundingClientRect()
            chars.push({ ch, x: cr.left - origin.left, top: cr.top - origin.top })
          }
          if (chars.length) items.push({ kind: 'text', weight: Number(s.fontWeight), size: parseFloat(s.fontSize), color: s.color, chars })
        } else if (node instanceof Element) {
          walk(node)
        }
      }
    }
    walk(root)
    // Where the baseline sits inside a character box, as a share of the font
    // size: measured with a zero-size inline-block, which sits on the baseline.
    const probe = document.createElement('span')
    probe.style.cssText = "font:700 100px 'IBM Plex Sans';line-height:1;position:absolute;left:0;top:0"
    probe.textContent = 'H'
    const mark = document.createElement('span')
    mark.style.cssText = 'display:inline-block;width:0;height:0'
    probe.appendChild(mark)
    document.body.appendChild(probe)
    const range = document.createRange()
    range.selectNodeContents(probe.firstChild!)
    const ascent = (mark.getBoundingClientRect().bottom - range.getBoundingClientRect().top) / 100
    probe.remove()
    return { items, ascent }
  })
}

const rgba = (c: string) => {
  const m = /rgba?\(([^)]+)\)/.exec(c)
  if (!m) return { fill: c, alpha: 1 }
  const [r, g, b, a] = m[1]!.split(',').map((v) => parseFloat(v))
  const hex = '#' + [r, g, b].map((v) => Math.round(v ?? 0).toString(16).padStart(2, '0')).join('')
  return { fill: hex, alpha: a === undefined ? 1 : a }
}
const paint = (c: string, attr = 'fill') => {
  const { fill, alpha } = rgba(c)
  return alpha < 1 ? `${attr}="${fill}" ${attr}-opacity="${+alpha.toFixed(3)}"` : `${attr}="${fill}"`
}
const n = (v: number) => +v.toFixed(2)

/**
 * SVG path data for a glyph outline. opentype.js's own toPathData() can emit
 * "NaN" when a coordinate's fraction is tiny (it builds "1e-14e+2" as a
 * string), so the commands are serialized here.
 */
function pathData(path: Path): string {
  const f = (v: number | undefined) => String(n(v ?? 0))
  return path.commands
    .map((c) => {
      switch (c.type) {
        case 'M':
        case 'L':
          return `${c.type}${f(c.x)} ${f(c.y)}`
        case 'Q':
          return `Q${f(c.x1)} ${f(c.y1)} ${f(c.x)} ${f(c.y)}`
        case 'C':
          return `C${f(c.x1)} ${f(c.y1)} ${f(c.x2)} ${f(c.y2)} ${f(c.x)} ${f(c.y)}`
        default:
          return 'Z'
      }
    })
    .join('')
}

function bannerSvg(w: number, h: number, items: Item[], ascent: number, fonts: Map<number, Font>): string {
  const defs: string[] = []
  const body: string[] = []
  const glyphIds = new Map<string, string>()
  let filters = 0
  for (const it of items) {
    if (it.kind === 'svg') {
      body.push(`<svg x="${n(it.x)}" y="${n(it.y)}" width="${n(it.w)}" height="${n(it.h)}" viewBox="0 0 32 32">${it.markup.replace(/\s+aria-hidden="true"/g, '')}</svg>`)
      continue
    }
    if (it.kind === 'box') {
      const op = it.opacity < 1 ? ` opacity="${it.opacity}"` : ''
      let filter = ''
      const sh = /rgba?\([^)]+\)\s+(-?[\d.]+)px\s+(-?[\d.]+)px\s+([\d.]+)px/.exec(it.shadow)
      if (sh) {
        const id = `shadow${++filters}`
        const color = rgba(/rgba?\([^)]+\)/.exec(it.shadow)![0])
        defs.push(
          `<filter id="${id}" x="-50%" y="-50%" width="200%" height="200%"><feDropShadow dx="${sh[1]}" dy="${sh[2]}" stdDeviation="${+(Number(sh[3]) / 2).toFixed(2)}" flood-color="${color.fill}" flood-opacity="${color.alpha}"/></filter>`,
        )
        filter = ` filter="url(#${id})"`
      }
      const rx = it.radius ? ` rx="${n(it.radius)}"` : ''
      if (it.radius >= Math.min(it.w, it.h) / 2 - 0.01 && it.w === it.h) {
        body.push(`<circle cx="${n(it.x + it.w / 2)}" cy="${n(it.y + it.h / 2)}" r="${n(it.w / 2)}" ${paint(it.bg)}${op}/>`)
        continue
      }
      const [top, right, bottom, left] = it.borders
      const uniform = top.w > 0 && [right, bottom, left].every((b) => b.w === top.w && b.color === top.color)
      if (rgba(it.bg).alpha > 0) body.push(`<rect x="${n(it.x)}" y="${n(it.y)}" width="${n(it.w)}" height="${n(it.h)}"${rx} ${paint(it.bg)}${op}${filter}/>`)
      if (uniform) {
        const half = top.w / 2
        const srx = it.radius ? ` rx="${n(Math.max(0, it.radius - half))}"` : ''
        body.push(`<rect x="${n(it.x + half)}" y="${n(it.y + half)}" width="${n(it.w - top.w)}" height="${n(it.h - top.w)}"${srx} fill="none" ${paint(top.color, 'stroke')} stroke-width="${top.w}"${op}/>`)
      } else {
        if (top.w) body.push(`<rect x="${n(it.x)}" y="${n(it.y)}" width="${n(it.w)}" height="${top.w}" ${paint(top.color)}/>`)
        if (bottom.w) body.push(`<rect x="${n(it.x)}" y="${n(it.y + it.h - bottom.w)}" width="${n(it.w)}" height="${bottom.w}" ${paint(bottom.color)}/>`)
      }
      continue
    }
    const font = fonts.get(it.weight) ?? fonts.get(400)!
    const uses = it.chars.map((c) => {
      const key = `${it.weight}:${it.size}:${c.ch}`
      let id = glyphIds.get(key)
      if (!id) {
        id = `g${glyphIds.size}`
        glyphIds.set(key, id)
        defs.push(`<path id="${id}" d="${pathData(font.charToGlyph(c.ch).getPath(0, 0, it.size))}"/>`)
      }
      return `<use href="#${id}" x="${n(c.x)}" y="${n(c.top + ascent * it.size)}"/>`
    })
    body.push(`<g ${paint(it.color)}>${uses.join('')}</g>`)
  }
  return [
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" role="img" aria-label="${BANNER_LABEL}">`,
    `<title>${BANNER_LABEL}</title>`,
    `<defs>${defs.join('')}</defs>`,
    `<clipPath id="frame"><rect width="${w}" height="${h}"/></clipPath>`,
    `<g clip-path="url(#frame)">${body.join('')}</g>`,
    '</svg>',
    '',
  ].join('\n')
}

/** A Windows .ico holding PNG images (supported since Vista and by every browser). */
function ico(pngs: { size: number; data: Buffer }[]): Buffer {
  const header = Buffer.alloc(6 + 16 * pngs.length)
  header.writeUInt16LE(0, 0)
  header.writeUInt16LE(1, 2)
  header.writeUInt16LE(pngs.length, 4)
  let offset = header.length
  pngs.forEach((p, i) => {
    const e = 6 + 16 * i
    header.writeUInt8(p.size >= 256 ? 0 : p.size, e)
    header.writeUInt8(p.size >= 256 ? 0 : p.size, e + 1)
    header.writeUInt8(0, e + 2)
    header.writeUInt8(0, e + 3)
    header.writeUInt16LE(1, e + 4)
    header.writeUInt16LE(32, e + 6)
    header.writeUInt32LE(p.data.length, e + 8)
    header.writeUInt32LE(offset, e + 12)
    offset += p.data.length
  })
  return Buffer.concat([header, ...pngs.map((p) => p.data)])
}

/** Renders an SVG at size × size with a transparent background, in a fresh page. */
async function rasterize(browser: Browser, svg: string, size: number): Promise<Buffer> {
  const page = await browser.newPage({ viewport: { width: size, height: size }, deviceScaleFactor: 1 })
  try {
    const sized = svg.replace(/<svg([^>]*?)\swidth="[^"]*"/, '<svg$1').replace(/<svg([^>]*?)\sheight="[^"]*"/, '<svg$1').replace('<svg', `<svg width="${size}" height="${size}"`)
    await page.setContent(`<html><body style="margin:0;background:transparent">${sized}</body></html>`)
    return await page.screenshot({ omitBackground: true, clip: { x: 0, y: 0, width: size, height: size } })
  } finally {
    await page.close()
  }
}

async function main() {
  mkdirSync(pub, { recursive: true })
  mkdirSync(docsAssets, { recursive: true })
  const brand = (f: string) => readFileSync(join(repo, 'brand', 'logo', f), 'utf8')
  const favicon = brand('reputecast-favicon.svg')
  const appIcon = brand('reputecast-app-icon.svg')
  // iOS and Android masks round the corners themselves: full-bleed square.
  const fullBleed = brand('reputecast-mark.svg').replace(/rx="8"/, 'rx="0"')

  copyFileSync(join(repo, 'brand', 'logo', 'reputecast-favicon.svg'), join(pub, 'favicon.svg'))

  const browser = await chromium.launch({ channel: 'chrome', headless: true })
  try {
    const page = await browser.newPage({ deviceScaleFactor: 1 })
    const icons = await Promise.all([16, 32, 48].map(async (size) => ({ size, data: await rasterize(browser, favicon, size) })))
    writeFileSync(join(pub, 'favicon.ico'), ico(icons))
    writeFileSync(join(pub, 'apple-touch-icon.png'), await rasterize(browser, fullBleed, 180))
    writeFileSync(join(pub, 'icon-192.png'), await rasterize(browser, appIcon, 192))
    writeFileSync(join(pub, 'icon-512.png'), await rasterize(browser, appIcon, 512))
    writeFileSync(join(pub, 'icon-maskable-512.png'), await rasterize(browser, fullBleed, 512))

    await page.setViewportSize({ width: 1280, height: 640 })
    await load(page, bannerHtml(1280, 640, 'Built by HostRepute · github.com/hostrepute/reputecast'))
    await page.locator('#banner').screenshot({ path: join(docsAssets, 'readme-banner.png') })
    const { items, ascent } = await scene(page)
    const fonts = new Map<number, Font>()
    for (const w of WEIGHTS) {
      const buf = readFileSync(join(fontDir('sans'), `ibm-plex-sans-latin-${w}-normal.woff`))
      fonts.set(w, opentype.parse(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength) as ArrayBuffer))
    }
    writeFileSync(join(docsAssets, 'readme-banner.svg'), bannerSvg(1280, 640, items, ascent, fonts))

    await page.setViewportSize({ width: 1200, height: 630 })
    await load(page, bannerHtml(1200, 630, 'Built by HostRepute · reputecast.com', 48))
    await page.locator('#banner').screenshot({ path: join(pub, 'og.png') })
  } finally {
    await browser.close()
  }
  console.log('wrote public/ icons and og.png, docs/assets/readme-banner.{png,svg}')
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
