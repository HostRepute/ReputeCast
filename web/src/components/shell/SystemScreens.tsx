import { useQueryClient } from '@tanstack/react-query'
import { Link, useLocation, useRouter, type ErrorComponentProps } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { formatList, T, t, useT } from '@/i18n'
import { ago } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { Icon, LogoMark } from '../icons/Icon'
import { Badge, type Tone } from '../ui/Badge'
import { Button, ButtonLink } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Note } from '../ui/Note'
import { toast } from '../ui/Toast'
import { m } from './messages'
import { Page } from './Page'

const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>

/** A full-page state card: icon, status badge, sentence, actions, footnote. */
export function FullPageState({
  icon,
  tone = 'neutral',
  badge,
  title,
  children,
  actions,
  footnote,
}: {
  icon: 'searchLarge' | 'lockLarge' | 'warningTriangle'
  tone?: Tone
  badge: ReactNode
  title: ReactNode
  children?: ReactNode
  actions?: ReactNode
  footnote?: ReactNode
}) {
  const ico = { neutral: undefined, warn: { background: '#FEF3C7', color: '#92400E' }, bad: { background: '#FEE2E2', color: '#991B1B' } }[tone as 'neutral' | 'warn' | 'bad']
  return (
    <section className="card full">
      <div className="ico" style={ico}>
        <Icon name={icon} size={22} />
      </div>
      <Badge tone={tone}>{badge}</Badge>
      <h2>{title}</h2>
      {children ? <p>{children}</p> : null}
      {actions ? <div className="flex flex-wrap justify-center gap-2.5">{actions}</div> : null}
      {footnote ? <span className="k">{footnote}</span> : null}
    </section>
  )
}

/** Centered frame for states shown before a workspace is known. */
function Frame({ children }: { children: ReactNode }) {
  const tr = useT()
  return (
    <div className="flex min-h-dvh flex-col items-center gap-6 px-4 py-14">
      <Link to="/" className="flex items-center gap-2.5 text-[16px] font-bold text-ink hover:text-ink" aria-label={tr(m.homeLabel)}>
        <LogoMark size={28} />
        {tr(m.brandName)}
      </Link>
      <main id="main" className="w-full max-w-[640px]">
        {children}
      </main>
    </div>
  )
}

export function NotFoundScreen() {
  const tr = useT()
  useDocumentTitle(tr(m.docTitle, { title: tr(m.stateNotFoundTitle) }))
  const path = typeof window === 'undefined' ? '' : window.location.pathname
  return (
    <Frame>
      <FullPageState
        icon="searchLarge"
        badge="404"
        title={tr(m.stateNotFoundHeading, { path })}
        actions={
          <Link to="/" className="btn pri">
            {tr(m.stateGoToOverview)}
          </Link>
        }
      >
        {tr(m.stateNotFoundBody)}
      </FullPageState>
    </Frame>
  )
}

export function ErrorScreen({ error, reset }: ErrorComponentProps) {
  const tr = useT()
  useDocumentTitle(tr(m.docTitle, { title: tr(m.stateErrorTitle) }))
  const router = useRouter()
  const ref = error instanceof ApiError ? (error.problem as { instance?: string }).instance : undefined
  return (
    <Frame>
      <FullPageState
        icon="warningTriangle"
        tone="bad"
        badge={error instanceof ApiError ? String(error.status) : '500'}
        title={tr(m.stateErrorHeading)}
        actions={
          <>
            <Button
              variant="primary"
              onClick={() => {
                reset()
                void router.invalidate()
              }}
            >
              {tr(m.stateRetry)}
            </Button>
            <Link to="/" className="btn">
              {tr(m.stateBackToOverview)}
            </Link>
          </>
        }
        footnote={ref ? <T msg={m.stateErrorFootnoteRef} values={{ ref, mono }} /> : <T msg={m.stateErrorFootnote} values={{ mono }} />}
      >
        {tr(m.stateErrorBody)}
      </FullPageState>
    </Frame>
  )
}

// In-shell states (SystemStates board): the page keeps the sidebar and the
// top bar; the content column shows the state.

/** Request id or error reference from a problem, when the server gave one. */
export function problemReference(error: unknown): string | undefined {
  if (!(error instanceof ApiError)) return undefined
  const p = error.problem as { instance?: string; meta?: { requestId?: string; errorId?: string } }
  return p.meta?.errorId ?? p.meta?.requestId ?? p.instance
}

/** 404 for a path or a thing that doesn't exist (any more). */
export function NotFoundState({
  path,
  children,
  primary,
  secondary,
  reference,
}: {
  /** As the person sees it inside the workspace: "/campaigns/weekly-digest-143". */
  path: string
  children?: ReactNode
  primary?: ReactNode
  secondary?: ReactNode
  reference?: string
}) {
  const tr = useT()
  return (
    <FullPageState
      icon="searchLarge"
      badge="404"
      title={tr(m.stateNotFoundHeading, { path })}
      actions={
        primary || secondary ? (
          <>
            {primary}
            {secondary}
          </>
        ) : undefined
      }
      footnote={reference ? <T msg={m.stateRequestId} values={{ ref: reference, mono }} /> : undefined}
    >
      {children ?? tr(m.stateNotFoundBody)}
    </FullPageState>
  )
}

/** 500: what failed, what is safe, how to retry and where the log is. */
export function ServerErrorState({
  reference,
  savedNote,
  onRetry,
  back,
}: {
  reference?: string
  /** What is safe: "Your draft from 09:38 is saved." */
  savedNote?: string
  onRetry: () => void
  back?: ReactNode
}) {
  const tr = useT()
  const body = savedNote ? (reference ? m.stateErrorBodySavedRef : m.stateErrorBodySaved) : reference ? m.stateErrorBodyRef : m.stateErrorBody
  return (
    <FullPageState
      icon="warningTriangle"
      tone="bad"
      badge="500"
      title={tr(m.stateErrorHeading)}
      actions={
        <>
          <Button variant="primary" onClick={onRetry}>
            {tr(m.stateRetry)}
          </Button>
          {back}
        </>
      }
      footnote={reference ? <T msg={m.stateErrorFootnoteRef} values={{ ref: reference, mono }} /> : <T msg={m.stateErrorFootnote} values={{ mono }} />}
    >
      <T msg={body} values={{ saved: savedNote ?? '', ref: reference ?? '', mono }} />
    </FullPageState>
  )
}

/** "just now", "5 min ago", "yesterday", "on Oct 3" */
function askedWhen(at: string): string {
  const w = ago(at)
  if (w === 'now') return t(m.stateAskedJustNow)
  return w.endsWith(' ago') || w === 'yesterday' ? w : t(m.stateAskedOn, { date: w })
}

function names(list: string[]): string {
  return formatList(list)
}

/** Asks the members who manage the team for the role a permission needs. */
export function RequestAccessDialog({
  open,
  onOpenChange,
  permission,
  roleName,
  managers,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  permission: string
  roleName: string
  managers: string[]
}) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [note, setNote] = useState('')
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/access-requests')
  const [error, setError] = useState<string>()
  const tr = useT()
  const submit = async () => {
    setError(undefined)
    try {
      const r = await send.mutateAsync({ params: { path: { workspace: slug } }, body: { permission, note: note.trim() || undefined } })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/access-requests/suggestion'] })
      toast({ kind: 'success', title: tr(m.requestSent), detail: tr(m.requestSentDetail, { names: names(r.notified) }) })
      setNote('')
      onOpenChange(false)
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  const id = useId()
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !send.isPending && onOpenChange(o)}
      layout="confirm"
      eyebrow={<Badge tone="info">{tr(m.requestBadge)}</Badge>}
      title={tr(m.requestTitle, { role: roleName })}
      description={managers.length ? tr(m.requestManagers, { names: names(managers) }) : tr(m.requestNobody)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} disabled={send.isPending}>
            {tr(m.requestCancel)}
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={send.isPending} disabledReason={managers.length ? undefined : tr(m.requestNobodyReason)}>
            {tr(m.requestSend)}
          </Button>
        </>
      }
    >
      <div className="lblc">
        <label htmlFor={`${id}-note`}>
          <T msg={m.requestWhy} values={{ k: (chunks) => <span className="k font-medium">{chunks}</span> }} />
        </label>
        <textarea id={`${id}-note`} className="in" value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/**
 * 403 for a member whose role can't do this: say what they can't do, who
 * can, offer "Request Editor access" and the read-only alternative.
 */
export function ForbiddenState({
  permission,
  title,
  readOnly,
}: {
  /** The missing permission, from the API's 403 problem. */
  permission: string
  /** What they can't do, as the heading: "You can look, but not change routing". */
  title: ReactNode
  /** The read-only way in: <ButtonLink …>View routing read-only</ButtonLink>. */
  readOnly?: ReactNode
}) {
  const { slug, me } = useWorkspace()
  const [asking, setAsking] = useState(false)
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/access-requests/suggestion', {
    params: { path: { workspace: slug }, query: { permission } },
  })
  const tr = useT()
  const s = q.data
  const role = me.workspace.roleName
  const sentence = s
    ? s.owners.length
      ? tr(m.stateForbiddenNeedsOwners, { workspace: me.workspace.name, role, action: s.action, needed: s.roleName, owners: s.owners.join(', ') })
      : tr(m.stateForbiddenNeeds, { workspace: me.workspace.name, role, action: s.action, needed: s.roleName })
    : tr(m.stateForbiddenDenied, { workspace: me.workspace.name, role, reason: deniedReason(permission) })
  return (
    <>
      <FullPageState
        icon="lockLarge"
        tone="warn"
        badge={tr(m.stateForbiddenBadge, { role })}
        title={title}
        actions={
          <>
            {s?.requestedAt ? null : (
              <Button variant="primary" onClick={() => setAsking(true)} disabledReason={q.isPending ? tr(m.stateChecking) : undefined}>
                {s ? tr(m.stateRequestRole, { role: s.roleName }) : tr(m.stateRequestAccess)}
              </Button>
            )}
            {readOnly}
          </>
        }
        footnote={
          s?.requestedAt ? tr(m.stateAsked, { when: askedWhen(s.requestedAt), names: names(s.managers) }) : tr(m.stateRequestsGo)
        }
      >
        {sentence}
      </FullPageState>
      {s ? <RequestAccessDialog open={asking} onOpenChange={setAsking} permission={permission} roleName={s.roleName} managers={s.managers} /> : null}
    </>
  )
}

/** The 404 for unknown paths inside a workspace, in the app shell. */
export function WorkspaceNotFound() {
  const { slug } = useWorkspace()
  const canAudit = useCan('audit:view')
  const pathname = useLocation({ select: (l) => l.pathname })
  const tr = useT()
  const path = pathname.replace(new RegExp(`^/w/${slug}`), '') || '/'
  return (
    <Page title={tr(m.stateNotFoundTitle)}>
      <NotFoundState
        path={path}
        primary={
          <ButtonLink variant="primary" to="/w/$workspace" params={{ workspace: slug }}>
            {tr(m.stateGoToOverview)}
          </ButtonLink>
        }
        secondary={
          canAudit ? (
            <ButtonLink to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
              {tr(m.stateSearchAudit)}
            </ButtonLink>
          ) : null
        }
      />
    </Page>
  )
}

/**
 * A screen failed to load: 403, 404 or 500 inside the shell, by status.
 * Used as the error component of workspace routes.
 */
export function WorkspaceRouteError({ error, reset }: ErrorComponentProps) {
  const router = useRouter()
  const { slug } = useWorkspace()
  const pathname = useLocation({ select: (l) => l.pathname })
  const tr = useT()
  const retry = () => {
    reset()
    void router.invalidate()
  }
  if (error instanceof ApiError && error.status === 403 && error.problem.permission) {
    return (
      <Page title={tr(m.stateNoAccessTitle)}>
        <ForbiddenState permission={error.problem.permission} title={tr(m.stateNoAccessHeading)} />
      </Page>
    )
  }
  if (error instanceof ApiError && error.status === 404) {
    return (
      <Page title={tr(m.stateNotFoundTitle)}>
        <NotFoundState path={pathname.replace(new RegExp(`^/w/${slug}`), '') || '/'} reference={problemReference(error)}>
          {error.problem.detail ? tr(m.stateNotFoundDetail, { detail: error.problem.detail }) : undefined}
        </NotFoundState>
      </Page>
    )
  }
  return (
    <Page title={tr(m.stateErrorTitle)}>
      <ServerErrorState
        reference={problemReference(error)}
        onRetry={retry}
        back={
          <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
            {tr(m.stateBackToOverview)}
          </ButtonLink>
        }
      />
    </Page>
  )
}
