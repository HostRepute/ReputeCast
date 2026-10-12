import { createFileRoute } from '@tanstack/react-router'
import { Prototype } from '@/features/composer/Prototype'

export const Route = createFileRoute('/w/$workspace/campaigns/$campaignId/edit')({
  component: () => <Prototype />,
})
