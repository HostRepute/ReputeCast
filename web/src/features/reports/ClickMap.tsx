import { useCallback, useEffect, useRef, useState } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { clickNote } from './words'

type Link = Schemas['ReportLink']

/** The email's own width; the map shows it scaled to the card. */
const EMAIL_WIDTH = 600
/** Bubble size, for keeping bubbles inside the map. */
const BUBBLE_W = 42
const BUBBLE_H = 22

interface Bubble {
  key: string
  label: string
  share: number
  hot: boolean
  left: number
  top: number
}

/** The links an email has, in document order, as the sending engine numbers them. */
function anchorsOf(doc: Document): HTMLAnchorElement[] {
  return Array.from(doc.querySelectorAll<HTMLAnchorElement>('a[href]')).filter((a) => {
    const href = (a.getAttribute('href') ?? '').trim()
    return href !== '' && !href.startsWith('#') && !href.toLowerCase().startsWith('mailto:')
  })
}

const OUTPUT_TAG = /\{\{-?\s*([^{}]*?)\s*-?\}\}/g
const DEFAULT_FILTER = /\|\s*default\s*:\s*(?:"([^"]*)"|'([^']*)')/

/**
 * A Liquid output tag with a fallback reads as its fallback: the map shows
 * the email to no one in particular ("Hi there"). Tags without one stay.
 */
export function withFallbacks(text: string): string {
  return text.replace(OUTPUT_TAG, (tag, inner: string) => {
    const m = DEFAULT_FILTER.exec(inner)
    return m ? (m[1] ?? m[2] ?? '') : tag
  })
}

/** Applies withFallbacks to the email's text (never to its links). */
function showFallbacks(doc: Document) {
  if (!doc.body) return
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
  for (let n = walker.nextNode(); n; n = walker.nextNode()) {
    const t = n.nodeValue
    if (t?.includes('{{')) n.nodeValue = withFallbacks(t)
  }
}

/** The app's IBM Plex @font-face rules, so the email renders in the same fonts. */
function fontFaces(): string {
  const out: string[] = []
  for (const sheet of Array.from(document.styleSheets)) {
    let rules: CSSRuleList
    try {
      rules = sheet.cssRules
    } catch {
      continue
    }
    for (const r of Array.from(rules)) {
      if (r instanceof CSSFontFaceRule && r.cssText.includes('IBM Plex')) out.push(r.cssText)
    }
  }
  return out.join('\n')
}

/**
 * Where a link sits in the email (unscaled px): its box, the end of its line
 * of links (a navigation row), the block around it, and whether it is drawn
 * as a button.
 */
export interface AnchorBox {
  top: number
  bottom: number
  right: number
  /** Right edge of the last link on the same line. */
  rowRight: number
  /** The box of the closest block-level element around the link. */
  blockTop: number
  blockRight: number
  button: boolean
}

/**
 * Bubbles as on the Report board: a button's bubble overlaps its bottom-right
 * corner, a text link's sits just after the link (after the whole row for a
 * line of links), and "Other" sits on the top edge of the block holding the
 * last small link (the footer). A text link that ends its line has no room
 * after it, so its bubble sits just under its end instead of over the words.
 */
export function placeBubbles(links: Link[], anchors: AnchorBox[], other: Schemas['ReportLinkGroup'] | undefined, scale: number, width: number): Bubble[] {
  const clampX = (x: number) => Math.max(2, Math.min(width - BUBBLE_W - 2, x))
  const clampY = (y: number) => Math.max(2, y)
  const out: Bubble[] = []
  let lastGrouped: AnchorBox | undefined
  for (const l of links) {
    const a = anchors[l.position]
    if (!a) continue
    if (l.grouped) {
      if (!lastGrouped || a.top >= lastGrouped.top) lastGrouped = a
      continue
    }
    const after = Math.max(a.right, a.rowRight) * scale + 4
    const pos = a.button
      ? { left: a.right * scale - 8, top: a.bottom * scale - 8 }
      : after + BUBBLE_W <= width - 2
        ? { left: after, top: ((a.top + a.bottom) / 2) * scale - BUBBLE_H / 2 }
        : { left: a.right * scale - BUBBLE_W, top: a.bottom * scale + 2 }
    out.push({ key: `${l.position}`, label: l.label, share: l.share, hot: l.hot, left: clampX(pos.left), top: clampY(pos.top) })
  }
  if (other && lastGrouped) {
    out.push({ key: 'other', label: 'Other', share: other.share, hot: false, left: clampX(lastGrouped.blockRight * scale - BUBBLE_W - 8), top: clampY(lastGrouped.blockTop * scale - 8) })
  }
  return out
}

/** Measures a link inside the email document. */
function anchorBox(a: HTMLAnchorElement, all: HTMLAnchorElement[]): AnchorBox {
  const win = a.ownerDocument.defaultView
  const r = a.getBoundingClientRect()
  let rowRight = r.right
  for (const b of all) {
    const o = b.getBoundingClientRect()
    if (b !== a && Math.abs(o.top - r.top) < 4 && o.left >= r.right) rowRight = Math.max(rowRight, o.right)
  }
  let block: Element | null = a.parentElement
  while (block && win && block !== a.ownerDocument.body) {
    const d = win.getComputedStyle(block).display
    if (!d.startsWith('inline') && d !== 'contents') break
    block = block.parentElement
  }
  const br = (block ?? a.ownerDocument.body).getBoundingClientRect()
  const st = win?.getComputedStyle(a)
  const bg = st?.backgroundColor ?? ''
  const button = Boolean(st && bg !== '' && bg !== 'transparent' && !bg.endsWith(', 0)') && parseFloat(st.paddingLeft) >= 4)
  return { top: r.top, bottom: r.bottom, right: r.right, rowRight, blockTop: br.top, blockRight: br.right, button }
}

function MapFrame({ m }: { m: Schemas['ReportClickMap'] }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [layout, setLayout] = useState<{ height: number; scale: number; bubbles: Bubble[] }>()

  const measure = useCallback(() => {
    const doc = frame.current?.contentDocument
    const el = box.current
    if (!doc || !el || !doc.body) return
    const width = el.clientWidth
    const scale = width / EMAIL_WIDTH
    const height = Math.max(doc.documentElement.scrollHeight, doc.body.scrollHeight)
    // Match each stored link to the anchor at its position; fall back to the
    // first anchor with the same address.
    const anchors = anchorsOf(doc)
    const byPosition: AnchorBox[] = []
    for (const l of m.links) {
      let a = anchors[l.position]
      if (!a || (a.getAttribute('href') ?? '').trim() !== l.url) a = anchors.find((x) => (x.getAttribute('href') ?? '').trim() === l.url) ?? a
      if (a) byPosition[l.position] = anchorBox(a, anchors)
    }
    setLayout({ height, scale, bubbles: placeBubbles(m.links, byPosition, m.other, scale, width) })
  }, [m.links, m.other])

  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (doc) {
      const style = doc.createElement('style')
      style.textContent = `${fontFaces()}\nhtml,body{margin:0;overflow:hidden}`
      doc.head?.appendChild(style)
      showFallbacks(doc)
    }
    measure()
    // Web fonts change line breaks; measure again once they are in.
    void doc?.fonts?.ready.then(measure)
  }

  useEffect(() => {
    const el = box.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(() => measure())
    ro.observe(el)
    return () => ro.disconnect()
  }, [measure])

  const h = layout ? layout.height * layout.scale : 280
  return (
    <div ref={box} className={cn('rep-map', h > 420 && 'scroll')} style={{ height: Math.min(h, 420) }}>
      <div style={{ height: h, position: 'relative' }}>
        <iframe
          ref={frame}
          title="The email as sent, with the share of clicks on each link"
          sandbox="allow-same-origin"
          srcDoc={m.html}
          onLoad={onLoad}
          tabIndex={-1}
          aria-hidden="true"
          style={{ height: layout?.height ?? 500, transform: `scale(${layout?.scale ?? 340 / EMAIL_WIDTH})` }}
        />
        {layout?.bubbles.map((b) => (
          <span key={b.key} className={cn('rep-ck', b.hot && 'hot')} style={{ left: b.left, top: b.top }} title={`${b.label}: ${Math.round(b.share)}% of clicks`}>
            {Math.round(b.share)}%
          </span>
        ))}
      </div>
    </div>
  )
}

/** Clicks per link as a table (the click map's table view). */
function LinkTable({ m }: { m: Schemas['ReportClickMap'] }) {
  return (
    <div className="tscroll">
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <th scope="col">Link</th>
            <th scope="col" className="r">
              Clicks
            </th>
            <th scope="col" className="r">
              Share
            </th>
          </tr>
        </thead>
        <tbody>
          {m.links.map((l) => (
            <tr key={`${l.position}-${l.url}`}>
              <td className="wrap">
                <div className="font-medium">{l.label}</div>
                <div className="k mono">{l.url}</div>
              </td>
              <td className="r n">{count(l.clicks)}</td>
              <td className="r n">{Math.round(l.share)}%</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** "Click map": the email as sent with the share of clicks on each link. */
export function ClickMap({ m }: { m: Schemas['ReportClickMap'] }) {
  const [table, setTable] = useState(false)
  const note = clickNote(m)
  const noLinks = m.links.length === 0
  return (
    <section className={cn('card flex min-w-0 flex-[1_1_380px] flex-col gap-3 px-5 py-4.5', !m.available && !noLinks && !table && 'border-bad-line')} aria-labelledby="rep-map-h">
      <div className="flex items-center justify-between max-md:flex-wrap">
        <h2 className="h2" id="rep-map-h">
          Click map
        </h2>
        <span className="k">share of {count(m.total)} clicks</span>
      </div>
      {noLinks ? (
        <div className="flex flex-col items-center gap-2 py-8 text-center">
          <span className="font-semibold">No tracked links</span>
          <span className="k max-w-[320px]">Clicks are counted on links in the email. This campaign had none, or link tracking was off.</span>
        </div>
      ) : table ? (
        <LinkTable m={m} />
      ) : m.available ? (
        <>
          <MapFrame m={m} />
          <table className="vh">
            <caption>Share of clicks per link</caption>
            <thead>
              <tr>
                <th scope="col">Link</th>
                <th scope="col">Clicks</th>
                <th scope="col">Share</th>
              </tr>
            </thead>
            <tbody>
              {m.links.map((l) => (
                <tr key={`${l.position}-${l.url}`}>
                  <td>{l.label}</td>
                  <td>{count(l.clicks)}</td>
                  <td>{Math.round(l.share)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      ) : (
        <div className="flex flex-col items-center gap-2 py-4 text-center">
          <span className="font-semibold">Click map couldn't render</span>
          <span className="k">The stored content of this campaign is missing, so there's no email to draw the clicks on. Link-level clicks are still counted.</span>
          <Button size="sm" onClick={() => setTable(true)}>
            Show clicks as a table
          </Button>
        </div>
      )}
      {note && !noLinks && m.available && !table ? <span className="k">{note}</span> : null}
    </section>
  )
}
