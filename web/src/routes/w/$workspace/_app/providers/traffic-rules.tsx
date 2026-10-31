import { createFileRoute } from '@tanstack/react-router'
import { TrafficRulesPage } from '@/features/traffic/TrafficRulesPage'

/** Sending providers › Traffic rules (C9). */
export const Route = createFileRoute('/w/$workspace/_app/providers/traffic-rules')({
  component: TrafficRulesPage,
})
