import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ContactsPage } from '@/features/audience/ContactsPage'

const search = z.object({
  q: z.string().max(200).optional().catch(undefined),
  list: z.string().uuid().optional().catch(undefined),
  tier: z.enum(['active', 'cooling', 'at_risk', 'sunset']).optional().catch(undefined),
  mailbox: z.enum(['gmail', 'outlook', 'yahoo', 'icloud', 'other']).optional().catch(undefined),
  consent: z.enum(['double', 'single', 'pending']).optional().catch(undefined),
  source: z.string().max(80).optional().catch(undefined),
  tag: z.string().max(80).optional().catch(undefined),
  joinedAfter: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/)
    .optional()
    .catch(undefined),
  segment: z.string().uuid().optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
  size: z.coerce.number().int().min(1).max(100).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/')({
  validateSearch: search,
  component: function ContactsRoute() {
    return <ContactsPage search={Route.useSearch()} />
  },
})
