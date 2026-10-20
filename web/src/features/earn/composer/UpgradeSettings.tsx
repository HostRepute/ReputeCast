import type { Component } from 'grapesjs'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { ButtonLink } from '@/components/ui/Button'
import { T, useT } from '@/i18n'
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
 * tier whose price and trial it reads, and the button. The price line
 * follows the tier: it is written again when the tier's prices changed.
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
  // A new block takes the first tier; a tier whose prices changed rewrites its line.
  const first = tiers[0]
  const fill = cfg && !disabled && data.data ? (!cfg.tier && first ? first : tier && tier.priceLine !== cfg.price ? tier : null) : null
  useEffect(() => {
    if (!cfg || !fill) return
    const fresh = !cfg.tier
    applyUpgrade(comp, {
      ...cfg,
      tier: fill.id,
      price: fill.priceLine,
      headline: fresh ? t(mc.defaultHeadline, { tier: fill.short }) : cfg.headline,
      button: fresh ? (fill.trialDays > 0 ? t(mc.defaultButton) : t(mc.defaultButtonNoTrial, { tier: fill.short })) : cfg.button,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per tier and price
  }, [fill?.id, fill?.priceLine])
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
        hint={tiers.length === 0 ? t(mc.noTiers) : tier && !tier.priceLine ? t(mc.tierPriceNone) : cfg.price ? t(mc.tierPriceHint, { price: cfg.price }) : null}
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
    </>
  )
}
