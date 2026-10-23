import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { mh } from './homenav.messages'

/** The six-dot grip of a draggable row. */
export function GripIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <circle cx="6" cy="3.5" r="1.2" />
      <circle cx="10" cy="3.5" r="1.2" />
      <circle cx="6" cy="8" r="1.2" />
      <circle cx="10" cy="8" r="1.2" />
      <circle cx="6" cy="12.5" r="1.2" />
      <circle cx="10" cy="12.5" r="1.2" />
    </svg>
  )
}

/**
 * Rows that reorder by dragging, or from the keyboard with the arrow keys on
 * a row's grip (the move is announced). Each row's content comes from
 * `render`.
 */
export function ReorderList<T>({
  items,
  keyOf,
  nameOf,
  onChange,
  render,
  label,
  disabled,
  rowClass,
}: {
  items: T[]
  keyOf: (it: T) => string
  nameOf: (it: T) => string
  onChange: (next: T[]) => void
  render: (it: T, index: number) => ReactNode
  label: string
  disabled?: boolean
  rowClass?: (it: T) => string | undefined
}) {
  const t = useT()
  const [dragging, setDragging] = useState<number | null>(null)
  const [over, setOver] = useState<number | null>(null)
  const [said, setSaid] = useState('')
  const grips = useRef(new Map<string, HTMLButtonElement>())
  const refocus = useRef<string | null>(null)
  useEffect(() => {
    if (!refocus.current) return
    grips.current.get(refocus.current)?.focus()
    refocus.current = null
  })
  const move = (from: number, to: number) => {
    if (from === to || to < 0 || to >= items.length) return
    const next = items.slice()
    const [x] = next.splice(from, 1)
    next.splice(to, 0, x as T)
    onChange(next)
    setSaid(t(mh.moved, { name: nameOf(x as T), n: to + 1 }))
    return x as T
  }
  return (
    <>
      <div className="ws-dlist" role="list" aria-label={label}>
        {items.map((it, i) => {
          const key = keyOf(it)
          return (
            <div
              key={key}
              role="listitem"
              className={cn('ws-drag', over === i && dragging !== null && dragging !== i && 'over', dragging === i && 'dragging', rowClass?.(it))}
              draggable={!disabled}
              onDragStart={(e) => {
                setDragging(i)
                e.dataTransfer.effectAllowed = 'move'
                e.dataTransfer.setData('text/plain', key)
              }}
              onDragOver={(e) => {
                if (dragging === null) return
                e.preventDefault()
                setOver(i)
              }}
              onDrop={(e) => {
                e.preventDefault()
                if (dragging !== null) move(dragging, i)
                setDragging(null)
                setOver(null)
              }}
              onDragEnd={() => {
                setDragging(null)
                setOver(null)
              }}
            >
              <button
                type="button"
                className="gr"
                aria-label={t(mh.dragKeysFor, { name: nameOf(it) })}
                disabled={disabled}
                ref={(el) => {
                  if (el) grips.current.set(key, el)
                  else grips.current.delete(key)
                }}
                onKeyDown={(e) => {
                  const to = e.key === 'ArrowUp' ? i - 1 : e.key === 'ArrowDown' ? i + 1 : -2
                  if (to === -2) return
                  e.preventDefault()
                  if (move(i, to) !== undefined) refocus.current = key
                }}
              >
                <GripIcon />
              </button>
              {render(it, i)}
            </div>
          )
        })}
      </div>
      <span className="vh" role="status" aria-live="polite">
        {said}
      </span>
    </>
  )
}
