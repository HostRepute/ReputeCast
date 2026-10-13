import { useEffect, useSyncExternalStore, type ReactNode } from 'react'

export type ToastKind = 'success' | 'deleted' | 'error' | 'progress' | 'info'

/**
 * Toasts (Feedback board): bottom-left, one at a time, above the phone tab
 * bar. Success and progress toasts use role="status"; errors use
 * role="alert" and stay until dismissed. Focus never moves to a toast.
 * Never toast something the page already shows.
 */
export interface ToastInput {
  kind?: ToastKind
  title: ReactNode
  detail?: ReactNode
  action?: { label: string; onClick: () => void }
  /** Offered for deletes, moves and unsubscribes; the toast then stays 10 s. */
  undo?: () => void
  /** Offered on errors: try the failed action again. */
  retry?: () => void
  /** Errors and progress stay until dismissed; undo toasts 10 s; others 6 s. */
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

/** Changes a toast in place, e.g. a progress toast that finished. */
export function updateToast(id: number, patch: Partial<ToastInput>) {
  if (!items.some((t) => t.id === id)) return
  items = items.map((t) => (t.id === id ? { ...t, ...patch } : t))
  emit()
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

export function toastDuration(t: ToastInput): number {
  if (t.duration !== undefined) return t.duration
  const kind = t.kind ?? 'success'
  if (kind === 'error' || kind === 'progress') return 0
  return t.undo ? 10_000 : 6000
}

function ToastView({ t }: { t: ToastItem }) {
  const kind = t.kind ?? 'success'
  const duration = toastDuration(t)
  useEffect(() => {
    if (!duration) return
    const h = window.setTimeout(() => dismissToast(t.id), duration)
    return () => window.clearTimeout(h)
  }, [t.id, duration])
  const ic = kind === 'progress' ? null : icons[kind]
  const act =
    t.action ??
    (t.undo ? { label: 'Undo', onClick: t.undo } : t.retry ? { label: 'Retry', onClick: t.retry } : kind === 'progress' ? { label: 'Hide', onClick: () => {} } : undefined)
  return (
    <div className="toast" role={kind === 'error' ? 'alert' : 'status'}>
      {ic ? (
        <span className="ti" style={{ background: ic.bg }}>
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="#fff" strokeWidth={ic.sw} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d={ic.path} />
          </svg>
        </span>
      ) : (
        <span className="spin sm on-dark" aria-hidden="true" />
      )}
      <span className="t">
        {t.title} {t.detail ? <span className="k">· {t.detail}</span> : null}
      </span>
      {act ? (
        <button
          type="button"
          className="btn"
          onClick={() => {
            act.onClick()
            dismissToast(t.id)
          }}
        >
          {act.label}
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
