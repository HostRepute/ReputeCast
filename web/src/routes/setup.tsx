import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/setup')({
  component: () => <NotBuilt screen="Set up" />,
})
