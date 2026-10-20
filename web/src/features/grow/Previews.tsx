import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { GROW } from './shared'

export type PreviewPage = 'invite' | 'hub' | 'leaderboard' | 'recommends'

function pageName(t: ReturnType<typeof useT>, page: PreviewPage): string {
  switch (page) {
    case 'invite':
      return t(m.previewInvite)
    case 'hub':
      return t(m.previewHub)
    case 'leaderboard':
      return t(m.previewBoard)
    case 'recommends':
      return t(m.previewRecommends)
  }
  return page
}

/** A button that opens a public page's preview. */
export function PreviewButton({ page, label, program, size = 'sm', icon }: { page: PreviewPage; label: string; program?: string; size?: 'sm' | 'md'; icon?: 'eye' | 'external' }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button size={size} icon={icon} onClick={() => setOpen(true)}>
        {label}
      </Button>
      {open ? <PreviewDialog page={page} program={program} onClose={() => setOpen(false)} /> : null}
    </>
  )
}

/** The page as the server renders it for a referrer, inert, in a frame sized to its content. */
export function PreviewDialog({ page, program, onClose }: { page: PreviewPage; program?: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/previews/{page}`, { params: { path: { workspace: slug, page }, query: program ? { program } : undefined } })
  const frame = useRef<HTMLIFrameElement>(null)
  const observer = useRef<ResizeObserver | null>(null)
  const [height, setHeight] = useState(560)
  useEffect(() => () => observer.current?.disconnect(), [])
  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (!doc?.body) return
    observer.current?.disconnect()
    const measure = () => setHeight(Math.max(doc.documentElement.scrollHeight, 200))
    measure()
    observer.current = new ResizeObserver(measure)
    observer.current.observe(doc.body)
  }
  const name = pageName(t, page)
  const d = q.data
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="lg"
      title={t(m.previewTitle, { page: name })}
      description={d ? (d.contact && page !== 'invite' ? t(m.previewFor, { who: d.contact }) : t(m.previewForAnyone)) : undefined}
      footer={<Button onClick={onClose}>{t(m.close)}</Button>}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.previewError)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {q.error instanceof ApiError && q.error.status === 404 ? t(m.previewNoLink) : errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <Skeleton h={480} r={10} />
      ) : (
        <div className="flex flex-col gap-2">
          {d.url ? <span className="k">{t(m.previewAt, { url: d.url.replace(/^https?:\/\//, '') })}</span> : null}
          <div className="overflow-hidden rounded-[10px] border border-line">
            <iframe ref={frame} title={t(m.previewFrame, { page: name })} srcDoc={d.html} sandbox="allow-same-origin" className="block w-full border-0" style={{ height }} onLoad={onLoad} />
          </div>
        </div>
      )}
    </Dialog>
  )
}
