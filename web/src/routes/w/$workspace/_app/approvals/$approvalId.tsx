import { createFileRoute } from '@tanstack/react-router'
import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/w/$workspace/_app/approvals/$approvalId')({
  component: () => (
    <Page title="Approvals">
      <NotBuilt screen="Approvals" />
    </Page>
  ),
})
