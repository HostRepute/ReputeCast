import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppEventsPage } from '@/features/whatsapp/EventsPage'

/** Sending providers › WhatsApp › Events (design expansion phase 8, X22). */
export const Route = createFileRoute('/w/$workspace/_app/providers/whatsapp/events')({
  component: WhatsAppEventsPage,
})
