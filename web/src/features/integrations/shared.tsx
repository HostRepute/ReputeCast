import { useQueryClient } from '@tanstack/react-query'
import { useCallback, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { T, useT, formatList, type MessageDescriptor } from '@/i18n'
import { clock, dayMonthDate, dayMonthYear } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { aiCatalogCapabilities, aiEntryLine, aiEntryName, aiMaintainer } from '@/features/ai/entries'
import { cm } from '@/features/commerce/messages'
import { m } from './messages'
import './integrations.css'

export type Hub = Schemas['IntegrationHub']
export type Entry = Schemas['IntegrationEntry']
export type EntryKey = Entry['key']
export type GroupKey = Schemas['IntegrationGroup']['key']
export type Reason = Schemas['IntegrationReason']
export type App = Schemas['IntegrationApp']
export type KeyView = Schemas['IntegrationKeyView']
export type ItemState = Schemas['IntegrationItemState']
export type ItemLock = Schemas['IntegrationItemLock']
export type Secret = Schemas['IntegrationSecret']
export type Site = Schemas['IntegrationSite']
export type WordPressPage = Schemas['IntegrationWordPressPage']
export type PlatformKind = App['kind']
export type ConnectKind = PlatformKind | 'wordpress'
export type Status = Entry['status']

export const base = '/api/v1/workspaces/{workspace}/integrations' as const
export const API_REFERENCE = 'https://github.com/hostrepute/reputecast/blob/main/docs/api/README.md'
export const PLUGIN_URL = 'https://wordpress.org/plugins/reputecast/'
export const KNOWN_PLUGIN = '1.4.0'

export function isPlatform(k: string): k is PlatformKind {
  return k === 'zapier' || k === 'make' || k === 'n8n'
}

/** The catalog: groups, entries and the attention banners. */
export function useHub() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/catalog`, { params: { path: { workspace: slug } } })
}

/** Zapier's, Make's or n8n's page. */
export function useApp(kind: PlatformKind) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/apps/{kind}`, { params: { path: { workspace: slug, kind } } })
}

/** The WordPress sites. */
export function useWordPress() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/wordpress`, { params: { path: { workspace: slug } } })
}

/** Refreshes every integrations screen (the catalog, the pages, the call logs). */
export function useRefreshHub() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => typeof q.queryKey[1] === 'string' && (q.queryKey[1] as string).startsWith(`${base}/`),
      }),
    [qc],
  )
}

export const entryName: Record<EntryKey, MessageDescriptor> = {
  zapier: m.zapier,
  make: m.make,
  n8n: m.n8n,
  wordpress: m.wordpress,
  hostrepute: m.hostrepute,
  api: m.api,
  webhooks: m.webhooks,
  mcp: m.mcp,
  ...aiEntryName,
  zerobounce: m.zerobounce,
  neverbounce: m.neverbounce,
  kickbox: m.kickbox,
  bouncer: m.bouncer,
  emailable: m.emailable,
  shopify: cm.shopify,
  woocommerce: cm.woocommerce,
}

export const maintainer: Record<EntryKey, MessageDescriptor> = {
  zapier: m.byApp,
  make: m.byApp,
  n8n: m.byNode,
  wordpress: m.byPlugin,
  hostrepute: m.byHostRepute,
  api: m.builtIn,
  webhooks: m.builtIn,
  mcp: m.builtIn,
  ...aiMaintainer,
  zerobounce: m.byOwnAccount,
  neverbounce: m.byOwnAccount,
  kickbox: m.byOwnAccount,
  bouncer: m.byOwnAccount,
  emailable: m.byOwnAccount,
  shopify: cm.byShopify,
  woocommerce: cm.byWoo,
}

export const entryLine: Record<EntryKey, MessageDescriptor> = {
  zapier: m.zapierLine,
  make: m.makeLine,
  n8n: m.n8nLine,
  wordpress: m.wordpressLine,
  hostrepute: m.hostreputeLine,
  api: m.apiLine,
  webhooks: m.webhooksLine,
  mcp: m.mcpLine,
  ...aiEntryLine,
  zerobounce: m.zerobounceLine,
  neverbounce: m.neverbounceLine,
  kickbox: m.kickboxLine,
  bouncer: m.bouncerLine,
  emailable: m.emailableLine,
  shopify: cm.shopifyLine,
  woocommerce: cm.wooLine,
}

export const capabilities: Record<EntryKey, MessageDescriptor[]> = {
  zapier: [m.capTriggers, m.capActions],
  make: [m.capTriggers, m.capActions],
  n8n: [m.capTriggers, m.capActions, m.capSelfHosted],
  wordpress: [m.capForms, m.capDrafts],
  hostrepute: [m.capBlocklists, m.capPostmaster, m.capSnds, m.capDmarc],
  api: [m.capScopedKeys],
  webhooks: [m.capSigned, m.capReplay],
  mcp: [m.capScopedAgents],
  ...aiCatalogCapabilities,
  zerobounce: [m.capBulkLists, m.capRealtime, m.capCreditBalance],
  neverbounce: [m.capBulkLists, m.capRealtime, m.capCreditBalance],
  kickbox: [m.capBulkLists, m.capRealtime, m.capCreditBalance],
  bouncer: [m.capBulkLists, m.capRealtime, m.capCreditBalance],
  emailable: [m.capBulkLists, m.capRealtime, m.capCreditBalance],
  shopify: [cm.capOrders, cm.capProducts, cm.capCustomers, cm.capCheckouts, cm.capWriteBack, cm.capAttribution],
  woocommerce: [cm.capOrders, cm.capProducts, cm.capCustomers, cm.capCheckouts, cm.capAttribution],
}

/** The store platforms (the Commerce group, X20). */
export const isStore = (k: string): k is 'shopify' | 'woocommerce' => k === 'shopify' || k === 'woocommerce'

/** The address verification services' keys (X14). */
export const verificationKeys = ['zerobounce', 'neverbounce', 'kickbox', 'bouncer', 'emailable'] as const
export type VerificationKey = (typeof verificationKeys)[number]
export const isVerification = (k: string): k is VerificationKey => (verificationKeys as readonly string[]).includes(k)

/** Where "Open Zapier" goes: the platform's own app (n8n: the address it was connected with). */
export function platformUrl(kind: PlatformKind, address?: string): string | undefined {
  if (kind === 'zapier') return 'https://zapier.com/app/zaps'
  if (kind === 'make') return 'https://www.make.com/en/login'
  return address || undefined
}

/** The board's logo tiles: letters for the apps, icons for the built-in ones. */
export function Tile({ k, size }: { k: EntryKey; size?: 'sm' | 'lg' }) {
  const px = size === 'lg' ? 22 : size === 'sm' ? 14 : 18
  const letters: Partial<Record<EntryKey, string>> = {
    zapier: 'Z',
    make: 'M',
    n8n: 'n8n',
    wordpress: 'W',
    zerobounce: 'ZB',
    neverbounce: 'NB',
    kickbox: 'K',
    bouncer: 'B',
    emailable: 'E',
    shopify: 'Sh',
    woocommerce: 'Woo',
  }
  let inner: ReactNode = letters[k]
  if (k === 'hostrepute') inner = <ShieldTick size={px} />
  if (k === 'api') inner = <Icon name="code" size={px} strokeWidth={1.7} />
  if (k === 'webhooks') inner = <Icon name="bolt" size={px} strokeWidth={1.7} />
  if (k === 'mcp') inner = <ServerIcon size={px} />
  return (
    <span className={`ih-tile${size ? ` ${size}` : ''}`} aria-hidden="true">
      {inner}
    </span>
  )
}

/** HostRepute's shield with a tick, in accent teal (as on the boards). */
export function ShieldTick({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="var(--accent)" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" />
      <path d="m5.5 8 1.8 1.8L10.8 6" />
    </svg>
  )
}

function ServerIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2" y="2" width="12" height="5" rx="1.5" />
      <rect x="2" y="9" width="12" height="5" rx="1.5" />
      <path d="M5 4.5h.01M5 11.5h.01" strokeWidth="2" />
    </svg>
  )
}

/** The amber circle with an exclamation mark of the banners and the "why" lines. */
export function AttentionIcon({ size = 16, color = 'var(--warn-bar)' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

/** The blue information circle before an error's explanation (IntegrationsZapier). */
export function InfoIcon({ size = 15 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="#1D4ED8" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="mt-0.5 flex-none">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

/** Connected, Not connected, Waiting for first call, Needs attention. */
export function StatusBadge({ status }: { status: Status | Site['status'] }) {
  const t = useT()
  switch (status) {
    case 'connected':
      return (
        <Badge tone="ok" dot>
          {t(m.statusConnected)}
        </Badge>
      )
    case 'waiting':
      return (
        <Badge tone="info" dot>
          {t(m.statusWaiting)}
        </Badge>
      )
    case 'attention':
      return (
        <Badge tone="warn" dot>
          {t(m.statusAttention)}
        </Badge>
      )
    case 'not_connected':
      return <Badge tone="neutral">{t(m.statusNotConnected)}</Badge>
  }
  return null
}

/** "14:18" today, else "13 Oct 22:10" (the boards' activity times). */
export function stamp(d: string | Date, now: Date = new Date()): string {
  const at = typeof d === 'string' ? new Date(d) : d
  return dayMonthDate(at) === dayMonthDate(now) ? clock(at) : `${dayMonthDate(at)} ${clock(at)}`
}

/** "15 Oct 12:06": the call log always shows the day. */
export function dayStamp(d: string | Date): string {
  return `${dayMonthDate(d)} ${clock(d)}`
}

/** "24 Sep 2026" */
export const longDay = dayMonthYear

const itemNames: Record<string, MessageDescriptor> = {
  'contact.subscribed': m.item_contact_subscribed,
  'contact.unsubscribed': m.item_contact_unsubscribed,
  'contact.updated': m.item_contact_updated,
  'form.submitted': m.item_form_submitted,
  'campaign.completed': m.item_campaign_completed,
  'message.bounced': m.item_message_bounced,
  upsert_contact: m.item_upsert_contact,
  add_to_list: m.item_add_to_list,
  add_tag: m.item_add_tag,
  record_event: m.item_record_event,
  send_transactional: m.item_send_transactional,
  list_forms: m.item_list_forms,
  create_drafts: m.item_create_drafts,
}

const itemLines: Record<string, MessageDescriptor> = {
  upsert_contact: m.item_upsert_contact_line,
  add_to_list: m.item_add_to_list_line,
  add_tag: m.item_add_tag_line,
  record_event: m.item_record_event_line,
  send_transactional: m.item_send_transactional_line,
  list_forms: m.item_list_forms_line,
  create_drafts: m.item_create_drafts_line,
}

/** A trigger's or an action's name ("Contact subscribed", "Add a tag"). */
export function itemName(t: ReturnType<typeof useT>, key: string): string {
  const d = itemNames[key]
  return d ? t(d) : key
}

/** An action's one line; triggers show their event instead. */
export function itemLine(t: ReturnType<typeof useT>, key: string): string | undefined {
  const d = itemLines[key]
  return d ? t(d) : undefined
}

/** Permission chips, ⚑ flagged, with "its own endpoints" after webhooks:manage. */
export function PermChips({ perms, sensitive = [], notes }: { perms: string[]; sensitive?: string[]; notes?: Record<string, string> }) {
  const t = useT()
  return (
    <div className="ih-perms">
      {perms.map((p) => (
        <span key={p}>
          <span className="ih-mk">{p}</span>
          {sensitive.includes(p) ? (
            <span title={t(m.sensitivePermission)}>
              <SensitiveFlag>{t(m.sensitive)}</SensitiveFlag>
            </span>
          ) : null}
          {notes?.[p] ? <span className="k">{notes[p]}</span> : p === 'webhooks:manage' ? <span className="k">{t(m.itsOwnEndpoints)}</span> : null}
        </span>
      ))}
    </div>
  )
}

/** One permission chip, as the trigger and action rows show them. */
export function PermChip({ perm }: { perm: string }) {
  return <span className="ih-mk">{perm}</span>
}

/** "Marcus Chen or Linh Nguyen" */
export function helpersText(helpers: string[]): string {
  return formatList(helpers, 'disjunction')
}

/** Rich-text tags shared by the messages: bold, mono and the failing count in red. */
export const tags = {
  b: (c: ReactNode[]) => <b>{c}</b>,
  mono: (c: ReactNode[]) => <span className="mono">{c}</span>,
  bad: (c: ReactNode[]) => <span className="font-semibold text-bad-fg">{c}</span>,
}

/** Why an integration needs attention, in a few words (its card, its page). */
export function reasonWhy(t: ReturnType<typeof useT>, r: Reason): string {
  const since = r.since ? stamp(r.since) : ''
  switch (r.code) {
    case 'site_silent':
      return t(m.whySilent, { host: r.host ?? '', since })
    case 'deliveries_failing':
      return r.status ? t(m.whyFailing, { host: r.host ?? '', since, status: r.status, statusText: r.statusText ?? '' }) : t(m.whyFailingPlain, { host: r.host ?? '', since })
    case 'key_revoked':
      return t(m.whyRevoked)
    case 'key_expired':
      return t(m.whyExpired)
    case 'owner_left':
      return t(m.whyOwnerLeft)
    case 'key_expiring':
      return t(m.whyExpiring, { date: r.since ? dayMonthDate(r.since) : '' })
    case 'calls_failing':
      return t(m.whyCallsFailing, { failed: r.failed ?? 0, calls: r.calls ?? 0, since })
  }
  return ''
}

/** The catalog's banner sentence for a problem, with the integration's name in bold. */
export function ReasonSentence({ r, name }: { r: Reason; name: string }) {
  const since = r.since ? stamp(r.since) : ''
  const values = { ...tags, name, host: r.host ?? '', since, time: since }
  switch (r.code) {
    case 'site_silent':
      return <T msg={m.bannerSilent} values={values} />
    case 'deliveries_failing':
      return r.retrying && r.nextAt ? (
        <T msg={m.bannerFailingRetry} values={{ ...values, status: r.status ?? '', statusText: r.statusText ?? '', retrying: r.retrying, next: stamp(r.nextAt) }} />
      ) : (
        <T msg={m.bannerFailing} values={{ ...values, status: r.status ?? '', statusText: r.statusText ?? '' }} />
      )
    case 'key_revoked':
      return <T msg={m.bannerRevoked} values={values} />
    case 'key_expired':
      return <T msg={m.bannerExpired} values={values} />
    case 'owner_left':
      return <T msg={m.bannerOwnerLeft} values={{ ...values, by: r.by ?? '' }} />
    case 'key_expiring':
      return <T msg={m.bannerExpiring} values={{ ...values, date: r.since ? dayMonthDate(r.since) : '' }} />
    case 'calls_failing':
      return <T msg={m.bannerCallsFailing} values={{ ...values, failed: r.failed ?? 0, calls: r.calls ?? 0 }} />
  }
  return null
}

/** "Managed by n8n": the badge on rows of Keys & MCP and Webhooks that an integration owns. */
export function ManagedByBadge({ kind, label, to }: { kind: string; label: string; to?: ReactNode }) {
  const letters: Record<string, string> = { zapier: 'Z', make: 'M', n8n: 'n8n', wordpress: 'W' }
  const inner = (
    <>
      <span className="ih-mt" aria-hidden="true">
        {letters[kind] ?? kind.slice(0, 1).toUpperCase()}
      </span>
      {label}
    </>
  )
  return to ? <>{to}</> : <span className="ih-mby">{inner}</span>
}

/** The managed-by badge's inside, for callers that render it as a link. */
export function ManagedByInner({ kind, label }: { kind: string; label: string }) {
  const letters: Record<string, string> = { zapier: 'Z', make: 'M', n8n: 'n8n', wordpress: 'W' }
  return (
    <>
      <span className="ih-mt" aria-hidden="true">
        {letters[kind] ?? kind.slice(0, 1).toUpperCase()}
      </span>
      {label}
    </>
  )
}
