import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, http } from '@/api/client'
import { ForbiddenState, NotFoundState, problemReference, ServerErrorState } from '@/components/shell/SystemScreens'
import { ButtonLink } from '@/components/ui/Button'
import { useDocumentTitle } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { Composer, type ComposerSearch } from './Composer'
import { useOpenLanguage } from '@/features/languages/composerParts'
import './composer.css'

/** Loading state (ComposerStates): the composer's frame with skeletons while a campaign opens. */
export function ComposerSkeleton({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="cmp rc-app" aria-busy="true">
      <h1 className="vh">Opening the composer</h1>
      <div className="cmp-head" style={{ gap: 12 }}>
        <span className="sk" style={{ width: 32, height: 32 }} />
        <span className="sk" style={{ width: 180, height: 18 }} />
        <span className="sk" style={{ width: 120, height: 22, borderRadius: 999 }} />
        <span className="sk" style={{ width: 300, maxWidth: '40%', height: 32, marginLeft: 'auto' }} />
      </div>
      <div className="cmp-body" style={{ minHeight: 300 }}>
        <div className="cpanel left" style={{ padding: 12, display: 'grid', gridTemplateColumns: 'repeat(3,minmax(0,1fr))', gap: 6, alignContent: 'start' }}>
          {Array.from({ length: 6 }, (_, i) => (
            <span key={i} className="sk" style={{ height: 58 }} />
          ))}
        </div>
        <div className="cmp-main" style={{ flexBasis: 400, alignItems: 'center', justifyContent: 'center', flexDirection: 'row', gap: 12, padding: 24 }} role="status">
          <span className="spin" style={{ width: 24, height: 24 }} aria-hidden="true" />
          <div className="flex flex-col">
            <span className="font-semibold">{title}</span>
            <span className="k">{detail}</span>
          </div>
        </div>
        <div className="cpanel right" style={{ padding: 12, gap: 10 }}>
          <span className="sk" style={{ height: 14, width: '60%' }} />
          <span className="sk" style={{ height: 32 }} />
          <span className="sk" style={{ height: 32 }} />
          <span className="sk" style={{ height: 32 }} />
        </div>
      </div>
    </div>
  )
}

/**
 * The composer is full screen, so its 403/404/500 states use the shared
 * system states in a centred frame instead of the app shell.
 */
function ComposerError({ error, path, campaignId, retry }: { error: unknown; path: string; campaignId?: string; retry?: () => void }) {
  const { slug } = useWorkspace()
  const canAudit = useCan('audit:view')
  const back = (
    <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
      Back to campaigns
    </ButtonLink>
  )
  const status = error instanceof ApiError ? error.status : 0
  useDocumentTitle(`${status === 404 ? 'Not found' : status === 403 ? 'No access' : 'Something broke'} · ReputeCast`)
  let body
  if (error instanceof ApiError && status === 403 && error.problem.permission) {
    body = <ForbiddenState permission={error.problem.permission} title={path.endsWith('/new') ? "Your role can't create campaigns" : "Your role can't open this campaign"} readOnly={back} />
  } else if (status === 404) {
    // SystemStates' 404: "Go to campaigns" and "Search the audit log". (The board says deleted
    // campaigns stay in the log for 90 days; the retention is the workspace's own, 365 by default.)
    body = (
      <NotFoundState
        path={path}
        reference={problemReference(error)}
        primary={
          <ButtonLink variant="primary" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
            Go to campaigns
          </ButtonLink>
        }
        secondary={
          canAudit ? (
            <ButtonLink to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ q: campaignId, range: 'all' }}>
              Search the audit log
            </ButtonLink>
          ) : undefined
        }
      >
        The campaign may have been deleted, or the link was copied before it was saved. Deleted campaigns stay in the audit log.
      </NotFoundState>
    )
  } else {
    body = <ServerErrorState reference={problemReference(error)} savedNote={retry ? undefined : 'Nothing was created.'} onRetry={retry ?? (() => window.location.reload())} back={back} />
  }
  return (
    <div className="cmp rc-app items-center" style={{ padding: '56px 16px' }}>
      <main id="main" className="w-full max-w-[640px]">
        {body}
      </main>
    </div>
  )
}

/** /w/$workspace/campaigns/$campaignId/edit: loads the campaign and the composer's context. */
export function ComposerPage({ campaignId, search, onSearch }: { campaignId: string; search: ComposerSearch; onSearch: (s: ComposerSearch) => void }) {
  const { slug } = useWorkspace()
  const lang = useOpenLanguage(campaignId, search.language)
  const campaign = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}',
    { params: { path: { workspace: slug, campaignId }, query: lang.load ? { language: lang.load } : undefined } },
    { refetchOnWindowFocus: false, retry: (n, e) => !(e as unknown as ApiError)?.status && n < 2, enabled: lang.ready },
  )
  const context = $api.useQuery('get', '/api/v1/workspaces/{workspace}/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  useDocumentTitle(campaign.data ? `${campaign.data.name} · Composer · ReputeCast` : 'Composer · ReputeCast')
  const path = `/campaigns/${campaignId}/edit`
  if (campaign.isError) return <ComposerError error={campaign.error} path={path} campaignId={campaignId} retry={() => void campaign.refetch()} />
  if (context.isError) return <ComposerError error={context.error} path={path} retry={() => void context.refetch()} />
  if (!campaign.data || !context.data) return <ComposerSkeleton title="Opening the campaign" detail="Content, audience and sender · usually under a second" />
  return (
    <Composer key={`${campaign.data.id}:${campaign.data.language}:${lang.missing ?? ''}`} campaign={campaign.data} context={context.data} search={search} onSearch={onSearch} missingLanguage={lang.missing} />
  )
}

/** /w/$workspace/campaigns/new: creates the draft, then opens it (blank, from a template or a copy). */
export function NewCampaignPage({ template, from }: { template?: string; from?: string }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const started = useRef(false)
  const [error, setError] = useState<unknown>(null)
  const [attempt, setAttempt] = useState(0)
  useDocumentTitle('New campaign · ReputeCast')
  useEffect(() => {
    if (started.current) return
    started.current = true
    http
      .POST('/api/v1/workspaces/{workspace}/campaigns', { params: { path: { workspace: slug } }, body: { templateId: template, fromCampaignId: from } })
      .then(({ data }) => {
        if (data) void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: data.id }, replace: true })
      })
      .catch((e: unknown) => setError(e))
  }, [slug, template, from, navigate, attempt])
  if (error)
    return (
      <ComposerError
        error={error}
        path="/campaigns/new"
        retry={() => {
          started.current = false
          setError(null)
          setAttempt((n) => n + 1)
        }}
      />
    )
  return <ComposerSkeleton title="Creating a draft" detail="New draft · not saved yet" />
}
