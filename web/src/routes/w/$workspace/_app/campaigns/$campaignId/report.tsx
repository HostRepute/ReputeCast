import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/report')({
  component: () => (
    <Page title="Campaign report">
      <NotBuilt screen="Campaign report" />
    </Page>
  ),
})
