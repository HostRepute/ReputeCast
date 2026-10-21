import { createFileRoute } from '@tanstack/react-router'
import { SmsReviewPage } from '@/features/sms/ReviewPage'

/** Review & send for a text campaign (SmsReview). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/sms-review')({
  component: function SmsReviewRoute() {
    const { campaignId } = Route.useParams()
    return <SmsReviewPage key={campaignId} campaignId={campaignId} />
  },
})
