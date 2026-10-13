import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ComposerPage } from '@/features/composer/ComposerPage'

const search = z.object({
  view: z.enum(['visual', 'mjml', 'html', 'preview']).optional().catch(undefined),
  device: z.enum(['desktop', 'mobile']).optional().catch(undefined),
  variant: z.enum(['A', 'B', 'C']).optional().catch(undefined),
})

/** The campaign composer: full screen, outside the app shell. */
export const Route = createFileRoute('/w/$workspace/campaigns/$campaignId/edit')({
  validateSearch: search,
  component: function ComposerRoute() {
    const { campaignId } = Route.useParams()
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <ComposerPage
        campaignId={campaignId}
        search={s}
        onSearch={(next) => void navigate({ search: { view: next.view === 'visual' ? undefined : next.view, device: next.device, variant: next.variant as 'B' | 'C' | undefined }, replace: true })}
      />
    )
  },
})
