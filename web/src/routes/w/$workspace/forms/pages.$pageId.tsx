import { createFileRoute, Outlet, useChildMatches } from '@tanstack/react-router'
import { z } from 'zod'
import { PageBuilderPage } from '@/features/pages/builder/PageBuilderPage'

const search = z.object({
  /** Opens the preview (Pages › Preview links here with view=preview). */
  view: z.enum(['preview']).optional().catch(undefined),
  tab: z.enum(['block', 'page', 'checks']).optional().catch(undefined),
  device: z.enum(['desktop', 'tablet', 'phone']).optional().catch(undefined),
  /** Opens the publish dialog (an approval notification links here with publish=1). */
  publish: z
    .union([z.literal('1'), z.literal(1), z.literal(true)])
    .optional()
    .catch(undefined),
  /** A block to select once the canvas is ready, by its id (b-faq). */
  select: z
    .string()
    .regex(/^[a-z][a-z0-9-]{0,60}$/)
    .optional()
    .catch(undefined),
})

/** The page builder (design expansion phase 7, X19): full screen, outside the app shell, like the composer. */
export const Route = createFileRoute('/w/$workspace/forms/pages/$pageId')({
  validateSearch: search,
  component: function PageBuilderRoute() {
    const { pageId } = Route.useParams()
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    // The preview link (/preview) is a page of its own under this one.
    const children = useChildMatches()
    if (children.length > 0) return <Outlet />
    return (
      <PageBuilderPage
        pageId={pageId}
        search={{ view: s.view, tab: s.tab, device: s.device, publish: s.publish !== undefined, select: s.select }}
        onSearch={(next) =>
          void navigate({
            // The deep links are used once: later changes drop them.
            search: { tab: next.tab, device: next.device, view: next.view },
            replace: true,
            ignoreBlocker: true,
          })
        }
      />
    )
  },
})
