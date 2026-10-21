import { createFileRoute } from '@tanstack/react-router'
import { SmsReportPage } from '@/features/sms/ReportPage'

/** A text campaign's report, live while it sends (SmsReport). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/sms-report')({
  component: function SmsReportRoute() {
    const { campaignId } = Route.useParams()
    return <SmsReportPage key={campaignId} campaignId={campaignId} />
  },
})
