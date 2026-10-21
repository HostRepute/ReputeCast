import { createFileRoute } from '@tanstack/react-router'
import { PushEventsPage } from '@/features/push/EventsPage'

/** Sending providers › Push › Events (design expansion phase 8, X22). */
export const Route = createFileRoute('/w/$workspace/_app/providers/push/events')({
  component: PushEventsPage,
})
