import { createFileRoute } from '@tanstack/react-router'
import { SourceStep } from '@/features/imports/SourceStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/source')({
  component: function SourceRoute() {
    const { importId } = Route.useParams()
    return <SourceStep key={importId} importId={importId} />
  },
})
