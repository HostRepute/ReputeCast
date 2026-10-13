import { createFileRoute } from '@tanstack/react-router'
import { WarmupPage } from '@/features/providers/WarmupPage'

export const Route = createFileRoute('/w/$workspace/_app/providers/warmup')({
  component: WarmupPage,
})
