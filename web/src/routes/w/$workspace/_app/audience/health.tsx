import { createFileRoute } from '@tanstack/react-router'
import { HealthPage } from '@/features/audience/HealthPage'

export const Route = createFileRoute('/w/$workspace/_app/audience/health')({
  component: HealthPage,
})
