import { createFileRoute } from '@tanstack/react-router'
import { PushSetupPage } from '@/features/push/SetupPage'

/** Sending providers › Push › Setup (design expansion phase 8, X22). */
export const Route = createFileRoute('/w/$workspace/_app/providers/push/')({
  component: PushSetupPage,
})
