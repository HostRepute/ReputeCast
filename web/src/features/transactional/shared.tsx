import { Fragment, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs, type NavTab } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { getI18n, t as tt, useI18n, useT } from '@/i18n'
import { weekdayLabel } from '@/i18n/regions'
import { clock, count, dayMonthDate, decimal, pct, zonedParts } from '@/lib/format'
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

/** "1.3 s"; a minute or more reads "3 min". Numbers follow the region format. */
export function secs(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || ms < 0) return '—'
  if (ms >= 60_000) return tt(mt.minutes, { value: count(ms / 60_000) })
  // A local mail catcher answers in milliseconds.
  if (ms < 100) return tt(mt.milliseconds, { value: count(ms) })
  return tt(mt.seconds, { value: decimal(ms / 1000, 1) })
}

/** A size as the log says it: "38 KB", "912 B". */
export function sizeText(bytes: number): string {
  return bytes >= 1024 ? tt(mt.sizeKB, { value: count(bytes / 1024) }) : tt(mt.sizeBytes, { value: count(bytes) })
}

/** Delivery rates keep one decimal, except a whole 100%: "99.9%", "100%". */
export function deliveredPct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v >= 99.995) return pct(100, 0)
  if (v === 0) return pct(0, 0)
  return pct(v, 1)
}

/** Bounce rates keep two decimals ("0.09%", "1.41%"); none reads "0%" (or "0.00%" with `keepZero`). */
export function bouncePct(v: number | null | undefined, keepZero = false): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v === 0 && !keepZero) return pct(0, 0)
  return pct(v, 2)
}

/** Complaint rates are tiny: "0.007%". */
export function complaintPct(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  if (v === 0) return pct(0, 0)
  return pct(v, v < 0.01 ? 3 : 2)
}

const DAY = 86_400_000

/** When a template last sent: "10:42" today, "Mon 08:00" this week, else "9 Oct". */
export function lastSent(d: string | null | undefined, now: Date = new Date()): string {
  if (!d) return '—'
  const t = new Date(d)
  if (dayMonthDate(t) === dayMonthDate(now) && now.getTime() - t.getTime() < DAY) return clock(t)
  if (now.getTime() - t.getTime() < 7 * DAY) return `${weekdayLabel(getI18n().regionFormat, zonedParts(t).weekday, 'short')} ${clock(t)}`
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

/**
 * Items joined as the app language lists them ("a, b and c"; Vietnamese
 * "a, b và c"), each drawn by `render`: a list with markup, which
 * formatList can't give.
 */
export function RichList({ items, render }: { items: string[]; render: (item: string) => ReactNode }) {
  const { locale } = useI18n()
  // English lists (and the pseudo-locale's) follow the house style, without the serial comma.
  const parts = new Intl.ListFormat(locale.startsWith('en') ? 'en-GB' : locale, { style: 'long', type: 'conjunction' }).formatToParts(items)
  return <>{parts.map((p, i) => (p.type === 'element' ? <Fragment key={i}>{render(p.value)}</Fragment> : p.value))}</>
}

/** "3,212" */
export const n = count
