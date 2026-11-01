import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { CampaignsPage } from '@/features/campaigns/CampaignsPage'

const search = z.object({
  status: z.enum(['all', 'drafts', 'scheduled', 'sending', 'sent', 'recurring', 'paid', 'archived', 'deleted']).optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
  channel: z.enum(['email', 'sms', 'push', 'whatsapp']).optional().catch(undefined),
  // Review 2026-11 CA-2: when it went or goes out, its sender, a list of its audience, a tag.
  when: z.enum(['7d', '30d', 'quarter', 'custom']).optional().catch(undefined),
  from: z.string().regex(/^\d{4}-\d{2}-\d{2}$/).optional().catch(undefined),
  to: z.string().regex(/^\d{4}-\d{2}-\d{2}$/).optional().catch(undefined),
  sender: z.string().uuid().optional().catch(undefined),
  list: z.string().uuid().optional().catch(undefined),
  tag: z.string().max(40).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/campaigns/')({
  validateSearch: search,
  component: function CampaignsRoute() {
    return <CampaignsPage search={Route.useSearch()} />
  },
})
