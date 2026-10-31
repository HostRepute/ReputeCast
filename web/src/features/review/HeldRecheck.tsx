import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { clock, count, shortDate } from '@/lib/format'
import { m } from './messages'

/**
 * A send the send-time recheck held on what nobody confirmed (SR-3,
 * SendReadinessStates "Held"): why, both counts, who was told, and when it
 * expires back to Draft if nobody reviews it. "Review again" confirms it
 * on the fresh checks and count.
 */
export function HeldRecheckCard({ held, notified, actions }: { held: Schemas['ReviewHeld']; notified?: ReactNode; actions: ReactNode }) {
  const t = useT()
  const first = held.holds?.[0]
  const since = held.since
  const reviewed = held.reviewed
  const rechecked = held.rechecked
  const diff = reviewed && rechecked ? rechecked.recipients - reviewed.recipients : 0
  const pct = reviewed && reviewed.recipients > 0 ? (diff / reviewed.recipients) * 100 : 0
  return (
    <section className="card border-bad-line" aria-labelledby="rv-held">
      <div className="rv-cardh h-bad">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="rv-held">
            {t(m.heldRecheckTitle)}
          </h2>
        </div>
        <Badge tone="bad">{t(m.heldBadge)}</Badge>
      </div>
      <div className="pad col gap-3!">
        <div className="note b" role="alert">
          <span>
            {t(m.heldAt, { time: clock(since), reason: lowerFirst(first?.text ?? held.reason) })}{' '}
            {held.expiresAt ? t(m.heldExpires, { time: clock(held.expiresAt) }) : null}
          </span>
        </div>
        {actions ? <div className="row gap-2!">{actions}</div> : null}
        <dl className="kv m-0 text-[12.5px]">
          {reviewed ? (
            <>
              <dt>{t(m.heldReviewed, { when: `${shortDate(reviewed.at)}, ${clock(reviewed.at)}` })}</dt>
              <dd>{t(m.heldRecipients, { count: count(reviewed.recipients) })}</dd>
            </>
          ) : null}
          {rechecked ? (
            <>
              <dt>{t(m.heldRechecked, { when: `${shortDate(rechecked.at)}, ${clock(rechecked.at)}` })}</dt>
              <dd>
                {count(rechecked.recipients)}
                {diff !== 0 ? <span className={diff > 0 ? 'rv-up' : 'rv-down'}> {t(m.heldDiff, { sign: diff > 0 ? 'up' : 'down', count: count(Math.abs(diff)), pct: pct.toFixed(1) })}</span> : null}
              </dd>
            </>
          ) : null}
          <dt>{t(m.heldWhy)}</dt>
          <dd>{t(m.heldWhyText, { kind: first?.kind ?? 'blocked' })}</dd>
          {notified ? (
            <>
              <dt>{t(m.notified)}</dt>
              <dd>{notified}</dd>
            </>
          ) : null}
          {held.expiresAt ? (
            <>
              <dt>{t(m.heldIfNobody)}</dt>
              <dd>{t(m.heldIfNobodyText, { time: clock(held.expiresAt) })}</dd>
            </>
          ) : null}
        </dl>
      </div>
    </section>
  )
}

function lowerFirst(s: string): string {
  return s.length > 1 && s[1] === s[1]!.toLowerCase() ? s[0]!.toLowerCase() + s.slice(1) : s
}
