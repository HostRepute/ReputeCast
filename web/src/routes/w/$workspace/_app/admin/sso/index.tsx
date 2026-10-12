import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/admin/sso/')({
  component: () => (
    <Page title="Instance admin">
      <NotBuilt screen="Instance admin" />
    </Page>
  ),
})
