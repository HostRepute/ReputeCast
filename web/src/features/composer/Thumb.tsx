import type { Schemas } from '@/api/client'

type Item = Schemas['MJMLThumbItem']

function Bar({ it }: { it: Item }) {
  if (it.row?.length) {
    return (
      <div style={{ display: 'flex', gap: 6 }}>
        {it.row.map((r, i) => (
          <i key={i} style={{ height: r.h, flex: 1 }} />
        ))}
      </div>
    )
  }
  return <i style={{ height: it.h, width: it.w ? `${it.w}%` : undefined, ...(it.accent ? { borderRadius: 7, background: 'var(--accent)' } : {}) }} />
}

/** A template's wireframe (the gallery's grey bars). */
export function Thumb({ items, blank, small }: { items: Item[]; blank?: boolean; small?: boolean }) {
  if (blank) {
    return (
      <div className="thumb" style={{ borderStyle: 'dashed', alignItems: 'center', justifyContent: 'center', color: 'var(--ink-3)', fontSize: 12, ...(small ? { height: 64 } : {}) }} aria-hidden="true">
        Empty mj-body
      </div>
    )
  }
  return (
    <div className="thumb" style={small ? { height: 64, padding: 6, gap: 4 } : undefined} aria-hidden="true">
      {items.map((it, i) => (
        <Bar key={i} it={small ? { ...it, h: it.h ? Math.max(4, Math.round(it.h / 2)) : it.h, row: it.row?.map((r) => ({ ...r, h: r.h ? Math.round(r.h / 2) : r.h })) } : it} />
      ))}
    </div>
  )
}
