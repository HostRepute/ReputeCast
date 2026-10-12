import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/settings/')({
  component: () => (
    <Page title="Settings">
      <NotBuilt screen="Settings" />
    </Page>
  ),
})
