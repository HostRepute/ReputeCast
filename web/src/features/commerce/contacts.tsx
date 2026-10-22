import { Field, Input, Select } from '@/components/ui/Field'
import { useT } from '@/i18n'
import '@/features/scoring/scoring.css'
import { money } from './api'
import { useCatalog } from './composer'
import { cf } from './elsewhere.messages'
import './commerce.css'

/*
 * Contacts › More filters (design expansion phase 7): "Store customer" and
 * "Total spent", with their chips. Total spent is orders less refunds, in
 * the stores' currency.
 */

export const storeMoreKeys = ['storeCustomer', 'spentFrom', 'spentTo'] as const

export interface StoreSearch {
  storeCustomer?: string
  spentFrom?: string
  spentTo?: string
}

export type StoreDraft = StoreSearch

const amount = (v: string) => {
  const s = v.trim().replace(/^\$/, '')
  return s === '' || Number.isNaN(Number(s)) ? undefined : s
}

/** The filters in More filters; nothing shows without a store. */
export function StoreContactFields({ draft, onChange, open }: { draft: StoreDraft; onChange: (d: StoreDraft) => void; open: boolean }) {
  const t = useT()
  const catalog = useCatalog()
  const stores = catalog.data?.stores ?? []
  if (!open || stores.length === 0) return null
  return (
    <>
      <Field label={t(cf.storeCustomer)}>
        <Select value={draft.storeCustomer ?? ''} onChange={(e) => onChange({ ...draft, storeCustomer: e.target.value || undefined })}>
          <option value="">{t(cf.anyone)}</option>
          <option value="any">{t(cf.anyStore)}</option>
          {stores.map((s) => (
            <option key={s.id} value={s.id}>
              {t(cf.ofStore, { store: s.name })}
            </option>
          ))}
        </Select>
      </Field>
      <div className="flex flex-col gap-1">
        <span className="lbl">{t(cf.spent)}</span>
        <div className="flex items-center gap-2">
          <Input className="num" inputMode="decimal" style={{ width: 96 }} aria-label={t(cf.from)} placeholder="0" value={draft.spentFrom ?? ''} onChange={(e) => onChange({ ...draft, spentFrom: amount(e.target.value) ?? (e.target.value === '' ? undefined : draft.spentFrom) })} />
          <span className="k">{t(cf.between)}</span>
          <Input className="num" inputMode="decimal" style={{ width: 96 }} aria-label={t(cf.to)} placeholder="500" value={draft.spentTo ?? ''} onChange={(e) => onChange({ ...draft, spentTo: amount(e.target.value) ?? (e.target.value === '' ? undefined : draft.spentTo) })} />
        </div>
        <span className="k">{t(cf.spentHint)}</span>
      </div>
    </>
  )
}

/** The chips of the store filters, each with its × to remove it. */
export function StoreChips({ search, onChange }: { search: StoreSearch; onChange: (patch: Partial<StoreSearch>) => void }) {
  const t = useT()
  const catalog = useCatalog()
  const chips: { key: string; label: string; clear: Partial<StoreSearch> }[] = []
  if (search.storeCustomer) {
    const s = catalog.data?.stores.find((x) => x.id === search.storeCustomer)
    chips.push({ key: 'sc', label: search.storeCustomer === 'any' || !s ? t(cf.chipAny) : t(cf.chipCustomer, { store: s.name }), clear: { storeCustomer: undefined } })
  }
  if (search.spentFrom || search.spentTo) {
    const cur = catalog.data?.stores[0]?.currency || 'USD'
    const show = (v: string) => money(Math.round(Number(v) * 100), cur, 'auto')
    const label =
      search.spentFrom && search.spentTo
        ? t(cf.chipSpent, { from: show(search.spentFrom), to: show(search.spentTo) })
        : search.spentFrom
          ? t(cf.chipSpentFrom, { from: show(search.spentFrom) })
          : t(cf.chipSpentTo, { to: show(search.spentTo ?? '0') })
    chips.push({ key: 'sp', label, clear: { spentFrom: undefined, spentTo: undefined } })
  }
  if (!chips.length) return null
  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      {chips.map((c) => (
        <span key={c.key} className="sc-chip">
          {c.label}
          <button type="button" aria-label={t(cf.remove, { name: c.label })} onClick={() => onChange(c.clear)}>
            ×
          </button>
        </span>
      ))}
    </div>
  )
}
