import { createFileRoute } from '@tanstack/react-router'
import { PreviewStep } from '@/features/imports/PreviewStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/preview')({
  component: function PreviewRoute() {
    const { importId } = Route.useParams()
    return <PreviewStep key={importId} importId={importId} />
  },
})
