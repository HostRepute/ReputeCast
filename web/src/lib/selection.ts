import { useCallback, useState } from 'react'

/** Rows picked for a bulk action (useSelection). */
export interface Selection {
  /** The picked ids among those shown, in the order shown. */
  selected: string[]
  has: (id: string) => boolean
  toggle: (id: string, on: boolean) => void
  /** Picks or drops several rows at once (a group's or the page's). */
  setMany: (ids: readonly string[], on: boolean) => void
  clear: () => void
  /** Every row shown is picked / some are. */
  all: boolean
  some: boolean
}

/**
 * The one bulk-selection pattern (review 2026-11 FE-3b): rows picked by
 * id, and only rows still shown stay picked, so another page, a filter or
 * a refresh that drops a row drops it from the selection too. Lists that
 * select every row matching a filter across pages (Contacts) keep their
 * table's own selection.
 */
export function useSelection(shown: readonly string[]): Selection {
  const key = shown.join('\n')
  const [state, setState] = useState<{ key: string; picked: ReadonlySet<string> }>(() => ({ key, picked: new Set() }))
  let picked = state.picked
  if (state.key !== key) {
    const visible = new Set(shown)
    picked = new Set([...state.picked].filter((id) => visible.has(id)))
    setState({ key, picked })
  }
  const setMany = useCallback((ids: readonly string[], on: boolean) => {
    setState((s) => {
      const next = new Set(s.picked)
      for (const id of ids) {
        if (on) next.add(id)
        else next.delete(id)
      }
      return { key: s.key, picked: next }
    })
  }, [])
  const toggle = useCallback((id: string, on: boolean) => setMany([id], on), [setMany])
  const clear = useCallback(() => setState((s) => ({ key: s.key, picked: new Set() })), [])
  const selected = shown.filter((id) => picked.has(id))
  return {
    selected,
    has: (id) => picked.has(id),
    toggle,
    setMany,
    clear,
    all: shown.length > 0 && selected.length === shown.length,
    some: selected.length > 0,
  }
}
