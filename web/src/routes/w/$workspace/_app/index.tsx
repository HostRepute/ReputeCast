import { createFileRoute } from '@tanstack/react-router'
import { OverviewPage } from '@/features/overview/OverviewPage'
import { overviewFirst } from '@/features/overview/queries'
import { prefetchFirst } from '@/lib/prefetch'

export const Route = createFileRoute('/w/$workspace/_app/')({
  // The cards' data loads alongside the screen's code, not after it.
  loader: ({ context, params }) => prefetchFirst(context.queryClient, params.workspace, overviewFirst(params.workspace)),
  component: OverviewPage,
})
