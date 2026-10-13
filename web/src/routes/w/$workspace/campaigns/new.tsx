import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { NewCampaignPage } from '@/features/composer/ComposerPage'

const search = z.object({
  /** Start from a template: starter:<key> or a saved template id. */
  template: z.string().max(80).optional().catch(undefined),
  /** Start from a copy of another campaign. */
  from: z.string().uuid().optional().catch(undefined),
})

/** Creates a draft and opens it in the composer (empty canvas with the template gallery). */
export const Route = createFileRoute('/w/$workspace/campaigns/new')({
  validateSearch: search,
  component: function NewCampaignRoute() {
    const s = Route.useSearch()
    return <NewCampaignPage template={s.template} from={s.from} />
  },
})
