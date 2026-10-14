import { createFileRoute } from '@tanstack/react-router'
import { PolicyPage } from '@/features/approvals/PolicyPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/approvals')({
  component: PolicyPage,
})
