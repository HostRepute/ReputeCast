import { createFileRoute, Navigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { errorMessage } from '@/api/client'
import { currentStep, ImportFrame, stepPath, useImportDetail } from '@/features/imports/ImportFrame'
import { ReadingSource } from '@/features/imports/states'
import '@/features/imports/jobsSource'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { mr } from '@/routes/-routes.messages'

export const Route = createFileRoute('/w/$workspace/import/$importId/')({
  component: ImportResume,
})

/** Opens an import on the step it is at (links from the jobs tray, history). */
function ImportResume() {
  const t = useT()
  const { workspace, importId } = Route.useParams()
  const q = useImportDetail(importId)
  if (q.data) return <Navigate to={stepPath[currentStep(q.data)]} params={{ workspace, importId }} replace />
  return (
    <ImportFrame title={t(mr.importTitle)}>
      {q.isError ? (
        <ErrorPanel title={t(mr.importOpenFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <ReadingSource title={t(mr.importOpening)} />
      )}
    </ImportFrame>
  )
}
