import { useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { getI18n, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mb } from './bulk.messages'
import { streamName, type FailoverPolicy, type Stream } from './common'
import { providerOption, setPrimary, type BulkIssue, type Draft, type RoutingConfig, type RoutingProvider } from './routing-model'

type Fix = 'create' | 'other' | 'none' | 'keep'

const bulkBadge: Record<RoutingProvider['bulk'], { tone: 'ok' | 'bad' | 'warn'; readonly text: string }> = {
  allowed: {
    tone: 'ok',
    get text() {
      return tt(mb.termsAllowed)
    },
  },
  your_policy: {
    tone: 'ok',
    get text() {
      return tt(mb.termsYourPolicy)
    },
  },
  forbidden: {
    tone: 'bad',
    get text() {
      return tt(mb.termsNotAllowed)
    },
  },
  broadcast_streams_only: {
    tone: 'warn',
    get text() {
      return tt(mb.termsBroadcastOnly)
    },
  },
}

/** "Postmark · Transactional stream", "Postal (self-hosted)". */
function termName(p: RoutingProvider, variant: string) {
  if (!variant) return p.name
  return variant === 'self-hosted' ? tt(mb.selfHosted, { name: p.name }) : `${p.name} · ${variant}`
}

/** "Transactional" from "Transactional stream" (the adapter's own words). */
const variantWord = (label: string) => label.replace(/ stream$/, '')

/** One way to fix the route (`label.opt`): title and explanation flow inline, as on the board. */
function FixOption({
  value,
  fix,
  onPick,
  title,
  description,
  dashed,
  disabled,
}: {
  value: Fix
  fix: Fix
  onPick: (f: Fix) => void
  title: ReactNode
  description: ReactNode
  dashed?: boolean
  disabled?: boolean
}) {
  return (
    // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its radio; title and description are its text
    <label className={cn('opt pv-opt', dashed && 'border-dashed', disabled && 'cursor-not-allowed opacity-60')}>
      <input type="radio" name="pv-fix" value={value} checked={fix === value} disabled={disabled} onChange={() => onPick(value)} />
      <span>
        <span className="font-semibold">{title}</span>
        <br />
        <span className="k">{description}</span>
      </span>
    </label>
  )
}

/**
 * The routing draft would send Broadcasts through a provider whose terms
 * forbid bulk mail: explain it, offer the fixes, show every provider's terms.
 */
export function BulkWarningView({
  cfg,
  draft,
  saved,
  issues,
  saving,
  onDraft,
  onSave,
  readOnlyReason,
}: {
  cfg: RoutingConfig
  draft: Draft
  saved: Draft
  issues: BulkIssue[]
  saving: boolean
  onDraft: (d: Draft) => void
  onSave: (next: Draft, acknowledge: boolean) => Promise<void>
  readOnlyReason?: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const issue = issues[0]!
  const bad = byId.get(issue.providerId)!
  const b = draft.broadcast
  const primary = b.splits[0] ? byId.get(b.splits[0].providerId) : undefined
  const others = b.splits
    .slice(1)
    .map((s) => ({ s, p: byId.get(s.providerId) }))
    .filter((x) => x.p)
  const alternatives = cfg.providers.filter((p) => p.bulkAllowed && p.id !== bad.id && p.id !== primary?.id)
  const alt = alternatives.find((p) => !p.warmingUp) ?? alternatives[0]
  const [fix, setFix] = useState<Fix>(bad.broadcastStreamCreatable ? 'create' : alt ? 'other' : 'none')
  const [busy, setBusy] = useState(false)
  const view = cfg.streams.find((s) => s.stream === 'broadcast')
  const fallbackIssue = issue.where === 'fallback'
  const variant = variantWord(bad.label)
  const role = fallbackIssue ? 'fallback' : 'route'
  const setBroadcast = (patch: Partial<Draft['broadcast']>) => onDraft({ ...draft, broadcast: { ...b, ...patch } })
  // A choice whose label can pass about 32 characters is a radio card, not a
  // select, outside English (CONVENTIONS › Text expansion, rule 1).
  const cards = getI18n().locale !== 'en'
  const policies: { value: FailoverPolicy; label: string }[] = [
    { value: 'pause_and_ask', label: t(mb.policyPause) },
    { value: 'warm_only', label: t(mb.policyWarm) },
    { value: 'automatic', label: t(mb.policyAuto) },
  ]

  const apply = async () => {
    if (fix === 'other' && alt) {
      const next: Draft = { ...draft, broadcast: { ...b, fallback: alt.id } }
      onDraft(next)
      await onSave(next, false)
      return
    }
    if (fix === 'none') {
      const next: Draft = { ...draft, broadcast: { ...b, fallback: '', failover: 'pause_and_ask' } }
      onDraft(next)
      await onSave(next, false)
      return
    }
    if (fix === 'keep') {
      await onSave(draft, true)
      return
    }
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/providers/{providerId}/broadcast-stream', { params: { path: { workspace: slug, providerId: bad.id } } })
      toast({ kind: 'success', title: t(mb.streamCreated, { name: bad.name }), detail: t(mb.streamCreatedDetail) })
      await onSave(draft, false)
    } catch (e) {
      toast({ kind: 'error', title: t(mb.streamFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const action: Record<Fix, string> = {
    create: t(mb.actionCreate),
    other: alt ? t(mb.actionOther, { provider: alt.short }) : t(common.save),
    none: t(mb.actionNone),
    keep: t(mb.actionKeep),
  }
  const body = [
    t(mb.bodySet, { where: issue.where, name: bad.name, variant, hasVariant: variant ? 'yes' : 'no' }),
    t(mb.bodyTerms, { name: bad.name, hasVariant: variant ? 'yes' : 'no', variantLower: variant.toLowerCase() }),
    ...(bad.broadcastStreamCreatable ? [t(mb.bodyCreatable, { name: bad.name })] : []),
  ].join(' ')

  return (
    <>
      <section role="alert" className="flex flex-wrap items-start gap-[18px] rounded-[12px] border border-[#FCD34D] bg-warn-tint px-[22px] py-[18px]">
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
          <circle cx="20" cy="20" r="19" fill="#FDE68A" />
          <path d="M20 11v11M20 27v1" stroke="#92400E" strokeWidth="3" strokeLinecap="round" />
        </svg>
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[18px] font-bold tracking-[-0.01em] text-[#78350F]">{fallbackIssue ? t(mb.titleFallback) : t(mb.titleRoute)}</span>
          <span className="text-[#78350F]">{body}</span>
        </div>
      </section>

      <section className="card flex flex-col" aria-labelledby="pv-bw-route">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
          <h2 className="h2" id="pv-bw-route">
            {t(mb.routeTitle)}
          </h2>
          <span className="k">
            {view?.volume?.smallest
              ? t(mb.routeSubVolume, { smallest: count(view.volume.smallest), largest: count(view.volume.largest ?? view.volume.smallest) })
              : t(mb.routeSub)}
          </span>
        </div>
        <Grid min={260} gap={16} className="px-5 py-[18px]">
          <Field
            label={t(mb.primary)}
            hint={
              others.length
                ? t(mb.weightedWith, {
                    list: others
                      .map((o) => (o.p!.warmingUp ? t(mb.weightItemWarming, { provider: o.p!.short, share: pct(o.s.weight, 0) }) : t(mb.weightItem, { provider: o.p!.short, share: pct(o.s.weight, 0) })))
                      .join('; '),
                  })
                : undefined
            }
          >
            <Select value={primary?.id ?? ''} disabled={Boolean(readOnlyReason)} onChange={(e) => onDraft({ ...draft, broadcast: setPrimary(b, e.target.value) })}>
              {primary ? null : <option value="">{t(mb.chooseProvider)}</option>}
              {cfg.providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {providerOption(p)}
                  {b.splits.length > 1 ? ` · ${pct(b.splits[0]!.weight, 0)}` : ''}
                </option>
              ))}
            </Select>
          </Field>
          <div className={cn('lbl', fallbackIssue && 'text-warn-fg')}>
            <label htmlFor="pv-bw-fallback">{t(mb.fallback)}</label>
            <Select
              id="pv-bw-fallback"
              value={b.fallback}
              disabled={Boolean(readOnlyReason)}
              aria-invalid={fallbackIssue || undefined}
              aria-describedby={fallbackIssue ? 'pv-bw-fallback-terms' : undefined}
              className={cn(fallbackIssue && 'border-[#F59E0B] bg-warn-tint')}
              onChange={(e) => setBroadcast({ fallback: e.target.value })}
            >
              <option value="">{t(mb.nonePauseAsk)}</option>
              {cfg.providers
                .filter((p) => p.id !== primary?.id)
                .map((p) => (
                  <option key={p.id} value={p.id}>
                    {providerOption(p, { warming: true, stream: true })}
                  </option>
                ))}
            </Select>
            {fallbackIssue ? (
              <Badge tone="warn" id="pv-bw-fallback-terms" className="mt-0.5 self-start">
                {t(mb.termsBadge)}
              </Badge>
            ) : null}
          </div>
          {cards ? (
            // Outside English the policies run past what a select shows (rule 1): radio cards.
            <div className="lbl" role="radiogroup" aria-labelledby="pv-bw-policy" aria-describedby="pv-bw-policy-hint">
              <span id="pv-bw-policy">{t(mb.failoverPolicy)}</span>
              {policies.map((p) => (
                <label key={p.value} className="opt pv-opt">
                  <input
                    type="radio"
                    name="pv-bw-failover"
                    value={p.value}
                    checked={b.failover === p.value}
                    disabled={Boolean(readOnlyReason)}
                    onChange={() => setBroadcast({ failover: p.value })}
                  />
                  <span className="text-[13px] font-medium text-ink">{p.label}</span>
                </label>
              ))}
              <span className="k" id="pv-bw-policy-hint">
                {t(mb.failoverHint)}
              </span>
            </div>
          ) : (
            <Field label={t(mb.failoverPolicy)} hint={t(mb.failoverHint)}>
              <Select value={b.failover} disabled={Boolean(readOnlyReason)} onChange={(e) => setBroadcast({ failover: e.target.value as FailoverPolicy })}>
                {policies.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </Select>
            </Field>
          )}
        </Grid>
        <div className="flex flex-col gap-2.5 px-5 pb-[18px]" id="pv-fix">
          <span className="text-[13px] font-semibold" id="pv-fix-title">
            {t(mb.fixTitle)}
          </span>
          <Grid min={280} gap={12} role="radiogroup" aria-labelledby="pv-fix-title">
            {bad.broadcastStreamCreatable ? (
              <FixOption
                value="create"
                fix={fix}
                onPick={setFix}
                title={t(mb.createTitle, { name: bad.name })}
                description={t(mb.createDescription, { name: bad.name, role })}
              />
            ) : null}
            <FixOption
              value="other"
              fix={fix}
              onPick={setFix}
              disabled={!alt}
              title={t(mb.otherTitle, { role })}
              description={
                !alt
                  ? t(mb.otherNone)
                  : alternatives.length === 1 && alt.warmingUp
                    ? t(mb.otherWarming, { provider: alt.short, day: alt.warmupDays ?? 21 })
                    : t(mb.otherUse, { name: alt.name })
              }
            />
            {fallbackIssue ? (
              <FixOption
                value="none"
                fix={fix}
                onPick={setFix}
                title={t(mb.noneTitle)}
                description={t(mb.noneDescription, { primary: primary?.short ?? t(mb.thePrimary) })}
              />
            ) : null}
            <FixOption
              value="keep"
              fix={fix}
              onPick={setFix}
              dashed
              title={<span className="text-bad-fg">{t(mb.keepTitle, { name: bad.name, variant })}</span>}
              description={t(mb.keepDescription)}
            />
          </Grid>
          <div className="flex flex-wrap justify-end gap-2.5">
            <Button
              onClick={() => onDraft({ ...draft, broadcast: { ...b, fallback: saved.broadcast.fallback, splits: saved.broadcast.splits, conditions: saved.broadcast.conditions } })}
            >
              {t(common.cancel)}
            </Button>
            <Button variant="primary" disabledReason={readOnlyReason} loading={busy || saving} onClick={() => void apply()}>
              {action[fix]}
            </Button>
          </div>
        </div>
      </section>

      <section className="card" aria-labelledby="pv-terms">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
          <h2 className="h2" id="pv-terms">
            {t(mb.termsTitle)}
          </h2>
          <span className="k">{t(mb.termsSub)}</span>
        </div>
        <Table className="pv-11 pv-top" phoneWrap>
          <thead>
            <tr>
              <Th>{t(mb.colProvider)}</Th>
              <Th>{t(mb.colBulk)}</Th>
              <Th>{t(mb.colNotes)}</Th>
              <Th>{t(mb.colStreams)}</Th>
            </tr>
          </thead>
          <tbody>
            {cfg.providers.flatMap((p) =>
              p.terms.map((term) => {
                const flagged = p.id === bad.id && term.active && term.bulk === 'forbidden'
                const badge = bulkBadge[term.bulk]
                return (
                  <tr key={`${p.id}-${term.variant}`} className={cn(flagged && 'bg-warn-tint')}>
                    <Td className="font-semibold">{termName(p, term.variant)}</Td>
                    <Td>
                      <Badge tone={badge.tone}>{badge.text}</Badge>
                    </Td>
                    <Td>
                      {term.note}
                      {!term.available ? ` ${t(mb.notCreated)}` : ''}
                    </Td>
                    <Td>{term.streams.map((s) => streamName[s as Stream]).join(' · ')}</Td>
                  </tr>
                )
              }),
            )}
          </tbody>
        </Table>
      </section>
    </>
  )
}
