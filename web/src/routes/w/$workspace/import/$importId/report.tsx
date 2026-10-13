import { createFileRoute } from '@tanstack/react-router'
import { ReportStep } from '@/features/imports/ReportStep'
import '@/features/imports/jobsSource'

export const Route = createFileRoute('/w/$workspace/import/$importId/report')({
  component: function ReportRoute() {
    const { importId } = Route.useParams()
    return <ReportStep key={importId} importId={importId} />
  },
})
