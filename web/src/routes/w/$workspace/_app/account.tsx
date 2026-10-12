import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/account')({
  component: () => (
    <Page title="Account & security">
      <NotBuilt screen="Account & security" />
    </Page>
  ),
})
