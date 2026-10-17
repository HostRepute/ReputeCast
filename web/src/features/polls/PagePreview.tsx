import { useEffect, useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { useT } from '@/i18n'
import { m } from './messages'
import './polls.css'

/**
 * "Preview the answer page" (the composer) and "Open the answer page" (the
 * report): the public page as the poll draws it, inert.
 */
export function AnswerPagePreview({ slug, config, campaign, onClose }: { slug: string; config: string; campaign: string; onClose: () => void }) {
  const t = useT()
  // The settings as they were when the preview opened.
  const [body] = useState(() => ({ config, campaign }))
  const [html, setHtml] = useState('')
  const [error, setError] = useState('')
  useEffect(() => {
    let cancelled = false
    void http
      .POST('/api/v1/workspaces/{workspace}/public-pages/poll-preview', { params: { path: { workspace: slug } }, body })
      .then(({ data }) => !cancelled && data && setHtml(data.html))
      .catch((e: unknown) => !cancelled && setError(errorMessage(e)))
    return () => {
      cancelled = true
    }
  }, [body, slug])
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={t(m.previewTitle)} size="lg">
      {error ? (
        <p role="alert">{t(m.previewFailed, { detail: error })}</p>
      ) : html ? (
        <iframe title={t(m.previewTitle)} className="pl-preview" sandbox="" srcDoc={html} />
      ) : (
        <p className="k" role="status">
          {t(m.previewLoading)}
        </p>
      )}
    </Dialog>
  )
}
