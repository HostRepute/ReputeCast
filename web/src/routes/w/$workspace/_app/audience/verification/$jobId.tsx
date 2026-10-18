import { createFileRoute } from '@tanstack/react-router'
import { JobPage } from '@/features/verification/JobPage'

export const Route = createFileRoute('/w/$workspace/_app/audience/verification/$jobId')({
  component: function JobRoute() {
    return <JobPage jobId={Route.useParams().jobId} />
  },
})
