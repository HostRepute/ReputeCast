import { createFileRoute } from '@tanstack/react-router'
import { WhatsAppTemplatesPage } from '@/features/whatsapp/TemplatesPage'

/** Sending providers › WhatsApp › Templates (design expansion phase 8, X22). */
export const Route = createFileRoute('/w/$workspace/_app/providers/whatsapp/templates')({
  component: WhatsAppTemplatesPage,
})
