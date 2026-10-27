import { forwardRef, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Segmented } from '@/components/ui/Tabs'
import { cn } from '@/lib/cn'
import { useT } from '@/i18n'
import { andList, count, pct, trimPct } from '@/lib/format'
import { m as rm } from './messages'
import { pct1 } from './words'

type Link = Schemas['ReportLink']
type Row = Schemas['ReportLinkRow']
type Map = Schemas['ReportClickMap']

/** The email's own width; the map shows it scaled to the card. */
const EMAIL_WIDTH = 600
/** Bubble height (.rep-ck). */
const BUBBLE_H = 22

/** A bubble's width: its text in 11.5px bold IBM Plex Sans with 8px padding on each side. */
export function bubbleWidth(text: string): number {
  let w = 16
  for (const ch of text) w += ch === '%' ? 10.5 : ch === ',' || ch === '.' ? 3.5 : 7.3
  return Math.ceil(w)
}

/** A place on the map that gets a bubble: a link, or a row of links (drawn after the row). */
export interface Spot {
  key: string
  label: string
  /** What the bubble says: a share of human clicks, or a count of bot clicks. */
  text: string
  /** The link the bubble sits by: the link, or a row's first link. */
  anchor: number
  hot: boolean
}

interface Bubble {
  key: string
  label: string
  text: string
  hot: boolean
  left: number
  top: number
}

/**
 * The email's tracked links by position. The report marks each with the
 * position the sending engine gave it (data-rc-link, internal/linkrule), so
 * the map follows the engine's rule for which links are tracked.
 */
export function anchorsOf(doc: Document): HTMLAnchorElement[] {
  const out: HTMLAnchorElement[] = []
  for (const a of Array.from(doc.querySelectorAll<HTMLAnchorElement>('a[data-rc-link]'))) {
    const pos = Number(a.getAttribute('data-rc-link'))
    if (Number.isInteger(pos) && pos >= 0 && !out[pos]) out[pos] = a
  }
  return out
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
 * of the band holding the last of its links (the footer), at its block's right.
 */
export function placeBubbles(
  spots: Spot[],
  anchors: AnchorBox[],
  other: { text: string; label: string; anchors: number[] } | undefined,
  scale: number,
  width: number,
): Bubble[] {
  const clampX = (x: number, w: number) => Math.max(2, Math.min(width - w - 2, x))
  const clampY = (y: number) => Math.max(2, y)
  const out: Bubble[] = []
  for (const s of spots) {
    const a = anchors[s.anchor]
    if (!a) continue
    const w = bubbleWidth(s.text)
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
    out.push({ key: s.key, label: s.label, text: s.text, hot: s.hot, left: clampX(pos.left, w), top: clampY(pos.top) })
  }
  let last: AnchorBox | undefined
  for (const i of other?.anchors ?? []) {
    const a = anchors[i]
    if (a && (!last || a.top >= last.top)) last = a
  }
  if (other && last) {
    const w = bubbleWidth(other.text)
    out.push({ key: 'other', label: other.label, text: other.text, hot: false, left: clampX(last.blockRight * scale - w, w), top: clampY(last.bandTop * scale - 8) })
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

/** What a map shows: one variant's clicks, or everyone's. */
interface Shown {
  /** Links under this share (percent) are grouped as Other. */
  threshold: number
  html: string
  available: boolean
  total: number
  botTotal: number
  links: Link[]
  rows: Row[]
  other?: Schemas['ReportLinkGroup']
}

/** The row label: "Header links", "Footer links", "Links in one row". */
function rowLabel(t: ReturnType<typeof useT>, r: Row): string {
  return t(r.place === 'header' ? rm.rowHeader : r.place === 'footer' ? rm.rowFooter : rm.rowBody)
}

/** The entries of the map and its table, in the board's order: links by share, rows, then Other. */
export function entries(s: Shown): { links: Link[]; rows: Row[]; grouped: Link[] } {
  const inRow = new Set(s.rows.flatMap((r) => r.positions))
  const links = s.links.filter((l) => !l.grouped && !inRow.has(l.position)).sort((a, b) => b.share - a.share || a.position - b.position)
  const rows = [...s.rows].sort((a, b) => b.share - a.share)
  const grouped = s.links.filter((l) => l.grouped && !inRow.has(l.position))
  return { links, rows, grouped }
}

function MapFrame({ s, view, polls }: { s: Shown; view: 'human' | 'bot'; polls?: MapPoll[] }) {
  const t = useT()
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
    const anchors = anchorsOf(doc)
    const all = anchors.filter(Boolean)
    const boxes: AnchorBox[] = []
    anchors.forEach((a, i) => {
      if (a) boxes[i] = anchorBox(a, all)
    })
    const { links, rows, grouped } = entries(s)
    const bot = view === 'bot'
    const spots: Spot[] = [
      ...links.map((l) => ({ key: `${l.position}`, label: l.label, text: bot ? count(l.botClicks) : pct(Math.round(l.share), 0), anchor: l.position, hot: !bot && l.hot })),
      ...rows.map((r) => ({ key: `row-${r.positions[0]}`, label: andList(r.labels), text: bot ? count(r.botClicks) : pct(Math.round(r.share), 0), anchor: r.positions[0]!, hot: false })),
    ]
    const otherBots = grouped.reduce((n, l) => n + l.botClicks, 0)
    const other =
      s.other && grouped.length
        ? { text: bot ? count(otherBots) : pct(Math.round(s.other.share), 0), label: t(rm.other), anchors: grouped.map((l) => l.position) }
        : undefined
    setLayout({ height, scale, bubbles: placeBubbles(spots, boxes, other, scale, width), regions: polls?.length ? pollRegions(doc, polls, scale) : [] })
  }, [s, view, polls, t])

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
    measure()
  }, [measure])

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
          title={t(rm.frameTitle)}
          sandbox="allow-same-origin"
          srcDoc={s.html}
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
          <span
            key={b.key}
            className={cn('rep-ck', b.hot && 'hot', view === 'bot' && 'bot')}
            style={{ left: b.left, top: b.top }}
            title={view === 'bot' ? t(rm.bubbleBots, { label: b.label, count: b.text }) : t(rm.bubbleTitle, { label: b.label, share: b.text })}
          >
            {b.text}
          </span>
        ))}
      </div>
    </div>
  )
}

/** Clicks per link (human or bot) with the rows, Other and the total, as on the board. */
function LinkTable({ s, view, focus }: { s: Shown; view: 'human' | 'bot'; focus?: number }) {
  const t = useT()
  const bot = view === 'bot'
  const { links, rows, grouped } = entries(s)
  const total = bot ? s.botTotal : s.total
  const share = (n: number, human: number) => (bot ? (total > 0 ? pct1((n / total) * 100) : '—') : pct1(human))
  const each = (n: number, k: number) => count(Math.round(n / Math.max(k, 1)))
  const otherBots = grouped.reduce((n, l) => n + l.botClicks, 0)
  const otherHuman = grouped.reduce((n, l) => n + l.uniqueClicks, 0)
  const labels = s.other?.labels ?? []
  const focusRow = useRef<HTMLTableRowElement>(null)
  useEffect(() => {
    if (focus !== undefined) focusRow.current?.focus({ preventScroll: true })
  }, [focus])
  return (
    <div className="overflow-x-auto">
      <table className="tbl rep-tbl rep-links">
        <caption className="vh">{t(bot ? rm.botPerLink : rm.humanPerLink)}</caption>
        <thead>
          <tr>
            <th scope="col">{t(rm.colLink)}</th>
            <th scope="col" className="r">
              {t(bot ? rm.colBotClicks : rm.colHumanClicks)}
            </th>
            <th scope="col" className="r">
              {t(rm.colShare)}
            </th>
          </tr>
        </thead>
        <tbody>
          {links.map((l) => (
            <tr key={`l-${l.position}`} ref={focus === l.position ? focusRow : undefined} tabIndex={focus === l.position ? -1 : undefined} className={cn(focus === l.position && 'rep-focus')}>
              <th scope="row">
                {l.label}
                <span className="k mono block font-normal">{l.url.replace(/^https?:\/\//, '')}</span>
              </th>
              <td className="r n">{count(bot ? l.botClicks : l.uniqueClicks)}</td>
              <td className="r n">{share(l.botClicks, l.share)}</td>
            </tr>
          ))}
          {rows.map((r) => (
            <tr key={`r-${r.positions[0]}`}>
              <th scope="row">
                {rowLabel(t, r)}
                <span className="k block font-normal">{bot ? t(rm.rowBotSub, { n: r.positions.length, each: each(r.botClicks, r.positions.length) }) : andList(r.labels)}</span>
              </th>
              <td className="r n">{count(bot ? r.botClicks : r.clicks)}</td>
              <td className="r n">{share(r.botClicks, r.share)}</td>
            </tr>
          ))}
          {s.other && grouped.length ? (
            <tr>
              <th scope="row">
                {t(rm.otherLinks, { n: grouped.length })}
                <span className="k block font-normal">
                  {bot
                    ? t(rm.otherBotSub, { each: each(otherBots, grouped.length) })
                    : labels.length
                      ? t(rm.otherUnderIncluding, { threshold: trimPct(s.threshold), labels: andList(labels) })
                      : t(rm.otherUnder, { threshold: trimPct(s.threshold) })}
                </span>
              </th>
              <td className="r n">{count(bot ? otherBots : otherHuman)}</td>
              <td className="r n">{share(otherBots, s.other.share)}</td>
            </tr>
          ) : null}
        </tbody>
        <tfoot>
          <tr>
            <th scope="row">{t(rm.allLinks)}</th>
            <td className="r n">{count(total)}</td>
            <td className="r n">{pct(100, 0)}</td>
          </tr>
        </tfoot>
      </table>
    </div>
  )
}

/**
 * "Click map": the email as sent with the share of human clicks on each
 * link, or where the bot clicks landed, beside the clicks per link (Report
 * board). clicks is the headline count of people who clicked; imported
 * history may have clicks but no links.
 */
export const ClickMap = forwardRef<
  HTMLElement,
  {
    m: Map
    clicks: number
    imported: boolean
    /** False when a read-only link leaves the email out. */
    emailShared?: boolean
    /** The language picker of a campaign sent in several languages. */
    picker?: ReactNode
    /** Who else the version shown went to (readers who got the fallback). */
    languageNote?: ReactNode
    /** The email's polls, drawn as one region each with their answers. */
    polls?: MapPoll[]
    pollNote?: ReactNode
    view: 'human' | 'bot'
    onView: (v: 'human' | 'bot') => void
    /** A link to point at (an insight's "Show on the click map"). */
    focus?: number
  }
>(function ClickMap({ m, clicks, imported, emailShared = true, picker, languageNote, polls, pollNote, view, onView, focus }, ref) {
  const t = useT()
  const [table, setTable] = useState(false)
  const [variant, setVariant] = useState('')
  const v = m.variants.find((x) => x.key === variant)
  // One object per map and variant: the frame measures again when it changes.
  const s: Shown = useMemo(
    () =>
      v
        ? {
            threshold: m.threshold,
            html: v.sameEmail ? m.html : v.html,
            available: v.sameEmail ? m.available : v.available,
            total: v.total,
            botTotal: v.botTotal,
            links: v.links,
            rows: v.rows,
            other: v.other,
          }
        : { threshold: m.threshold, html: m.html, available: m.available, total: m.total, botTotal: m.botTotal, links: m.links, rows: m.rows, other: m.other },
    [m, v],
  )
  const noLinks = m.links.length === 0
  const bot = view === 'bot'
  const failed = !s.available && emailShared && !table
  const controls = noLinks ? null : (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
      {picker}
      {m.variants.length > 1 ? (
        <label className="flex items-center gap-2 text-[12px] font-semibold text-ink-2">
          {t(rm.variantToShow)}
          <select className="in h-[30px]" value={variant} onChange={(e) => setVariant(e.target.value)}>
            <option value="">{t(rm.allRecipients)}</option>
            {m.variants.map((x) => (
              <option key={x.key} value={x.key}>
                {t(rm.variantRecipients, { key: x.key })}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      <Segmented
        className="rep-seg"
        value={view}
        onChange={onView}
        label={t(rm.clicksToShow)}
        items={[
          { value: 'human', label: t(rm.colHumanClicks) },
          { value: 'bot', label: t(rm.colBotClicks) },
        ]}
      />
    </div>
  )
  const people = clicks
  const right = (
    <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-2.5">
      {bot ? (
        <p className="m-0 text-[13px] text-ink-2">
          {s.botTotal > 0
            ? t(rm.botNote, { count: count(s.botTotal), each: count(Math.round(s.botTotal / Math.max(1, s.links.filter((l) => l.botClicks > 0).length))) })
            : t(rm.botNone)}
        </p>
      ) : v ? (
        <p className="k m-0">{t(rm.mapCaptionVariant, { total: count(s.total), key: v.key })}</p>
      ) : (
        <p className="k m-0">{t(rm.mapCaption, { n: people, total: count(s.total), people: count(people) })}</p>
      )}
      <LinkTable s={s} view={view} focus={focus} />
      {bot || v ? null : (
        <p className="k m-0">
          {t(rm.humanNote, { people: count(people), total: count(s.total) })}
          {s.botTotal > 0 ? ` ${t(rm.humanNoteBots, { count: count(s.botTotal) })}` : ''}
        </p>
      )}
      {!emailShared ? <span className="k">{t(rm.emailNotShared)}</span> : null}
    </div>
  )
  return (
    <section
      ref={ref}
      id="rep-click-map"
      tabIndex={-1}
      className={cn('card flex min-w-0 flex-col gap-3.5 px-5 py-4.5', failed && !noLinks && 'border-bad-line')}
      aria-labelledby="rep-map-h"
    >
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 className="h2" id="rep-map-h">
          {t(rm.clickMap)}
        </h2>
        {failed ? null : controls}
      </div>
      {noLinks ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-center">
          <span className="font-semibold">{imported && clicks > 0 ? t(rm.noLinkClicks) : t(rm.noTracked)}</span>
          <span className="k max-w-[320px]">{imported && clicks > 0 ? t(rm.noLinkClicksBody, { count: count(clicks) }) : t(rm.noTrackedBody)}</span>
        </div>
      ) : failed ? (
        // The email is missing: the notice first, the table on request (ReportStates).
        <div className="flex flex-1 flex-col items-center justify-center gap-2 py-4 text-center">
          <span className="font-semibold">{t(rm.mapFailed)}</span>
          <span className="k">{t(rm.mapFailedBody)}</span>
          <Button size="sm" className="rep-noprint" onClick={() => setTable(true)}>
            {t(rm.showTable)}
          </Button>
        </div>
      ) : (
        <div className="flex flex-wrap items-start gap-5">
          {s.available ? (
            <div className="mx-auto min-w-0 flex-[0_1_340px]">
              <MapFrame s={s} view={view} polls={polls} />
            </div>
          ) : null}
          {right}
        </div>
      )}
      {polls?.length && s.available && !noLinks ? pollNote : null}
      {languageNote}
    </section>
  )
})
