import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { LiveMonitorPage } from '@/features/live/LiveMonitorPage'

const search = z.object({
  // While an auto-pause or an outage waits for a decision, the decision
  // view shows by default; "live" keeps the live view instead.
  view: z.enum(['live']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/live')({
  validateSearch: search,
  component: function LiveRoute() {
    const { campaignId } = Route.useParams()
    return <LiveMonitorPage key={campaignId} campaignId={campaignId} search={Route.useSearch()} />
  },
})
