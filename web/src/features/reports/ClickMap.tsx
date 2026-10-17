import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { TableScroll } from '@/components/ui/Table'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { clickNote } from './words'

type Link = Schemas['ReportLink']

/** The email's own width; the map shows it scaled to the card. */
const EMAIL_WIDTH = 600
/** Bubble height (.rep-ck). */
const BUBBLE_H = 22

/** A bubble's width: "38%" in 11.5px bold IBM Plex Sans with 8px padding on each side. */
export function bubbleWidth(share: number): number {
  return Math.ceil(26.5 + 7.3 * String(Math.round(share)).length)
}

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
    // Poll answers and survey links aren't tracked: a poll is one region.
    return href !== '' && !href.startsWith('#') && !href.toLowerCase().startsWith('mailto:') && !href.includes('rc_poll_') && !href.includes('rc_survey_')
  })
}

/** A poll on the map: its block id and its label ("Poll · 2,412 answers"). */
export interface MapPoll {
  id: string
  label: string
}

interface PollRegion {
  id: string
  label: string
  left: number
  top: number
  width: number
  height: number
}

/** Each poll's region: the column holding its answer links, scaled. */
function pollRegions(doc: Document, polls: MapPoll[], scale: number): PollRegion[] {
  const out: PollRegion[] = []
  for (const p of polls) {
    const links = Array.from(doc.querySelectorAll<HTMLAnchorElement>('a[href]')).filter((a) => (a.getAttribute('href') ?? '').includes(`rc_poll_${p.id}`))
    if (!links.length) continue
    const column = links[0]!.closest('[class*="mj-column-per"]') ?? links[0]!.parentElement
    const r = column?.getBoundingClientRect()
    if (!r) continue
    out.push({ id: p.id, label: p.label, left: r.left * scale, top: r.top * scale, width: r.width * scale, height: r.height * scale })
  }
  return out
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

/**
 * Email CSS puts its desktop layout behind "@media only screen and
 * (min-width:480px)" (MJML does). The map's frame is always 600px wide, so the
 * rules hold without the screen guard; dropping it keeps the same layout when
 * the report is printed to PDF, where the bubbles were measured on screen.
 */
export function screenRulesForPrint(css: string): string {
  return css.replace(/@media\s+(?:only\s+)?screen\s+and\s+/gi, '@media ').replace(/@media\s+(?:only\s+)?screen\s*\{/gi, '@media all {')
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
 * of links (a navigation row), whether anything follows it on its line, the
 * block around it, and whether it is drawn as a button.
 */
export interface AnchorBox {
  top: number
  bottom: number
  right: number
  /** Right edge of the last link on the same line. */
  rowRight: number
  /** Nothing but space follows the link (or its row of links) on its line. */
  endsLine: boolean
  /** Top of the band the link sits on: the closest box around it that paints a background (an email's section), else its block. */
  bandTop: number
  /** Right edge of the closest block-level element around the link. */
  blockRight: number
  button: boolean
}

/**
 * Bubbles as on the Report board: a button's bubble overlaps its bottom-right
 * corner; a text link that ends its line has its bubble at the right edge of
 * its block, on its line, when that leaves the words clear; otherwise the
 * bubble sits just after the link (after the whole row for a line of links)
 * or, with no room there, just under its end. "Other" sits on the top edge
 * of the band holding the last small link (the footer), at its block's right.
 */
export function placeBubbles(links: Link[], anchors: AnchorBox[], other: Schemas['ReportLinkGroup'] | undefined, scale: number, width: number): Bubble[] {
  const clampX = (x: number, w: number) => Math.max(2, Math.min(width - w - 2, x))
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
    const w = bubbleWidth(l.share)
    const after = Math.max(a.right, a.rowRight) * scale + 4
    const line = ((a.top + a.bottom) / 2) * scale - BUBBLE_H / 2
    const edge = Math.min(a.blockRight * scale, width - 2) - w
    const pos = a.button
      ? { left: a.right * scale - 8, top: a.bottom * scale - 8 }
      : a.endsLine && edge >= after
        ? { left: edge, top: line }
        : after + w <= width - 2
          ? { left: after, top: line }
          : { left: a.right * scale - w, top: a.bottom * scale + 2 }
    out.push({ key: `${l.position}`, label: l.label, share: l.share, hot: l.hot, left: clampX(pos.left, w), top: clampY(pos.top) })
  }
  if (other && lastGrouped) {
    const w = bubbleWidth(other.share)
    out.push({ key: 'other', label: 'Other', share: other.share, hot: false, left: clampX(lastGrouped.blockRight * scale - w, w), top: clampY(lastGrouped.bandTop * scale - 8) })
  }
  return out
}

/** Whether only space follows a link inside its block on the link's line. */
function lastOnLine(a: HTMLAnchorElement, block: Element): boolean {
  const r = a.getBoundingClientRect()
  const range = a.ownerDocument.createRange()
  range.setStartAfter(a)
  range.setEnd(block, block.childNodes.length)
  return !Array.from(range.getClientRects()).some((t) => t.width > 0.5 && t.left >= r.right - 1 && t.top < r.bottom - 2 && t.bottom > r.top + 2)
}

/** Measures a link inside the email document. */
function anchorBox(a: HTMLAnchorElement, all: HTMLAnchorElement[]): AnchorBox {
  const win = a.ownerDocument.defaultView
  const r = a.getBoundingClientRect()
  let block: Element | null = a.parentElement
  while (block && win && block !== a.ownerDocument.body) {
    const d = win.getComputedStyle(block).display
    if (!d.startsWith('inline') && d !== 'contents') break
    block = block.parentElement
  }
  const outer = block ?? a.ownerDocument.body
  const br = outer.getBoundingClientRect()
  let band: Element | null = outer
  while (band && win && band !== a.ownerDocument.body) {
    const bg = win.getComputedStyle(band).backgroundColor
    if (bg !== '' && bg !== 'transparent' && !bg.endsWith(', 0)')) break
    band = band.parentElement
  }
  const bandTop = band && band !== a.ownerDocument.body ? band.getBoundingClientRect().top : br.top
  // A row is the links after this one on its line in the same block (a
  // navigation bar), not links in a neighbouring column.
  let rowRight = r.right
  let last = a
  for (const b of all) {
    const o = b.getBoundingClientRect()
    if (b !== a && outer.contains(b) && Math.abs(o.top - r.top) < 4 && o.left >= r.right && o.right > rowRight) {
      rowRight = o.right
      last = b
    }
  }
  const st = win?.getComputedStyle(a)
  const bg = st?.backgroundColor ?? ''
  const button = Boolean(st && bg !== '' && bg !== 'transparent' && !bg.endsWith(', 0)') && parseFloat(st.paddingLeft) >= 4)
  return { top: r.top, bottom: r.bottom, right: r.right, rowRight, endsLine: lastOnLine(last, outer), bandTop, blockRight: br.right, button }
}

function MapFrame({ m, polls }: { m: Schemas['ReportClickMap']; polls?: MapPoll[] }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [layout, setLayout] = useState<{ height: number; scale: number; bubbles: Bubble[]; regions: PollRegion[] }>()

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
    setLayout({ height, scale, bubbles: placeBubbles(m.links, byPosition, m.other, scale, width), regions: polls?.length ? pollRegions(doc, polls, scale) : [] })
  }, [m.links, m.other, polls])

  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (doc) {
      for (const st of Array.from(doc.querySelectorAll('style'))) {
        const css = st.textContent ?? ''
        const kept = screenRulesForPrint(css)
        if (kept !== css) st.textContent = kept
      }
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
        {layout?.regions.map((r) => (
          <span key={r.id} className="rep-poll" style={{ left: r.left - 3, top: r.top - 3, width: r.width + 6, height: r.height + 6 }}>
            <b>{r.label}</b>
          </span>
        ))}
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
    <TableScroll label="Clicks per link">
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
    </TableScroll>
  )
}

/**
 * "Click map": the email as sent with the share of clicks on each link.
 * clicks is the headline count; imported history may have clicks but no links.
 */
export function ClickMap({
  m,
  clicks,
  imported,
  picker,
  languageNote,
  polls,
  pollNote,
}: {
  m: Schemas['ReportClickMap']
  clicks: number
  imported: boolean
  /** The language picker of a campaign sent in several languages. */
  picker?: ReactNode
  /** Who else the version shown went to (readers who got the fallback). */
  languageNote?: ReactNode
  /** The email's polls, drawn as one region each with their answers. */
  polls?: MapPoll[]
  pollNote?: ReactNode
}) {
  const [table, setTable] = useState(false)
  const note = clickNote(m)
  const noLinks = m.links.length === 0
  return (
    <section className={cn('card flex min-w-0 flex-[1_1_380px] flex-col gap-3 px-5 py-4.5', !m.available && !noLinks && !table && 'border-bad-line')} aria-labelledby="rep-map-h">
      <div className="flex items-center justify-between max-md:flex-wrap">
        {picker ? (
          <>
            <div className="flex min-w-0 flex-col gap-0.5">
              <h2 className="h2" id="rep-map-h">
                Click map
              </h2>
              <span className="k">{noLinks ? null : `share of ${count(m.total)} clicks`}</span>
            </div>
            {picker}
          </>
        ) : (
          <>
            <h2 className="h2" id="rep-map-h">
              Click map
            </h2>
            {noLinks ? null : <span className="k">share of {count(m.total)} clicks</span>}
          </>
        )}
      </div>
      {noLinks ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-center">
          <span className="font-semibold">{imported && clicks > 0 ? 'No clicks per link' : 'No tracked links'}</span>
          <span className="k max-w-[320px]">
            {imported && clicks > 0
              ? `Clicks per link weren’t part of this campaign’s import; its ${count(clicks)} clicks are in the tiles above.`
              : 'Clicks are counted on links in the email. This campaign had none, or link tracking was off.'}
          </span>
        </div>
      ) : table ? (
        <LinkTable m={m} />
      ) : m.available ? (
        <>
          <MapFrame m={m} polls={polls} />
          {/* A table ignores width: 1px, so the wrapper hides it without widening the page. */}
          <div className="vh">
            <table>
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
          </div>
        </>
      ) : (
        // The card stretches to the chart beside it; the notice sits in its middle.
        <div className="flex flex-1 flex-col items-center justify-center gap-2 py-4 text-center">
          <span className="font-semibold">Click map couldn't render</span>
          <span className="k">The stored content of this campaign is missing, so there's no email to draw the clicks on. Link-level clicks are still counted.</span>
          <Button size="sm" className="rep-noprint" onClick={() => setTable(true)}>
            Show clicks as a table
          </Button>
        </div>
      )}
      {note && !noLinks && m.available && !table ? <span className="k">{note}</span> : null}
      {polls?.length && m.available && !table ? pollNote : null}
      {languageNote}
    </section>
  )
}
