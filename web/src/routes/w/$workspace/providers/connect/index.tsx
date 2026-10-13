import { createFileRoute } from '@tanstack/react-router'
import { CatalogPage } from '@/features/connect/CatalogPage'

/** 7.2 step 1: the provider catalog. */
export const Route = createFileRoute('/w/$workspace/providers/connect/')({
  component: CatalogPage,
})
