import { useEffect, useSyncExternalStore, type ReactNode } from 'react'

export type ToastKind = 'success' | 'deleted' | 'error' | 'progress' | 'info'

export interface ToastInput {
  kind?: ToastKind
  title: ReactNode
  detail?: ReactNode
  action?: { label: string; onClick: () => void }
  /** Errors and progress stay until dismissed; others close after 6 s. */
  duration?: number
}

interface ToastItem extends ToastInput {
  id: number
}

let items: ToastItem[] = []
let nextId = 1
const listeners = new Set<() => void>()
const emit = () => listeners.forEach((l) => l())

/** Shows a toast. One at a time: a new toast replaces the current one. */
export function toast(t: ToastInput): number {
  const id = nextId++
  items = [{ ...t, id }]
  emit()
  return id
}

export function dismissToast(id?: number) {
  items = id === undefined ? [] : items.filter((t) => t.id !== id)
  emit()
}

function subscribe(l: () => void) {
  listeners.add(l)
  return () => listeners.delete(l)
}

const icons: Record<Exclude<ToastKind, 'progress'>, { bg: string; path: string; sw: number }> = {
  success: { bg: '#166534', path: 'm3 8 3.5 3.5L13 5', sw: 2.4 },
  deleted: { bg: '#4B5563', path: 'M3 4h10M6 4V3h4v1M5 4l1 9h4l1-9', sw: 2.2 },
  error: { bg: '#B91C1C', path: 'M8 4v5M8 11.5v.5', sw: 2.4 },
  info: { bg: '#1E40AF', path: 'M8 7.5V11M8 5v.5', sw: 2.4 },
}

function ToastView({ t }: { t: ToastItem }) {
  const kind = t.kind ?? 'success'
  const sticky = kind === 'error' || kind === 'progress'
  const duration = t.duration ?? (sticky ? 0 : 6000)
  useEffect(() => {
    if (!duration) return
    const h = window.setTimeout(() => dismissToast(t.id), duration)
    return () => window.clearTimeout(h)
  }, [t.id, duration])
  const ic = kind === 'progress' ? null : icons[kind]
  return (
    <div className="toast" role={kind === 'error' ? 'alert' : 'status'}>
      {ic ? (
        <span className="ti" style={{ background: ic.bg }}>
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="#fff" strokeWidth={ic.sw} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d={ic.path} />
          </svg>
        </span>
      ) : (
        <span className="spin" aria-hidden="true" />
      )}
      <span className="t">
        {t.title} {t.detail ? <span className="k">· {t.detail}</span> : null}
      </span>
      {t.action ? (
        <button
          type="button"
          className="btn"
          onClick={() => {
            t.action?.onClick()
            dismissToast(t.id)
          }}
        >
          {t.action.label}
        </button>
      ) : null}
      {kind !== 'progress' ? (
        <button type="button" className="x" aria-label="Dismiss" onClick={() => dismissToast(t.id)}>
          ×
        </button>
      ) : null}
    </div>
  )
}

/** Mount once near the root. Focus never moves to a toast. */
export function Toaster() {
  const list = useSyncExternalStore(subscribe, () => items, () => items)
  return (
    <div className="toasts" aria-live="polite">
      {list.map((t) => (
        <ToastView key={t.id} t={t} />
      ))}
    </div>
  )
}
