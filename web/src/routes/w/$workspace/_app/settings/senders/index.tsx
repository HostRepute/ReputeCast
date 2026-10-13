import { createFileRoute } from '@tanstack/react-router'
import { SendersPage } from '@/features/senders/SendersPage'

/** 7.13 Settings › Senders: sender identities, domains and defaults. */
export const Route = createFileRoute('/w/$workspace/_app/settings/senders/')({
  component: SendersPage,
})
