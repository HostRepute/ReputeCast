import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/live')({
  component: () => (
    <Page title="Live sending monitor">
      <NotBuilt screen="Live sending monitor" />
    </Page>
  ),
})
