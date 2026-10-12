import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/providers/connect/$adapter')({
  component: () => <NotBuilt screen="Connect a sending provider" />,
})
