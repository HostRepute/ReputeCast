import { createFileRoute } from '@tanstack/react-router'
import { PushComposerPage } from '@/features/push/ComposerPage'

/** The push composer (board PushComposer): full screen, outside the app shell. */
export const Route = createFileRoute('/w/$workspace/campaigns/$campaignId/push')({
  component: function PushComposerRoute() {
    const { campaignId } = Route.useParams()
    return <PushComposerPage key={campaignId} campaignId={campaignId} />
  },
})
