import { createFileRoute } from '@tanstack/react-router'
import { ScoringPage } from '@/features/scoring/ScoringPage'

export const Route = createFileRoute('/w/$workspace/_app/audience/scoring')({
  component: ScoringPage,
})
