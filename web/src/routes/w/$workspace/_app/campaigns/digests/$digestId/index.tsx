import { createFileRoute } from '@tanstack/react-router'
import { DigestOverviewPage } from '@/features/digests/OverviewPage'

/** A recurring digest: its next issue, feed, settings, issues and history (DigestOverview). */
export const Route = createFileRoute('/w/$workspace/_app/campaigns/digests/$digestId/')({
  component: function DigestOverviewRoute() {
    const { digestId } = Route.useParams()
    return <DigestOverviewPage digestId={digestId} />
  },
})
