import { useRef, useState, type DragEvent, type KeyboardEvent } from 'react'

/**
 * Rows the boards reorder by a grip: drag a row by its grip, or focus the
 * grip and press the up and down arrows. `move(from, to)` applies it.
 */
export function useReorder(move: (from: number, to: number) => void, count: number) {
  const dragging = useRef<number | null>(null)
  const [over, setOver] = useState<number | null>(null)
  const row = (i: number) => ({
    onDragOver: (e: DragEvent) => {
      if (dragging.current === null) return
      e.preventDefault()
      setOver(i)
    },
    onDragLeave: () => setOver((o) => (o === i ? null : o)),
    onDrop: (e: DragEvent) => {
      e.preventDefault()
      if (dragging.current !== null && dragging.current !== i) move(dragging.current, i)
      dragging.current = null
      setOver(null)
    },
    'data-over': over === i ? 'true' : undefined,
  })
  const grip = (i: number) => ({
    draggable: true,
    onDragStart: (e: DragEvent) => {
      dragging.current = i
      e.dataTransfer.effectAllowed = 'move'
      const tr = (e.currentTarget as HTMLElement).closest('[data-row]')
      if (tr) e.dataTransfer.setDragImage(tr, 12, 12)
    },
    onDragEnd: () => {
      dragging.current = null
      setOver(null)
    },
    onKeyDown: (e: KeyboardEvent) => {
      const to = e.key === 'ArrowUp' ? i - 1 : e.key === 'ArrowDown' ? i + 1 : -1
      if (to < 0 || to >= count) return
      e.preventDefault()
      move(i, to)
      // The grip moves with its row: focus follows.
      const list = (e.currentTarget as HTMLElement).closest('[data-rows]')
      window.requestAnimationFrame(() => list?.querySelectorAll<HTMLElement>('.ea-grip')[to]?.focus())
    },
  })
  return { row, grip }
}

/** The six-dot grip (the boards' `gripb`). */
export function Grip({ label, disabled, ...props }: { label: string; disabled?: boolean } & Partial<ReturnType<ReturnType<typeof useReorder>['grip']>>) {
  return (
    <button type="button" className="ea-grip" aria-label={label} disabled={disabled} {...(disabled ? {} : props)}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
        <circle cx="6" cy="4" r=".9" />
        <circle cx="10" cy="4" r=".9" />
        <circle cx="6" cy="8" r=".9" />
        <circle cx="10" cy="8" r=".9" />
        <circle cx="6" cy="12" r=".9" />
        <circle cx="10" cy="12" r=".9" />
      </svg>
    </button>
  )
}

/** Moves one item of a list to another place. */
export function moved<T>(list: T[], from: number, to: number): T[] {
  const next = [...list]
  const [x] = next.splice(from, 1)
  if (x === undefined) return list
  next.splice(to, 0, x)
  return next
}
