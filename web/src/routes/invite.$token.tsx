import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/invite/$token')({
  component: () => <NotBuilt screen="Accept invitation" />,
})
