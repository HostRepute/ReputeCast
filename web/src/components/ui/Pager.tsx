import type { ReactNode } from 'react'
import { Button, type ButtonSize } from './Button'

/** What Pager needs of a list read page by page (TanStack Query's infinite queries). */
export interface PagedQuery {
  hasNextPage: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => Promise<unknown>
}

/**
 * "Show more" under a list read page by page, the one way to draw it
 * (review 2026-11 FE-3b): nothing once the last page is in, and a button
 * that says it's loading and can't fetch the same page twice. The label is
 * the page's own ("Load 50 more", "Show more"); the list's own line says
 * how many are shown.
 */
export function Pager({ query, label, size = 'sm', className }: { query: PagedQuery; label: ReactNode; size?: ButtonSize; className?: string }) {
  if (!query.hasNextPage) return null
  return (
    <Button size={size} className={className} loading={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>
      {label}
    </Button>
  )
}
