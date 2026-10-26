import type { Component, Editor } from 'grapesjs'
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode, type RefObject } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Popover } from '@/components/ui/Menu'
import { clock, count } from '@/lib/format'
import { CanvasPreviewAs, ComposerLanguageBanner, LanguageTags } from '@/features/languages/composerParts'
import { DigestPreviewWith, useDigestMode } from '@/features/digests/composer/DigestMode'
import { DigestCanvasCards } from '@/features/digests/composer/IssueMode'
import { dc } from '@/features/digests/composer/messages'
import { AltTextNote, WHOLE_ZOOM, WholeIssue } from '@/features/digests/composer/WholeIssue'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { RewriteActions } from '@/features/ai/composer/rewrite'
import { newTest, withVariant } from './ABPanel'
import { mc } from './canvas.messages'
import { m } from './messages'
import { useComposer, type Issue } from './context'
import { DEVICES, EMAIL_DEVICES, type DeviceId } from './editor/grapes'
import { componentLabel, mjChildren, tagOf } from './editor/labels'
import { headCss } from './editor/mjmlDoc'
import { applyTextStyle, currentTextStyle, TEXT_STYLES, type TextEditing, type TextStyle } from './editor/textStyle'
import { chipHtml } from './editor/liquid'
import { componentAt, insertFooter, reveal } from './fixes'
import { EmptyGallery } from './Gallery'
import { ToolIcon } from './icons'
import { canvasIssues, contentIssues } from './RightPanel'
import { LiquidPopover } from './LiquidChips'
import { StateCards } from './StateCards'
import { AccessibilityOverlays } from '@/features/accessibility/composer/Overlays'
import { ConditionalOverlays, ConditionalToggle } from '@/features/conditional/Canvas'
import { VersionsPanel } from '@/features/conditional/Versions'
import { SurveyLinkChoices } from '@/features/surveys/LinkChoices'
import { EmailWebToggle, WebOverlays } from '@/features/website/composer/campaign'
import { GrowPreviewAs } from '@/features/grow/composer/Panel'
import { mb as growMessages } from '@/features/grow/composer/messages'
import { mc as earnComposer } from '@/features/earn/composer/composer.messages'
import { useScrollRegion } from '@/lib/hooks'

interface Box {
  top: number
  left: number
  width: number
  height: number
}

/** Position of an element of the canvas frame, relative to `host`. */
export function boxOf(editor: Editor, el: Element | null | undefined, host: HTMLElement | null): Box | null {
  if (!el || !host || !el.isConnected) return null
  const frame = editor.Canvas.getFrameEl()
  if (!frame) return null
  const zoom = editor.Canvas.getZoomDecimal()
  const fr = frame.getBoundingClientRect()
  const hr = host.getBoundingClientRect()
  const r = el.getBoundingClientRect()
  if (!r.width && !r.height) return null
  return { top: fr.top - hr.top + r.top * zoom, left: fr.left - hr.left + r.left * zoom, width: r.width * zoom, height: r.height * zoom }
}

/** Re-renders the overlays whenever the canvas changes size or content. */
function useCanvasTick(editor: Editor | null, host: HTMLElement | null): number {
  const [tick, setTick] = useState(0)
  useEffect(() => {
    if (!editor || !host) return
    let raf = 0
    const bump = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(() => setTick((t) => t + 1))
    }
    const events = 'component:update component:toggled component:add component:remove component:mount canvas:frame:load:body rte:enable rte:disable undo redo'
    editor.on(events, bump)
    const ro = new ResizeObserver(bump)
    ro.observe(host)
    const frame = editor.Canvas.getFrameEl()
    if (frame) ro.observe(frame)
    const win = editor.Canvas.getWindow() as (Window & typeof globalThis) | undefined
    const inner = win?.ResizeObserver ? new win.ResizeObserver(bump) : null
    const body = editor.Canvas.getBody()
    if (inner && body) inner.observe(body)
    window.addEventListener('resize', bump)
    bump()
    return () => {
      editor.off(events, bump)
      ro.disconnect()
      inner?.disconnect()
      window.removeEventListener('resize', bump)
      cancelAnimationFrame(raf)
    }
  }, [editor, host])
  return tick
}

function UndoRedo() {
  const c = useComposer()
  const t = useT()
  const um = c.editor?.UndoManager
  const canUndo = !!um?.hasUndo()
  const canRedo = !!um?.hasRedo()
  void c.rev
  return (
    <div className="flex gap-1">
      <button type="button" className="btn ic" aria-label={t(mc.undo)} title={t(mc.undoTitle)} disabled={!canUndo || !c.canEdit} style={!canUndo ? { opacity: 0.5 } : undefined} onClick={() => um?.undo()}>
        <ToolIcon name="undo" strokeWidth={1.7} />
      </button>
      <button type="button" className="btn ic" aria-label={t(mc.redo)} title={t(mc.redoTitle)} disabled={!canRedo || !c.canEdit} style={!canRedo ? { opacity: 0.5 } : undefined} onClick={() => um?.redo()}>
        <ToolIcon name="redo" strokeWidth={1.7} />
      </button>
    </div>
  )
}

function variantLabel(kind: string | undefined, key: string): string {
  if (key === 'A') return tt(mc.variantA)
  if (kind === 'subject') return tt(mc.variantSubject, { key })
  if (kind === 'send_time') return tt(mc.variantSendTime, { key })
  return tt(mc.variantContent, { key })
}

function VariantSeg() {
  const c = useComposer()
  const tr = useT()
  const t = c.abTest
  const keys = t ? t.variants.map((v) => v.key) : ['A']
  const add = () => {
    if (!c.canEdit) return
    // A new subject variant starts from A's subject (CO-6).
    if (!t) {
      c.setABTest(newTest('subject', c.subject, c.currentMjml))
      c.setVariant('B')
      return
    }
    if (t.variants.length >= 3) {
      c.setView('preview')
      return
    }
    const next = withVariant(t, c.subject, c.currentMjml)
    c.setABTest(next)
    c.setVariant(next.variants[next.variants.length - 1]!.key)
  }
  return (
    <div className="seg" role="group" aria-label={tr(mc.variantGroup)}>
      {keys.map((k) => (
        <button key={k} type="button" aria-pressed={c.variant === k} onClick={() => c.setVariant(k)}>
          {variantLabel(t?.kind, k)}
        </button>
      ))}
      {keys.length < 3 && c.canEdit ? (
        <button type="button" aria-label={t ? tr(mc.addVariant) : tr(mc.startAb)} title={t ? tr(mc.addVariant) : tr(mc.startAbTitle)} onClick={add}>
          +
        </button>
      ) : null}
    </div>
  )
}

export function DeviceSeg({ dim }: { dim?: boolean }) {
  const c = useComposer()
  const t = useT()
  return (
    <div className="seg" role="group" aria-label={t(mc.device)}>
      {EMAIL_DEVICES.map((d) => (
        <button key={d} type="button" aria-pressed={c.device === d} disabled={dim} onClick={() => c.setDevice(d)}>
          {d === 'desktop' ? t(mc.desktop) : t(mc.mobile)}
        </button>
      ))}
    </div>
  )
}

export function ViewSeg({ dim }: { dim?: boolean }) {
  const c = useComposer()
  const t = useT()
  if (c.views) {
    return (
      <div className="seg" role="group" aria-label={t(mc.view)}>
        {c.views.map((it) => (
          <button key={it.key} type="button" aria-pressed={it.on} disabled={dim} onClick={() => c.selectView?.(it.key)}>
            {it.label}
          </button>
        ))}
      </div>
    )
  }
  const items: { v: 'visual' | 'mjml' | 'html'; label: string }[] = [
    { v: 'visual', label: t(mc.viewVisual) },
    { v: 'mjml', label: 'MJML' },
    { v: 'html', label: 'HTML' },
  ]
  return (
    <div className="seg" role="group" aria-label={t(mc.view)}>
      {items.map((it) => (
        <button key={it.v} type="button" aria-pressed={c.view === it.v} disabled={dim} onClick={() => c.setView(it.v)}>
          {it.label}
        </button>
      ))}
    </div>
  )
}

function Toolbar({ zoom }: { zoom: number }) {
  const c = useComposer()
  const t = useT()
  const digest = useDigestMode()
  const issues = contentIssues(canvasIssues(c))
  const blocks = issues.filter((i) => i.severity === 'block').length
  const warns = issues.length - blocks
  const mobile = c.device === 'mobile'
  return (
    <div className="cmp-tools">
      <DeviceSeg />
      <ViewSeg />
      <EmailWebToggle />
      {digest ? <DigestPreviewWith /> : <CanvasPreviewAs />}
      <GrowPreviewAs editor={c.editor} slug={c.slug} rev={c.rev} campaignId={c.kind === 'transactional' ? undefined : c.campaign.id} />
      <ConditionalToggle />
      {digest?.preview === 'six' ? (
        // "6 items" shows the whole issue, zoomed out, instead of the canvas.
        <span className="k" style={{ marginLeft: 'auto' }}>
          {tt(dc.zoomWhole, { pct: Math.round(WHOLE_ZOOM * 100) })}
        </span>
      ) : issues.length ? (
        <>
          {blocks ? (
            <Badge tone="bad" dot>
              {t(mc.blocking, { count: blocks })}
            </Badge>
          ) : null}
          {warns ? (
            <Badge tone="warn" dot>
              {t(mc.warnings, { count: warns })}
            </Badge>
          ) : null}
          <span className="k" style={{ marginLeft: 'auto' }}>
            {t(mc.warningsNote)}
          </span>
        </>
      ) : mobile ? (
        <>
          <UndoRedo />
          <Badge tone="info">{t(mc.columnsStack)}</Badge>
          <span className="k inline-flex items-center gap-1.5" style={{ marginLeft: 'auto' }}>
            {t(mc.darkMode)}{' '}
            <button type="button" className="btn" aria-pressed={c.darkCanvas} style={{ minHeight: 26, padding: '0 8px', fontSize: 11.5 }} onClick={() => c.setDarkCanvas(!c.darkCanvas)}>
              {c.darkCanvas ? t(common.on) : t(common.off)}
            </button>
          </span>
        </>
      ) : (
        <>
          <UndoRedo />
          {/* A digest's issues have no A/B test: each issue's subject comes from its posts. */}
          {digest || c.campaign.digestRole ? null : <VariantSeg />}
          <span className="k" style={{ marginLeft: 'auto' }}>
            {t(mc.zoomHint, { zoom: Math.round(zoom * 100) })}
          </span>
        </>
      )}
    </div>
  )
}

function SelectionOverlay({ host, tick }: { host: HTMLElement | null; tick: number }) {
  const c = useComposer()
  const t = useT()
  const ed = c.editor
  const sel = c.selected
  const [rte, setRte] = useState(false)
  useEffect(() => {
    if (!ed) return
    const on = () => setRte(true)
    const off = () => setRte(false)
    ed.on('rte:enable', on)
    ed.on('rte:disable', off)
    return () => {
      ed.off('rte:enable', on)
      ed.off('rte:disable', off)
    }
  }, [ed])
  const box = useMemo(() => (ed && sel ? boxOf(ed, sel.getEl(), host) : null), [ed, sel, host, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  // A selected section shows its columns as dashed guides.
  const cols = useMemo(() => {
    if (!ed || !sel || (tagOf(sel) !== 'mj-section' && tagOf(sel) !== 'mj-group')) return []
    return mjChildren(sel)
      .filter((x) => tagOf(x) === 'mj-column')
      .map((x) => boxOf(ed, x.getEl(), host))
      .filter((b): b is Box => !!b)
  }, [ed, sel, host, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  if (!ed || !sel || !box || !tagOf(sel) || tagOf(sel) === 'mj-body' || tagOf(sel) === 'mjml') return null
  const parent = sel.parent()
  const canParent = !!parent && !!tagOf(parent) && tagOf(parent) !== 'mjml'
  // Keyboard alternative to dragging: swap with the previous or next element.
  const move = (dir: -1 | 1) => {
    const p = sel.parent()
    if (!p) return
    const kids = mjChildren(p)
    const target = kids[kids.indexOf(sel) + dir]
    if (!target) return
    const at = p.components().indexOf(target)
    sel.move(p, { at: dir < 0 ? at : at + 1 })
    ed.select(sel)
  }
  return (
    <>
      {cols.map((b, i) => (
        <div key={i} className="cmp-col" style={{ top: b.top, left: b.left, width: b.width, height: b.height }} aria-hidden="true" />
      ))}
      <div className="cmp-sel" style={{ top: box.top, left: box.left, width: box.width, height: box.height }}>
        <div className="cmp-sel-box" aria-hidden="true" />
        {rte ? null : <span className="lab">{componentLabel(sel)}</span>}
        {c.canEdit && !rte ? (
          <div className="tb" role="toolbar" aria-label={t(mc.componentTools)}>
            {canParent ? (
              <button type="button" aria-label={t(mc.selectParent)} title={t(mc.selectParent)} onClick={() => parent && ed.select(parent)}>
                <ToolIcon name="parent" />
              </button>
            ) : null}
            <button
              type="button"
              aria-label={t(mc.moveLabel)}
              title={t(mc.moveTitle)}
              style={{ cursor: 'grab' }}
              onMouseDown={(e) => {
                e.preventDefault()
                ed.runCommand('tlb-move', { target: sel, event: e.nativeEvent })
              }}
              onKeyDown={(e) => {
                if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
                  e.preventDefault()
                  move(e.key === 'ArrowUp' ? -1 : 1)
                }
              }}
            >
              <ToolIcon name="move" />
            </button>
            <button type="button" aria-label={t(mc.duplicate)} title={t(mc.duplicate)} onClick={() => ed.runCommand('tlb-clone')}>
              <ToolIcon name="duplicate" />
            </button>
            <button type="button" aria-label={t(common.delete)} title={t(common.delete)} onClick={() => ed.runCommand('core:component-delete')}>
              <ToolIcon name="delete" />
            </button>
          </div>
        ) : null}
      </div>
    </>
  )
}

const TAGGED = new Set(['content.alt_text', 'links.blocklist', 'links.shortener', 'links.broken', 'links.empty', 'personalization.unknown', 'content.raw_html', 'store.product_removed'])

function WarningOverlays({ host, tick }: { host: HTMLElement | null; tick: number }) {
  const c = useComposer()
  const ed = c.editor
  const issues = canvasIssues(c).filter((i) => (i.severity === 'block' || i.severity === 'warn') && i.tag && i.path.length && TAGGED.has(i.id))
  const placed = useMemo(() => {
    if (!ed) return []
    const out: { issue: Issue; box: Box; target: Component }[] = []
    for (const issue of issues) {
      const target = componentAt(c, issue.path)
      const el = target?.getEl()
      // Mark the image or the button itself, not the row around it.
      const inner = tagOf(target) === 'mj-image' ? el?.querySelector('img') : tagOf(target) === 'mj-button' ? el?.querySelector('a, p') : null
      const box = target ? boxOf(ed, inner ?? el, host) : null
      if (target && box) out.push({ issue, box, target })
    }
    return out
  }, [ed, host, tick, c.checks, c.variant]) // eslint-disable-line react-hooks/exhaustive-deps
  return (
    <>
      {placed.map(({ issue, box, target }, i) => {
        const tone = issue.severity === 'block' ? 'b' : 'w'
        const isImage = tagOf(target) === 'mj-image'
        // As on the board: an image's tag sits on its top-left corner, a link's above it, centred.
        const tagStyle = isImage ? { top: box.top - 12, left: box.left - 12 } : { top: box.top - 14, left: box.left + box.width / 2, transform: 'translateX(-50%)' }
        return (
          <div key={`${issue.id}-${i}`}>
            <div className={`wbox ${tone}`} style={{ top: box.top, left: box.left, width: box.width, height: box.height }} aria-hidden="true" />
            <button type="button" className={`wtag ${tone}`} style={tagStyle} onClick={() => reveal(c, issue.path)} title={issue.title}>
              <ToolIcon name="warn" size={12} strokeWidth={2} />
              {issue.tag}
            </button>
            {issue.url && issue.id.startsWith('links.') ? (
              <span className="mono" style={{ position: 'absolute', zIndex: 2, top: box.top + box.height + 10, left: box.left - 40, width: box.width + 80, textAlign: 'center', fontSize: 11, color: tone === 'b' ? '#991B1B' : '#92400E', pointerEvents: 'none' }}>
                {issue.url.replace(/^https?:\/\//, '')}
              </span>
            ) : null}
          </div>
        )
      })}
    </>
  )
}

function MissingFooter({ width }: { width: number }) {
  const c = useComposer()
  const t = useT()
  const issue = canvasIssues(c).find((i) => i.id === 'compliance.footer')
  if (!issue) return null
  const when = c.checks?.footerRemovedAt
  return (
    <div className="wfoot" style={{ width, marginTop: 0 }}>
      <button type="button" className="wtag b" style={{ top: -13, left: '50%', transform: 'translateX(-50%)' }} onClick={() => c.setRightTab('deliverability')}>
        <ToolIcon name="warn" size={12} strokeWidth={2} />
        {issue.tag || t(mc.noFooterTag)}
      </button>
      <span style={{ fontSize: 12.5, color: '#991B1B' }}>{when ? t(mc.footerDeleted, { time: clock(when) }) : t(mc.footerMissing)}</span>
      <br />
      {c.canEdit ? (
        <button type="button" className="btn sm" style={{ marginTop: 8, borderColor: '#FCA5A5', color: '#991B1B' }} onClick={() => insertFooter(c)}>
          {t(mc.insertFooter)}
        </button>
      ) : null}
    </div>
  )
}

type Rte = TextEditing & { insertHTML(html: string): void }

const styleLabels: Record<TextStyle, MessageDescriptor> = { p: mc.styleParagraph, heading: mc.styleHeading, subheading: mc.styleSubheading, small: mc.styleSmall }

/**
 * GrapesJS ends text editing on any mousedown in the page outside the text.
 * Popovers of the text toolbar render in a portal, so they stop the event
 * themselves; `keepFocus` also keeps the focus (and selection) in the text.
 */
function MouseShield({ children, keepFocus }: { children: ReactNode; keepFocus?: boolean }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const stop = (e: Event) => {
      if (keepFocus) e.preventDefault()
      e.stopPropagation()
    }
    el.addEventListener('mousedown', stop)
    return () => el.removeEventListener('mousedown', stop)
  }, [keepFocus])
  return (
    <div ref={ref} className="contents">
      {children}
    </div>
  )
}

function RteToolbar({ host, tick }: { host: HTMLElement | null; tick: number }) {
  const c = useComposer()
  const tr = useT()
  const ed = c.editor
  const [rte, setRte] = useState<{ rte: Rte; el: HTMLElement } | null>(null)
  const [, setSel] = useState(0)
  const [link, setLink] = useState('')
  const [linkOpen, setLinkOpen] = useState(false)
  const range = useRef<Range | null>(null)
  useEffect(() => {
    if (!ed) return
    const on = (view: { el: HTMLElement }, inst: Rte) => setRte({ rte: inst, el: view.el })
    const off = () => setRte(null)
    ed.on('rte:enable', on)
    ed.on('rte:disable', off)
    return () => {
      ed.off('rte:enable', on)
      ed.off('rte:disable', off)
    }
  }, [ed])
  useEffect(() => {
    if (!rte) return
    const doc = rte.rte.doc
    const onSel = () => setSel((n) => n + 1)
    doc.addEventListener('selectionchange', onSel)
    return () => doc.removeEventListener('selectionchange', onSel)
  }, [rte])
  const box = useMemo(() => (ed && rte ? boxOf(ed, rte.el, host) : null), [ed, rte, host, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  if (!ed || !rte || !box) return null
  const doc = rte.rte.doc
  const state = (cmd: string) => {
    try {
      return doc.queryCommandState(cmd)
    } catch {
      return false
    }
  }
  // As in MouseShield: stop GrapesJS ending the edit, and keep the focus in the text
  // (a select still opens: it gives the focus back once a style is picked).
  const keep = (e: MouseEvent) => {
    if (!(e.target instanceof HTMLSelectElement)) e.preventDefault()
    e.stopPropagation()
  }
  // The text lives in the canvas iframe: give it focus back before formatting.
  const inText = (fn: () => void) => {
    const s = rte.rte.selection()
    const r = s && s.rangeCount ? s.getRangeAt(0).cloneRange() : null
    rte.rte.el.focus()
    if (r && s) {
      s.removeAllRanges()
      s.addRange(r)
    }
    fn()
    setSel((n) => n + 1)
  }
  const exec = (cmd: string) => inText(() => rte.rte.exec(cmd))
  const saveRange = () => {
    const s = rte.rte.selection()
    range.current = s && s.rangeCount ? s.getRangeAt(0).cloneRange() : null
  }
  const restoreRange = () => {
    const s = rte.rte.selection()
    if (s && range.current) {
      s.removeAllRanges()
      s.addRange(range.current)
    }
  }
  const currentLink = (): HTMLAnchorElement | null => {
    const s = rte.rte.selection()
    let n: Node | null | undefined = s?.anchorNode
    while (n && n !== rte.el) {
      if ((n as HTMLElement).tagName === 'A') return n as HTMLAnchorElement
      n = n.parentNode
    }
    return null
  }
  const applyLink = () => {
    restoreRange()
    const url = link.trim()
    const a = currentLink()
    if (a) {
      if (url) a.setAttribute('href', url)
      else a.replaceWith(...Array.from(a.childNodes))
    } else if (url) {
      const text = rte.rte.selection()?.toString() || url
      rte.rte.insertHTML(`<a href="${url.replace(/"/g, '&quot;')}">${text.replace(/&/g, '&amp;').replace(/</g, '&lt;')}</a>`)
    }
    setLinkOpen(false)
    rte.rte.el.focus()
  }
  const tokens = c.context.tokens
  return (
    <div className="rte" role="toolbar" aria-label={tr(mc.textFormatting)} style={{ top: Math.max(0, box.top - 40), left: box.left }} onMouseDown={keep}>
      {/* Headings and small print, as Write mode's formatting bar offers them (CO-8). */}
      <select aria-label={tr(mc.textStyle)} value={currentTextStyle(rte.rte)} onChange={(e) => inText(() => applyTextStyle(rte.rte, e.target.value as TextStyle))}>
        {TEXT_STYLES.map((s) => (
          <option key={s} value={s}>
            {tr(styleLabels[s])}
          </option>
        ))}
      </select>
      <span className="rsep" aria-hidden="true" />
      <button type="button" aria-label={tr(mc.bold)} aria-pressed={state('bold')} onClick={() => exec('bold')}>
        <b>B</b>
      </button>
      <button type="button" aria-label={tr(mc.italic)} aria-pressed={state('italic')} onClick={() => exec('italic')}>
        <i style={{ fontFamily: 'Georgia, serif' }}>I</i>
      </button>
      <Popover
        open={linkOpen}
        onOpenChange={(o) => {
          if (o) {
            saveRange()
            setLink(currentLink()?.getAttribute('href') ?? '')
          }
          setLinkOpen(o)
        }}
        label={tr(mc.link)}
        className="w-[300px]"
        trigger={
          <button type="button" aria-label={tr(mc.link)} aria-pressed={!!currentLink()}>
            {tr(mc.link)}
          </button>
        }
      >
        <MouseShield>
          <form
            className="flex flex-col gap-2 p-1"
            onSubmit={(e) => {
              e.preventDefault()
              applyLink()
            }}
          >
            <label className="lbl">
              {tr(mc.linkUrl)}
              <input className="in mono" value={link} onChange={(e) => setLink(e.target.value)} placeholder={'https://'} autoFocus />
            </label>
            <span className="k">{tr(mc.linkHint, { token: '{{ unsubscribe_url }}' })}</span>
            {/* A campaign links to a survey with each contact's token (design expansion, X11). */}
            {c.kind !== 'transactional' ? <SurveyLinkChoices value={link} onPick={setLink} /> : null}
            <div className="flex justify-end gap-2">
              <button type="button" className="btn sm" onClick={() => setLinkOpen(false)}>
                {tr(common.cancel)}
              </button>
              <button type="submit" className="btn sm pri">
                {tr(m.apply)}
              </button>
            </div>
          </form>
        </MouseShield>
      </Popover>
      <span className="rsep" aria-hidden="true" />
      <button type="button" aria-label={tr(mc.bulletedList)} title={tr(mc.bulletedList)} aria-pressed={state('insertUnorderedList')} onClick={() => exec('insertUnorderedList')}>
        <svg viewBox="0 0 16 16" strokeWidth={1.7} strokeLinecap="round" aria-hidden="true">
          <path d="M6 4h8M6 8h8M6 12h8" />
          <circle cx="2.5" cy="4" r=".6" />
          <circle cx="2.5" cy="8" r=".6" />
          <circle cx="2.5" cy="12" r=".6" />
        </svg>
      </button>
      <button type="button" aria-label={tr(mc.numberedList)} title={tr(mc.numberedList)} aria-pressed={state('insertOrderedList')} onClick={() => exec('insertOrderedList')}>
        <svg viewBox="0 0 16 16" strokeWidth={1.6} strokeLinecap="round" aria-hidden="true">
          <path d="M6.5 4h7.5M6.5 8h7.5M6.5 12h7.5M2 3l1-.5v3M2 7.5h2L2 10h2M2 12h1.6l-.9 1 .9 1H2" />
        </svg>
      </button>
      <span className="rsep" aria-hidden="true" />
      <Popover
        label={tr(mc.insertToken)}
        className="w-[280px]"
        onOpenChange={(o) => o && saveRange()}
        trigger={
          <button type="button" aria-label={tr(mc.insertToken)}>
            {'{{ }}'}
          </button>
        }
      >
        <span className="mh">{tr(mc.insertTokenHeading)}</span>
        <MouseShield keepFocus>
          <div className="flex max-h-[280px] flex-col overflow-auto">
            {tokens.map((t, i) => [
              // Grow's tokens under their own heading (design expansion, X15).
              t.kind === 'referral' && tokens[i - 1]?.kind !== 'referral' ? (
                <span key="referral-group" className="mh">
                  {tt(growMessages.tokensReferral)}
                </span>
              ) : null,
              // Earn's paid.* tokens and the reader's links (design expansion, X16).
              t.kind === 'paid' && tokens[i - 1]?.kind !== 'paid' ? (
                <span key="paid-group" className="mh">
                  {tt(earnComposer.tokensPaid)}
                </span>
              ) : null,
              <button
                key={t.key}
                type="button"
                className="flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-left hover:bg-ground"
                style={{ border: 0, background: 'none', font: 'inherit', cursor: 'pointer' }}
                onClick={() => {
                  restoreRange()
                  const token = t.fallback !== undefined && t.fallback !== '' ? `{{ ${t.key} | default: "${t.fallback}" }}` : `{{ ${t.key} }}`
                  // A chip, as the canvas shows tokens (it exports as the token).
                  rte.rte.insertHTML(`${chipHtml(token, { label: t.label, fallback: t.fallback || undefined, unknown: false })}&nbsp;`)
                }}
              >
                <span>{t.label}</span>
                <span className="mono k min-w-0 text-right [overflow-wrap:anywhere]">{t.key}</span>
              </button>,
            ])}
          </div>
        </MouseShield>
      </Popover>
      <RewriteActions />
    </div>
  )
}

/** Fits the device width into the canvas column (tablet and phone screens). */
function useZoom(editor: Editor | null, stage: HTMLElement | null, device: DeviceId): number {
  const [zoom, setZoom] = useState(1)
  useLayoutEffect(() => {
    if (!stage) return
    const width = parseInt(DEVICES[device].width, 10)
    const measure = () => {
      const cs = getComputedStyle(stage)
      const avail = stage.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight)
      setZoom(avail >= width ? 1 : Math.max(0.4, Math.floor((avail / width) * 100) / 100))
    }
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(stage)
    return () => ro.disconnect()
  }, [stage, device])
  useEffect(() => {
    if (editor && Math.abs(editor.Canvas.getZoom() - zoom * 100) > 0.5) editor.Canvas.setZoom(zoom * 100)
  }, [editor, zoom])
  return zoom
}

/** Keeps the scaled frame from reserving its unscaled height. */
function useZoomHeight(editor: Editor | null, zoom: number, tick: number, device: DeviceId, host: HTMLElement | null) {
  useLayoutEffect(() => {
    const frames = editor?.Canvas.getElement()?.querySelector<HTMLElement>('.gjs-cv-canvas__frames')
    const frame = editor?.Canvas.getFrameEl()
    if (!frames || !frame) return
    const h = frame.offsetHeight
    const width = parseInt(DEVICES[device].width, 10)
    const zoomed = zoom < 1
    // Scaled from its top-left corner and centred by hand: a frame wider
    // than the column can't be centred with auto margins.
    frames.style.width = zoomed ? `${width}px` : ''
    frames.style.transformOrigin = zoomed ? '0 0' : ''
    frames.style.marginLeft = zoomed && host ? `${Math.max(0, Math.floor((host.clientWidth - width * zoom) / 2))}px` : ''
    frames.style.marginBottom = zoomed ? `${-Math.round((1 - zoom) * h)}px` : ''
  }, [editor, zoom, tick, device, host])
}

/** GrapesJS's frame has no title of its own; screen readers announce the canvas by it. */
function useFrameTitle(editor: Editor | null) {
  useEffect(() => {
    if (!editor) return
    const set = () => editor.Canvas.getFrameEl()?.setAttribute('title', tt(mc.frameTitle))
    set()
    editor.on('canvas:frame:load', set)
    return () => {
      editor.off('canvas:frame:load', set)
    }
  }, [editor])
}

/** CSS of the head's mj-style (responsive rules) and the dark-mode check, inside the canvas frame. */
function useFrameStyles(editor: Editor | null, head: string, dark: boolean) {
  useEffect(() => {
    const doc = editor?.Canvas.getDocument()
    if (!doc?.head) return
    let el = doc.getElementById('rc-head-style') as HTMLStyleElement | null
    if (!el) {
      el = doc.createElement('style')
      el.id = 'rc-head-style'
      doc.head.appendChild(el)
    }
    el.textContent = headCss(head)
    doc.documentElement.classList.toggle('rc-dark', dark)
  }, [editor, head, dark])
}

/**
 * Parts the transactional mode replaces: the toolbar, the save-state cards,
 * the empty canvas and what sits under the email (the campaign's missing
 * footer warning). Each one left out keeps the campaign's.
 */
export interface CanvasSlots {
  /** Gets the canvas zoom (1 = 100 %). */
  toolbar?: (zoom: number) => ReactNode
  cards?: ReactNode
  empty?: ReactNode
  below?: ReactNode
}

export function CanvasArea({
  containerRef,
  savedCard,
  onCloseSaved,
  onReview,
  slots,
}: {
  containerRef: RefObject<HTMLDivElement | null>
  savedCard: { version: number; at: string; template?: string } | null
  onCloseSaved: () => void
  onReview: () => void
  slots?: CanvasSlots
}) {
  const c = useComposer()
  const t = useT()
  // Digest mode's "6 items" covers the canvas with the whole issue (the editor stays loaded underneath).
  const whole = useDigestMode()?.preview === 'six'
  const [host, setHost] = useState<HTMLDivElement | null>(null)
  const [stage, setStage] = useState<HTMLDivElement | null>(null)
  const ed = c.editor
  const zoom = useZoom(ed, stage, c.device)
  const tick = useCanvasTick(ed, host)
  useZoomHeight(ed, zoom, tick, c.device, host)
  useFrameTitle(ed)
  useFrameStyles(ed, c.head, c.darkCanvas && c.device === 'mobile')
  const frameWidth = parseInt(DEVICES[c.device].width, 10) * zoom
  const setContainer = useCallback(
    (el: HTMLDivElement | null) => {
      containerRef.current = el
    },
    [containerRef],
  )
  // When the canvas scrolls (on phones), it's a named tab stop; it stays the skip link's target.
  const [canvasRef, canvasScroll] = useScrollRegion<HTMLDivElement>(t(m.canvasRegion), true)

  if (c.isEmpty) {
    return (
      <>
        <div className="cmp-tools" style={{ opacity: 0.55 }} aria-hidden="true" inert>
          <DeviceSeg dim />
          <ViewSeg dim />
        </div>
        {slots?.empty ?? <EmptyGallery />}
      </>
    )
  }
  return (
    <>
      {slots?.toolbar ? slots.toolbar(zoom) : <Toolbar zoom={zoom} />}
      <div ref={canvasRef} className="cmp-canvas" id={c.view === 'visual' ? 'main' : undefined} tabIndex={-1} {...canvasScroll}>
        {whole ? <WholeIssue /> : null}
        <div
          className="cmp-stage"
          ref={setStage}
          style={whole ? { visibility: 'hidden', position: 'absolute', inset: '0 0 auto 0', height: 0, overflow: 'hidden' } : undefined}
          aria-hidden={whole || undefined}
        >
          {slots?.cards ?? <StateCards saved={savedCard} onCloseSaved={onCloseSaved} onReview={onReview} />}
          <DigestCanvasCards onReview={onReview} />
          {/* A digest's template and issues have one language: no language banner. */}
          {c.campaign.digestRole ? null : <ComposerLanguageBanner width={frameWidth} />}
          <div ref={setHost} className="relative">
            <div ref={setContainer} role="region" aria-label={t(mc.canvasRegion, { device: c.device })} />
            {ed ? null : (
              // ComposerStates: "Compiling Weekly Digest #142 from MJML · Version 15 · 212 lines".
              <div className="mx-auto flex items-center justify-center gap-3 bg-surface" style={{ width: frameWidth, minHeight: 320, boxShadow: '0 1px 3px rgba(11,18,32,.12)' }} aria-busy="true" role="status">
                <span className="spin" style={{ width: 24, height: 24 }} aria-hidden="true" />
                <div className="flex flex-col">
                  <span className="font-semibold">{c.name ? t(mc.compilingName, { name: c.name }) : t(mc.compilingCampaign)}</span>
                  <span className="k">{t(mc.compilingDetail, { version: c.campaign.version, lines: count(c.campaign.mjml.split('\n').length) })}</span>
                </div>
              </div>
            )}
            <WarningOverlays host={host} tick={tick} />
            <AccessibilityOverlays host={host} tick={tick} boxOf={boxOf} />
            <ConditionalOverlays host={host} tick={tick} boxOf={boxOf} />
            <WebOverlays host={host} tick={tick} boxOf={boxOf} />
            <LanguageTags host={host} tick={tick} />
            <SelectionOverlay host={host} tick={tick} />
            <RteToolbar host={host} tick={tick} />
            <LiquidPopover host={host} tick={tick} boxOf={boxOf} />
          </div>
          {ed ? (
            <div className="mx-auto" style={{ width: frameWidth }}>
              {slots && 'below' in slots ? slots.below : <MissingFooter width={frameWidth} />}
              <AltTextNote />
            </div>
          ) : null}
        </div>
      </div>
      {/* The versions of conditional content, over the bottom of the canvas. */}
      <VersionsPanel />
    </>
  )
}
