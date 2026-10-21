import { createFileRoute, Navigate } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { ImportFrame } from '@/features/imports/ImportFrame'
import { SourceStep } from '@/features/imports/SourceStep'
import { ReadingSource } from '@/features/imports/states'
// The jobs tray lists imports from the server once this module loads.
import '@/features/imports/jobsSource'
import { useT } from '@/i18n'
import { mr } from '@/routes/-routes.messages'

export const Route = createFileRoute('/w/$workspace/import/')({
  component: ImportIndex,
})

/** Resumes the draft in progress, or starts at the source step. */
function ImportIndex() {
  const t = useT()
  const { workspace } = Route.useParams()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/imports', { params: { path: { workspace } } })
  if (list.data?.active) {
    return <Navigate to="/w/$workspace/import/$importId" params={{ workspace, importId: list.data.active }} replace />
  }
  if (list.isPending) {
    return (
      <ImportFrame title={t(mr.importTitle)}>
        <ReadingSource title={t(mr.importLooking)} />
      </ImportFrame>
    )
  }
  return <SourceStep />
}
