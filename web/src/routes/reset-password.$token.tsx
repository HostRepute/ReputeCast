import { createFileRoute } from '@tanstack/react-router'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/reset-password/$token')({
  component: () => <NotBuilt screen="Reset password" />,
})
