import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useCan, useWorkspace } from '@/lib/session'
import { PRODUCTS } from './api'
import { mp } from './elsewhere.messages'

type T = ReturnType<typeof useT>

export interface Option {
  value: string
  label: string
}

/**
 * The "Bought a product" filter's options: no filter, any product, then
 * each product that can have buyers (drafts can't). A link or a saved view
 * may name a product the list doesn't have (yet): it stays selectable.
 */
export function boughtOptions(t: T, products: { id: string; name: string; status: string }[], value?: string): Option[] {
  const out: Option[] = [
    { value: '', label: t(mp.filterAll) },
    { value: 'any', label: t(mp.filterAny) },
  ]
  for (const p of products) if (p.status !== 'draft') out.push({ value: p.id, label: t(mp.filterProduct, { product: p.name }) })
  if (value && !out.some((o) => o.value === value)) out.push({ value, label: t(mp.filterUnknown) })
  return out
}

/**
 * Audience › Contacts' "Bought a product" filter (publish-sell brief §6):
 * buyers of any product, or of one (the `bought` search parameter,
 * params.ts). Hidden without products:view, and once it's clear the
 * workspace has nothing anyone could have bought.
 */
export function BoughtFilter({ value, onChange, className }: { value?: string; onChange: (v: string | undefined) => void; className?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', PRODUCTS, { params: { path: { workspace: slug } } }, { enabled: can, staleTime: 60_000 })
  if (!can) return null
  const options = boughtOptions(t, q.data?.products ?? [], value)
  if (!value && q.data && options.length <= 2) return null
  return (
    <select className={className} aria-label={t(mp.filterLabel)} value={value ?? ''} onChange={(e) => onChange(e.target.value || undefined)}>
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}
