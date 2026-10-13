import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { useDocumentTitle } from '@/lib/hooks'
import { setDisplayTimeZone, shortDate } from '@/lib/format'
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
  useDocumentTitle(r ? `${r.campaign.name} · report · ${q.data!.workspace}` : 'Shared report · ReputeCast')
  const badge = r ? statusBadge(r.campaign) : undefined
  const caption = r ? headerCaption(r.campaign) : ''
  const gone = q.error instanceof ApiError && q.error.status === 404
  const limited = q.error instanceof ApiError && q.error.status === 429
  return (
    <div className="rc-app min-h-dvh bg-ground">
      <a href="#main" className="skip">
        Skip to content
      </a>
      <header className="top">
        <span className="flex items-center gap-2.5 text-ink" aria-label="ReputeCast">
          <LogoMark size={26} />
          <span className="text-[15px] font-bold tracking-[-0.01em]">ReputeCast</span>
        </span>
        <span className="k">Read-only report</span>
        {q.data ? <span className="k ml-auto">Shared by {q.data.workspace}</span> : null}
      </header>
      <main className="content" id="main" tabIndex={-1}>
        {q.isPending ? (
          <ReportSkeleton />
        ) : gone || limited || q.isError ? (
          <div className="mx-auto w-full max-w-[640px] py-10">
            <FullPageState
              icon={gone ? 'searchLarge' : 'warningTriangle'}
              tone={gone ? 'neutral' : 'warn'}
              badge={gone ? 'Link not working' : limited ? 'Too many requests' : 'Something went wrong'}
              title={gone ? 'This link doesn’t open a report' : limited ? 'Too many requests from your network' : 'The report didn’t load'}
              actions={!gone ? <Button onClick={() => void q.refetch()}>Try again</Button> : undefined}
            >
              {gone
                ? 'It may have been revoked, or it expired. Ask the person who shared it for a new one.'
                : limited
                  ? 'Wait a minute, then try again.'
                  : `${errorMessage(q.error)} Try again in a moment.`}
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
              Aggregate results only: this link never shows who received, opened or clicked.
              {q.data!.expiresAt ? ` It stops working on ${shortDate(q.data!.expiresAt)}.` : ''} Made with ReputeCast, the self-hosted newsletter platform.
            </p>
          </>
        ) : null}
      </main>
    </div>
  )
}
