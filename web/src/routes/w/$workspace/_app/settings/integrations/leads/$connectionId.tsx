import { createFileRoute } from '@tanstack/react-router'
import { LeadAdsPage } from '@/features/leads/LeadAdsPage'

/** Settings › Integrations › Meta Lead Ads · Acme Marketing: a Meta connection (M66). */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/leads/$connectionId')({
  component: function LeadAdsRoute() {
    const { connectionId } = Route.useParams()
    return <LeadAdsPage key={connectionId} id={connectionId} />
  },
})
