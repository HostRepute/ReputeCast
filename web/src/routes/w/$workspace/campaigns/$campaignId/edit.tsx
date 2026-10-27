import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ComposerPage } from '@/features/composer/ComposerPage'

const search = z.object({
  view: z.enum(['visual', 'write', 'mjml', 'html', 'preview', 'compare']).optional().catch(undefined),
  /** A language version (its code, such as de); none is the source. */
  language: z
    .string()
    .regex(/^[a-z]{2,3}$/)
    .optional()
    .catch(undefined),
  device: z.enum(['desktop', 'mobile']).optional().catch(undefined),
  variant: z.enum(['A', 'B', 'C']).optional().catch(undefined),
  /** Deep link to a component: its element path from mj-body ("2" is the third section, "1-0-0" its first element). */
  select: z.coerce
    .string()
    .regex(/^\d+(-\d+){0,8}$/)
    .optional()
    .catch(undefined),
  /** Panel to open with it: the inspector tab, and the left panel tab. */
  panel: z.enum(['style', 'settings', 'deliverability', 'accessibility', 'web', 'comments']).optional().catch(undefined),
  /** A comment thread to show with panel=comments (a mention's link). */
  thread: z.string().uuid().optional().catch(undefined),
  left: z.enum(['blocks', 'layers', 'templates']).optional().catch(undefined),
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
        onSearch={(next) =>
          void navigate({
            // The deep link is used once: later changes drop it.
            search: { view: next.view === 'visual' ? undefined : next.view, device: next.device, variant: next.variant as 'B' | 'C' | undefined, language: next.language },
            replace: true,
            ignoreBlocker: true,
          })
        }
      />
    )
  },
})
