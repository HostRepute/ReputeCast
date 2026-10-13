import { useQueryClient } from '@tanstack/react-query'
import { Link, useLocation, useRouter, type ErrorComponentProps } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { ago } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { Icon, LogoMark } from '../icons/Icon'
import { Badge, type Tone } from '../ui/Badge'
import { Button, ButtonLink } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Note } from '../ui/Note'
import { toast } from '../ui/Toast'
import { Page } from './Page'

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
  return (
    <div className="flex min-h-dvh flex-col items-center gap-6 px-4 py-14">
      <Link to="/" className="flex items-center gap-2.5 text-[16px] font-bold text-ink hover:text-ink" aria-label="ReputeCast home">
        <LogoMark size={28} />
        ReputeCast
      </Link>
      <main id="main" className="w-full max-w-[640px]">
        {children}
      </main>
    </div>
  )
}

export function NotFoundScreen() {
  useDocumentTitle('Not found · ReputeCast')
  const path = typeof window === 'undefined' ? '' : window.location.pathname
  return (
    <Frame>
      <FullPageState
        icon="searchLarge"
        badge="404"
        title={`There's nothing at ${path}`}
        actions={
          <Link to="/" className="btn pri">
            Go to overview
          </Link>
        }
      >
        The page may have been deleted, or the link was copied before it was saved.
      </FullPageState>
    </Frame>
  )
}

export function ErrorScreen({ error, reset }: ErrorComponentProps) {
  useDocumentTitle('Something broke · ReputeCast')
  const router = useRouter()
  const ref = error instanceof ApiError ? (error.problem as { instance?: string }).instance : undefined
  return (
    <Frame>
      <FullPageState
        icon="warningTriangle"
        tone="bad"
        badge={error instanceof ApiError ? String(error.status) : '500'}
        title="Something broke on our side"
        actions={
          <>
            <Button
              variant="primary"
              onClick={() => {
                reset()
                void router.invalidate()
              }}
            >
              Retry
            </Button>
            <Link to="/" className="btn">
              Back to overview
            </Link>
          </>
        }
        footnote={
          <>
            Self-hosted? Check <span className="mono">docker compose logs app</span>
            {ref ? <> for {ref}</> : null}.
          </>
        }
      >
        The page couldn't load. Retry now, or wait a minute; the error is already in the server log.
      </FullPageState>
    </Frame>
  )
}

// ------------------------------------------------------------------------
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
  return (
    <FullPageState
      icon="searchLarge"
      badge="404"
      title={`There's nothing at ${path}`}
      actions={
        primary || secondary ? (
          <>
            {primary}
            {secondary}
          </>
        ) : undefined
      }
      footnote={
        reference ? (
          <>
            Request id <span className="mono">{reference}</span> · include it if you contact support
          </>
        ) : undefined
      }
    >
      {children ?? 'The page may have been deleted, or the link was copied before it was saved.'}
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
  return (
    <FullPageState
      icon="warningTriangle"
      tone="bad"
      badge="500"
      title="Something broke on our side"
      actions={
        <>
          <Button variant="primary" onClick={onRetry}>
            Retry
          </Button>
          {back}
        </>
      }
      footnote={
        <>
          Self-hosted? Check <span className="mono">docker compose logs app</span>
          {reference ? <> for {reference}</> : null}.
        </>
      }
    >
      The page couldn&apos;t load. {savedNote ? `${savedNote} ` : null}Retry now, or wait a minute; the error is already in the server log
      {reference ? (
        <>
          {' '}
          with id <span className="mono">{reference}</span>
        </>
      ) : null}
      .
    </FullPageState>
  )
}

function names(list: string[]): string {
  if (list.length <= 1) return list[0] ?? ''
  return `${list.slice(0, -1).join(', ')} and ${list[list.length - 1]}`
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
  const submit = async () => {
    setError(undefined)
    try {
      const r = await send.mutateAsync({ params: { path: { workspace: slug } }, body: { permission, note: note.trim() || undefined } })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/access-requests/suggestion'] })
      toast({ kind: 'success', title: 'Request sent', detail: `${names(r.notified)} will see it` })
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
      eyebrow={<Badge tone="info">Request access</Badge>}
      title={`Ask for ${roleName} access`}
      description={
        managers.length
          ? `${names(managers)} can change roles. They'll see your note in their notifications; nothing changes until one of them does.`
          : 'Nobody in this workspace can change roles. Ask an instance admin.'
      }
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} disabled={send.isPending}>
            Cancel
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={send.isPending} disabledReason={managers.length ? undefined : 'Nobody here can change roles.'}>
            Send request
          </Button>
        </>
      }
    >
      <div className="lblc">
        <label htmlFor={`${id}-note`}>
          Why you need it <span className="k font-medium">· optional</span>
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
  const s = q.data
  const role = me.workspace.roleName
  const sentence = s
    ? `Your role in ${me.workspace.name} is ${role}. ${s.action} needs ${s.roleName} or above.${s.owners.length ? ` Owners: ${s.owners.join(', ')}.` : ''}`
    : `Your role in ${me.workspace.name} is ${role}. ${deniedReason(permission)}`
  return (
    <>
      <FullPageState
        icon="lockLarge"
        tone="warn"
        badge={`403 · ${role} role`}
        title={title}
        actions={
          <>
            {s?.requestedAt ? null : (
              <Button variant="primary" onClick={() => setAsking(true)} disabledReason={q.isPending ? 'Checking who can grant it…' : undefined}>
                Request {s?.roleName ?? 'access'}
                {s ? ' access' : ''}
              </Button>
            )}
            {readOnly}
          </>
        }
        footnote={
          s?.requestedAt
            ? `You asked ${ago(s.requestedAt)}; ${names(s.managers)} will see it. Ask them directly if it's urgent.`
            : 'Requests go to the owners with a note; they usually answer within the day.'
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
  const path = pathname.replace(new RegExp(`^/w/${slug}`), '') || '/'
  return (
    <Page title="Not found">
      <NotFoundState
        path={path}
        primary={
          <ButtonLink variant="primary" to="/w/$workspace" params={{ workspace: slug }}>
            Go to overview
          </ButtonLink>
        }
        secondary={
          canAudit ? (
            <ButtonLink to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
              Search the audit log
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
  const retry = () => {
    reset()
    void router.invalidate()
  }
  if (error instanceof ApiError && error.status === 403 && error.problem.permission) {
    return (
      <Page title="No access">
        <ForbiddenState permission={error.problem.permission} title="Your role can't open this page" />
      </Page>
    )
  }
  if (error instanceof ApiError && error.status === 404) {
    return (
      <Page title="Not found">
        <NotFoundState path={pathname.replace(new RegExp(`^/w/${slug}`), '') || '/'} reference={problemReference(error)}>
          {error.problem.detail ? `${error.problem.detail} It may have been deleted, or the link was copied before it was saved.` : undefined}
        </NotFoundState>
      </Page>
    )
  }
  return (
    <Page title="Something broke">
      <ServerErrorState
        reference={problemReference(error)}
        onRetry={retry}
        back={
          <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
            Back to overview
          </ButtonLink>
        }
      />
    </Page>
  )
}
