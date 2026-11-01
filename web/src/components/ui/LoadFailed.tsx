import type { CSSProperties } from 'react'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { Button } from './Button'
import { m } from './messages'

/**
 * A part of a page whose request failed: says so and offers Try again,
 * instead of looking empty (review 2026-11 AP-2; TransactionalComplete
 * board, "Comments that didn't load").
 */
export function LoadFailed({ title, onRetry, style }: { title: string; onRetry: () => void; style?: CSSProperties }) {
  const t = useT()
  return (
    <div className="flex flex-col items-start gap-2 rounded-[10px] border border-bad-line bg-bad-tint px-4 py-3.5 text-[13px] text-bad-fg" role="alert" style={style}>
      <b>{title}</b>
      <span>{t(m.loadFailedBody)}</span>
      <Button size="sm" onClick={onRetry}>
        {t(common.retry)}
      </Button>
    </div>
  )
}
