import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppReviewPage } from '@/features/whatsapp/ReviewPage'

/** Review & send for a WhatsApp campaign (board WhatsAppChannel). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/whatsapp-review')({
  component: function WhatsAppReviewRoute() {
    const { campaignId } = Route.useParams()
    return <WhatsAppReviewPage key={campaignId} campaignId={campaignId} />
  },
})
