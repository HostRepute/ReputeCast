import { createFileRoute, Outlet, useChildMatches } from '@tanstack/react-router'
import { z } from 'zod'
import { TEMPLATE_KEYS } from '@/features/pages/format'
import { PagesTab } from '@/features/pages/PagesTab'

const search = z.object({
  // New page: the template picker, open on this template.
  new: z.enum(TEMPLATE_KEYS).optional().catch(undefined),
})

// Forms & pages › Pages (design expansion phase 7, X19): your pages with
// their numbers, the template picker and the built-in pages. Each page opens
// in the page builder, /w/$workspace/forms/pages/$pageId.
export const Route = createFileRoute('/w/$workspace/_app/forms/pages')({
  validateSearch: search,
  component: function PagesRoute() {
    const s = Route.useSearch()
    // A screen nested under /forms/pages renders in place of the list.
    const children = useChildMatches()
    return children.length > 0 ? <Outlet /> : <PagesTab search={s} />
  },
})
