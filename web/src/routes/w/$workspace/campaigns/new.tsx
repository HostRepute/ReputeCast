import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/campaigns/new')({
  component: () => <NotBuilt screen="New campaign" />,
})
