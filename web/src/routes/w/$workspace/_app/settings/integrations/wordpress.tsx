import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { WordPressPage } from '@/features/integrations/WordPressPage'

/** ?connect opens "Connect a site" (the catalog's Connect). */
const search = z.object({
  connect: z.coerce.boolean().optional().catch(undefined),
})

/** Settings › Integrations › WordPress: the sites. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/wordpress')({
  validateSearch: search,
  component: function WordPressRoute() {
    return <WordPressPage connect={Route.useSearch().connect} />
  },
})
