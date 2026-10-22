import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Feedback'
import { t as tt, useT } from '@/i18n'
import { callTime, lastUsed, usDate } from '@/lib/format'
import { scopeLabel } from '../access/parts'
import { m } from './messages'

export type Key = Schemas['APIKeyView']

/** The empty card of the developer settings (SettingsStates): dashed, centred, one action. */
export function DashedEmpty({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return (
    <section className="card flex flex-col items-center gap-2 border-dashed px-6 py-7 text-center" aria-label={title}>
      <span className="text-[16px] font-bold">{title}</span>
      <span className="k max-w-[420px]">{children}</span>
      {action ? <div className="mt-1">{action}</div> : null}
    </section>
  )
}

/** Loading rows of the keys and webhook tables (SettingsStates): text bars and a pill. */
export function TableSkeleton({ rows = 3, label }: { rows?: number; label: string }) {
  const widths = [
    [160, 140, 220],
    [190, 140, 180],
    [120, 140, 200],
  ]
  return (
    <div className="flex flex-col" aria-busy="true">
      <span className="vh" role="status">
        {label}
      </span>
      {Array.from({ length: rows }, (_, i) => {
        const [a, b, c] = widths[i % widths.length]!
        return (
          <div key={i} className="row flex-nowrap gap-3 border-t border-line-2 px-4 py-2.5 first:border-t-0">
            <Skeleton w={a} h={14} />
            <Skeleton w={b} h={14} />
            <Skeleton w={c} h={22} r={999} />
            <Skeleton w={120} h={14} className="ml-auto" />
          </div>
        )
      })}
    </div>
  )
}

/** Permission keys as grey mono badges (the keys tables). */
export function PermBadges({ keys }: { keys: string[] }) {
  return (
    <div className="row gap-1.5">
      {keys.map((p) => (
        <Badge key={p} tone="neutral" className="mono font-medium!">
          {p}
        </Badge>
      ))}
    </div>
  )
}

/** "3 min ago" over "198.51.100.12", or "09:36 · get_campaign_report" for an agent. */
export function LastUsedCell({ k, now }: { k: Key; now: Date }) {
  const t = useT()
  if (!k.lastUsedAt) return <span className="k">{t(m.neverUsed)}</span>
  if (k.status === 'expired') return <>{usDate(k.lastUsedAt)}</>
  const head = k.kind === 'mcp_agent' ? callTime(k.lastUsedAt, k.lastCall, now) : lastUsed(k.lastUsedAt, now)
  const tail = [k.lastUsedIp, k.kind === 'mcp_agent' ? undefined : k.lastUsedClient].filter(Boolean).join(' · ')
  return (
    <div className="col gap-0">
      <span>{head}</span>
      {tail ? <span className="k mono">{tail}</span> : null}
    </div>
  )
}

/** "Never", "Jan 1, 2027", or Expired with the date. */
export function ExpiresCell({ k }: { k: Key }) {
  const t = useT()
  if (k.status === 'expired')
    return (
      <span className="row gap-1.5">
        <Badge tone="bad">{t(m.expiredBadge)}</Badge>
        {k.expiresAt ? <span className="k">{usDate(k.expiresAt)}</span> : null}
      </span>
    )
  return k.expiresAt ? <>{usDate(k.expiresAt)}</> : <span className="k">{t(m.neverExpires)}</span>
}

/**
 * "Weekly Digest only" or "All lists". The words come from access/parts,
 * which builds the server's English label in the app language.
 */
export function scopeText(k: Pick<Key, 'scope'> & Partial<Pick<Key, 'transactional'>>) {
  const tx = k.transactional
  if (tx) {
    // Transactional keys are scoped by template or category, and test keys by address.
    const what = tx.templates.length
      ? tt(m.scopeTemplates, { names: tx.templates.join(', ') })
      : tx.categories.length
        ? tt(m.scopeCategories, { names: tx.categories.map((c) => c.name).join(', ') })
        : tt(m.scopeAllTemplates)
    return tx.mode === 'test' ? tt(m.scopeTestOnly, { scope: what }) : what
  }
  return k.scope.all || !k.scope.label ? tt(m.scopeAllLists) : scopeLabel(k.scope, { listWord: false })
}

/** One-line summary used in dialogs: “Website signup form” · subscribers:create · Weekly Digest only · no expiry. */
export function keySummary(k: Key) {
  const values = { name: k.name, permissions: k.permissions.join(', '), scope: scopeText(k) }
  return k.expiresAt ? tt(m.summary, { ...values, date: usDate(k.expiresAt) }) : tt(m.summaryNoExpiry, values)
}
