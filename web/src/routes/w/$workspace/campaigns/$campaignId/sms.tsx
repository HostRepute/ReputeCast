import { createFileRoute } from '@tanstack/react-router'
import { SmsComposerPage } from '@/features/sms/ComposerPage'

/** The SMS composer (SmsComposer): full screen, outside the app shell. */
export const Route = createFileRoute('/w/$workspace/campaigns/$campaignId/sms')({
  component: function SmsComposerRoute() {
    const { campaignId } = Route.useParams()
    return <SmsComposerPage key={campaignId} campaignId={campaignId} />
  },
})
