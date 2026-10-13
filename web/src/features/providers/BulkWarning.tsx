import { useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { streamName, type FailoverPolicy, type Stream } from './common'
import { providerOption, setPrimary, type BulkIssue, type Draft, type RoutingConfig, type RoutingProvider } from './routing-model'

type Fix = 'create' | 'other' | 'none' | 'keep'

const bulkBadge: Record<RoutingProvider['bulk'], { tone: 'ok' | 'bad' | 'warn'; text: string }> = {
  allowed: { tone: 'ok', text: 'Allowed' },
  your_policy: { tone: 'ok', text: 'Your own policy' },
  forbidden: { tone: 'bad', text: 'Not allowed' },
  broadcast_streams_only: { tone: 'warn', text: 'Broadcast streams only' },
}

/** "Postmark · Transactional stream", "Postal (self-hosted)". */
function termName(p: RoutingProvider, variant: string) {
  if (!variant) return p.name
  return variant === 'self-hosted' ? `${p.name} (self-hosted)` : `${p.name} · ${variant}`
}

/** "Transactional" from "Transactional stream". */
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
  const { slug } = useWorkspace()
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const issue = issues[0]!
  const bad = byId.get(issue.providerId)!
  const b = draft.broadcast
  const primary = b.splits[0] ? byId.get(b.splits[0].providerId) : undefined
  const others = b.splits.slice(1).map((s) => ({ s, p: byId.get(s.providerId) })).filter((x) => x.p)
  const alternatives = cfg.providers.filter((p) => p.bulkAllowed && p.id !== bad.id && p.id !== primary?.id)
  const alt = alternatives.find((p) => !p.warmingUp) ?? alternatives[0]
  const [fix, setFix] = useState<Fix>(bad.broadcastStreamCreatable ? 'create' : alt ? 'other' : 'none')
  const [busy, setBusy] = useState(false)
  const view = cfg.streams.find((s) => s.stream === 'broadcast')
  const fallbackIssue = issue.where === 'fallback'
  const variant = variantWord(bad.label)
  const setBroadcast = (patch: Partial<Draft['broadcast']>) => onDraft({ ...draft, broadcast: { ...b, ...patch } })

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
      toast({ kind: 'success', title: `${bad.name} Broadcast stream created`, detail: 'Webhooks are registered for it too.' })
      await onSave(draft, false)
    } catch (e) {
      toast({ kind: 'error', title: "The Broadcast stream wasn't created", detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const action: Record<Fix, string> = {
    create: 'Create Broadcast stream and save',
    other: alt ? `Use ${alt.short} and save` : 'Save',
    none: 'Save without a fallback',
    keep: 'Keep it and save',
  }

  return (
    <>
      <section role="alert" className="flex flex-wrap items-start gap-[18px] rounded-[12px] border border-[#FCD34D] bg-warn-tint px-[22px] py-[18px]">
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
          <circle cx="20" cy="20" r="19" fill="#FDE68A" />
          <path d="M20 11v11M20 27v1" stroke="#92400E" strokeWidth="3" strokeLinecap="round" />
        </svg>
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[18px] font-bold tracking-[-0.01em] text-[#78350F]">
            Broadcasts would {fallbackIssue ? 'fail over to' : 'go through'} a provider whose terms don&apos;t allow bulk marketing
          </span>
          <span className="text-[#78350F]">
            You set the Broadcasts {issue.where === 'fallback' ? 'fallback' : issue.where === 'split' ? 'split' : 'condition'} to {bad.name}&apos;s &quot;{variant}&quot;
            {variant ? ' message stream' : ''}. {bad.name}&apos;s terms restrict {variant ? `${variant.toLowerCase()} streams` : 'this account'} to one-to-one mail; newsletters on it
            risk account suspension.
            {bad.broadcastStreamCreatable ? ` ${bad.name} does allow bulk mail on a Broadcast message stream, and ReputeCast can create one.` : ''}
          </span>
        </div>
      </section>

      <section className="card flex flex-col" aria-labelledby="pv-bw-route">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
          <h2 className="h2" id="pv-bw-route">
            Broadcasts route
          </h2>
          <span className="k">
            campaigns
            {view?.volume?.smallest ? ` · ${count(view.volume.smallest)} to ${count(view.volume.largest ?? view.volume.smallest)} recipients per send` : ''}
          </span>
        </div>
        <Grid min={260} gap={16} className="px-5 py-[18px]">
          <Field label="Primary" hint={others.length ? `Weighted with ${others.map((o) => `${o.p!.short} · ${o.s.weight}%${o.p!.warmingUp ? ', warming up' : ''}`).join('; ')}` : undefined}>
            <Select value={primary?.id ?? ''} disabled={Boolean(readOnlyReason)} onChange={(e) => onDraft({ ...draft, broadcast: setPrimary(b, e.target.value) })}>
              {primary ? null : <option value="">Choose a provider</option>}
              {cfg.providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {providerOption(p)}
                  {b.splits.length > 1 ? ` · ${b.splits[0]!.weight}%` : ''}
                </option>
              ))}
            </Select>
          </Field>
          <div className={cn('lbl', fallbackIssue && 'text-warn-fg')}>
            <label htmlFor="pv-bw-fallback">Fallback</label>
            <Select
              id="pv-bw-fallback"
              value={b.fallback}
              disabled={Boolean(readOnlyReason)}
              aria-invalid={fallbackIssue || undefined}
              aria-describedby={fallbackIssue ? 'pv-bw-fallback-terms' : undefined}
              className={cn(fallbackIssue && 'border-[#F59E0B] bg-warn-tint')}
              onChange={(e) => setBroadcast({ fallback: e.target.value })}
            >
              <option value="">None · pause and ask</option>
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
                Terms: bulk marketing not allowed
              </Badge>
            ) : null}
          </div>
          <Field label="Failover policy" hint="Applies when the primary fails during a send.">
            <Select value={b.failover} disabled={Boolean(readOnlyReason)} onChange={(e) => setBroadcast({ failover: e.target.value as FailoverPolicy })}>
              <option value="pause_and_ask">Pause and ask (default for Broadcasts)</option>
              <option value="warm_only">Automatic, only to warmed-up providers</option>
              <option value="automatic">Automatic</option>
            </Select>
          </Field>
        </Grid>
        <div className="flex flex-col gap-2.5 px-5 pb-[18px]" id="pv-fix">
          <span className="text-[13px] font-semibold" id="pv-fix-title">
            Fix this
          </span>
          <Grid min={280} gap={12} role="radiogroup" aria-labelledby="pv-fix-title">
            {bad.broadcastStreamCreatable ? (
              <FixOption
                value="create"
                fix={fix}
                onPick={setFix}
                title={`Create a ${bad.name} Broadcast stream (recommended)`}
                description={`ReputeCast creates "ReputeCast Broadcasts" in your ${bad.name} server, registers webhooks and uses it as the ${fallbackIssue ? 'fallback' : 'route'}. About 1 minute.`}
              />
            ) : null}
            <FixOption
              value="other"
              fix={fix}
              onPick={setFix}
              disabled={!alt}
              title={`Use a different ${fallbackIssue ? 'fallback' : 'provider'}`}
              description={
                !alt
                  ? "No other connected provider's terms allow bulk mail yet."
                  : alternatives.length === 1 && alt.warmingUp
                    ? `${alt.short} is the only other provider, and it's warming up: failover to it would be blocked by the warm-up guard until day ${alt.warmupDays ?? 21}.`
                    : `Use ${alt.name} instead.`
              }
            />
            {fallbackIssue ? (
              <FixOption
                value="none"
                fix={fix}
                onPick={setFix}
                title="No fallback, pause and ask"
                description={`If ${primary?.short ?? 'the primary'} fails mid-send, the campaign holds and you decide.`}
              />
            ) : null}
            <FixOption
              value="keep"
              fix={fix}
              onPick={setFix}
              dashed
              title={<span className="text-bad-fg">{`Keep ${bad.name} ${variant} anyway`}</span>}
              description="Recorded in the audit log. Send Readiness will show a Block on every broadcast until this is changed."
            />
          </Grid>
          <div className="flex flex-wrap justify-end gap-2.5">
            <Button
              onClick={() => onDraft({ ...draft, broadcast: { ...b, fallback: saved.broadcast.fallback, splits: saved.broadcast.splits, conditions: saved.broadcast.conditions } })}
            >
              Cancel
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
            What each connected provider&apos;s terms allow
          </h2>
          <span className="k">from each adapter&apos;s manifest · reviewed quarterly</span>
        </div>
        <Table className="pv-11 pv-top" phoneWrap>
          <thead>
            <tr>
              <Th>Provider</Th>
              <Th>Bulk marketing</Th>
              <Th>Notes from the adapter</Th>
              <Th>Streams it can serve</Th>
            </tr>
          </thead>
          <tbody>
            {cfg.providers.flatMap((p) =>
              p.terms.map((t) => {
                const flagged = p.id === bad.id && t.active && t.bulk === 'forbidden'
                const badge = bulkBadge[t.bulk]
                return (
                  <tr key={`${p.id}-${t.variant}`} className={cn(flagged && 'bg-warn-tint')}>
                    <Td className="font-semibold">{termName(p, t.variant)}</Td>
                    <Td>
                      <Badge tone={badge.tone}>{badge.text}</Badge>
                    </Td>
                    <Td>
                      {t.note}
                      {!t.available ? ' Not created yet in this workspace.' : ''}
                    </Td>
                    <Td>{t.streams.map((s) => streamName[s as Stream]).join(' · ')}</Td>
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
