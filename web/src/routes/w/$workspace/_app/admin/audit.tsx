import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { InstanceAuditPage } from '@/features/admin/audit/InstanceAuditPage'

const day = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .optional()
  .catch(undefined)

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  /** A workspace slug, or "instance" for instance-level entries only. */
  workspace: z.string().max(64).optional().catch(undefined),
  actorType: z.enum(['user', 'api_key', 'mcp_agent', 'system']).optional().catch(undefined),
  result: z.enum(['allowed', 'denied']).optional().catch(undefined),
  range: z.enum(['7d', '90d', 'custom']).optional().catch(undefined),
  /** Custom range, as dates in the admin's timezone. */
  since: day,
  until: day,
  /** The entry shown in the details dialog. */
  event: z.string().max(64).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/admin/audit')({
  validateSearch: search,
  component: function InstanceAuditRoute() {
    return <InstanceAuditPage search={Route.useSearch()} />
  },
})
