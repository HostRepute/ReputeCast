import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/providers/warmup')({
  component: () => (
    <Page title="Sending providers & routing">
      <NotBuilt screen="Sending providers & routing" />
    </Page>
  ),
})
