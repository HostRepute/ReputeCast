import { createFileRoute } from '@tanstack/react-router'
import { DigestSettingsPage } from '@/features/digests/SettingsPage'

/** A recurring digest's settings: feed, filters, schedule, audience and sending mode (DigestSetup). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/digests/$digestId/settings')({
  component: function DigestSettingsRoute() {
    const { digestId } = Route.useParams()
    return <DigestSettingsPage digestId={digestId} />
  },
})
