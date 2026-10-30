import { useEffect, useMemo, useRef, useState, type RefObject } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'
import { useT } from '@/i18n'
import { mh } from './chrome.messages'
import { stripActive } from './inert'

type Item = Schemas['MJMLThumbItem']

/** The width an email is laid out at; its picture is drawn smaller. */
const EMAIL_WIDTH = 600

/**
 * An email's HTML as a picture (the gallery's previews, R4.U4): nothing in
 * it runs, links, takes focus or scrolls. The server already took most of
 * it out of a template's preview; a campaign's preview comes as a reader
 * sees it, so it's done here too.
 */
export function pictureDocument(html: string): string {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  stripActive(doc.documentElement)
  doc.querySelectorAll('form, input, button, select, textarea, audio, video').forEach((el) => el.remove())
  doc.querySelectorAll('a[href], area[href]').forEach((el) => el.removeAttribute('href'))
  doc.querySelectorAll('[tabindex], [contenteditable], [autofocus], [accesskey]').forEach((el) => {
    for (const a of ['tabindex', 'contenteditable', 'autofocus', 'accesskey']) el.removeAttribute(a)
  })
  const style = doc.createElement('style')
  style.textContent = 'html,body{overflow:hidden!important;pointer-events:none}'
  doc.head.appendChild(style)
  return `<!doctype html>${doc.documentElement.outerHTML}`
}

/** Whether an element has come near the screen (once): what loads lazily waits for it. */
function useSeen<T extends Element>(): [RefObject<T | null>, boolean] {
  const ref = useRef<T>(null)
  // Without IntersectionObserver everything counts as seen.
  const [seen, setSeen] = useState(() => typeof IntersectionObserver === 'undefined')
  useEffect(() => {
    const el = ref.current
    if (!el || seen) return
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setSeen(true)
          io.disconnect()
        }
      },
      { rootMargin: '200px' },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [seen])
  return [ref, seen]
}

/** The size of a box as the layout changes it (the gallery's columns reflow). */
function useSize<T extends HTMLElement>(): [RefObject<T | null>, { w: number; h: number }] {
  const ref = useRef<T>(null)
  const [size, setSize] = useState({ w: 0, h: 0 })
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(() => setSize({ w: el.clientWidth, h: el.clientHeight }))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  return [ref, size]
}

/**
 * An email drawn small, as a picture (Composer and ComposerEmpty boards):
 * laid out at 600 px in a sandboxed frame, scaled to the tile and cropped
 * to its height. Nothing in it runs, takes focus or takes clicks; the
 * tile's button names the template.
 */
export function LivePicture({ html, label, small }: { html: string; label: string; small?: boolean }) {
  const [box, size] = useSize<HTMLSpanElement>()
  const doc = useMemo(() => pictureDocument(html), [html])
  const scale = size.w / EMAIL_WIDTH
  return (
    <div className={`thumb mini live${small ? ' small' : ''}`} aria-hidden="true">
      <span ref={box} className="mpic">
        {scale > 0 ? (
          <iframe
            title={label}
            srcDoc={doc}
            sandbox=""
            tabIndex={-1}
            inert
            loading="lazy"
            style={{ width: EMAIL_WIDTH, height: Math.ceil(size.h / scale), transform: `scale(${scale})` }}
          />
        ) : null}
      </span>
    </div>
  )
}

/**
 * A template's picture in the gallery (R4.U4): its real email, compiled by
 * the server, once the tile comes near the screen; the miniature drawn
 * from its parts until then, if the preview can't be made, and for Blank.
 */
export function TemplatePicture({ id, name, thumb, blank, small }: { id: string; name: string; thumb: Item[]; blank?: boolean; small?: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const [ref, seen] = useSeen<HTMLDivElement>()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/templates/{templateId}/preview',
    { params: { path: { workspace: slug, templateId: id } } },
    { enabled: seen && !blank, staleTime: 5 * 60_000, retry: false },
  )
  return (
    <div ref={ref} className="tpl-pic">
      {q.data?.html ? <LivePicture html={q.data.html} label={t(mh.thumbPreview, { name })} small={small} /> : <Thumb items={thumb} blank={blank} small={small} />}
    </div>
  )
}

/** The last sent campaign's picture ("Copy of …"), as a reader without a first name sees it. */
export function CampaignPicture({ id, name, thumb }: { id: string; name: string; thumb: Item[] }) {
  const { slug } = useWorkspace()
  const t = useT()
  const [ref, seen] = useSeen<HTMLDivElement>()
  const q = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/preview',
    { params: { path: { workspace: slug, campaignId: id } }, body: { sample: 'no_first_name' } },
    { enabled: seen, staleTime: 5 * 60_000, retry: false },
  )
  return (
    <div ref={ref} className="tpl-pic">
      {q.data?.html ? <LivePicture html={q.data.html} label={t(mh.thumbPreview, { name })} /> : <Thumb items={thumb} />}
    </div>
  )
}

/** An image in a miniature email: a grey box with a mountain, as the boards draw images. */
function MiniImage({ h }: { h?: number }) {
  return (
    <span className="mimg" style={{ height: h }}>
      <svg viewBox="0 0 24 14" aria-hidden="true">
        <path d="M2 13 9 5l5 5 3-3 5 6z" />
      </svg>
    </span>
  )
}

type Kind = NonNullable<Item['kind']>

/**
 * The kind of a part saved before parts had one (templates saved earlier
 * store their thumbnail), from the sizes the server gave each kind.
 */
function withKind(it: Item): Item {
  if (it.kind) return it
  if (it.row?.length) return { ...it, row: it.row.map((r) => (r.kind ? r : { ...r, kind: (r.h ?? 0) >= 28 ? 'image' : 'text' })) }
  let kind: Kind = 'text'
  if (it.accent) kind = 'button'
  else if (it.h === 10 && it.w === 40) kind = 'header'
  else if ((it.h ?? 0) >= 28) kind = 'image'
  else if (it.h === 8 && !it.w) kind = 'footer'
  return { ...it, kind }
}

/** One part of a miniature email, by the kind of block it stands for. */
function Part({ it, scale }: { it: Item; scale: number }) {
  const h = it.h ? Math.max(3, Math.round(it.h * scale)) : undefined
  const w = it.w ? `${it.w}%` : undefined
  if (it.row?.length) {
    return (
      <span className="mrow">
        {it.row.map((r, i) => (
          <span key={i} className="mcol">
            <Part it={r} scale={scale} />
            {r.kind === 'image' ? <i className="mtx" style={{ width: '80%' }} /> : null}
            {/* Two columns read as article cards, each with its button. */}
            {r.kind === 'image' && it.row?.length === 2 ? <i className="mbt" style={{ width: '45%' }} /> : null}
          </span>
        ))}
      </span>
    )
  }
  switch (it.kind) {
    case 'header':
      return (
        <span className="mhead">
          <i className="mlogo" />
          <span className="mnav">
            <i />
            <i />
          </span>
        </span>
      )
    case 'footer':
      return (
        <span className="mfoot">
          <i />
          <i />
        </span>
      )
    case 'image':
      return <MiniImage h={h} />
    case 'heading':
      return <i className="mhd" style={{ width: w ?? '80%' }} />
    case 'button':
      return <i className="mbt" style={{ width: w }} />
    default:
      return <i className="mtx" style={{ width: w }} />
  }
}

/**
 * A template as a miniature email (Composer and ComposerEmpty boards): its
 * header, images, headings, text, buttons and footer, drawn from the MJML.
 */
export function Thumb({ items, blank, small }: { items: Item[]; blank?: boolean; small?: boolean }) {
  const t = useT()
  const scale = small ? 0.5 : 1
  const parts = items.map(withKind)
  const head = parts.find((it) => it.kind === 'header')
  const foot = parts.find((it) => it.kind === 'footer')
  const body = parts.filter((it) => it !== head && it !== foot)
  return (
    <div className={`thumb mini${small ? ' small' : ''}`} aria-hidden="true">
      <span className="memail">
        {blank || head ? <Part it={head ?? { kind: 'header' }} scale={scale} /> : null}
        <span className="mbody">
          {blank ? <span className="mempty">{small ? null : t(mh.thumbEmpty)}</span> : body.map((it, i) => <Part key={i} it={it} scale={scale} />)}
        </span>
      </span>
      {blank || foot ? <Part it={foot ?? { kind: 'footer' }} scale={scale} /> : null}
    </div>
  )
}
