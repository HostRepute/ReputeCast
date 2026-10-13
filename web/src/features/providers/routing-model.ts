import type { Schemas } from '@/api/client'
import { money } from '@/lib/format'
import type { FailoverPolicy, Stream } from './common'

export type RoutingConfig = Schemas['RoutingConfig']
export type RoutingProvider = Schemas['RoutingProvider']
export type Condition = Schemas['RoutingCondition'] & { valueLabels?: string[] }

/** One stream of the routing being edited. */
export interface StreamDraft {
  splits: { providerId: string; weight: number }[]
  /** Provider id, or '' for none. */
  fallback: string
  failover: FailoverPolicy
  conditions: Condition[]
}

export type Draft = Record<Stream, StreamDraft>

export const streams: Stream[] = ['broadcast', 'automation', 'system']

/** The routing being edited starts from what is saved. */
export function toDraft(cfg: RoutingConfig): Draft {
  const out = {} as Draft
  for (const s of streams) {
    const v = cfg.streams.find((x) => x.stream === s)
    out[s] = {
      splits: v ? v.splits.map((x) => ({ providerId: x.providerId, weight: x.weight })) : [],
      fallback: v?.fallback ?? '',
      failover: (v?.failover ?? (s === 'broadcast' ? 'pause_and_ask' : 'automatic')) as FailoverPolicy,
      conditions: v ? v.conditions.map((c) => ({ field: c.field, values: [...c.values], providerId: c.providerId, note: c.note, valueLabels: c.valueLabels })) : [],
    }
  }
  return out
}

/** The save body for a draft. */
export function toInput(d: Draft, acknowledgeBulkTerms = false): Schemas['RoutingSaveInput'] {
  return {
    acknowledgeBulkTerms: acknowledgeBulkTerms || undefined,
    streams: streams.map((s) => ({
      stream: s,
      splits: d[s].splits.filter((x) => x.weight > 0),
      fallback: d[s].fallback || undefined,
      failover: d[s].failover,
      conditions: d[s].conditions.map((c) => ({ field: c.field, values: c.values, providerId: c.providerId, note: c.note || undefined })),
    })),
  }
}

/** Whether two drafts route the same way. */
export function sameDraft(a: Draft, b: Draft): boolean {
  return JSON.stringify(toInput(a)) === JSON.stringify(toInput(b))
}

/** Changes of the draft compared to what is saved, counted per stream. */
export function changedStreams(d: Draft, saved: Draft): Stream[] {
  return streams.filter((s) => JSON.stringify(toInput({ ...saved, [s]: d[s] }).streams) !== JSON.stringify(toInput(saved).streams))
}

/**
 * Sets a stream's primary provider, keeping the split weights: a provider
 * already in the split swaps places with the old primary, any other
 * provider takes the primary's place.
 */
export function setPrimary(sd: StreamDraft, providerId: string): StreamDraft {
  const fallback = sd.fallback === providerId ? '' : sd.fallback
  const [first, ...rest] = sd.splits
  if (!first) return { ...sd, splits: [{ providerId, weight: 100 }], fallback }
  if (first.providerId === providerId) return sd
  const i = rest.findIndex((x) => x.providerId === providerId)
  const others = i < 0 ? rest : rest.map((x, j) => (j === i ? { providerId: first.providerId, weight: x.weight } : x))
  return { ...sd, splits: [{ providerId, weight: first.weight }, ...others], fallback }
}

/** Sets the share of a secondary provider; the primary keeps the rest. */
export function setShare(sd: StreamDraft, providerId: string, share: number): StreamDraft {
  const pct = Math.max(0, Math.min(100, Math.round(share)))
  const others = sd.splits.slice(1).map((x) => (x.providerId === providerId ? { ...x, weight: pct } : x))
  const used = others.reduce((n, x) => n + x.weight, 0)
  return { ...sd, splits: [{ ...sd.splits[0]!, weight: Math.max(0, 100 - used) }, ...others] }
}

/** A provider that the draft sends Broadcasts through although its terms forbid bulk mail. */
export interface BulkIssue {
  stream: Stream
  providerId: string
  where: 'fallback' | 'split' | 'condition'
}

/** Bulk-terms problems: Broadcasts may only use providers whose terms allow bulk mail. */
export function bulkIssues(d: Draft, providers: RoutingProvider[]): BulkIssue[] {
  const byId = new Map(providers.map((p) => [p.id, p]))
  const bad = (id: string) => {
    const p = byId.get(id)
    return Boolean(p && !p.bulkAllowed)
  }
  const out: BulkIssue[] = []
  const b = d.broadcast
  if (b.fallback && bad(b.fallback)) out.push({ stream: 'broadcast', providerId: b.fallback, where: 'fallback' })
  for (const s of b.splits) if (s.weight > 0 && bad(s.providerId)) out.push({ stream: 'broadcast', providerId: s.providerId, where: 'split' })
  for (const c of b.conditions) if (bad(c.providerId)) out.push({ stream: 'broadcast', providerId: c.providerId, where: 'condition' })
  return out
}

/** "Amazon SES · us-east-1", "Postal · mta.acme.io", "Postmark · Transactional". */
export function providerOption(p: RoutingProvider, opts: { warming?: boolean; stream?: boolean } = {}): string {
  if (opts.warming && p.warmingUp) return `${p.name} · warming up`
  let label = p.label.replace(/^self-hosted · /, '')
  if (!opts.stream) label = label.replace(/ stream$/, '')
  return label ? `${p.name} · ${label}` : p.name
}

export const mailboxName: Record<string, string> = { gmail: 'Gmail', outlook: 'Outlook', yahoo: 'Yahoo', icloud: 'iCloud', other: 'Other' }

export const fieldText: Record<Condition['field'], string> = {
  mailbox_provider: 'recipient mailbox provider',
  list: 'list',
  sender_domain: 'sender domain',
}

const streamTitle: Record<Stream, string> = { broadcast: 'Broadcasts', automation: 'Automations', system: 'System' }

/** One step of the route tester as a sentence. */
export function stepText(st: Schemas['RouteTestStep'], stream: Stream, hasConditions: boolean): string {
  switch (st.kind) {
    case 'stream':
      return st.conditions > 0 || stream === 'broadcast'
        ? `Stream is ${streamTitle[stream]}, so the ${streamTitle[stream]} rule set applies.`
        : `Stream is ${streamTitle[stream]}: primary ${st.provider ?? 'not set'}, no conditions.`
    case 'condition': {
      const value = st.value ?? ''
      const head = st.field === 'list' ? `list is ${value}` : st.field === 'sender_domain' ? `sender domain is ${value}` : `recipient mailbox provider is ${value}`
      return `Condition matched: ${head}, so ${st.provider} only.${st.warming ? ` ${st.warming} is still warming up.` : ''}`
    }
    case 'split': {
      const split = (st.splits ?? []).map((x) => `${x.short} ${x.weight}`).join(' / ')
      return `${hasConditions ? 'No condition matched. ' : ''}Weighted split ${split}: this address hashes to bucket ${st.bucket} of 100.`
    }
    case 'primary':
      return `Primary ${st.provider}.`
    case 'suppression':
      return st.suppressed ? `Suppression list checked: ${st.email} is suppressed, so nothing is sent to it.` : `Suppression list checked: ${st.email} is not suppressed.`
    case 'held':
      return st.reason === 'webhooks_silent'
        ? `${st.provider}'s webhooks are silent, so its share waits until events resume.`
        : `${st.provider} is down right now, so this email would wait for it or its fallback.`
  }
}

/** What happens if the chosen provider fails. */
export function fallbackText(r: Schemas['RouteTestResult'], stream: Stream): string {
  const fb = r.fallback
  if (fb.policy === 'pause_and_ask') {
    return stream === 'broadcast' ? 'pause and ask. Broadcast failover to a cold provider is never automatic.' : 'pause and ask: the send holds until you decide.'
  }
  if (!fb.provider) return 'no fallback is set, so the send holds until the provider recovers.'
  if (fb.policy === 'warm_only') return `automatic failover to ${fb.provider} once it is warmed up; until then, pause and ask.`
  return stream === 'automation'
    ? `automatic failover to ${fb.provider} after 3 failed sends or 2 failed health checks.`
    : `automatic failover to ${fb.provider}.`
}

/** "$0.0001 · $0.10 per 1,000", "plan · about $0.0044", "$0 metered · self-hosted". */
export function costText(c: Schemas['RouteTestCost']): string {
  if (c.basis === 'self_hosted') return '$0 metered · self-hosted'
  if (c.basis === 'plan') return c.perMessage !== undefined ? `plan · about $${c.perMessage.toFixed(4)}` : 'plan'
  if (c.per1000 === undefined) return '—'
  return `$${(c.per1000 / 1000).toFixed(4)} · ${money(c.per1000)} per 1,000`
}
