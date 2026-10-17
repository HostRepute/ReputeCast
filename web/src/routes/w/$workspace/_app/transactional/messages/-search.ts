import { z } from 'zod'

/** The message log's filters, shared by the log and the message detail's list. */
export const logSearch = z.object({
  q: z.string().max(254).optional().catch(undefined),
  template: z.string().max(64).optional().catch(undefined),
  provider: z.string().max(36).optional().catch(undefined),
  key: z.string().max(36).optional().catch(undefined),
  range: z.enum(['1h', '24h', '7d', '30d', 'custom']).optional().catch(undefined),
  from: z.string().max(10).optional().catch(undefined),
  to: z.string().max(10).optional().catch(undefined),
  status: z.enum(['all', 'queued', 'sent', 'delivered', 'deferred', 'bounced', 'complained', 'rejected', 'failed', 'test']).optional().catch(undefined),
})
