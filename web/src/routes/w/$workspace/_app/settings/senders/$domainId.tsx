import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { DomainPage } from '@/features/senders/DomainPage'

const search = z.object({
  provider: z.uuid().optional().catch(undefined),
})

/** A sending domain's DNS records, checked live (Records and Verify in Settings › Senders). */
export const Route = createFileRoute('/w/$workspace/_app/settings/senders/$domainId')({
  validateSearch: search,
  component: DomainRoute,
})

function DomainRoute() {
  const { domainId } = Route.useParams()
  const { provider } = Route.useSearch()
  const navigate = Route.useNavigate()
  const onProvider = useCallback((p?: string) => void navigate({ search: { provider: p }, replace: true }), [navigate])
  return <DomainPage key={domainId} domainId={domainId} provider={provider} onProvider={onProvider} />
}
