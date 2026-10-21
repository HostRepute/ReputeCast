import { createFileRoute } from '@tanstack/react-router'
import { GrowOverviewPage } from '@/features/grow/OverviewPage'

// Grow › Overview (design expansion phase 5): the last 30 days' numbers,
// Needs you, new subscribers by source and the program and recommendations
// cards.
export const Route = createFileRoute('/w/$workspace/_app/grow/')({
  component: GrowOverviewPage,
})
