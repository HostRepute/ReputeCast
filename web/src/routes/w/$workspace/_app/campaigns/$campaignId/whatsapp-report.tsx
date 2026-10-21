import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppReportPage } from '@/features/whatsapp/ReportPage'

/** A WhatsApp campaign's report. */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/whatsapp-report')({
  component: function WhatsAppReportRoute() {
    const { campaignId } = Route.useParams()
    return <WhatsAppReportPage key={campaignId} campaignId={campaignId} />
  },
})
