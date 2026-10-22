import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { useDocumentTitle } from '@/lib/hooks'
import { setDisplayTimeZone, shortDate } from '@/lib/format'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './messages'
import { ReportBody } from './ReportBody'
import { ReportSkeleton } from './ReportSkeleton'
import { headerCaption, statusBadge } from './words'
import './reports.css'

/** Keeps the page out of search engines (the server says so too). */
function useNoIndex() {
  useEffect(() => {
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    const ref = document.createElement('meta')
    ref.name = 'referrer'
    ref.content = 'no-referrer'
    document.head.appendChild(ref)
    return () => {
      meta.remove()
      ref.remove()
    }
  }, [])
}

/**
 * A read-only shared report (`/r/{token}`): the campaign's aggregate numbers
 * for anyone with the link. No app shell, no sign-in, no contact data.
 */
export function SharedReportPage({ token }: { token: string }) {
  const t = useT()
  useNoIndex()
  const q = useQuery({
    queryKey: ['shared-report', token],
    queryFn: async () => {
      const { data } = await http.GET('/api/v1/shared-reports/{token}', { params: { path: { token } } })
      return data!
    },
    retry: false,
    refetchOnWindowFocus: false,
  })
  const r = q.data?.report
  // Times read in the browser's timezone for someone outside the workspace.
  useEffect(() => setDisplayTimeZone(undefined), [])
  useDocumentTitle(r ? t(m.sharedDocTitle, { name: r.campaign.name, workspace: q.data!.workspace }) : t(m.sharedDocTitleLoading))
  const badge = r ? statusBadge(r.campaign) : undefined
  const caption = r ? headerCaption(r.campaign) : ''
  const gone = q.error instanceof ApiError && q.error.status === 404
  const limited = q.error instanceof ApiError && q.error.status === 429
  return (
    <div className="rc-app min-h-dvh bg-ground">
      <a href="#main" className="skip">
        {t(m.skip)}
      </a>
      <header className="top">
        <div className="mx-auto flex w-full max-w-[1152px] flex-wrap items-center gap-3">
          <span className="flex items-center gap-2.5 text-ink" aria-label={'ReputeCast'}>
            <LogoMark size={26} />
            <span className="text-[15px] font-bold tracking-[-0.01em]">ReputeCast</span>
          </span>
          <span className="k">{t(m.readOnlyReport)}</span>
          {q.data ? <span className="k ml-auto">{t(m.sharedBy, { workspace: q.data.workspace })}</span> : null}
        </div>
      </header>
      {/* As wide as the report inside the app at 1440 px, so the tiles and cards fall the same way. */}
      <main className="content mx-auto max-w-[1152px]" id="main" tabIndex={-1}>
        {q.isPending ? (
          <ReportSkeleton />
        ) : gone || limited || q.isError ? (
          <div className="mx-auto w-full max-w-[640px] py-10">
            <FullPageState
              icon={gone ? 'searchLarge' : 'warningTriangle'}
              tone={gone ? 'neutral' : 'warn'}
              badge={gone ? t(m.goneBadge) : limited ? t(m.limitedBadge) : t(m.errorBadge)}
              title={gone ? t(m.goneTitle) : limited ? t(m.limitedTitle) : t(m.errorTitle)}
              actions={!gone ? <Button onClick={() => void q.refetch()}>{t(common.tryAgain)}</Button> : undefined}
            >
              {gone ? t(m.goneBody) : limited ? t(m.limitedBody) : `${errorMessage(q.error)} ${t(m.errorBody)}`}
            </FullPageState>
          </div>
        ) : r ? (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="m-0 text-[20px] font-bold tracking-[-0.01em]">{r.campaign.name}</h1>
              {badge ? (
                <Badge tone={badge.tone} dot>
                  {badge.text}
                </Badge>
              ) : null}
              {caption ? <span className="k">{caption}</span> : null}
            </div>
            <ReportBody r={r} />
            <p className="k m-0">
              {[t(m.aggregateOnly), ...(q.data!.expiresAt ? [t(m.stopsOn, { date: shortDate(q.data!.expiresAt) })] : []), t(m.madeWith)].join(' ')}
            </p>
          </>
        ) : null}
      </main>
    </div>
  )
}
