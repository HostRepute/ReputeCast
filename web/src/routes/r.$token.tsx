import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/r/$token')({
  component: () => <NotBuilt screen="Shared report" />,
})
