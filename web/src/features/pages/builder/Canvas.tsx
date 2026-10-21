import type { Component, Editor } from 'grapesjs'
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode, type RefObject } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Popover } from '@/components/ui/Menu'
import { formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { modKeyLabel } from '@/lib/shortcuts'
import { useBuilder, type PageCheck } from './context'
import { useForms, useProducts, useTheme } from './data'
import { canMove } from './editor/actions'
import { attrOf, attrsOf, findById, isBlock, liveTagOf } from './editor/nodes'
import { DEVICE_WIDTH, type PageDevice } from './editor/pageEditor'
import { UiIcon } from './icons'
import { selectionLabel } from './labels'
import { parseList, splitAt, zonedInstant } from './model'
import { m } from './messages'
import { whenText } from './wording'

interface Box {
  top: number
  left: number
  width: number
  height: number
}

/** Where an element of the canvas frame is, relative to `host` (as the composer measures it). */
function boxOf(editor: Editor, el: Element | null | undefined, host: HTMLElement | null): Box | null {
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
      raf = requestAnimationFrame(() => setTick((x) => x + 1))
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

/** The stage's side gutter (builder.css): 70% of 1200 px fits the 1440 board's canvas column; the stacked layout keeps 21 px (66% at 834), phones 8 px. */
function gutter(): number {
  if (typeof window === 'undefined') return 4
  if (window.matchMedia('(max-width: 767px)').matches) return 8
  return window.matchMedia('(max-width: 1100px)').matches ? 21 : 4
}

/** Fits the device's width into the canvas column. */
function useZoom(editor: Editor | null, stage: HTMLElement | null, device: PageDevice): number {
  const [zoom, setZoom] = useState(1)
  useLayoutEffect(() => {
    if (!stage) return
    const width = DEVICE_WIDTH[device]
    const measure = () => {
      const avail = stage.clientWidth - 2 * gutter()
      setZoom(avail >= width ? 1 : Math.max(0.25, Math.floor((avail / width) * 100) / 100))
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

/** Keeps the scaled frame from reserving its unscaled height, and centres it. */
function useZoomHeight(editor: Editor | null, zoom: number, tick: number, device: PageDevice, host: HTMLElement | null) {
  useLayoutEffect(() => {
    const frames = editor?.Canvas.getElement()?.querySelector<HTMLElement>('.gjs-cv-canvas__frames')
    const frame = editor?.Canvas.getFrameEl()
    if (!frames || !frame) return
    const h = frame.offsetHeight
    const width = DEVICE_WIDTH[device]
    const zoomed = zoom < 1
    frames.style.width = zoomed ? `${width}px` : ''
    frames.style.transformOrigin = zoomed ? '0 0' : ''
    frames.style.marginLeft = zoomed && host ? `${Math.max(0, Math.floor((host.clientWidth - width * zoom) / 2))}px` : ''
    frames.style.marginBottom = zoomed ? `${-Math.round((1 - zoom) * h)}px` : ''
  }, [editor, zoom, tick, device, host])
}

/** Now, every 30 seconds (countdowns end while the page is open). */
function useNow(): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const h = window.setInterval(() => setNow(new Date()), 30_000)
    return () => window.clearInterval(h)
  }, [])
  return now
}

function Toolbar({ zoom }: { zoom: number }) {
  const t = useT()
  const b = useBuilder()
  const theme = useTheme()
  const um = b.editor?.UndoManager
  void b.rev
  const canUndo = Boolean(um?.hasUndo()) && b.canEdit
  const canRedo = Boolean(um?.hasRedo()) && b.canEdit
  const mod = modKeyLabel()
  const device = b.device === 'desktop' ? t(m.deviceDesktop) : b.device === 'tablet' ? t(m.deviceTablet) : t(m.devicePhone)
  return (
    <div className="cmp-tools pb-tools">
      <button type="button" className="btn sm ic" aria-label={t(m.undo)} title={t(m.undoTitle, { mod })} disabled={!canUndo} onClick={() => um?.undo()}>
        <UiIcon name="undo" />
      </button>
      <button type="button" className="btn sm ic" aria-label={t(m.redo)} title={t(m.redoTitle, { mod })} disabled={!canRedo} onClick={() => um?.redo()}>
        <UiIcon name="redo" />
      </button>
      <span className="k">{t(m.deviceInfo, { device, width: DEVICE_WIDTH[b.device], zoom: Math.round(zoom * 100) })}</span>
      {theme ? (
        <span className="chip pb-chip">
          <span className="pb-swatch" style={{ background: theme.accent }} aria-hidden="true" />
          {t(m.themeChip, { name: theme.name })}
        </span>
      ) : null}
      <span className="k" style={{ marginLeft: 'auto' }}>
        {t(m.shortcutsHint, { mod })}
      </span>
    </div>
  )
}

function Card({ tone, badge, title, children, actions }: { tone: 'ok' | 'warn' | 'bad'; badge: string; title: string; children: ReactNode; actions: ReactNode }) {
  const border = tone === 'ok' ? '#BBF7D0' : tone === 'warn' ? '#FCD34D' : '#FCA5A5'
  return (
    <div className="card pb-card" role={tone === 'ok' ? 'status' : 'alert'} style={{ borderColor: border }}>
      <div className="flex flex-wrap items-start gap-2.5">
        <Badge tone={tone} dot>
          {badge}
        </Badge>
        <span className="min-w-0 font-semibold" style={{ flex: '1 1 260px' }}>
          {title}
        </span>
      </div>
      <span className="k" style={{ color: '#374151' }}>
        {children}
      </span>
      <div className="flex flex-wrap gap-2">{actions}</div>
    </div>
  )
}

/**
 * Save states above the page (PageBuilderStates · Autosave): the
 * confirmation of a manual save, a conflict with someone else's save, an
 * outage and a failed save.
 */
function StateCards() {
  const t = useT()
  const b = useBuilder()
  const s = b.status
  const cards: ReactNode[] = []
  if (s.kind === 'conflict') {
    const who = s.theirs.by || t(m.someone)
    const next = s.theirs.version + 1
    const blocks = b.conflict?.both.length ? b.conflict.both : b.conflict?.theirs ?? []
    const words = blocks.length ? formatList(blocks) : ''
    cards.push(
      <Card
        key="conflict"
        tone="warn"
        badge={t(m.conflictBadge)}
        title={t(m.conflictTitle, { name: who, version: s.theirs.version, time: s.theirs.at ? whenText(t, s.theirs.at) : '' })}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void b.save({ force: true, manual: true })}>
              {t(m.keepMine, { version: next })}
            </button>
            <button type="button" className="btn sm" onClick={() => void b.reloadFromServer()}>
              {t(m.loadTheirs, { name: who, version: s.theirs.version })}
            </button>
            <button type="button" className="btn sm" onClick={() => b.openDialog({ kind: 'compare' })}>
              {t(m.compare)}
            </button>
          </>
        }
      >
        {b.conflict?.both.length ? t(m.conflictBoth, { blocks: words, name: who, next }) : blocks.length ? t(m.conflictTheirs, { blocks: words, name: who, next }) : t(m.conflictPlain, { name: who, next })}
      </Card>,
    )
  } else if (s.kind === 'offline') {
    cards.push(
      <Card
        key="offline"
        tone="bad"
        badge={t(m.offlineBadge)}
        title={t(m.offlineTitle, { version: s.version + 1 })}
        actions={
          <button type="button" className="btn sm pri" onClick={() => void b.save({ manual: true })}>
            {t(m.retryNow)}
          </button>
        }
      >
        {t(m.offlineBody, { time: whenText(t, s.since) })}
      </Card>,
    )
  } else if (s.kind === 'error') {
    cards.push(
      <Card
        key="error"
        tone="bad"
        badge={t(m.offlineBadge)}
        title={t(m.errorTitle, { version: s.version + 1 })}
        actions={
          <button type="button" className="btn sm pri" onClick={() => void b.save({ manual: true })}>
            {t(m.retryNow)}
          </button>
        }
      >
        {t(m.errorBody, { message: s.message })}
      </Card>,
    )
  } else if (b.savedCard) {
    cards.push(
      <Card
        key="saved"
        tone="ok"
        badge={t(m.savedBadge)}
        title={t(m.savedTitle, { version: b.savedCard.version, time: whenText(t, b.savedCard.at) })}
        actions={
          <>
            <button type="button" className="btn sm" onClick={b.closeSavedCard}>
              {t(m.backToPage)}
            </button>
            <button type="button" className="btn sm" onClick={() => b.openDialog({ kind: 'versions' })}>
              {t(m.versionHistory)}
            </button>
          </>
        }
      >
        {t(m.savedBody)}
      </Card>,
    )
  }
  if (!cards.length) return null
  return <div className="pb-cards">{cards}</div>
}

/** The blocks a blocking check is about: a paused or deleted form, a product that isn't published. */
function blockedChecks(checks: readonly PageCheck[]): PageCheck[] {
  return checks.filter((c) => c.status === 'block' && (c.key === 'form' || c.key === 'product') && c.ref)
}

function checkFor(c: Component, blocked: readonly PageCheck[]): PageCheck | undefined {
  const tag = liveTagOf(c)
  if (tag === 'rc-form') return blocked.find((x) => x.key === 'form' && x.ref === attrOf(c, 'form'))
  if (tag === 'rc-product') return blocked.find((x) => x.key === 'product' && x.ref === attrOf(c, 'product'))
  return undefined
}

function SelectionOverlay({ host, tick, zoom, now }: { host: HTMLElement | null; tick: number; zoom: number; now: Date }) {
  const t = useT()
  const b = useBuilder()
  const ed = b.editor
  const sel = b.selected
  const products = useProducts()
  const forms = useForms()
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
  if (!ed || !sel || !box || sel.get('type') === 'wrapper') return null
  const off = 8 * zoom
  const bad = Boolean(checkFor(sel, blockedChecks(b.page.checks)))
  const label = selectionLabel(t, sel, { products: products.items, forms: forms.items, now })
  return (
    <div className={cn('cmp-sel pb-sel', bad && 'bad', !b.canEdit && 'ro')} style={{ top: box.top - off, left: box.left - off, width: box.width + 2 * off, height: box.height + 2 * off }}>
      <div className="cmp-sel-box" style={{ borderRadius: 16 * zoom }} aria-hidden="true" />
      {rte ? null : (
        <span className="lab" title={label}>
          {label}
        </span>
      )}
      {b.canEdit && !rte ? (
        <div className="tb" role="toolbar" aria-label={t(m.blockActions)}>
          <button type="button" aria-label={t(m.moveUp)} title={t(m.moveUp)} disabled={!canMove(sel, -1)} onClick={() => b.moveSelected(-1)}>
            <UiIcon name="up" color="#fff" />
          </button>
          <button type="button" aria-label={t(m.moveDown)} title={t(m.moveDown)} disabled={!canMove(sel, 1)} onClick={() => b.moveSelected(1)}>
            <UiIcon name="down" color="#fff" />
          </button>
          <button type="button" aria-label={t(m.duplicate)} title={t(m.duplicate)} disabled={sel.get('copyable') === false} onClick={b.duplicateSelected}>
            <UiIcon name="duplicate" color="#fff" />
          </button>
          <button type="button" aria-label={t(m.delete)} title={t(m.delete)} disabled={sel.get('removable') === false} onClick={b.removeSelected}>
            <UiIcon name="trash" color="#fff" />
          </button>
        </div>
      ) : null}
    </div>
  )
}

type Note = {
  key: string
  box: Box
  kind: 'revealed' | 'hides' | 'hidden' | 'blocked'
  label: string
  block: boolean
  /** What sits right above the block in its column (its previous element). */
  above: Box | null
}

/**
 * What the canvas marks besides the selection: the blocks a countdown
 * reveals or hides (PageBuilderStates · Launch), and the blocks a blocking
 * check is about, outlined in red with what's wrong (a paused form).
 */
function Annotations({ host, tick, zoom, now }: { host: HTMLElement | null; tick: number; zoom: number; now: Date }) {
  const t = useT()
  const b = useBuilder()
  const ed = b.editor
  const products = useProducts()
  const forms = useForms()
  const notes = useMemo(() => {
    if (!ed) return []
    const out: Note[] = []
    const wrapper = ed.getWrapper()
    const seen = new Set<string>()
    const push = (c: Component, kind: Note['kind'], label: string) => {
      if (c === b.selected || seen.has(c.cid)) return
      const box = boxOf(ed, c.getEl(), host)
      if (!box) return
      seen.add(c.cid)
      out.push({ key: `${kind}-${c.cid}`, box, kind, label, block: isBlock(c), above: boxOf(ed, c.getEl()?.previousElementSibling, host) })
    }
    const blocked = blockedChecks(b.page.checks)
    wrapper?.onAll((c: Component) => {
      if (blocked.length && checkFor(c, blocked)) push(c, 'blocked', selectionLabel(t, c, { products: products.items, forms: forms.items, now }))
    })
    wrapper?.onAll((c: Component) => {
      if (liveTagOf(c) !== 'rc-countdown') return
      const a = attrsOf(c)
      if (a.end !== 'reveal') return
      const { date, time } = splitAt(a.at)
      const at = zonedInstant(date, time, a.tz || 'UTC')
      const ended = Boolean(at && at.getTime() <= now.getTime())
      // Once it ended, what it revealed shows as readers see it: nothing to mark.
      for (const id of ended ? [] : parseList(a.reveal)) {
        const x = findById(wrapper, id)
        if (x) push(x, 'revealed', t(m.revealedWhenEnds))
      }
      for (const id of parseList(a.hide)) {
        const x = findById(wrapper, id)
        if (x) push(x, ended ? 'hidden' : 'hides', t(ended ? m.hiddenSinceEnded : m.hiddenWhenEnds))
      }
    })
    return out
  }, [ed, host, tick, b.page.checks, b.selected, now, t, products.items, forms.items]) // eslint-disable-line react-hooks/exhaustive-deps
  const selBox = useMemo(() => (ed && b.selected ? boxOf(ed, b.selected.getEl(), host) : null), [ed, b.selected, host, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  const off = 8 * zoom
  /** Whether the room above a block is taken (the selection or another marked block right above it): its pill goes inside. */
  const crowded = (n: Note) => {
    const top = n.box.top - off - 30
    const left = n.box.left - off - 2
    const right = left + n.label.length * 6.5 + 44
    const others = [selBox, n.above, ...notes.filter((x) => x !== n).map((x) => x.box)]
    return others.some((o) => o && o.left - off < right && o.left + o.width + off > left && o.top - off < top + 24 && o.top + o.height + off > top)
  }
  return (
    <>
      {notes.map((n) => {
        const dim = n.kind === 'revealed' || n.kind === 'hidden'
        if (n.kind === 'blocked')
          return (
            <div key={n.key} className="pb-err" style={{ top: n.box.top - off, left: n.box.left - off, width: n.box.width + 2 * off, height: n.box.height + 2 * off, borderRadius: 16 * zoom }}>
              <span className="lab">{n.label}</span>
            </div>
          )
        // A block says what happens to it just above its top left corner (a dimmed one in a pill, inside
        // its top right corner when the room above is taken). Text lines that swap (an eyebrow, a lead)
        // are only dimmed.
        const place = !dim
          ? { top: n.box.top - 18, left: n.box.left }
          : crowded(n)
            ? { top: n.box.top + 6, left: n.box.left + n.box.width - 6, transform: 'translateX(-100%)' }
            : { top: n.box.top - off - 30, left: n.box.left - off - 2 }
        return (
          <div key={n.key}>
            {dim ? <div className="pb-dim" style={{ top: n.box.top - off, left: n.box.left - off, width: n.box.width + 2 * off, height: n.box.height + 2 * off, borderRadius: 16 * zoom }} aria-hidden="true" /> : null}
            {n.block ? (
              <span className={cn('pb-hbtag', !dim && 'plain')} style={place}>
                {dim ? <UiIcon name="eyeOff" size={12} /> : null}
                {n.label}
              </span>
            ) : null}
          </div>
        )
      })}
    </>
  )
}

type Rte = { exec(cmd: string, value?: string): void; insertHTML(html: string): void; selection(): Selection | null; doc: Document; el: HTMLElement }

/** GrapesJS ends text editing on any mousedown outside the text: the link popover stops the event itself. */
function MouseShield({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const stop = (e: Event) => e.stopPropagation()
    el.addEventListener('mousedown', stop)
    return () => el.removeEventListener('mousedown', stop)
  }, [])
  return (
    <div ref={ref} className="contents">
      {children}
    </div>
  )
}

/** Bold, italic and links for the text being edited (the composer's text toolbar). */
function RteToolbar({ host, tick }: { host: HTMLElement | null; tick: number }) {
  const t = useT()
  const b = useBuilder()
  const ed = b.editor
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
  const keep = (e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
  }
  const exec = (cmd: string) => {
    const s = rte.rte.selection()
    const r = s && s.rangeCount ? s.getRangeAt(0).cloneRange() : null
    rte.rte.el.focus()
    if (r && s) {
      s.removeAllRanges()
      s.addRange(r)
    }
    rte.rte.exec(cmd)
    setSel((n) => n + 1)
  }
  const currentLink = (): HTMLAnchorElement | null => {
    let n: Node | null | undefined = rte.rte.selection()?.anchorNode
    while (n && n !== rte.el) {
      if ((n as HTMLElement).tagName === 'A') return n as HTMLAnchorElement
      n = n.parentNode
    }
    return null
  }
  const applyLink = () => {
    const s = rte.rte.selection()
    if (s && range.current) {
      s.removeAllRanges()
      s.addRange(range.current)
    }
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
  return (
    <div className="rte" role="toolbar" aria-label={t(m.textTools)} style={{ top: Math.max(0, box.top - 40), left: box.left }} onMouseDown={keep}>
      <button type="button" aria-label={t(m.bold)} aria-pressed={state('bold')} onClick={() => exec('bold')}>
        <b aria-hidden="true">B</b>
      </button>
      <button type="button" aria-label={t(m.italic)} aria-pressed={state('italic')} onClick={() => exec('italic')}>
        <i aria-hidden="true" style={{ fontFamily: 'Georgia, serif' }}>
          I
        </i>
      </button>
      <Popover
        open={linkOpen}
        onOpenChange={(o) => {
          if (o) {
            const s = rte.rte.selection()
            range.current = s && s.rangeCount ? s.getRangeAt(0).cloneRange() : null
            setLink(currentLink()?.getAttribute('href') ?? '')
          }
          setLinkOpen(o)
        }}
        label={t(m.link)}
        className="w-[300px]"
        trigger={
          <button type="button" aria-label={t(m.link)} aria-pressed={Boolean(currentLink())}>
            {t(m.link)}
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
              {t(m.linkUrl)}
              <input className="in mono" value={link} onChange={(e) => setLink(e.target.value)} placeholder={'https://'} autoFocus />
            </label>
            <span className="k">{t(m.linkUrlHint)}</span>
            <div className="flex justify-end gap-2">
              <button type="button" className="btn sm" onClick={() => setLinkOpen(false)}>
                {t(common.cancel)}
              </button>
              <button type="submit" className="btn sm pri">
                {t(m.apply)}
              </button>
            </div>
          </form>
        </MouseShield>
      </Popover>
    </div>
  )
}

/** The canvas column (PageBuilder): undo and redo, the device and zoom, the theme; the page on its theme with the selection outlined. */
export function Canvas({ containerRef }: { containerRef: RefObject<HTMLDivElement | null> }) {
  const t = useT()
  const b = useBuilder()
  const [stage, setStage] = useState<HTMLDivElement | null>(null)
  const [host, setHost] = useState<HTMLDivElement | null>(null)
  const zoom = useZoom(b.editor, stage, b.device)
  const tick = useCanvasTick(b.editor, host)
  const now = useNow()
  useZoomHeight(b.editor, zoom, tick, b.device, host)
  const width = DEVICE_WIDTH[b.device]
  const device = b.device === 'desktop' ? t(m.deviceDesktop) : b.device === 'tablet' ? t(m.deviceTablet) : t(m.devicePhone)
  const setContainer = useCallback(
    (el: HTMLDivElement | null) => {
      containerRef.current = el
    },
    [containerRef],
  )
  const empty = b.editor ? !b.editor.getWrapper()?.components().length : false
  return (
    <section className="cmp-main pb-main" aria-label={t(m.canvasRegion, { device, width })}>
      <Toolbar zoom={zoom} />
      <div className="cmp-canvas pb-canvas" id="main" tabIndex={-1}>
        <div className="cmp-stage pb-stage" ref={setStage}>
          <StateCards />
          <div ref={setHost} className="relative">
            <div ref={setContainer} />
            {b.editor ? null : (
              <div className="pb-loading" style={{ width: width * zoom }} aria-busy="true" role="status">
                <span className="spin" aria-hidden="true" />
                <span className="font-semibold">{t(m.loadingCanvas)}</span>
              </div>
            )}
            {empty ? (
              <p className="pb-emptyhint k" style={{ width: width * zoom }}>
                {t(m.emptyPage)}
              </p>
            ) : null}
            <Annotations host={host} tick={tick} zoom={zoom} now={now} />
            <SelectionOverlay host={host} tick={tick} zoom={zoom} now={now} />
            <RteToolbar host={host} tick={tick} />
          </div>
        </div>
      </div>
    </section>
  )
}
