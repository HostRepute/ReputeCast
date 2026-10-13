import { createFileRoute } from '@tanstack/react-router'
import { AdaptersPage } from '@/features/providers/AdaptersPage'

export const Route = createFileRoute('/w/$workspace/_app/providers/adapters')({
  component: AdaptersPage,
})
