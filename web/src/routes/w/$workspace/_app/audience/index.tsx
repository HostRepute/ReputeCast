import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ContactsPage } from '@/features/audience/ContactsPage'
// The jobs tray lists imports from the server once this module loads.
import '@/features/imports/jobsSource'

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
  language: z
    .string()
    .regex(/^([a-z]{2,3}|none|not_offered)$/)
    .optional()
    .catch(undefined),
  // Scoring and Growth (X13).
  band: z.enum(['cold', 'warm', 'hot', 'none']).optional().catch(undefined),
  scoreFrom: z.coerce.number().int().min(0).max(100).optional().catch(undefined),
  scoreTo: z.coerce.number().int().min(0).max(100).optional().catch(undefined),
  growthSource: z.string().max(200).optional().catch(undefined),
  utmSource: z.string().max(200).optional().catch(undefined),
  utmMedium: z.string().max(200).optional().catch(undefined),
  utmCampaign: z.string().max(200).optional().catch(undefined),
  referringDomain: z.string().max(200).optional().catch(undefined),
  sort: z.enum(['activity', 'score']).optional().catch(undefined),
  /** The latest address check's result (X14). */
  check: z.enum(['valid', 'invalid', 'risky', 'unknown', 'never']).optional().catch(undefined),
  /** Paid status (Earn, X16). */
  paid: z.enum(['with_access', 'trialing', 'past_due', 'complimentary', 'gift', 'ended', 'free']).optional().catch(undefined),
  paidTier: z.string().uuid().optional().catch(undefined),
  /** Store customers and what they spent in the stores (X20). */
  storeCustomer: z
    .string()
    .regex(/^(any|[0-9a-f-]{36})$/)
    .optional()
    .catch(undefined),
  spentFrom: z
    .string()
    .regex(/^\d{1,9}(\.\d{1,2})?$/)
    .optional()
    .catch(undefined),
  spentTo: z
    .string()
    .regex(/^\d{1,9}(\.\d{1,2})?$/)
    .optional()
    .catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
  size: z.coerce.number().int().min(1).max(100).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/audience/')({
  validateSearch: search,
  component: function ContactsRoute() {
    return <ContactsPage search={Route.useSearch()} />
  },
})
