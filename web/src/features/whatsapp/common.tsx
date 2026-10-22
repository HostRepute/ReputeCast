import { useQueryClient } from '@tanstack/react-query'
import { Fragment, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs } from '@/components/ui/Tabs'
import { useT, type MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import '@/features/push/push.css'
import './whatsapp.css'

export { throwIfForbidden } from '@/features/push/common'

export type Overview = Schemas['WhatsAppOverview']
export type Account = Schemas['WhatsAppAccount']
export type Template = Schemas['WhatsAppTemplate']

/** Account · Templates · Events, inline after the channel switch. */
export function WhatsAppTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  return (
    <NavTabs
      label={t(m.sectionsLabel)}
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: t(m.tabAccount), to: '/w/$workspace/providers/whatsapp', params, exact: true },
        { label: t(m.tabTemplates), to: '/w/$workspace/providers/whatsapp/templates', params },
        { label: t(m.tabEvents), to: '/w/$workspace/providers/whatsapp/events', params },
      ]}
    />
  )
}

export const categoryBadge: Record<Template['category'], { tone: Tone; msg: MessageDescriptor }> = {
  utility: { tone: 'info', msg: m.catUtility },
  marketing: { tone: 'purple', msg: m.catMarketing },
  authentication: { tone: 'neutral', msg: m.catAuthentication },
}

export const statusBadge: Record<Template['status'], { tone: Tone; msg: MessageDescriptor }> = {
  approved: { tone: 'ok', msg: m.stApproved },
  pending: { tone: 'warn', msg: m.stPending },
  rejected: { tone: 'bad', msg: m.stRejected },
  paused: { tone: 'warn', msg: m.stPaused },
  disabled: { tone: 'bad', msg: m.stDisabled },
  in_appeal: { tone: 'warn', msg: m.stAppeal },
  other: { tone: 'neutral', msg: m.stOther },
}

/** A test message's status, in words: "delivered". */
export const testStatusWord: Record<Schemas['WhatsAppTestResult']['status'], MessageDescriptor> = {
  queued: m.stQueued,
  sending: m.stSending,
  sent: m.stSent,
  delivered: m.stDelivered,
  failed: m.stFailed,
  held: m.stHeld,
  cancelled: m.stCancelled,
  skipped: m.stSkipped,
  undelivered: m.stUndelivered,
}

const qualityWord: Record<Account['quality'], { tone: Tone; msg: MessageDescriptor }> = {
  high: { tone: 'ok', msg: m.qualityHigh },
  medium: { tone: 'warn', msg: m.qualityMedium },
  low: { tone: 'bad', msg: m.qualityLow },
  unknown: { tone: 'neutral', msg: m.qualityUnknown },
}

export function QualityBadge({ quality }: { quality: Account['quality'] | string }) {
  const t = useT()
  const q = qualityWord[(quality in qualityWord ? quality : 'unknown') as Account['quality']]
  return (
    <Badge tone={q.tone} dot>
      {t(q.msg)}
    </Badge>
  )
}

export function qualityLabel(t: ReturnType<typeof useT>, quality: string): string {
  return t((qualityWord[quality as Account['quality']] ?? qualityWord.unknown).msg)
}

/** A template's body with its variables as chips: "Hi {{1}}, see you…". */
export function Body({ text, values }: { text: string; values?: Record<string, ReactNode> }) {
  const parts = text.split(/(\{\{\d+\}\})/g)
  return (
    <>
      {parts.map((p, i) => {
        const k = /^\{\{(\d+)\}\}$/.exec(p)?.[1]
        if (!k) return <Fragment key={i}>{p}</Fragment>
        const v = values?.[k]
        return v !== undefined ? <b key={i}>{v}</b> : <span key={i} className="wa-tok">{`{{${k}}}`}</span>
      })}
    </>
  )
}

/** "Hi there" for a fallback in "Hi {{1}}, …": the words around the variable, filled. */
export function fallbackSentence(body: string, key: string, fallback: string): string {
  const i = body.indexOf(`{{${key}}}`)
  if (i < 0) return fallback
  const before = body.slice(0, i).split(/[.!?]\s|\n/).pop() ?? ''
  const words = before.trim().split(/\s+/).slice(-2).join(' ')
  return `${words} ${fallback}`.trim()
}

/** The preview's text with the filled variables in bold, found by matching the template around them. */
export function FilledBody({ body, text }: { body: string; text: string }) {
  const parts = body.split(/(\{\{\d+\}\})/g)
  const out: { s: string; v: boolean }[] = []
  let rest = text
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]!
    if (/^\{\{\d+\}\}$/.test(part)) {
      const next = parts[i + 1] ?? ''
      const end = next ? rest.indexOf(next) : rest.length
      if (end < 0) return <>{text}</>
      out.push({ s: rest.slice(0, end), v: true })
      rest = rest.slice(end)
    } else {
      if (!rest.startsWith(part)) return <>{text}</>
      out.push({ s: part, v: false })
      rest = rest.slice(part.length)
    }
  }
  return (
    <>
      {out.map((x, i) => (x.v ? <b key={i}>{x.s}</b> : <span key={i}>{x.s}</span>))}
    </>
  )
}

/** "{{1}} first name · {{2}} doors-open time". */
export function VarLine({ vars }: { vars: Template['variables'] }) {
  return (
    <>
      {vars.map((v, i) => (
        <Fragment key={v.key}>
          {i > 0 ? ' · ' : null}
          <span className="wa-tok">{`{{${v.key}}}`}</span> {v.label}
        </Fragment>
      ))}
    </>
  )
}

/** Refreshes every WhatsApp query after a change. */
export function useInvalidateWhatsApp() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const k = q.queryKey[1]
        return typeof k === 'string' && (k.includes('/whatsapp') || k.includes('/channels'))
      },
    })
}

/** Meta's WhatsApp Manager and Meta for Developers. */
export const MANAGER_URL = 'https://business.facebook.com/wa/manage/message-templates/'
export const DEVELOPERS_URL = 'https://developers.facebook.com/apps/'

export function WhatsAppIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 1.8a6.2 6.2 0 0 0-5.33 9.36L1.8 14.2l3.13-.84A6.2 6.2 0 1 0 8 1.8z" />
      <path d="M6.1 5.2c-.5 0-.95.45-.85 1.05.35 2.2 2.15 4.05 4.4 4.45.6.1 1.05-.35 1.05-.85v-.55l-1.5-.65-.6.6a3.7 3.7 0 0 1-2-2l.6-.6-.65-1.5z" strokeWidth="1.25" />
    </svg>
  )
}
