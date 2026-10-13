import { createFileRoute } from '@tanstack/react-router'
import { ConsentStep } from '@/features/imports/ConsentStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/consent')({
  component: function ConsentRoute() {
    const { importId } = Route.useParams()
    return <ConsentStep key={importId} importId={importId} />
  },
})
