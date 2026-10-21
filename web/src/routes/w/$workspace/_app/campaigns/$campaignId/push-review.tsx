import { createFileRoute } from '@tanstack/react-router'
import { PushReviewPage } from '@/features/push/ReviewPage'

/** Review & send for a push campaign (board PushStates). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/push-review')({
  component: function PushReviewRoute() {
    const { campaignId } = Route.useParams()
    return <PushReviewPage key={campaignId} campaignId={campaignId} />
  },
})
