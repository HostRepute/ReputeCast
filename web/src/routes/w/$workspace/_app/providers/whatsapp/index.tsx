import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppAccountPage } from '@/features/whatsapp/AccountPage'

/** Sending providers › WhatsApp › Account (design expansion phase 8, X22). */
export const Route = createFileRoute('/w/$workspace/_app/providers/whatsapp/')({
  component: WhatsAppAccountPage,
})
