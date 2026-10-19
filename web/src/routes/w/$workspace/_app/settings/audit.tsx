import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { AuditPage } from '@/features/audit/AuditPage'

const day = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .optional()
  .catch(undefined)

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  actorType: z.enum(['user', 'api_key', 'mcp_agent', 'system']).optional().catch(undefined),
  actor: z.string().max(64).optional().catch(undefined),
  action: z.string().max(64).optional().catch(undefined),
  category: z.enum(['sending', 'routing', 'audience', 'team', 'settings', 'integrations', 'transactional', 'scoring', 'digests', 'polls', 'ai', 'earn', 'verification', 'placement', 'website']).optional().catch(undefined),
  result: z.enum(['allowed', 'denied', 'error']).optional().catch(undefined),
  range: z.enum(['24h', '7d', '30d', '90d', 'all', 'custom']).optional().catch(undefined),
  /** Custom range, as dates in the workspace timezone. */
  since: day,
  until: day,
  /** The entry shown in the detail panel. */
  event: z.string().max(64).optional().catch(undefined),
})

export type AuditSearch = z.infer<typeof search>

export const Route = createFileRoute('/w/$workspace/_app/settings/audit')({
  validateSearch: search,
  component: function AuditRoute() {
    return <AuditPage search={Route.useSearch()} />
  },
})
