import { createFileRoute } from '@tanstack/react-router'
import { PushReportPage } from '@/features/push/ReportPage'

/** A push campaign's report (board PushStates). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/push-report')({
  component: function PushReportRoute() {
    const { campaignId } = Route.useParams()
    return <PushReportPage key={campaignId} campaignId={campaignId} />
  },
})
