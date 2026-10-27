import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { mh } from './chrome.messages'

type Item = Schemas['MJMLThumbItem']

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
