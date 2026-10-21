import type { Schemas } from '@/api/client'
import { T, useT } from '@/i18n'
import { ma } from './elsewhere.messages'

/*
 * The automation trigger and goal "Bought a product" (publish-sell brief
 * §3.5 and §6): which product, any when none is picked. The step panel
 * (features/automations/panels.tsx) shows these; the names on the canvas
 * come from elsewhere.ts.
 */

type Trigger = Schemas['AutomationTrigger']
type Condition = Schemas['AutomationCondition']
type ProductOption = Schemas['AutomationOptionItem']

/** "Any product", then the products (published first, then others that sold, as the options list them). */
function ProductSelect({ value, products, disabled, onChange }: { value?: string; products?: ProductOption[]; disabled?: boolean; onChange: (id: string | undefined) => void }) {
  const t = useT()
  const known = !value || (products ?? []).some((p) => p.id === value)
  return (
    <select className="in" aria-label={t(ma.product)} disabled={disabled} value={value ?? ''} onChange={(e) => onChange(e.target.value || undefined)}>
      <option value="">{t(ma.any)}</option>
      {(products ?? []).map((p) => (
        <option key={p.id} value={p.id}>
          {p.name}
        </option>
      ))}
      {known ? null : <option value={value}>{products ? t(ma.deleted) : t(ma.loading)}</option>}
    </select>
  )
}

// A Liquid tag, kept on one line.
const liquid = (tag: string) => <span className="mono whitespace-nowrap">{`{{ ${tag} }}`}</span>

/** The trigger's settings: the product, who enters, and the order data the flow's emails can use. */
export function BoughtTriggerProps({ t: trigger, disabled, set, products }: { t: Trigger; disabled: boolean; set: (patch: Partial<Trigger>) => void; products?: ProductOption[] }) {
  const t = useT()
  return (
    <>
      <ProductSelect value={trigger.productId} products={products} disabled={disabled} onChange={(productId) => set({ productId })} />
      <span className="k">{t(ma.triggerNote)}</span>
      <span className="k">
        <T msg={ma.liquid} values={{ name: liquid('order.product_name'), amount: liquid('order.amount'), number: liquid('order.number') }} />
      </span>
    </>
  )
}

/** The condition's settings, in branches, filters and goals: the product. */
export function BoughtConditionProps({ c, disabled, set, products }: { c: Condition; disabled?: boolean; set: (patch: Partial<Condition>) => void; products?: ProductOption[] }) {
  const t = useT()
  return (
    <>
      <ProductSelect value={c.productId} products={products} disabled={disabled} onChange={(productId) => set({ productId })} />
      <span className="k">{t(ma.conditionNote)}</span>
    </>
  )
}
