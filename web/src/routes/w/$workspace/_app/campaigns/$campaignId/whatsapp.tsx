import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppCampaignPage } from '@/features/whatsapp/CampaignPage'

/** A WhatsApp campaign (board WhatsAppChannel › A WhatsApp campaign). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/whatsapp')({
  component: function WhatsAppCampaignRoute() {
    const { campaignId } = Route.useParams()
    return <WhatsAppCampaignPage key={campaignId} campaignId={campaignId} />
  },
})
