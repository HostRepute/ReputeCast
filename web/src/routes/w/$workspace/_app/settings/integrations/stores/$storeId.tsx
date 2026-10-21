import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { StorePage } from '@/features/commerce/StorePage'

/**
 * ?connect opens the connect dialog where the store stopped; Shopify sends
 * the browser back here after its approval, with ?error when it failed.
 */
const search = z.object({
  connect: z.coerce.boolean().optional().catch(undefined),
  error: z.string().max(40).optional().catch(undefined),
})

/** Settings › Integrations › Shopify · Acme Merch: a connected store. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/stores/$storeId')({
  validateSearch: search,
  component: function StoreRoute() {
    const { storeId } = Route.useParams()
    const { connect, error } = Route.useSearch()
    return <StorePage key={storeId} id={storeId} connect={connect} error={error} />
  },
})
