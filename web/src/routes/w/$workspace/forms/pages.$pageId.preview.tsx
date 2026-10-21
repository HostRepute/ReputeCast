import { createFileRoute } from '@tanstack/react-router'
import { PagePreviewPage } from '@/features/pages/builder/PreviewPage'

/** The page builder's preview link: the saved draft as readers will see it, for the workspace's members only. */
export const Route = createFileRoute('/w/$workspace/forms/pages/$pageId/preview')({
  component: function PagePreviewRoute() {
    const { pageId } = Route.useParams()
    return <PagePreviewPage pageId={pageId} />
  },
})
