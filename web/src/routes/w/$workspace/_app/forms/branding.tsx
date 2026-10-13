import { createFileRoute } from '@tanstack/react-router'
import { BrandingPage } from '@/features/forms/BrandingPage'

export const Route = createFileRoute('/w/$workspace/_app/forms/branding')({
  component: BrandingPage,
})
