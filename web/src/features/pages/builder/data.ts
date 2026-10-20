import { $api, type Schemas } from '@/api/client'
import { money } from '@/features/earn/format'
import { useCan, useWorkspace } from '@/lib/session'

/**
 * What the Block tab's pickers list: products, signup forms, tags and
 * tiers, each behind its own permission (a list the person can't read is
 * simply empty), and the theme's name and colour for the toolbar's chip.
 */

export type ProductItem = Schemas['ProductListItem']
export type ProductDetail = Schemas['ProductDetail']
export type FormItem = Schemas['SignupFormSummary']
export type TagItem = Schemas['WebsiteTag']
export type TierItem = Schemas['EarnTier']

const long = { staleTime: 60_000, retry: false } as const

export function useProducts(): { items: ProductItem[]; ready: boolean } {
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/products', { params: { path: { workspace: slug }, query: { period: '30d' } } }, { ...long, enabled: can })
  return { items: q.data?.products ?? [], ready: !can || q.isFetched }
}

export function useProduct(id: string | undefined): ProductDetail | undefined {
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/products/{product}', { params: { path: { workspace: slug, product: id ?? '' } } }, { ...long, enabled: can && Boolean(id) })
  return q.data
}

export function useForms(): { items: FormItem[]; ready: boolean } {
  const { slug } = useWorkspace()
  const can = useCan('subscribers:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms', { params: { path: { workspace: slug } } }, { ...long, enabled: can })
  return { items: q.data?.forms ?? [], ready: !can || q.isFetched }
}

export function useTags(): TagItem[] {
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/website/tags', { params: { path: { workspace: slug } } }, { ...long, enabled: can })
  return q.data ?? []
}

export function useTiers(): TierItem[] {
  const { slug } = useWorkspace()
  const can = useCan('earn:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/earn/plans', { params: { path: { workspace: slug } } }, { ...long, enabled: can })
  return (q.data?.tiers ?? []).filter((x) => x.visibility !== 'archived')
}

/** The theme's publisher and accent (the toolbar's "Theme · Acme" chip), from Pages' overview. */
export function useTheme(): { name: string; accent: string } | null {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages', { params: { path: { workspace: slug }, query: { period: '30d' } } }, long)
  const th = q.data?.theme
  return th ? { name: th.publisher, accent: th.accent } : null
}

/** A webinar page's reminder automation: the offer, or the flow made. */
export function useReminders(page: string, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}/reminder-automation', { params: { path: { workspace: slug, page } } }, { ...long, enabled })
}

/** "$29", "$15.50": a price in minor units (the design system's money). */
export function moneyText(cents: number, currency: string): string {
  return money(cents, currency, 'auto')
}

/** A product's price as its button shows it: "$29", or the pay-what-you-want minimum with a plus ("$5+"), as the server does. */
export function buttonPrice(p: Pick<ProductItem, 'price' | 'currency'>): string {
  return p.price.kind === 'pwyw' ? `${moneyText(p.price.minimumCents, p.currency)}+` : moneyText(p.price.amountCents, p.currency)
}
