import type { Component } from 'grapesjs'
import { Link } from '@tanstack/react-router'
import { useState, useSyncExternalStore, type ReactNode } from 'react'
import { SwitchInput } from '@/components/ui/Field'
import '@/features/accessibility/composer/accessibility.css'
import { useComposer } from '@/features/composer/context'
import { T, useT } from '@/i18n'
import { count } from '@/lib/format'
import { discountText } from '../PoolsPage'
import { couponOf, couponPools, setCoupon, subscribeCouponPools, type CouponBlock } from './block'
import { mc } from './messages'

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

function Prop({ title, children, id }: { title: ReactNode; children: ReactNode; id: string }) {
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

/**
 * The Discount code block's inspector (ComposerTimedBlocks): the pool (with
 * its free codes), the label, the button and a way to the pools.
 */
export function CouponPanel({ c }: { c: Component }) {
  const t = useT()
  const composer = useComposer()
  const pools = useSyncExternalStore(subscribeCouponPools, couponPools)
  const [b, setB] = useState<CouponBlock>(() => couponOf(c))
  const save = (next: CouponBlock) => {
    setB(next)
    setCoupon(c, next)
  }
  const pool = pools?.find((p) => p.slug === b.pool)
  const active = (pools ?? []).filter((p) => p.status === 'active' || p.slug === b.pool)
  const token = pool ? `{{ coupons.${pool.slug}.url }}` : ''
  const facts = pool
    ? [
        pool.source === 'upload' ? t(mc.poolUpload) : t(mc.poolStore, { platform: pool.source === 'woocommerce' ? 'WooCommerce' : 'Shopify', store: pool.store?.name ?? '' }),
        pool.source === 'upload' ? '' : discountText(t, pool),
        pool.onePerContact ? t(mc.poolOnePer) : '',
      ].filter(Boolean)
    : []
  const hasUrl = pool && (pool.source !== 'upload' || !!pool.settings.applyUrl)
  return (
    <div className="flex flex-col">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-3 py-3">
        <span>
          <b className="font-semibold">{t(mc.name)}</b> {pool ? <span className="k">{pool.name}</span> : null}
        </span>
        <span className="k">{t(mc.oneCode)}</span>
      </div>
      <Prop title={t(mc.pool)} id="cp-pool">
        {pools && pools.length === 0 ? (
          <span className="k">
            <T
              msg={mc.noPools}
              values={{
                a: (x) => (
                  <Link to="/w/$workspace/campaigns/codes" params={{ workspace: composer.slug }} search={{ new: 1 }} target="_blank">
                    {x}
                  </Link>
                ),
              }}
            />
          </span>
        ) : (
          <select className="in" aria-labelledby="cp-pool" value={b.pool} onChange={(e) => save({ ...b, pool: e.target.value })}>
            {!pool ? <option value={b.pool}>{t(mc.poolNone)}</option> : null}
            {active.map((p) => (
              <option key={p.id} value={p.slug}>
                {t(mc.poolOption, { name: p.name, free: count(p.free) })}
              </option>
            ))}
          </select>
        )}
        {pool ? (
          <span className="k">
            {facts.join(' · ')} · <T msg={mc.poolLiquid} values={{ token: `coupons.${pool.slug}`, mono }} />
          </span>
        ) : null}
        {pool?.status === 'archived' ? (
          <span className="text-[12.5px] text-warn-fg" role="status">
            {t(mc.archivedNote, { name: pool.name })}
          </span>
        ) : null}
      </Prop>
      <Prop title={t(mc.label)} id="cp-label">
        <input className="in" aria-labelledby="cp-label" maxLength={80} value={b.label} onChange={(e) => setB({ ...b, label: e.target.value })} onBlur={() => save(b)} />
      </Prop>
      <Prop
        title={
          <span className="flex w-full items-center justify-between gap-2">
            <span id="cp-button-t">{t(mc.button)}</span>
            <SwitchInput aria-label={t(mc.buttonOn)} checked={b.buttonUrl} onChange={(e) => save({ ...b, buttonUrl: e.target.checked })} />
          </span>
        }
        id="cp-button"
      >
        <input
          className="in"
          aria-labelledby="cp-button-t"
          maxLength={40}
          disabled={!b.buttonUrl}
          value={b.button}
          onChange={(e) => setB({ ...b, button: e.target.value })}
          onBlur={() => save(b)}
        />
        {pool ? (
          <span className="k">
            {hasUrl ? (
              <T msg={pool.source === 'upload' ? mc.buttonApply : mc.buttonOpens} values={{ store: pool.store?.name ?? '', token, mono }} />
            ) : (
              t(mc.buttonNoUrl, { name: pool.name })
            )}
          </span>
        ) : null}
      </Prop>
      <div className="prop">
        <Link to="/w/$workspace/campaigns/codes" params={{ workspace: composer.slug }} target="_blank" className="font-semibold no-underline">
          {t(mc.manage)}
        </Link>
        <span className="k">{t(mc.manageNote)}</span>
      </div>
      <div className="flex items-start gap-2 px-3 py-3">
        <span className="a11y-cchip p">{t(mc.pass)}</span>
        <span className="k">{t(mc.passNote)}</span>
      </div>
    </div>
  )
}
