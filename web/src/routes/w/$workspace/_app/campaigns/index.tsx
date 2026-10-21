import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { CampaignsPage } from '@/features/campaigns/CampaignsPage'

const search = z.object({
  status: z.enum(['all', 'drafts', 'scheduled', 'sending', 'sent', 'recurring', 'paid']).optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
  channel: z.enum(['email', 'sms', 'push', 'whatsapp']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/campaigns/')({
  validateSearch: search,
  component: function CampaignsRoute() {
    return <CampaignsPage search={Route.useSearch()} />
  },
})
