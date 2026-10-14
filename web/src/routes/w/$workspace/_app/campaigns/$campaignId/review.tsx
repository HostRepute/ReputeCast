import { createFileRoute } from '@tanstack/react-router'
import { ReviewPage } from '@/features/review/ReviewPage'

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/review')({
  component: function ReviewRoute() {
    const { campaignId } = Route.useParams()
    return <ReviewPage key={campaignId} campaignId={campaignId} />
  },
})
