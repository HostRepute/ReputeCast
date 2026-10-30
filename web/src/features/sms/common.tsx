import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { ApiError, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { NavTabs } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { ChannelMenu } from '@/features/channels/ChannelMenu'
import { getI18n, useT, type MessageDescriptor } from '@/i18n'
import { dayMonthDate, dayMonthYear, money } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import './sms.css'

/** A 403 goes to the route's error screen; other errors stay on the page. */
export function throwIfForbidden(error: unknown) {
  if (error instanceof ApiError && error.status === 403) throw error
}

/** Refetches everything the SMS screens show after a change. */
export function useRefreshSms() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const path = q.queryKey[1]
        return typeof path === 'string' && (path.includes('/sms') || path.includes('/channels') || path.includes('/contacts/{contactId}/channels'))
      },
    })
}

/** Copies a text, for the registration helper and the webhook secret. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

export type SenderKind = Schemas['SmsSenderRow']['kind']
export type Registration = Schemas['SmsRegistrationView']

/** The SMS section's header: the channel menu, then its three tabs. */
export function SmsTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  return (
    <>
      <ChannelMenu current="sms" />
      <NavTabs
        label={t(m.sections)}
        className="ml-3 max-md:ml-0"
        tabs={[
          { label: t(m.tabProviders), to: '/w/$workspace/providers/sms', params, exact: true },
          { label: t(m.tabRules), to: '/w/$workspace/providers/sms/rules', params },
          { label: t(m.tabEvents), to: '/w/$workspace/providers/sms/events', params },
        ]}
      />
    </>
  )
}

/** Series colours of SMS providers, in the order they were connected. */
const series = ['var(--chart-gmail)', 'var(--chart-outlook)', 'var(--chart-yahoo)', 'var(--chart-icloud)', 'var(--chart-ses)']

/** A provider's colour: by its place among the workspace's providers. */
export function providerColor(order: string[], providerName: string): string {
  const i = order.indexOf(providerName)
  return i < 0 ? 'var(--chart-other)' : series[i % series.length]!
}

/** A provider's name with its coloured dot. */
export function ProviderDot({ name, color, gap = 7 }: { name: string; color: string; gap?: number }) {
  return (
    <span className="inline-flex items-center" style={{ gap }}>
      <span className="dot" style={{ color }} />
      {name}
    </span>
  )
}

/** A date-only value ("2026-10-06T00:00:00Z") as a calendar day. */
function calendarDay(s: string): Date {
  return new Date(`${s.slice(0, 10)}T12:00:00Z`)
}
export function dayYear(s: string | null | undefined): string {
  return s ? dayMonthYear(calendarDay(s)) : ''
}
export function dayShort(s: string | null | undefined): string {
  return s ? dayMonthDate(calendarDay(s)) : ''
}

/** A price per segment: $0.0110. */
export function price(v: number | null | undefined): string {
  return money(v, 4)
}

/** A country's name in the app's language ("United Kingdom"). */
export function countryName(code: string): string {
  try {
    return new Intl.DisplayNames([getI18n().locale], { type: 'region' }).of(code) ?? code
  } catch {
    return code
  }
}

/** How running text names a country: "UK", "US", else its name. */
export function shortCountry(code: string): string {
  if (getI18n().locale.startsWith('en')) {
    if (code === 'GB') return 'UK'
    if (code === 'US') return 'US'
  }
  return countryName(code)
}

/** A route's countries: "US and Canada", "UK, Germany, Netherlands". */
export function regionLabel(codes: string[]): string {
  const names = codes.map((c, i) => (i === 0 ? shortCountry(c) : c === 'US' || c === 'GB' ? shortCountry(c) : countryName(c)))
  if (names.length <= 1) return names[0] ?? ''
  // Two read as a pair ("US and Canada"), more as a list ("UK, Germany, Netherlands").
  const type = names.length === 2 ? 'conjunction' : 'unit'
  try {
    return new Intl.ListFormat(getI18n().locale.startsWith('en') ? 'en-GB' : getI18n().locale, { style: 'long', type }).format(names)
  } catch {
    return names.join(', ')
  }
}

/** Countries inside a sentence: "the route for UK, Germany and Netherlands". */
export function regionSentence(codes: string[]): string {
  const names = codes.map((c) => (c === 'US' || c === 'GB' ? shortCountry(c) : countryName(c)))
  if (names.length <= 1) return names[0] ?? ''
  try {
    return new Intl.ListFormat(getI18n().locale.startsWith('en') ? 'en-GB' : getI18n().locale, { style: 'long', type: 'conjunction' }).format(names)
  } catch {
    return names.join(', ')
  }
}

/** Country codes as pills: US CA, with the name on hover. */
export function CountryChips({ codes, nowrap, named }: { codes: string[]; nowrap?: boolean; named?: boolean }) {
  return (
    <span className="sms-ccs" style={nowrap ? { flexWrap: 'nowrap' } : undefined}>
      {codes.map((c) =>
        named ? (
          <span key={c} className="sms-ccy">
            <b>{c}</b>
            {countryName(c)}
          </span>
        ) : (
          <span key={c} className="sms-ccy s" title={countryName(c)}>
            <b>{c}</b>
          </span>
        ),
      )}
    </span>
  )
}

/** One country with its name: [US] United States. */
export function CountryName({ code }: { code: string }) {
  return (
    <span className="sms-ccy">
      <b>{code}</b>
      {countryName(code)}
    </span>
  )
}

const kindNames: Record<SenderKind, MessageDescriptor> = {
  toll_free: m.kindTollFree,
  '10dlc': m.kind10dlc,
  local: m.kindLocal,
  mobile: m.kindMobile,
  sender_id: m.kindSenderId,
}

const holdWords: Record<string, MessageDescriptor> = {
  pending: m.holdPending,
  rejected: m.holdRejected,
  not_registered: m.holdNot_registered,
  not_supported: m.holdNot_supported,
  no_route: m.holdNo_route,
}

/** Why a route can't send to a country (a route decision's hold): "pending", "not registered". */
export function holdWord(t: (d: MessageDescriptor) => string, code: string): string {
  const d = holdWords[code]
  return d ? t(d) : code.replace(/_/g, ' ')
}

export function useKindName(): (k: string) => string {
  const t = useT()
  return (k) => {
    const d = kindNames[k as SenderKind]
    return d ? t(d) : k
  }
}

/** A registration's status as its badge says it: "Verified 6 Oct 2026". */
export function registrationBadge(r: Registration, t: (d: MessageDescriptor, v?: Record<string, string | number>) => string, short = false): { tone: Tone; text: string } {
  const date = r.since ? (short ? dayShort(r.since) : dayYear(r.since)) : ''
  switch (r.status) {
    case 'verified':
      return { tone: 'ok', text: date ? t(m.regVerifiedOn, { date }) : t(m.regVerified) }
    case 'approved':
      return { tone: 'ok', text: date ? t(m.regApprovedOn, { date }) : t(m.regApproved) }
    case 'allowed':
      return { tone: 'ok', text: t(m.regAllowed) }
    case 'pending':
      return { tone: 'warn', text: date ? t(m.regPendingSince, { date }) : t(m.regPending) }
    case 'rejected':
      return { tone: 'bad', text: date ? t(m.regRejectedOn, { date }) : t(m.regRejected) }
    case 'not_supported':
      return { tone: 'neutral', text: t(m.regNotSupported) }
  }
  return { tone: 'neutral', text: t(m.regNotRegistered) }
}

/** A sender's registration badges: one, or a 10DLC number's brand and campaign. */
export function RegistrationBadges({ row, compact }: { row: Schemas['SmsSenderRow']; compact?: boolean }) {
  const t = useT()
  if (row.brand && row.campaign) {
    const b = row.brand
    const c = row.campaign
    const brandText =
      b.status === 'approved'
        ? compact
          ? t(m.brandApprovedShort, { date: dayShort(b.since) })
          : t(m.brandApprovedOn, { name: b.name, date: dayShort(b.since) })
        : registrationBadge(b, t, true).text
    const campaignText =
      c.status === 'pending'
        ? t(m.campaignPendingSince, { date: dayShort(c.since) })
        : c.status === 'approved'
          ? t(m.campaignApprovedOn, { date: dayShort(c.since) })
          : c.status === 'rejected'
            ? t(m.campaignRejectedOn, { date: dayShort(c.since) })
            : registrationBadge(c, t, true).text
    return (
      <>
        <Badge tone={registrationBadge(b, t).tone}>{brandText}</Badge>
        <Badge tone={registrationBadge(c, t).tone}>{campaignText}</Badge>
      </>
    )
  }
  const r = registrationBadge(row.registration, t)
  return <Badge tone={r.tone}>{r.text}</Badge>
}

const capNames: Record<string, MessageDescriptor> = {
  toll_free: m.capToll_free,
  '10dlc': m.cap10dlc,
  sender_ids: m.capSender_ids,
  receipts: m.capReceipts,
  inbound: m.capInbound,
  opt_out_sync: m.capOpt_out_sync,
  registration: m.capRegistration,
  pricing: m.capPricing,
}

const capShort: Record<string, MessageDescriptor> = {
  receipts: m.capShortReceipts,
  inbound: m.capShortInbound,
  registration: m.capShortRegistration,
}

/** With a note, "Registration status via API" reads "Registration status · set by hand". */
const capNoted: Record<string, MessageDescriptor> = {
  registration: m.capNotedRegistration,
  pricing: m.capNotedPricing,
}
const capNotedShort: Record<string, MessageDescriptor> = {
  registration: m.capNotedShortRegistration,
}

/**
 * A capability from an adapter's manifest (`.cap`): yes, partial or no.
 * `short` is the catalog's wording ("Receipts", "Keywords").
 */
export function Capability({ badge, short }: { badge: Schemas['Badge']; short?: boolean }) {
  const t = useT()
  const names = short && capShort[badge.capability] ? capShort : capNames
  const noted = badge.note ? (short && capShort[badge.capability] ? capNotedShort : capNoted)[badge.capability] : undefined
  const label = noted ? t(noted) : names[badge.capability] ? t(names[badge.capability]!) : badge.capability
  const text = badge.note ? t(m.capWithNote, { label, note: badge.note }) : label
  const cls = { yes: 'y', partial: 'p', no: 'n' }[badge.support]
  return (
    <span className={`sms-cap ${cls}`}>
      {badge.support === 'no' ? (
        <>
          <span className="vh">{t(m.capNotSupported, { capability: text })}</span>
          <span aria-hidden="true">{text}</span>
        </>
      ) : (
        text
      )}
    </span>
  )
}

/** The capabilities, in the manifest's order. */
export function Capabilities({ badges, short }: { badges: Schemas['Badge'][]; short?: boolean }): ReactNode {
  return (
    <div className="flex flex-wrap gap-1.5">
      {badges.map((b) => (
        <Capability key={b.capability} badge={b} short={short} />
      ))}
    </div>
  )
}

/**
 * The budget meter (`.bm`): spent, this send (striped), what goes over,
 * and the alert mark. `legend` adds the line under the track.
 */
export function BudgetMeter({
  spent,
  budget,
  alertPct,
  send = 0,
  month,
  legend,
}: {
  spent: number
  budget: number
  alertPct: number
  send?: number
  month: string
  legend?: boolean
}) {
  const t = useT()
  const cap = budget > 0 ? budget : 1
  const used = Math.min(100, (spent / cap) * 100)
  const sendPct = Math.max(0, Math.min(100 - used, (send / cap) * 100))
  const over = Math.max(0, spent + send - budget)
  const overPct = budget > 0 ? Math.max(0, Math.min(100 - used - sendPct, (over / cap) * 100)) : 0
  const label = send
    ? t(m.bmAriaSend, { month, spent: money(spent), budget: money(budget), send: money(send), alert: alertPct })
    : t(m.bmAria, { month, spent: money(spent), budget: money(budget), alert: alertPct })
  return (
    <div className="sms-bm">
      <div className="sms-bm-t" role="img" aria-label={label}>
        <i className="u" style={{ width: `${used.toFixed(2)}%` }} />
        {sendPct > 0 ? <i className="s" style={{ width: `${sendPct.toFixed(2)}%` }} /> : null}
        {overPct > 0 ? <i className="o" style={{ width: `${overPct.toFixed(2)}%` }} /> : null}
        <span className="bmk" style={{ left: `${alertPct}%` }} />
      </div>
      {legend ? (
        <div className="sms-bm-l">
          <span>
            <span className="sms-bsw" style={{ background: 'var(--accent)' }} />
            {t(m.bmSpent, { amount: money(spent) })}
          </span>
          {send ? (
            <span>
              <span className="sms-bsw" style={{ background: 'repeating-linear-gradient(135deg, #0f766e 0 3px, #99f6e4 3px 6px)' }} />
              {t(m.bmThisSend, { amount: money(send) })}
            </span>
          ) : null}
          {over > 0 && send ? (
            <span>
              <span className="sms-bsw" style={{ background: 'repeating-linear-gradient(135deg, #b91c1c 0 3px, #fca5a5 3px 6px)' }} />
              {t(m.bmOver, { amount: money(over) })}
            </span>
          ) : null}
          <span>
            <span aria-hidden="true" style={{ width: 2, height: 12, background: 'var(--ink)', borderRadius: 1, display: 'inline-block' }} />
            {t(m.bmAlert, { alert: alertPct })}
          </span>
        </div>
      ) : null}
    </div>
  )
}

/** A value to copy into a provider's form: its label, the text, Copy. */
export function CopyRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  const t = useT()
  return (
    <div className="sms-cpy">
      <div className="tx">
        <span className="k">{label}</span>
        <span className={mono ? 'mono' : undefined} style={{ fontSize: mono ? 12.5 : 13, overflowWrap: 'anywhere' }}>
          {value}
        </span>
      </div>
      <CopyButton value={value} label={t(m.copyLabel, { what: label.toLowerCase() })} />
    </div>
  )
}

/** Copy, then "Copied" for two seconds. */
export function CopyButton({ value, label }: { value: string; label: string }) {
  const t = useT()
  const [done, setDone] = useState(false)
  return (
    <button
      type="button"
      className="btn sm"
      aria-label={label}
      onClick={() => {
        void copyText(value).then((ok) => {
          if (!ok) {
            toast({ kind: 'error', title: t(m.copyFailed) })
            return
          }
          setDone(true)
          window.setTimeout(() => setDone(false), 2000)
        })
      }}
    >
      <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <rect x="5.5" y="5.5" width="8" height="8" rx="1.5" />
        <path d="M10.5 5.5V3.9A1.4 1.4 0 0 0 9.1 2.5H3.9A1.4 1.4 0 0 0 2.5 3.9v5.2a1.4 1.4 0 0 0 1.4 1.4h1.6" />
      </svg>
      <span aria-live="polite">{done ? t(m.copied) : t(m.copy)}</span>
    </button>
  )
}

/** A secret the viewer can't see: •••••• Hidden. */
export function Masked() {
  const t = useT()
  return (
    <span className="sms-mask">
      <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <rect x="3" y="7" width="10" height="7" rx="1.5" />
        <path d="M5.5 7V5a2.5 2.5 0 0 1 5 0v2" />
      </svg>
      <span aria-hidden="true">••••••</span>
      <Badge tone="neutral" className="h-5 text-[11px]">
        {t(m.hidden)}
      </Badge>
    </span>
  )
}

/** "Aisha Bello · Viewer": who is reading, and why they can only read. */
export function ReadOnlyNote({ children }: { children: ReactNode }) {
  const t = useT()
  const { me } = useWorkspace()
  const who = useMe().data
  const name = who?.user.name || who?.user.email || ''
  return (
    <div className="note n" role="note">
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(m.roleNote, { name, role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }) })}</span>
        <div>{children}</div>
      </div>
    </div>
  )
}
