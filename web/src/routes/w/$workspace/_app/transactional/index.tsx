import { createFileRoute } from '@tanstack/react-router'
import { m } from '@/components/shell/messages'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { Page } from '@/components/shell/Page'
import { useT } from '@/i18n'

// The sidebar's Transactional section; X02 builds its screens here.
export const Route = createFileRoute('/w/$workspace/_app/transactional/')({
  component: function TransactionalRoute() {
    const t = useT()
    return (
      <Page title={t(m.navTransactional)}>
        <NotBuilt screen={t(m.navTransactional)} />
      </Page>
    )
  },
})
