import { useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs, type NavTab } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, count, dayMonthDate, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mt } from './messages'
import './transactional.css'

export type TxCategoryRef = Schemas['TransactionalCategoryRef']
export type TxPeriod = '24h' | '7d' | '30d'

/** The Transactional section's tabs, inline after the title: Templates · Messages · Setup · Settings. */
export function TxTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  return <NavTabs label={t(mt.tabsLabel)} className="ml-3 max-md:ml-0" tabs={txTabs(slug, t)} />
}

function txTabs(workspace: string, t: ReturnType<typeof useT>): NavTab[] {
  const params = { workspace }
  return [
    { label: t(mt.tabTemplates), to: '/w/$workspace/transactional', params, exact: true },
    { label: t(mt.tabMessages), to: '/w/$workspace/transactional/messages', params },
    { label: t(mt.tabSetup), to: '/w/$workspace/transactional/setup', params },
    { label: t(mt.tabSettings), to: '/w/$workspace/transactional/settings', params },
  ]
}

/** "1.3 s"; a minute or more reads "3 min". */
export function secs(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || ms < 0) return '—'
  if (ms >= 60_000) return `${Math.round(ms / 60_000)} min`
  // A local mail catcher answers in milliseconds.
  if (ms < 100) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

/** Delivery rates keep one decimal, except a whole 100%: "99.9%", "100%". */
export function deliveredPct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v >= 99.995) return '100%'
  if (v === 0) return '0%'
  return `${v.toFixed(1)}%`
}

/** Bounce rates keep two decimals ("0.09%", "1.41%"); none reads "0%". */
export function bouncePct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v === 0) return '0%'
  return `${v.toFixed(2)}%`
}

/** Complaint rates are tiny: "0.007%". */
export function complaintPct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v === 0) return '0%'
  return v < 0.01 ? `${v.toFixed(3)}%` : `${v.toFixed(2)}%`
}

const DAY = 86_400_000

/** When a template last sent: "10:42" today, "Mon 08:00" this week, else "9 Oct". */
export function lastSent(d: string | null | undefined, now: Date = new Date()): string {
  if (!d) return '—'
  const t = new Date(d)
  if (dayMonthDate(t) === dayMonthDate(now) && now.getTime() - t.getTime() < DAY) return clock(t)
  if (now.getTime() - t.getTime() < 7 * DAY) return `${shortDate(t).split(' ')[0]} ${clock(t)}`
  return dayMonthDate(t)
}

/** Message status badges (design system §7). */
export function statusTone(status: string): Tone {
  switch (status) {
    case 'delivered':
      return 'ok'
    case 'sent':
      return 'info'
    case 'deferred':
      return 'warn'
    case 'bounced':
    case 'complained':
    case 'failed':
      return 'bad'
    case 'test':
      return 'purple'
  }
  return 'neutral'
}

export function useStatusLabel() {
  const t = useT()
  return (status: string): string => {
    switch (status) {
      case 'queued':
        return t(mt.statusQueued)
      case 'sent':
        return t(mt.statusSent)
      case 'delivered':
        return t(mt.statusDelivered)
      case 'deferred':
        return t(mt.statusDeferred)
      case 'bounced':
        return t(mt.statusBounced)
      case 'complained':
        return t(mt.statusComplained)
      case 'rejected':
        return t(mt.statusRejected)
      case 'failed':
        return t(mt.statusFailed)
      case 'test':
        return t(mt.statusTest)
      case 'cancelled':
        return t(mt.statusCancelled)
    }
    return status
  }
}

export function StatusBadge({ status, plain }: { status: string; plain?: boolean }) {
  const label = useStatusLabel()
  const tone = statusTone(status)
  // Lists of messages (the phone log, the detail's list) show the word only.
  const dot = !plain && (tone === 'ok' || tone === 'bad' || tone === 'warn' || tone === 'info')
  return (
    <Badge tone={tone} dot={dot}>
      {label(status)}
    </Badge>
  )
}

/** A category and its kind: "Account · Essential". */
export function CategoryCell({ category }: { category?: TxCategoryRef | null }) {
  const t = useT()
  if (!category) return <span className="k">—</span>
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <span>{category.name}</span>
      <Badge tone={category.kind === 'optional' ? 'info' : 'neutral'} className="tx-kind">
        {category.kind === 'optional' ? t(mt.optional) : t(mt.essential)}
      </Badge>
    </span>
  )
}

/** "v3 live", plus "Draft v2" when the draft differs. */
export function VersionBadges({ live, draftChanges, draftVersion, archived }: { live?: number; draftChanges: boolean; draftVersion: number; archived?: boolean }) {
  const t = useT()
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      {archived ? <Badge tone="neutral">{t(mt.archived)}</Badge> : null}
      {live ? <Badge tone="ok">{t(mt.live, { version: live })}</Badge> : <Badge tone="neutral">{t(mt.neverPublished)}</Badge>}
      {live && draftChanges ? <Badge tone="warn">{t(mt.draft, { version: draftVersion })}</Badge> : null}
    </span>
  )
}

/** The templates table's 72 × 22 trend of messages per day. */
export function MiniSpark({ values }: { values: number[] }) {
  if (values.length < 2) return null
  const w = 72
  const h = 22
  const max = Math.max(...values, 1)
  const min = Math.min(...values)
  const span = max - min || 1
  const pts = values.map((v, i) => {
    const x = 1 + (i / (values.length - 1)) * (w - 2)
    const y = 3 + (1 - (v - min) / span) * (h - 6)
    return [x, y] as const
  })
  const line = pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
  const area = `M1.0,${h} ${pts.map(([x, y]) => `L${x.toFixed(1)},${y.toFixed(1)}`).join(' ')} L${(w - 1).toFixed(1)},${h} Z`
  return (
    <svg className="block flex-none" width={w} height={h} viewBox={`0 0 ${w} ${h}`} aria-hidden="true">
      <path d={area} fill="#0F766E" opacity=".10" />
      <polyline points={line} fill="none" stroke="#0F766E" strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}

/** Key rows' badges: Send-only or SMTP, Live or Test. */
export function KeyBadges({ preset, mode }: { preset: string; mode: string }) {
  const t = useT()
  return (
    <span className="inline-flex flex-wrap gap-1.5">
      <Badge tone="neutral">{preset === 'smtp' ? t(mt.smtp) : t(mt.sendOnly)}</Badge>
      {mode === 'test' ? <Badge tone="purple">{t(mt.modeTest)}</Badge> : <Badge tone="ok">{t(mt.modeLive)}</Badge>}
    </span>
  )
}

export type CodeLang = 'curl' | 'node' | 'python' | 'go' | 'php'
export const codeLangs: { key: CodeLang; label: string }[] = [
  { key: 'curl', label: 'cURL' },
  { key: 'node', label: 'Node.js' },
  { key: 'python', label: 'Python' },
  { key: 'go', label: 'Go' },
  { key: 'php', label: 'PHP' },
]

/** A dark code box with language buttons and Copy (design system §7). */
export function CodeBox({ lang, onLang, code, children }: { lang: CodeLang; onLang: (l: CodeLang) => void; code: string; children: ReactNode }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  return (
    <div className="tx-cbx">
      <div className="tx-cbh" role="group" aria-label={t(mt.codeSample)}>
        {codeLangs.map((l) => (
          <button key={l.key} type="button" className={l.key === lang ? 'on' : undefined} aria-pressed={l.key === lang} onClick={() => onLang(l.key)}>
            {l.label}
          </button>
        ))}
        <button
          type="button"
          className="cp"
          aria-label={t(mt.copyCode)}
          onClick={() => {
            void navigator.clipboard?.writeText(code).then(
              () => {
                setCopied(true)
                window.setTimeout(() => setCopied(false), 1500)
              },
              () => toast({ kind: 'error', title: t(mt.copy) }),
            )
          }}
        >
          <Icon name={copied ? 'check' : 'copy'} size={14} />
          {copied ? t(mt.copied) : t(mt.copy)}
        </button>
      </div>
      <pre>{children}</pre>
    </div>
  )
}

/** "3,212" */
export const n = count
