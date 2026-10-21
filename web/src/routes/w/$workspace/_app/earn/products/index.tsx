import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { ProductsPage, type ProductsSearch } from '@/features/products/ProductsPage'

const search = z.object({
  /** The numbers' period: 30 days when absent. */
  period: z.enum(['90d', '12m']).optional().catch(undefined),
  /** The orders' search, product, status and date filters (30 days when absent), and page. */
  q: z.string().max(200).optional().catch(undefined),
  product: z.uuid().optional().catch(undefined),
  status: z.enum(['paid', 'refunded', 'partially_refunded', 'disputed', 'test']).optional().catch(undefined),
  date: z.enum(['7d', '90d', 'all']).optional().catch(undefined),
  page: z.number().int().min(1).max(100000).optional().catch(undefined),
  /** The order in the panel (the "product delivery failed" and "dispute opened" notifications link here). */
  order: z.uuid().optional().catch(undefined),
})

/** Earn › Products (Products): the numbers, the products, the orders and the order panel. */
export const Route = createFileRoute('/w/$workspace/_app/earn/products/')({
  validateSearch: search,
  component: function ProductsRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: ProductsSearch, replace?: boolean) => void navigate({ search: next as z.infer<typeof search>, replace }), [navigate])
    return <ProductsPage search={s} onSearch={onSearch} />
  },
})
