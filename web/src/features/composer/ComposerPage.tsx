import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { ErrorPanel } from '@/components/ui/Feedback'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { Composer, type ComposerSearch } from './Composer'
import './composer.css'

/** Loading state (ComposerStates): the composer's frame with skeletons while a campaign opens. */
export function ComposerSkeleton({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="cmp" aria-busy="true">
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

function ComposerError({ title, children, retry }: { title: string; children: string; retry?: () => void }) {
  const { slug } = useWorkspace()
  useDocumentTitle(`${title} · ReputeCast`)
  return (
    <div className="cmp items-center justify-center" style={{ padding: 24 }}>
      <ErrorPanel
        title={title}
        className="w-full max-w-[560px]"
        actions={
          <>
            {retry ? (
              <button type="button" className="btn pri" onClick={retry}>
                Try again
              </button>
            ) : null}
            <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="btn">
              Back to campaigns
            </Link>
          </>
        }
      >
        {children}
      </ErrorPanel>
    </div>
  )
}

/** /w/$workspace/campaigns/$campaignId/edit: loads the campaign and the composer's context. */
export function ComposerPage({ campaignId, search, onSearch }: { campaignId: string; search: ComposerSearch; onSearch: (s: ComposerSearch) => void }) {
  const { slug } = useWorkspace()
  const campaign = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId } } }, { refetchOnWindowFocus: false, retry: (n, e) => !(e as unknown as ApiError)?.status && n < 2 })
  const context = $api.useQuery('get', '/api/v1/workspaces/{workspace}/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  useDocumentTitle(campaign.data ? `${campaign.data.name} · Composer · ReputeCast` : 'Composer · ReputeCast')
  if (campaign.isError) {
    const e = campaign.error as unknown
    if (e instanceof ApiError && e.status === 404) return <ComposerError title="This campaign doesn't exist">It may have been deleted, or the link is from another workspace.</ComposerError>
    return (
      <ComposerError title="The campaign couldn't open" retry={() => void campaign.refetch()}>
        {errorMessage(e)}
      </ComposerError>
    )
  }
  if (context.isError)
    return (
      <ComposerError title="The composer couldn't load" retry={() => void context.refetch()}>
        {errorMessage(context.error)}
      </ComposerError>
    )
  if (!campaign.data || !context.data) return <ComposerSkeleton title="Opening the campaign" detail="Content, audience and sender · usually under a second" />
  return <Composer key={campaign.data.id} campaign={campaign.data} context={context.data} search={search} onSearch={onSearch} />
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
  if (error) {
    if (error instanceof ApiError && error.status === 403)
      return <ComposerError title="You can't create campaigns here">Creating a campaign needs the campaigns:edit permission. Ask an owner or admin of the workspace.</ComposerError>
    return (
      <ComposerError
        title="The draft couldn't be created"
        retry={() => {
          started.current = false
          setError(null)
          setAttempt((n) => n + 1)
        }}
      >
        {errorMessage(error)}
      </ComposerError>
    )
  }
  return <ComposerSkeleton title="Creating a draft" detail="New draft · not saved yet" />
}
