import { createFileRoute } from '@tanstack/react-router'
import { StaffPage } from '@/features/agency/StaffPage'

export const Route = createFileRoute('/w/$workspace/agency/staff')({
  component: StaffPage,
})
