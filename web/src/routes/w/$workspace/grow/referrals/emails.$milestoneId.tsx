import { createFileRoute } from '@tanstack/react-router'
import { MilestoneEmailPage } from '@/features/grow/MilestoneEmailPage'

/** A milestone email (the composer's milestone mode): full screen, outside the app shell. */
export const Route = createFileRoute('/w/$workspace/grow/referrals/emails/$milestoneId')({
  component: function MilestoneEmailRoute() {
    const { milestoneId } = Route.useParams()
    return <MilestoneEmailPage milestoneId={milestoneId} />
  },
})
