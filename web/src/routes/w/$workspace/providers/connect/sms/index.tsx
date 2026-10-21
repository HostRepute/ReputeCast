import { createFileRoute } from '@tanstack/react-router'
import { SmsCatalogPage } from '@/features/sms/Connect'

/** The connect wizard's SMS path, step 1: the SMS adapters. */
export const Route = createFileRoute('/w/$workspace/providers/connect/sms/')({
  component: SmsCatalogPage,
})
