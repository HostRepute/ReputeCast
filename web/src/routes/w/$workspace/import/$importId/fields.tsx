import { createFileRoute } from '@tanstack/react-router'
import { FieldsStep } from '@/features/imports/FieldsStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/fields')({
  component: function FieldsRoute() {
    const { importId } = Route.useParams()
    return <FieldsStep key={importId} importId={importId} />
  },
})
