import { createFileRoute } from '@tanstack/react-router'
import { ProgressStep } from '@/features/imports/ProgressStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/progress')({
  component: function ProgressRoute() {
    const { importId } = Route.useParams()
    return <ProgressStep key={importId} importId={importId} />
  },
})
