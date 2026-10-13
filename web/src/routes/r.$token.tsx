import { createFileRoute } from '@tanstack/react-router'
import { SharedReportPage } from '@/features/reports/SharedReportPage'

/** A read-only shared report: public, outside the app shell, never indexed. */
export const Route = createFileRoute('/r/$token')({
  component: function SharedReportRoute() {
    const { token } = Route.useParams()
    return <SharedReportPage token={token} />
  },
})
