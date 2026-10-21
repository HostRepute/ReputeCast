import type { Component } from 'grapesjs'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { ButtonLink } from '@/components/ui/Button'
import { T, useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { tagOf } from '@/features/composer/editor/labels'
import { mc } from './composer.messages'
import { useEarnComposer } from './PaidVisibility'
import { ATTR_UPGRADE, encodeUpgrade, parseUpgrade, upgradeColumn, type UpgradeConfig } from './upgrade'

/** The Upgrade block a component is, null when it isn't one. */
export function upgradeOf(c: Component | null | undefined): UpgradeConfig | null {
  if (!c || tagOf(c) !== 'mj-section') return null
  return parseUpgrade((c.getAttributes() as Record<string, unknown>)[ATTR_UPGRADE])
}

/** Writes an Upgrade block's settings: its attribute and the card they draw, in one undo step. */
export function applyUpgrade(comp: Component, cfg: UpgradeConfig) {
  comp.addAttributes({ [ATTR_UPGRADE]: encodeUpgrade(cfg) })
  comp.components(upgradeColumn(cfg))
}

function Prop({ title, children, hint }: { title: ReactNode; children?: ReactNode; hint?: ReactNode }) {
  const id = useId()
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
      {hint ? <span className="k">{hint}</span> : null}
    </div>
  )
}

/** A text input that commits on Enter or blur (one undo step per edit). */
function Draft({ value, onCommit, label, disabled }: { value: string; onCommit: (v: string) => void; label: string; disabled?: boolean }) {
  const [draft, setDraft] = useState(value)
  const focused = useRef(false)
  useEffect(() => {
    if (!focused.current) setDraft(value)
  }, [value])
  const commit = () => draft.trim() !== value && draft.trim() && onCommit(draft.trim())
  return (
    <input
      className="in"
      aria-label={label}
      value={draft}
      disabled={disabled}
      onFocus={() => (focused.current = true)}
      onBlur={() => {
        focused.current = false
        commit()
      }}
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          commit()
          ;(e.target as HTMLInputElement).blur()
        }
      }}
    />
  )
}

/**
 * The Upgrade block's settings (ComposerEarnBlocks): headline, line, the
 * tier whose price and trial it reads, the button and an offer (X17). The
 * price line follows the tier, or the offer: it is written again when their
 * prices changed, and goes back to the tier's once the offer ended.
 */
export function UpgradeSettings({ c: comp }: { c: Component }) {
  const t = useT()
  const c = useComposer()
  const canConnect = useCan('payments:manage')
  const cfg = upgradeOf(comp)
  const data = useEarnComposer()
  const tiers = data.data?.tiers ?? []
  const disabled = !c.canEdit
  const tier = cfg ? (tiers.find((x) => x.id === cfg.tier) ?? null) : null
  const offers = tier?.offers ?? []
  const offer = cfg?.offer ? (offers.find((o) => o.code === cfg.offer) ?? null) : null
  const defaultButton = (x: { trialDays: number; short: string }) => (x.trialDays > 0 ? t(mc.defaultButton) : t(mc.defaultButtonNoTrial, { tier: x.short }))
  // A new block takes the first tier; an offer that ended goes back to the
  // tier's price and button; a tier or offer whose prices changed rewrites
  // the line.
  const first = tiers[0]
  let next: UpgradeConfig | null = null
  if (cfg && !disabled && data.data) {
    if (!cfg.tier && first) {
      next = { ...cfg, tier: first.id, price: first.priceLine, headline: t(mc.defaultHeadline, { tier: first.short }), button: defaultButton(first), offer: '' }
    } else if (tier && cfg.offer && !offer) {
      next = { ...cfg, offer: '', price: tier.priceLine, button: defaultButton(tier) }
    } else if (tier && offer && offer.line !== cfg.price) {
      next = { ...cfg, price: offer.line }
    } else if (tier && !cfg.offer && tier.priceLine !== cfg.price) {
      next = { ...cfg, price: tier.priceLine }
    }
  }
  const nextKey = next ? encodeUpgrade(next) : ''
  useEffect(() => {
    if (next) applyUpgrade(comp, next)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per change of what the card should say
  }, [nextKey])
  if (!cfg || c.kind === 'transactional') return null
  const set = (patch: Partial<UpgradeConfig>) => applyUpgrade(comp, { ...cfg, ...patch })
  return (
    <>
      <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <span className="font-semibold">{t(mc.upgradeTitle)}</span>
        <span className="k">{t(mc.upgradeSub)}</span>
      </div>
      {data.data && !data.data.connected ? (
        <div className="prop">
          <div className="note w" role="note">
            <span>{t(mc.notConnected)}</span>
          </div>
          {canConnect ? (
            <ButtonLink size="sm" className="self-start" to="/w/$workspace/earn/connect" params={{ workspace: c.slug }}>
              {t(mc.connect)}
            </ButtonLink>
          ) : null}
        </div>
      ) : null}
      <Prop title={t(mc.headline)}>
        <Draft label={t(mc.headline)} value={cfg.headline} disabled={disabled} onCommit={(v) => set({ headline: v })} />
      </Prop>
      <Prop title={t(mc.line)}>
        <Draft label={t(mc.line)} value={cfg.line} disabled={disabled} onCommit={(v) => set({ line: v })} />
      </Prop>
      <Prop
        title={t(mc.tierPrice)}
        hint={tiers.length === 0 ? t(mc.noTiers) : tier && !tier.priceLine ? t(mc.tierPriceNone) : tier?.priceLine || cfg.price ? t(mc.tierPriceHint, { price: tier?.priceLine || cfg.price }) : null}
      >
        <select
          className="in"
          aria-label={t(mc.tierLabel)}
          value={cfg.tier}
          disabled={disabled || !tiers.length}
          onChange={(e) => {
            const next = tiers.find((x) => x.id === e.target.value)
            if (next) set({ tier: next.id, price: next.priceLine })
          }}
        >
          {!tier ? <option value={cfg.tier}>{t(mc.pickTier)}</option> : null}
          {tiers.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </select>
      </Prop>
      <Prop
        title={t(mc.button)}
        hint={<T msg={mc.buttonHint} values={{ token: <span className="tok mono">{'{{ upgrade_url }}'}</span> }} />}
      >
        <Draft label={t(mc.button)} value={cfg.button} disabled={disabled} onCommit={(v) => set({ button: v })} />
      </Prop>
      <Prop
        title={t(mc.offer)}
        hint={offer?.endsAt ? t(mc.offerEnds, { offer: offer.name, date: dayMonthDate(offer.endsAt) }) : tier && data.data && !offers.length ? t(mc.offerNoneLive) : null}
      >
        <select
          className="in"
          aria-label={t(mc.offer)}
          value={offer ? offer.code : ''}
          disabled={disabled || !tier || !offers.length}
          onChange={(e) => {
            if (!tier) return
            const o = offers.find((x) => x.code === e.target.value)
            set(o ? { offer: o.code, price: o.line, button: o.button } : { offer: '', price: tier.priceLine, button: defaultButton(tier) })
          }}
        >
          <option value="">{t(mc.offerNone)}</option>
          {offers.map((o) => (
            <option key={o.code} value={o.code}>
              {o.option}
            </option>
          ))}
        </select>
      </Prop>
    </>
  )
}
