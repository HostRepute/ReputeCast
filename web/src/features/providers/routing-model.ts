import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { count, money } from '@/lib/format'
import { streamName, type FailoverPolicy, type Stream } from './common'
import { mr } from './routing.messages'

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
  /** Transactional: first in line on providers that also send campaigns. */
  priority: boolean
}

export type Draft = Record<Stream, StreamDraft>

export const streams: Stream[] = ['broadcast', 'automation', 'system', 'transactional']

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
      priority: v?.priority ?? true,
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
      priority: s === 'transactional' ? d[s].priority : undefined,
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
  if (opts.warming && p.warmingUp) return t(mr.optionWarming, { name: p.name })
  let label = p.label.replace(/^self-hosted · /, '')
  if (!opts.stream) label = label.replace(/ stream$/, '')
  return label ? `${p.name} · ${label}` : p.name
}

/** Mailbox providers by key: names, and "Other" in the app language. */
export const mailboxName: Record<string, string> = {
  gmail: 'Gmail',
  outlook: 'Outlook',
  yahoo: 'Yahoo',
  icloud: 'iCloud',
  get other() {
    return t(mr.mailboxOther)
  },
}

/** A condition's field as its chip says it ("recipient mailbox provider"). */
export const fieldText: Record<Condition['field'], string> = {
  get mailbox_provider() {
    return t(mr.fieldMailboxChip)
  },
  get list() {
    return t(mr.fieldListChip)
  },
  get sender_domain() {
    return t(mr.fieldDomainChip)
  },
  get category() {
    return t(mr.fieldCategoryChip)
  },
}

/** One step of the route tester as a sentence. */
export function stepText(st: Schemas['RouteTestStep'], stream: Stream, hasConditions: boolean): string {
  const name = streamName[stream]
  switch (st.kind) {
    case 'stream':
      return st.conditions > 0 || stream === 'broadcast'
        ? t(mr.stepStreamRules, { stream: name })
        : st.provider
          ? t(mr.stepStreamPrimary, { stream: name, provider: st.provider })
          : t(mr.stepStreamNoPrimary, { stream: name })
    case 'condition': {
      const matched = t(mr.stepCondition, { field: st.field ?? 'mailbox_provider', value: st.value ?? '', provider: st.provider ?? '' })
      return st.warming ? `${matched} ${t(mr.stepStillWarming, { provider: st.warming })}` : matched
    }
    case 'split': {
      const split = (st.splits ?? []).map((x) => `${x.short} ${x.weight}`).join(' / ')
      const line = t(mr.stepSplit, { split, bucket: st.bucket })
      return hasConditions ? `${t(mr.stepNoMatch)} ${line}` : line
    }
    case 'primary':
      return t(mr.stepPrimary, { provider: st.provider ?? '' })
    case 'suppression':
      return st.suppressed ? t(mr.stepSuppressed, { email: st.email ?? '' }) : t(mr.stepNotSuppressed, { email: st.email ?? '' })
    case 'held':
      return st.reason === 'webhooks_silent' ? t(mr.stepSilent, { provider: st.provider ?? '' }) : t(mr.stepDown, { provider: st.provider ?? '' })
  }
}

/** What happens if the chosen provider fails. */
export function fallbackText(r: Schemas['RouteTestResult'], stream: Stream): string {
  const fb = r.fallback
  if (fb.policy === 'pause_and_ask') {
    return stream === 'broadcast' ? t(mr.fallbackPauseBroadcast) : t(mr.fallbackPause)
  }
  if (!fb.provider) return t(mr.fallbackNone)
  if (fb.policy === 'warm_only') return t(mr.fallbackWarmOnly, { provider: fb.provider })
  return stream === 'automation' ? t(mr.fallbackAutomation, { provider: fb.provider }) : t(mr.fallbackAutomatic, { provider: fb.provider })
}

/** "$0.0001 · $0.10 per 1,000", "plan · about $0.0044", "$0 metered · self-hosted". */
export function costText(c: Schemas['RouteTestCost']): string {
  if (c.basis === 'self_hosted') return t(mr.costSelfHosted, { amount: money(0, 0) })
  if (c.basis === 'plan') return c.perMessage !== undefined ? t(mr.costPlanAbout, { amount: money(c.perMessage, 4) }) : t(mr.costPlan)
  if (c.per1000 === undefined) return '—'
  return t(mr.costMetered, { each: money(c.per1000 / 1000, 4), per1000: money(c.per1000), thousand: count(1000) })
}

/** How the provider being removed is used in one stream. */
export type ProviderUse =
  | { kind: 'primary'; stream: Stream }
  | { kind: 'share'; stream: Stream; weight: number }
  | { kind: 'failover'; stream: Stream }
  | { kind: 'condition'; stream: Stream; values: string }

const order: ProviderUse['kind'][] = ['primary', 'share', 'failover', 'condition']

/** Where a provider is used: primaries first, then shares, failovers and conditions. */
export function providerUses(d: Draft, id: string): ProviderUse[] {
  const out: ProviderUse[] = []
  for (const s of streams) {
    const sd = d[s]
    const splits = sd.splits.filter((x) => x.weight > 0)
    if (splits[0]?.providerId === id) out.push({ kind: 'primary', stream: s })
    const share = splits.slice(1).find((x) => x.providerId === id)
    if (share) out.push({ kind: 'share', stream: s, weight: share.weight })
    if (sd.fallback === id) out.push({ kind: 'failover', stream: s })
    for (const c of sd.conditions) if (c.providerId === id) out.push({ kind: 'condition', stream: s, values: (c.valueLabels ?? c.values).join(', ') })
  }
  return out.sort((a, b) => order.indexOf(a.kind) - order.indexOf(b.kind))
}

/** The stream without the provider: the replacement takes its place, a share goes back to the primary. */
export function withoutProvider(sd: StreamDraft, id: string, primary: string, fallback: string): StreamDraft {
  let splits = sd.splits.filter((x) => x.weight > 0)
  const first = splits[0]
  if (first?.providerId === id) {
    const taken = splits.find((x) => x.providerId === primary)?.weight ?? 0
    splits = [{ providerId: primary, weight: first.weight + taken }, ...splits.slice(1).filter((x) => x.providerId !== primary)]
  } else {
    const share = splits.find((x) => x.providerId === id)
    splits = splits.filter((x) => x.providerId !== id)
    if (share && splits[0]) splits[0] = { ...splits[0], weight: splits[0].weight + share.weight }
  }
  let fb = sd.fallback === id ? fallback : sd.fallback
  // A stream never fails over to its own primary.
  if (fb && fb === splits[0]?.providerId) fb = ''
  return { ...sd, splits, fallback: fb, conditions: sd.conditions.filter((c) => c.providerId !== id) }
}
