import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Segmented } from '@/components/ui/Tabs'
import { formatList, T, t as tt, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './automation.messages'

/*
 * The automation triggers for paid members (M57, G-11): Becomes a paid
 * member, Changes tier, Stops paying and Trial ends soon. Earn queues their
 * events for Stripe subscriptions in live mode; the step panel
 * (features/automations/panels.tsx) shows these settings, and the canvas
 * names them with the text functions below, as
 * internal/automations/paid.go does.
 */

type Trigger = Schemas['AutomationTrigger']
type Kind = Trigger['kind']
type TierOption = Schemas['AutomationTierOption']

export const paidTriggerKinds = ['paid_started', 'paid_tier_changed', 'paid_ended', 'paid_trial_ending', 'paid_paused', 'paid_resumed'] as const satisfies readonly Kind[]

export function isPaidTrigger(kind: string | undefined): boolean {
  return (paidTriggerKinds as readonly string[]).includes(kind ?? '')
}

/** A paid trigger's settings when it's picked: a member who comes back can go through again. */
export function paidTriggerDefaults(kind: string): Partial<Trigger> {
  return isPaidTrigger(kind) ? { reentry: 'always' } : {}
}

/** "Becomes a paid member", "Changes tier": the palette's and the select's names. */
export function paidTriggerName(kind: Kind): string {
  switch (kind) {
    case 'paid_started':
      return tt(m.kindStarted)
    case 'paid_tier_changed':
      return tt(m.kindTierChanged)
    case 'paid_ended':
      return tt(m.kindEnded)
    case 'paid_paused':
      return tt(m.kindPaused)
    case 'paid_resumed':
      return tt(m.kindResumed)
  }
  return tt(m.kindTrialEnding)
}

/** The palette's heading over the paid triggers. */
export function paidGroupName(): string {
  return tt(m.group)
}

const reasonOrder = ['canceled', 'payment_failed', 'trial_ended'] as const
type Reason = (typeof reasonOrder)[number]
const reasonWords = { canceled: m.wordCanceled, payment_failed: m.wordFailed, trial_ended: m.wordTrialEnded }

/** What a paid trigger points at: "any tier", "Upgrades to Founding member", "canceled or payment failed". */
export function paidTriggerRef(trig: Trigger, tiers?: ReadonlyMap<string, string>): string {
  const tier = trig.tierId ? (tiers?.get(trig.tierId) ?? tt(m.refDeletedTier)) : ''
  if (trig.kind === 'paid_tier_changed') {
    if (!trig.tierId) return tt(trig.direction === 'upgrade' ? m.refAnyUpgrade : trig.direction === 'downgrade' ? m.refAnyDowngrade : m.refAnyChange)
    return tt(trig.direction === 'upgrade' ? m.refUpgrade : trig.direction === 'downgrade' ? m.refDowngrade : m.refChange, { tier })
  }
  if (trig.kind === 'paid_ended') {
    const picked = reasonOrder.filter((r) => trig.endReasons?.includes(r))
    const reasons = picked.length ? formatList(picked.map((r) => tt(reasonWords[r])), 'disjunction') : tt(m.refAnyReason)
    return tier ? tt(m.refTierReasons, { tier, reasons }) : reasons
  }
  return tier || tt(m.refAnyTier)
}

function TierSelect({ value, tiers, disabled, label, onChange }: { value?: string; tiers?: TierOption[]; disabled: boolean; label: string; onChange: (id: string | undefined) => void }) {
  const t = useT()
  const known = !value || (tiers ?? []).some((x) => x.id === value)
  return (
    <label className="lbl flex flex-col gap-1">
      {label}
      <select className="in" disabled={disabled} value={value ?? ''} onChange={(e) => onChange(e.target.value || undefined)}>
        <option value="">{t(m.anyTierOption)}</option>
        {(tiers ?? []).map((x) => (
          <option key={x.id} value={x.id}>
            {x.name}
          </option>
        ))}
        {known ? null : <option value={value}>{tiers ? t(m.deletedTierOption) : t(m.loadingTiers)}</option>}
      </select>
    </label>
  )
}

// A Liquid tag, kept on one line.
const liquid = (tag: string) => <span className="mono whitespace-nowrap">{`{{ ${tag} }}`}</span>

/**
 * A paid trigger's settings: the tier (any when none is picked; for Changes
 * tier, the tier moved to), Changes tier's direction and Stops paying's
 * reasons, then what starts the flow and the data its emails can use.
 */
export function PaidTriggerFields({ t: trig, set, disabled, tiers }: { t: Trigger; set: (patch: Partial<Trigger>) => void; disabled: boolean; tiers?: TierOption[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const changes = trig.kind === 'paid_tier_changed'
  const ended = trig.kind === 'paid_ended'
  const reasons = trig.endReasons ?? []
  const toggle = (r: Reason, on: boolean) => {
    const next = reasonOrder.filter((x) => (x === r ? on : reasons.includes(x)))
    set({ endReasons: next.length ? next : undefined })
  }
  const note = {
    paid_started: m.startedNote,
    paid_tier_changed: m.directionHint,
    paid_ended: m.endedNote,
    paid_trial_ending: m.trialEndingNote,
    paid_paused: m.pausedNote,
    paid_resumed: m.resumedNote,
  }[trig.kind as (typeof paidTriggerKinds)[number]]
  return (
    <div className="flex flex-col gap-2">
      {tiers && tiers.length === 0 ? (
        <span className="k">
          {t(m.noTiers)}{' '}
          <Link to="/w/$workspace/earn/plans" params={{ workspace: slug }}>
            {t(m.setUpTiers)}
          </Link>
        </span>
      ) : (
        <TierSelect value={trig.tierId} tiers={tiers} disabled={disabled} label={t(changes ? m.newTier : m.tier)} onChange={(tierId) => set({ tierId })} />
      )}
      {changes ? (
        <fieldset className="m-0 flex flex-col gap-1 border-0 p-0" disabled={disabled}>
          <legend className="lbl mb-1 p-0">{t(m.direction)}</legend>
          <Segmented
            label={t(m.direction)}
            value={trig.direction === 'upgrade' || trig.direction === 'downgrade' ? trig.direction : 'any'}
            onChange={(v) => set({ direction: v === 'any' ? undefined : v })}
            items={[
              { value: 'any', label: t(m.directionAny) },
              { value: 'upgrade', label: t(m.directionUp) },
              { value: 'downgrade', label: t(m.directionDown) },
            ]}
          />
        </fieldset>
      ) : null}
      {ended ? (
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0" disabled={disabled} aria-describedby="paid-reasons-hint">
          <legend className="lbl mb-1 p-0">{t(m.reasons)}</legend>
          {reasonOrder.map((r) => (
            <label key={r} className="fe-chk">
              <input type="checkbox" checked={reasons.includes(r)} onChange={(e) => toggle(r, e.target.checked)} />
              {t({ canceled: m.reasonCanceled, payment_failed: m.reasonFailed, trial_ended: m.reasonTrialEnded }[r])}
            </label>
          ))}
          <span className="k" id="paid-reasons-hint">
            {t(m.reasonsHint)}
          </span>
        </fieldset>
      ) : null}
      {note ? <span className="k">{t(note)}</span> : null}
      <span className="k">{t(m.liveOnly)}</span>
      <span className="k">
        {trig.kind === 'paid_paused' ? (
          <T msg={m.liquidPaused} values={{ tier: liquid('subscription.tier'), resumes: liquid('subscription.resumes_at') }} />
        ) : (
          <T msg={m.liquid} values={{ tier: liquid('subscription.tier'), previous: liquid('subscription.previous_tier'), trialEnds: liquid('subscription.trial_ends_at') }} />
        )}
      </span>
    </div>
  )
}
