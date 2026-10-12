import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/workspaces/new')({
  component: () => <NotBuilt screen="Create workspace" />,
})
