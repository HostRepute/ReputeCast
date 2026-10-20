import { useId, type ReactNode } from 'react'
import { $api } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { newRuleId, type VisibilityCondition, type VisibilityRule } from '@/features/conditional/rules'
import { mc } from './composer.messages'
import './composer.css'

/**
 * Visibility · Paid access (ComposerEarnBlocks, brief §3.7): a block shown
 * to any paid tier, to a tier and above, or to free subscribers only. The
 * server's paid access condition (internal/earn, kind "paid") decides it
 * per reader at send time.
 */

export const KIND_PAID = 'paid'
export type PaidOp = 'any' | 'at_least' | 'free'

type T = ReturnType<typeof useT>

/** The composer's paid content: tiers with their price lines, the connection, the upgrade page. */
export function useEarnComposer() {
  const { slug } = useComposer()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/earn/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
}

/** A rule that is Paid access alone: Visibility shows the Paid access choice for it. */
export function isPaidRule(r: VisibilityRule | null | undefined): boolean {
  return !!r && r.show === 'only' && r.conditions.length === 1 && r.conditions[0]!.kind === KIND_PAID
}

/** A Paid access rule, keeping the block's id. */
export function paidRule(base: VisibilityRule | null | undefined, cond: VisibilityCondition): VisibilityRule {
  return { id: base?.id ?? newRuleId(), show: 'only', match: 'all', conditions: [cond] }
}

/** What a tier choice says: "Insider and above", or the top tier alone ("Founding member"). */
export function tierChoice(t: T, tier: { short: string; top: boolean }): string {
  return tier.top ? tier.short : t(mc.tierChoice, { tier: tier.short })
}

/** The condition picker's words for the paid access kind. */
export function paidOpLabel(t: T, op: string): string {
  return op === 'free' ? t(mc.opFree) : op === 'at_least' ? t(mc.opTier) : t(mc.opAny)
}

/** A new paid access condition (the picker's default): any paid tier. */
export function paidDefault(): VisibilityCondition {
  return { kind: KIND_PAID, op: 'any' }
}

/** The value of a paid access condition: the tier, for "a tier and above". */
export function PaidTierSelect({ cond, onChange, disabled }: { cond: VisibilityCondition; onChange: (c: VisibilityCondition) => void; disabled?: boolean }) {
  const t = useT()
  const tiers = useEarnComposer().data?.tiers ?? []
  if (cond.op !== 'at_least') return <span className="k self-center">—</span>
  const known = tiers.some((x) => x.id === cond.value)
  return (
    <select
      className="in"
      aria-label={t(mc.tierLabel)}
      value={cond.value ?? ''}
      disabled={disabled}
      onChange={(e) => {
        const tier = tiers.find((x) => x.id === e.target.value)
        if (tier) onChange({ ...cond, value: tier.id, label: tierChoice(t, tier) })
      }}
    >
      {!known ? <option value={cond.value ?? ''}>{cond.label || t(mc.pickTier)}</option> : null}
      {tiers.map((x) => (
        <option key={x.id} value={x.id}>
          {tierChoice(t, x)}
        </option>
      ))}
    </select>
  )
}

/** Visibility's Paid access choice: any paid tier, a tier and above, or free subscribers only. */
export function PaidAccessChoice({ rule, disabled, onChange }: { rule: VisibilityRule; disabled?: boolean; onChange: (r: VisibilityRule) => void }) {
  const t = useT()
  const id = useId()
  const tiers = useEarnComposer().data?.tiers ?? []
  const cond = rule.conditions[0]!
  const op = (cond.op as PaidOp) || 'any'
  // The Upgrade block's own condition keeps its mark (hidden until payments are connected).
  const mark = cond.field ? { field: cond.field } : {}
  const set = (next: PaidOp, tierId?: string) => {
    if (next !== 'at_least') return onChange(paidRule(rule, { kind: KIND_PAID, op: next, ...mark }))
    const tier = tiers.find((x) => x.id === tierId) ?? tiers.find((x) => x.id === cond.value) ?? tiers[0]
    if (tier) onChange(paidRule(rule, { kind: KIND_PAID, op: next, value: tier.id, label: tierChoice(t, tier), ...mark }))
  }
  const current = op === 'at_least' ? tiers.find((x) => x.id === cond.value) : undefined
  const option = (value: PaidOp, label: string, extra?: ReactNode) => (
    <label className={`ea-popt${op === value ? ' on' : ''}`}>
      <input type="radio" name={id} checked={op === value} disabled={disabled || (value === 'at_least' && !tiers.length)} onChange={() => set(value)} />
      <span className="flex min-w-0 flex-col gap-1.5">
        {label}
        {extra}
      </span>
    </label>
  )
  return (
    <div className="ea-paid-sub">
      <div className="flex flex-col gap-1.5" role="radiogroup" aria-label={t(mc.show)}>
        {option('any', t(mc.anyTier))}
        {option(
          'at_least',
          t(mc.tierAndAbove),
          op === 'at_least' ? (
            <select className="in" aria-label={t(mc.tierLabel)} value={cond.value ?? ''} disabled={disabled} onChange={(e) => set('at_least', e.target.value)}>
              {!current ? <option value={cond.value ?? ''}>{cond.label || t(mc.pickTier)}</option> : null}
              {tiers.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
            </select>
          ) : null,
        )}
        {option('free', t(mc.freeOnly))}
      </div>
      {tiers.length ? null : <span className="k">{t(mc.noTiers)}</span>}
      <span className="k">
        {t(mc.includes)}
        {current?.includes.length ? ` ${t(mc.includesTier, { tier: current.short, lower: formatList(current.includes) })}` : ''}
      </span>
    </div>
  )
}
