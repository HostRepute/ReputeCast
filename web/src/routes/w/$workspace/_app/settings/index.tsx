import { createFileRoute } from '@tanstack/react-router'
import { GeneralPage } from '@/features/workspace/GeneralPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/')({
  component: GeneralPage,
})
