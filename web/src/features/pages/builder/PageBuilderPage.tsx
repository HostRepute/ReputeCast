import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { UiIcon } from './icons'
import { m } from './messages'
import { PageBuilder, type BuilderSearch } from './PageBuilder'
import '@/features/composer/composer.css'
import './builder.css'

/** The builder's frame while the page loads, or when it can't. */
function Shell({ children, title }: { children: ReactNode; title?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="cmp pb rc-app">
      <header className="cmp-head pb-head">
        <Link to="/w/$workspace/forms/pages" params={{ workspace: slug }} className="btn ic" aria-label={t(m.back)}>
          <UiIcon name="back" size={16} />
        </Link>
        <nav className="pb-crumb" aria-label={t(m.breadcrumb)}>
          <Link to="/w/$workspace/forms/pages" params={{ workspace: slug }}>
            {t(m.crumbPages)}
          </Link>
          {title ? (
            <>
              <span className="k" aria-hidden="true">
                ›
              </span>
              <b>{title}</b>
            </>
          ) : null}
        </nav>
      </header>
      <main id="main" className="pb-shellbody">
        {children}
      </main>
    </div>
  )
}

/**
 * /w/$workspace/forms/pages/$pageId: loads the page and the site's frame
 * for the canvas, then opens the builder. Full screen, outside the app
 * shell, like the composer.
 */
export function PageBuilderPage({ pageId, search, onSearch }: { pageId: string; search: BuilderSearch; onSearch: (s: BuilderSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const path = { params: { path: { workspace: slug, page: pageId } } }
  // The builder keeps its own copy once it's open: no refetch behind its back.
  const page = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}', path, { staleTime: Infinity, gcTime: 0, refetchOnWindowFocus: false, retry: false })
  const canvas = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}/canvas', path, { staleTime: 5 * 60_000, refetchOnWindowFocus: false, retry: 1 })
  useDocumentTitle(page.data ? t(m.documentTitle, { name: page.data.name }) : t(m.loadingPage))
  if (page.isError) {
    const missing = page.error instanceof ApiError && page.error.status === 404
    return (
      <Shell>
        <ErrorPanel
          title={missing ? t(m.pageNotFound) : t(m.loadFailed)}
          actions={
            <>
              {missing ? null : (
                <Button variant="primary" onClick={() => void page.refetch()}>
                  {t(common.retry)}
                </Button>
              )}
              <Link to="/w/$workspace/forms/pages" params={{ workspace: slug }} className="btn">
                {t(m.backToPages)}
              </Link>
            </>
          }
        >
          {missing ? t(m.pageNotFoundBody) : errorMessage(page.error)}
        </ErrorPanel>
      </Shell>
    )
  }
  if (canvas.isError) {
    return (
      <Shell title={page.data?.name}>
        <ErrorPanel
          title={t(m.canvasFailed)}
          actions={
            <Button variant="primary" onClick={() => void canvas.refetch()}>
              {t(common.retry)}
            </Button>
          }
        >
          {errorMessage(canvas.error)}
        </ErrorPanel>
      </Shell>
    )
  }
  if (!page.data || !canvas.data) {
    return (
      <Shell title={page.data?.name}>
        <div className="flex items-center gap-3" role="status" aria-busy="true">
          <span className="spin" aria-hidden="true" />
          <span className="font-semibold">{t(m.loadingPage)}</span>
        </div>
      </Shell>
    )
  }
  return <PageBuilder key={pageId} initial={page.data} canvas={canvas.data} search={search} onSearch={onSearch} />
}
