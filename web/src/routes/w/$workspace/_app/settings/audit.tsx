import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AuditPage } from '@/features/audit/AuditPage'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  actorType: z.enum(['user', 'api_key', 'mcp_agent', 'system']).optional().catch(undefined),
  actor: z.string().max(64).optional().catch(undefined),
  action: z.string().max(64).optional().catch(undefined),
  result: z.enum(['allowed', 'denied', 'error']).optional().catch(undefined),
  range: z.enum(['24h', '7d', '30d', '90d', 'all']).optional().catch(undefined),
  event: z.string().max(64).optional().catch(undefined),
})

export type AuditSearch = z.infer<typeof search>

export const Route = createFileRoute('/w/$workspace/_app/settings/audit')({
  validateSearch: search,
  component: function AuditRoute() {
    return <AuditPage search={Route.useSearch()} />
  },
})
