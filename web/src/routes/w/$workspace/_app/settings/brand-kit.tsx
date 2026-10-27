import { createFileRoute } from '@tanstack/react-router'
import { BrandKitPage } from '@/features/brandkit/BrandKitPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/brand-kit')({
  component: BrandKitPage,
})
