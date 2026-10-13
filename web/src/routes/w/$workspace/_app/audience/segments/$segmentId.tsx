import { createFileRoute } from '@tanstack/react-router'
import { SegmentBuilderPage } from '@/features/audience/SegmentBuilder'

export const Route = createFileRoute('/w/$workspace/_app/audience/segments/$segmentId')({
  component: function SegmentRoute() {
    const { segmentId } = Route.useParams()
    return <SegmentBuilderPage segmentId={segmentId} />
  },
})
