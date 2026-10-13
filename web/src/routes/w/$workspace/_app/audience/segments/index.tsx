import { createFileRoute } from '@tanstack/react-router'
import { SegmentsIndex } from '@/features/audience/SegmentBuilder'

export const Route = createFileRoute('/w/$workspace/_app/audience/segments/')({
  component: SegmentsIndex,
})
