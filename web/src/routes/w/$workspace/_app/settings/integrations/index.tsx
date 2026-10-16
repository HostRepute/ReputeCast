import { createFileRoute } from '@tanstack/react-router'
import { CatalogPage } from '@/features/integrations/CatalogPage'

/** Settings › Integrations: the catalog. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/')({
  component: CatalogPage,
})
