import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'

const filters = ['all', 'unread', 'approvals', 'sending', 'providers', 'team', 'hostrepute'] as const

export const Route = createFileRoute('/w/$workspace/_app/notifications')({
  validateSearch: z.object({ filter: z.enum(filters).optional() }),
  component: function NotificationsRoute() {
    const { filter } = Route.useSearch()
    return <NotificationsPage filter={filter ?? 'all'} />
  },
})
