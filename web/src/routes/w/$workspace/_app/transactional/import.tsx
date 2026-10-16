import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { z } from 'zod'
import { $api, errorMessage } from '@/api/client'
import { m as shell } from '@/components/shell/messages'
import { Page } from '@/components/shell/Page'
import { ErrorPanel, Loading } from '@/components/ui/Feedback'
import { ImportDialog } from '@/features/transactional-import/ImportDialog'
import { m } from '@/features/transactional-import/messages'
import { useT } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'

/** Import templates: a dialog over the Transactional section. ?id= resumes an import in progress. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/import')({
  validateSearch: z.object({ id: z.uuid().optional().catch(undefined) }),
  component: function TransactionalImportRoute() {
    const t = useT()
    const { workspace } = Route.useParams()
    const { id } = Route.useSearch()
    const navigate = Route.useNavigate()
    // An import opened by its link loads first; one started here keeps its dialog.
    const [resumed] = useState(Boolean(id))
    const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/transactional/imports/{importId}', { params: { path: { workspace, importId: id ?? '' } } }, { enabled: resumed && !!id })
    useDocumentTitle(t(m.pageTitle))
    const close = () => void navigate({ to: '/w/$workspace/transactional', params: { workspace } })
    const live = q.data && (q.data.status === 'pick' || q.data.status === 'review') ? q.data : undefined
    return (
      <Page title={t(shell.navTransactional)}>
        <p className="k m-0">{t(m.backdrop)}</p>
        {resumed && q.isPending ? (
          <Loading label={t(m.title)}>{t(m.title)}</Loading>
        ) : resumed && q.isError ? (
          <ErrorPanel title={t(m.loadFailed)}>{errorMessage(q.error)}</ErrorPanel>
        ) : (
          <ImportDialog initial={live} onClose={close} onSession={(next) => void navigate({ search: { id: next }, replace: true })} />
        )}
      </Page>
    )
  },
})
