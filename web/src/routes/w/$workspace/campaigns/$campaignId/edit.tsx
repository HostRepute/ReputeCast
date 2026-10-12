import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/campaigns/$campaignId/edit')({
  component: () => <NotBuilt screen="Composer" />,
})
