import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { DEVICE_WIDTH, type PageDevice } from './editor/pageEditor'
import { UiIcon } from './icons'
import { m } from './messages'
import { PreviewFrame } from './PreviewDialog'
import '@/features/composer/composer.css'
import './builder.css'

/**
 * The preview link (PagesStates · Daniel Park's draft, "Copy preview
 * link"): the saved draft as readers will see it, for anyone signed in to
 * the workspace. It's an app page: the public site never shows drafts.
 */
export function PagePreviewPage({ pageId }: { pageId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [device, setDevice] = useState<PageDevice>('desktop')
  const page = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}', { params: { path: { workspace: slug, page: pageId } } }, { retry: false })
  const preview = useQuery({
    queryKey: ['page-preview', slug, pageId, page.data?.draft.version],
    enabled: Boolean(page.data),
    retry: false,
    queryFn: async () => {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/preview', { params: { path: { workspace: slug, page: pageId } }, body: {} })
      return data?.html ?? ''
    },
  })
  useDocumentTitle(t(m.previewPageTitle, { name: page.data?.name ?? '' }))
  const p = page.data
  const devices: { d: PageDevice; label: string }[] = [
    { d: 'desktop', label: t(m.deviceDesktop) },
    { d: 'tablet', label: t(m.deviceTablet) },
    { d: 'phone', label: t(m.devicePhone) },
  ]
  const error = page.error ?? preview.error
  return (
    <div className="cmp pb rc-app">
      <header className="cmp-head pb-head">
        <Link to="/w/$workspace/forms/pages/$pageId" params={{ workspace: slug, pageId }} className="btn ic" aria-label={t(m.openBuilder)}>
          <UiIcon name="back" size={16} />
        </Link>
        <h1 className="pb-crumb m-0">
          <b>{p ? t(m.previewTitle, { name: p.name }) : t(m.loadingPage)}</b>
        </h1>
        {p ? <span className="k">{p.live ? t(m.previewBarLive, { version: p.draft.version, live: p.live.version }) : t(m.previewBar, { version: p.draft.version })}</span> : null}
        <div className="pb-acts">
          <div className="seg" role="group" aria-label={t(m.deviceLabel)}>
            {devices.map((x) => (
              <button key={x.d} type="button" aria-pressed={device === x.d} onClick={() => setDevice(x.d)}>
                <UiIcon name={x.d} />
                {x.label}
              </button>
            ))}
          </div>
          <Link to="/w/$workspace/forms/pages/$pageId" params={{ workspace: slug, pageId }} className="btn">
            {t(m.openBuilder)}
          </Link>
        </div>
      </header>
      <main id="main" className="pb-shellbody pb-previewpage">
        {error ? (
          <ErrorPanel
            title={t(m.previewFailed)}
            actions={
              <Button variant="primary" onClick={() => void (page.error ? page.refetch() : preview.refetch())}>
                {t(common.retry)}
              </Button>
            }
          >
            {errorMessage(error)}
          </ErrorPanel>
        ) : preview.data !== undefined && p ? (
          <PreviewFrame html={preview.data} width={DEVICE_WIDTH[device]} title={t(m.previewFrame, { name: p.name })} />
        ) : (
          <div className="flex items-center gap-3" role="status" aria-busy="true">
            <span className="spin" aria-hidden="true" />
            <span className="font-semibold">{t(m.loadingCanvas)}</span>
          </div>
        )}
      </main>
    </div>
  )
}
