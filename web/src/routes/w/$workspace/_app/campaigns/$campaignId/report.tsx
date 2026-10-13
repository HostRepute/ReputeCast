import { createFileRoute } from '@tanstack/react-router'
import { ReportPage } from '@/features/reports/ReportPage'

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/report')({
  component: function ReportRoute() {
    const { campaignId } = Route.useParams()
    return <ReportPage key={campaignId} campaignId={campaignId} />
  },
})
