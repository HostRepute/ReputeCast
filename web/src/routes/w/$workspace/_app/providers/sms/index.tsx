import { createFileRoute } from '@tanstack/react-router'
import { SmsProvidersPage } from '@/features/sms/ProvidersPage'

/** Sending providers › SMS › Providers (design expansion phase 8). */
export const Route = createFileRoute('/w/$workspace/_app/providers/sms/')({
  component: SmsProvidersPage,
})
