import { useParams } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { T, defineMessages, useT } from '@/i18n'
import { count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { commerceBase, money } from './api'
import './commerce.css'

type Trigger = Schemas['AutomationTrigger']
type Condition = Schemas['AutomationCondition']
type Graph = { nodes: { type: string; data: { condition?: Condition } }[] }
export type TriggerStore = Schemas['CommerceTriggerStore']

/**
 * Store triggers, the goal "Placed an order" and attributed revenue in
 * automations (design expansion phase 7; CommerceEmails' abandoned-cart
 * starter and its trigger panel).
 */
export const ct = defineMessages('commerce.automations', {
  store: 'Store',
  storeOption: '{name} · {address}',
  pickStore: 'Pick a store',
  loading: 'Loading the stores…',
  noStore: 'No store is connected. Connect Shopify or WooCommerce in Settings › Integrations.',
  noCheckouts: "{store} doesn't report abandoned checkouts: WooCommerce needs the ReputeCast plugin for them.",
  after: 'After',
  hours: 'Hours',
  afterHint: 'hours without an order · 1 to 24',
  whoEnters: 'Who enters',
  locked: 'Accepted marketing in {store}',
  lockedAny: 'Accepted marketing in the store',
  lockedNote: "Required, and it can't be removed: customers who didn't accept marketing never enter.",
  addFilter: 'Add a filter',
  removeFilters: 'Remove the filters',
  product: 'Product',
  anyProduct: 'Any product',
  collection: 'Collection',
  anyCollection: 'Any collection',
  minTotal: 'Total at least',
  firstOnly: "The contact's first order in the store only",
  goalTitle: 'Placing the order exits the flow',
  goalOn: 'The goal “Placed an order”',
  goalOff: 'Add a Goal step with “Placed an order” so an order ends the emails.',
  stats: 'Since {date}: {abandoned} checkouts abandoned, {withEmail} with an email, {entered} entered. Skip rule: the contact got an abandoned-cart email in the last {days, plural, one {# day} other {# days}}.',
  statsNoSkip: 'Since {date}: {abandoned} checkouts abandoned, {withEmail} with an email, {entered} entered.',
  notEmailed: "{count} not emailed: didn't accept marketing",
  goalOrders: '{count, plural, one {<n>#</n> order} other {<n>#</n> orders}}',
  orderedIn: 'Ordered in',
  anyStore: 'Any connected store',
  revenueColumn: 'Revenue · 30 d',
  revenueCell: '{amount} · {orders, plural, one {# order} other {# orders}}',
  triggerOrderPlaced: 'Order placed',
  triggerFirstOrder: 'First order',
  triggerAbandoned: 'Abandoned checkout',
  goalPlacedOrder: 'Placed an order',
})

export const storeTriggerKinds = ['order_placed', 'first_order', 'abandoned_checkout'] as const

export function isStoreTrigger(kind: string | undefined): boolean {
  return kind === 'order_placed' || kind === 'first_order' || kind === 'abandoned_checkout'
}

/** The store a flow's trigger watches, when a store trigger starts it. */
export function storeOfTrigger(g: { nodes: { type: string; data: { trigger?: Trigger } }[] }): string | undefined {
  const tr = g.nodes.find((n) => n.type === 'trigger')?.data.trigger
  return tr && isStoreTrigger(tr.kind) ? tr.store || undefined : undefined
}

/** The stores, their products and collections, for triggers and goals. */
export function useStoreTriggerOptions() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${commerceBase}/trigger-options`, { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
}

/** A store trigger's settings when it's picked: Abandoned checkout waits an hour and skips 7 days. */
export function storeTriggerDefaults(kind: string, stores: TriggerStore[] | undefined): Partial<Trigger> {
  if (!isStoreTrigger(kind)) return {}
  const connected = (stores ?? []).filter((s) => s.status !== 'disconnected')
  const store = connected.length === 1 ? connected[0]!.id : undefined
  return kind === 'abandoned_checkout' ? { store, afterHours: 1, skipDays: 7, reentry: 'always' } : { store }
}

function StoreSelect({ value, stores, loading, disabled, onPick, label, allowAny }: {
  value?: string
  stores: TriggerStore[]
  loading: boolean
  disabled: boolean
  onPick: (id: string | undefined) => void
  label: string
  allowAny?: boolean
}) {
  const t = useT()
  return (
    <select className="in" aria-label={label} disabled={disabled} value={value ?? ''} onChange={(e) => onPick(e.target.value || undefined)}>
      {value ? null : <option value="">{loading ? t(ct.loading) : allowAny ? t(ct.anyStore) : t(ct.pickStore)}</option>}
      {allowAny && value ? <option value="">{t(ct.anyStore)}</option> : null}
      {stores.map((s) => (
        <option key={s.id} value={s.id}>
          {t(ct.storeOption, { name: s.name, address: s.address })}
        </option>
      ))}
    </select>
  )
}

/**
 * The fields of Order placed, First order and Abandoned checkout: the store;
 * the delay, the locked "Accepted marketing" filter and the goal for
 * Abandoned checkout; product, collection, total and first-order filters.
 */
export function StoreTriggerFields({ t: trig, set, disabled, graph }: { t: Trigger; set: (patch: Partial<Trigger>) => void; disabled: boolean; graph: Graph }) {
  const t = useT()
  const opts = useStoreTriggerOptions()
  const stores = opts.data?.stores ?? []
  const store = stores.find((s) => s.id === trig.store)
  const abandoned = trig.kind === 'abandoned_checkout'
  const [adding, setAdding] = useState(false)
  const filtered = !!(trig.products?.length || trig.collections?.length || trig.minTotal || trig.firstOnly)
  const showFilters = filtered || adding || trig.kind === 'order_placed'
  const hasGoal = graph.nodes.some((n) => n.type === 'goal' && n.data.condition?.kind === 'placed_order')
  if (opts.isSuccess && stores.length === 0) {
    return <span className="k">{t(ct.noStore)}</span>
  }
  return (
    <>
      <label className="lbl">
        {t(ct.store)}
        <StoreSelect value={trig.store} stores={stores} loading={opts.isPending} disabled={disabled} label={t(ct.store)} onPick={(store) => set({ store, products: undefined, collections: undefined })} />
      </label>
      {abandoned && store && !store.checkouts ? <span className="k">{t(ct.noCheckouts, { store: store.name })}</span> : null}
      {abandoned ? (
        <>
          <div className="lbl">
            <span id="cm-after">{t(ct.after)}</span>
            <div className="flex flex-nowrap items-center gap-2">
              <input
                className="in w-16"
                type="number"
                min={1}
                max={24}
                aria-labelledby="cm-after"
                aria-describedby="cm-after-hint"
                disabled={disabled}
                value={trig.afterHours || 1}
                onChange={(e) => set({ afterHours: Math.min(24, Math.max(1, Number(e.target.value) || 1)) })}
              />
              <span className="k" id="cm-after-hint">
                {t(ct.afterHint)}
              </span>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="lbl">{t(ct.whoEnters)}</span>
            <div className="cm-lockf">
              <Icon name="lock" size={14} className="mt-0.5 flex-none text-ink-3" />
              <span className="flex flex-col gap-0.5">
                <b className="font-semibold">{store ? t(ct.locked, { store: store.name }) : t(ct.lockedAny)}</b>
                <span className="k">{t(ct.lockedNote)}</span>
              </span>
            </div>
          </div>
        </>
      ) : null}
      {trig.kind !== 'first_order' ? (
        showFilters ? (
          <StoreFilters
            trig={trig}
            store={store}
            set={set}
            disabled={disabled}
            onRemove={
              abandoned
                ? () => {
                    setAdding(false)
                    set({ products: undefined, collections: undefined, minTotal: undefined, firstOnly: undefined })
                  }
                : undefined
            }
          />
        ) : (
          <Button size="sm" icon="plus" className="self-start" disabled={disabled || !store} onClick={() => setAdding(true)}>
            {t(ct.addFilter)}
          </Button>
        )
      ) : null}
      {abandoned ? (
        <div className="cm-srow items-center">
          <span>
            <b className="text-[13px] font-semibold">{t(ct.goalTitle)}</b>
            <span className="k" id="cm-goal-sub">
              {hasGoal ? t(ct.goalOn) : t(ct.goalOff)}
            </span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(ct.goalTitle)} aria-describedby="cm-goal-sub" checked={hasGoal} disabled readOnly />
        </div>
      ) : null}
      {abandoned ? <TriggerStats skipDays={trig.skipDays ?? 0} /> : null}
    </>
  )
}

/** Order placed's (and Abandoned checkout's added) filters. */
function StoreFilters({ trig, store, set, disabled, onRemove }: { trig: Trigger; store?: TriggerStore; set: (patch: Partial<Trigger>) => void; disabled: boolean; onRemove?: () => void }) {
  const t = useT()
  const cur = store?.currency || 'USD'
  return (
    <div className="flex flex-col gap-2">
      <select className="in" aria-label={t(ct.product)} disabled={disabled || !store} value={trig.products?.[0] ?? ''} onChange={(e) => set({ products: e.target.value ? [e.target.value] : undefined })}>
        <option value="">{t(ct.anyProduct)}</option>
        {(store?.products ?? []).map((p) => (
          <option key={p.id} value={p.id}>
            {p.title}
          </option>
        ))}
      </select>
      <select className="in" aria-label={t(ct.collection)} disabled={disabled || !store} value={trig.collections?.[0] ?? ''} onChange={(e) => set({ collections: e.target.value ? [e.target.value] : undefined })}>
        <option value="">{t(ct.anyCollection)}</option>
        {(store?.collections ?? []).map((c) => (
          <option key={c.id} value={c.id}>
            {c.title}
          </option>
        ))}
      </select>
      <label className="flex items-center gap-2 text-[13px]">
        <span className="whitespace-nowrap">{t(ct.minTotal)}</span>
        <input
          className="in w-24"
          type="number"
          min={0}
          step="0.01"
          disabled={disabled}
          value={trig.minTotal ? trig.minTotal / 100 : ''}
          placeholder={money(0, cur, 'always')}
          onChange={(e) => set({ minTotal: Math.round((Number(e.target.value) || 0) * 100) || undefined })}
        />
      </label>
      {trig.kind === 'order_placed' ? (
        <label className="fe-chk">
          <input type="checkbox" checked={!!trig.firstOnly} disabled={disabled} onChange={(e) => set({ firstOnly: e.target.checked || undefined })} />
          {t(ct.firstOnly)}
        </label>
      ) : null}
      {onRemove ? (
        <Button size="sm" className="self-start" disabled={disabled} onClick={onRemove}>
          {t(ct.removeFilters)}
        </Button>
      ) : null}
    </div>
  )
}

const bold = (c: ReactNode[]) => <b className="n">{c}</b>

function useAutomationId(): string | undefined {
  const p = useParams({ strict: false }) as { automationId?: string }
  return p.automationId
}

/** The flow's store numbers: abandoned checkouts, entries, orders and revenue. */
export function useStoreStats() {
  const { slug } = useWorkspace()
  const id = useAutomationId()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/store-stats', { params: { path: { workspace: slug, id: id ?? '' } } }, { enabled: !!id, staleTime: 60_000 })
}

function TriggerStats({ skipDays }: { skipDays: number }) {
  const t = useT()
  const q = useStoreStats()
  const s = q.data
  if (!s?.since) return null
  const values = { date: dayMonthDate(s.since), abandoned: count(s.abandoned), withEmail: count(s.withEmail), entered: count(s.entered), days: skipDays }
  return <span className="k text-ink-nav">{skipDays > 0 ? t(ct.stats, values) : t(ct.statsNoSkip, values)}</span>
}

/** The canvas line under a store trigger or the "Placed an order" goal. */
export function StoreNodeLine({ kind }: { kind: 'trigger' | 'goal' }) {
  const t = useT()
  const q = useStoreStats()
  const s = q.data
  if (!s?.since) return null
  if (kind === 'trigger') return s.noConsent > 0 ? <span>{t(ct.notEmailed, { count: count(s.noConsent) })}</span> : null
  return (
    <>
      <span>
        <T msg={ct.goalOrders} values={{ count: s.orders, n: bold }} />
      </span>
      <span>
        <b className="n">{money(s.revenue, s.currency)}</b>
      </span>
    </>
  )
}

/** "Placed an order"'s settings in a goal, branch or filter: the store. */
export function PlacedOrderFields({ c, set, disabled }: { c: Condition; set: (patch: Partial<Condition>) => void; disabled?: boolean }) {
  const t = useT()
  const opts = useStoreTriggerOptions()
  return (
    <StoreSelect value={c.store} stores={opts.data?.stores ?? []} loading={opts.isPending} disabled={!!disabled} label={t(ct.orderedIn)} allowAny onPick={(store) => set({ store })} />
  )
}

/** The automations list's revenue column: "$472 · 8 orders". */
export function RevenueCell({ revenue }: { revenue?: Schemas['AutomationRevenue'] }) {
  const t = useT()
  if (!revenue || revenue.orders === 0) return null
  return <>{t(ct.revenueCell, { amount: money(revenue.revenue, revenue.currency), orders: revenue.orders })}</>
}
