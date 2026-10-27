import { createFileRoute } from '@tanstack/react-router'
import { ReviewLinkPage } from '@/features/reviewlinks/ReviewLinkPage'

/** A composer review link (CO-10): public, outside the app shell, never indexed. */
export const Route = createFileRoute('/review/$token')({
  component: function ReviewLinkRoute() {
    const { token } = Route.useParams()
    return <ReviewLinkPage token={token} />
  },
})
