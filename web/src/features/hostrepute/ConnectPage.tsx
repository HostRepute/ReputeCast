import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ConnectCard } from './ConnectCard'
import { plainNote, useIntegrations, useRefreshConnection, useStartOAuth, type Connection } from './shared'
import './hostrepute.css'

export interface ConnectSearch {
  result?: 'connected' | 'error'
  reason?: string
  key?: boolean
}

/** Why a handoff came back without a connection. */
function reasonText(reason: string | undefined, conn: Connection): string {
  // A workspace that was already connected keeps its connection.
  const kept = conn.status === 'connected' ? `Your connection to team “${conn.teamName}” is unchanged` : 'Nothing was connected'
  switch (reason) {
    case 'declined':
      return `You declined the request on HostRepute. ${kept}.`
    case 'wrong_user':
      return 'This handoff was started by someone else. Start it again from your own account.'
    case 'forbidden':
      return `Connecting HostRepute needs the integrations:hostrepute permission. ${kept}.`
    case 'exchange_failed':
      return `HostRepute didn't accept the approval (it expired or was already used). ${kept}; try again.`
    case 'unreachable':
      return `Couldn't reach HostRepute to finish connecting. ${kept}; try again in a minute.`
    case 'hostrepute_error':
      return `HostRepute answered with an error. ${kept}; try again.`
    case 'save_failed':
      return `ReputeCast couldn't save the connection. ${kept}; try again.`
  }
  return `HostRepute sent you back without connecting. ${kept}; try again.`
}

/** A5 5.15 Connect HostRepute: before connecting, the OAuth handoff and the return, or an API key. */
export function ConnectPage({ search }: { search: ConnectSearch }) {
  const { slug } = useWorkspace()
  const q = useIntegrations()
  const [redirect, setRedirect] = useState<string | null>(null)
  const oauth = useStartOAuth(setRedirect)
  const keyInput = useRef<HTMLInputElement>(null)
  // The API key card shows next to the handoff until it's dismissed; a
  // workspace that is already connected only sees it when asked for.
  const [keyOpen, setKeyOpen] = useState<boolean | null>(search.key ? true : null)
  useEffect(() => {
    if (search.key) keyInput.current?.focus()
  }, [search.key, q.data])
  const conn = q.data?.hostRepute
  const showKey = keyOpen ?? conn?.status !== 'connected'
  const useKey = () => {
    setKeyOpen(true)
    window.setTimeout(() => {
      keyInput.current?.scrollIntoView({ block: 'center', behavior: 'smooth' })
      keyInput.current?.focus()
    }, 0)
  }
  return (
    <Page
      title="Connect HostRepute"
      contentClassName="hr-crumbs"
      badges={
        <nav aria-label="Breadcrumb" className="flex flex-wrap items-center gap-2 text-[13px]">
          <Link to="/w/$workspace/deliverability" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
            Deliverability
          </Link>
          <span className="k" aria-hidden="true">
            /
          </span>
          <span className="font-semibold" aria-current="page">
            Connect HostRepute
          </span>
        </nav>
      }
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the HostRepute connection" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : !conn ? (
        <div className="grid items-start gap-5 [grid-template-columns:minmax(0,3fr)_minmax(320px,2fr)] max-md:grid-cols-1" aria-busy="true">
          <span className="vh" role="status">
            Loading
          </span>
          <div className="card flex flex-col gap-3 p-4.5">
            <Skeleton w={220} h={16} />
            <Skeleton w="80%" />
            <Skeleton w="100%" h={120} />
          </div>
          <div className="card flex flex-col gap-3 p-4.5">
            <Skeleton w={180} h={16} />
            <Skeleton w="100%" h={36} />
          </div>
        </div>
      ) : (
        <div className="grid items-start gap-5 [grid-template-columns:minmax(0,3fr)_minmax(320px,2fr)] max-md:grid-cols-1">
          <ConnectCard conn={conn} onConnect={oauth.start} onUseKey={useKey} connecting={oauth.pending} />
          <div className="col gap-5">
            {redirect ? <RedirectCard url={redirect} /> : null}
            {search.result === 'connected' && !redirect ? <BackCard conn={conn} /> : null}
            {search.result === 'error' && !redirect ? (
              <ErrorCard conn={conn} reason={search.reason} onRetry={oauth.start} retrying={oauth.pending} />
            ) : null}
            {search.result !== 'connected' && showKey ? <KeyCard inputRef={keyInput} onCancel={() => setKeyOpen(false)} /> : null}
          </div>
        </div>
      )}
    </Page>
  )
}

/** The handoff: the browser is on its way to app.hostrepute.com. */
function RedirectCard({ url }: { url: string }) {
  let host = 'app.hostrepute.com'
  try {
    const u = new URL(url, window.location.href)
    if (u.origin !== window.location.origin) host = u.host
  } catch {
    // keep the default
  }
  return (
    <section className="card" aria-labelledby="h-hr-redirect">
      <div className="sec">
        <h2 className="h2" id="h-hr-redirect">
          Redirecting to HostRepute…
        </h2>
        <Badge tone="info">OAuth</Badge>
      </div>
      <div className="pad col items-center gap-3 px-4.5 py-7 text-center" role="status">
        <span className="dot h-3.5 w-3.5 text-accent" aria-hidden="true" />
        <span className="font-semibold">Approve the request on {host}</span>
        <span className="k max-w-[360px]">
          You're leaving ReputeCast to sign in to HostRepute and confirm the scopes above. You'll come straight back here. Nothing is connected until you approve.
        </span>
        <span className="k">
          Didn't get redirected? <a href={url}>Open HostRepute</a>
        </span>
      </div>
    </section>
  )
}

/** "Use an API key instead": for installs that can't use the browser handoff. */
function KeyCard({ inputRef, onCancel }: { inputRef: React.RefObject<HTMLInputElement | null>; onCancel: () => void }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshConnection()
  const reason = usePermissionReason('integrations:hostrepute')
  const [key, setKey] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  async function connect(e: React.FormEvent) {
    e.preventDefault()
    if (reason) return
    if (!key.trim()) {
      setError('Paste the API key from HostRepute → Settings → API.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/api-key', { params: { path: { workspace: slug } }, body: { apiKey: key.trim() } })
      setKey('')
      await refresh()
      await navigate({ to: '/w/$workspace/deliverability/connect', params: { workspace: slug }, search: { result: 'connected' } })
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : errorMessage(err)
      setError(msg)
      if (!(err instanceof ApiError && err.status === 422)) toast({ kind: 'error', title: "Couldn't connect HostRepute", detail: msg })
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="card" aria-labelledby="h-hr-key">
      <div className="sec">
        <h2 className="h2" id="h-hr-key">
          Use an API key instead
        </h2>
        <span className="k">For installs that can't use the browser handoff</span>
      </div>
      <form className="pad col gap-3" onSubmit={connect} noValidate>
        <label className="lbl">
          HostRepute API key
          <input
            ref={inputRef}
            className="in mono"
            type="password"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="hr_live_…"
            autoComplete="off"
            spellCheck={false}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? 'hr-key-err' : undefined}
          />
          {error ? (
            <span className="err" id="hr-key-err" role="alert">
              {error}
            </span>
          ) : null}
        </label>
        <Note tone="neutral" className={cn(plainNote, 'text-[12.5px]')} role="note">
          Create a key in HostRepute → Settings → API with the monitor:read, alerts:read and assets:write scopes. The key is write-only here: it can be replaced but never read
          back.
        </Note>
        <div className="row justify-end gap-2">
          <Button
            onClick={() => {
              setKey('')
              setError(null)
              onCancel()
            }}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={busy} disabledReason={reason}>
            Connect
          </Button>
        </div>
      </form>
    </section>
  )
}

/** Back from HostRepute: connected, what is monitored and what could be. */
function BackCard({ conn }: { conn: Connection }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshConnection()
  const reason = usePermissionReason('integrations:hostrepute')
  const [adding, setAdding] = useState<string>()
  const [limit, setLimit] = useState<string | null>(null)
  const connected = conn.status === 'connected'
  useEffect(() => {
    // Once the first sync is done (it may finish before this page loads),
    // refresh once more so the sidebar and Deliverability show it.
    if (!conn.syncing) {
      void refresh()
      return
    }
    const h = window.setInterval(() => void refresh(), 2000)
    return () => window.clearInterval(h)
  }, [conn.syncing, refresh])
  const monitored = conn.assets.filter((a) => a.monitored)
  const suggested = conn.assets.filter((a) => !a.monitored)
  const scopes = conn.scopes.filter((s) => s.granted).map((s) => s.key)
  async function add(asset: string) {
    setAdding(asset)
    setLimit(null)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/assets', { params: { path: { workspace: slug } }, body: { asset } })
      await refresh()
      toast({ title: `${asset} added to monitoring`, detail: 'HostRepute checks it within the hour' })
    } catch (e) {
      if (e instanceof ApiError && e.code === 'monitor_limit_reached') setLimit(e.message)
      else toast({ kind: 'error', title: `Couldn't add ${asset}`, detail: errorMessage(e) })
    } finally {
      setAdding(undefined)
    }
  }
  if (!connected) return null
  return (
    <section className="card" aria-labelledby="h-hr-back">
      <div className="sec">
        <h2 className="h2" id="h-hr-back">
          Back from HostRepute
        </h2>
        <Badge tone="ok">Success</Badge>
      </div>
      <div className="pad col gap-2.5">
        <Note tone="ok" label="Connected">
          Team “{conn.teamName}” · scopes {scopes.join(', ')} · {conn.syncing ? 'first sync running (about 1 minute)' : 'first sync done'}
        </Note>
        <KeyValues
          className="text-[12.5px]"
          items={[
            ['Monitored now', monitored.length ? monitored.map((a) => a.asset).join(', ') : 'nothing yet: HostRepute adds them on the first sync'],
            ...(suggested.length
              ? ([
                  [
                    'Suggested',
                    <>
                      {suggested[0]!.asset} is not monitored yet ·{' '}
                      <Button size="sm" onClick={() => add(suggested[0]!.asset)} loading={adding === suggested[0]!.asset} disabledReason={reason}>
                        Add to monitoring
                      </Button>
                    </>,
                  ],
                ] as [string, React.ReactNode][])
              : []),
          ]}
        />
        {limit ? <Note tone="neutral" className={plainNote}>{limit} ReputeCast stays fully usable.</Note> : null}
        <ButtonLink size="sm" variant="primary" className="self-start" to="/w/$workspace/settings/integrations" params={{ workspace: slug }}>
          Finish
        </ButtonLink>
      </div>
    </section>
  )
}

function ErrorCard({ conn, reason, onRetry, retrying }: { conn: Connection; reason?: string; onRetry: () => void; retrying: boolean }) {
  const can = !usePermissionReason('integrations:hostrepute')
  return (
    <section className="card" aria-labelledby="h-hr-back">
      <div className="sec">
        <h2 className="h2" id="h-hr-back">
          Back from HostRepute
        </h2>
        {conn.status === 'connected' ? (
          <Badge tone="ok">Still connected</Badge>
        ) : conn.status === 'revoked' || conn.status === 'expired' ? (
          <Badge tone="warn">Needs reconnecting</Badge>
        ) : (
          <Badge tone="bad">Not connected</Badge>
        )}
      </div>
      <div className="pad col gap-2.5">
        <Note tone="bad">{reasonText(reason, conn)}</Note>
        {can ? (
          <Button size="sm" variant="primary" className="self-start" onClick={onRetry} loading={retrying}>
            Try again
          </Button>
        ) : null}
      </div>
    </section>
  )
}
