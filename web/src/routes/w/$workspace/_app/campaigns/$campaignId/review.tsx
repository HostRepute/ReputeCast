import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/campaigns/$campaignId/review')({
  component: () => (
    <Page title="Review & send">
      <NotBuilt screen="Review & send" />
    </Page>
  ),
})
