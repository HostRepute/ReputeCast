import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ago, usDate } from '@/lib/format'

type Session = Schemas['AccountSession']
const sessionsKey = ['get', '/api/v1/me/sessions'] as const

function lastActive(s: Session) {
  if (s.current) return 'now'
  const days = (Date.now() - new Date(s.lastActiveAt).getTime()) / 86_400_000
  return days >= 1 ? usDate(s.lastActiveAt) : ago(s.lastActiveAt)
}

function where(s: Session) {
  if (!s.ip) return <span className="k">Unknown</span>
  return (
    <>
      <span className="mono">{s.ip}</span>
      {s.location ? ` · ${s.location} (approx.)` : null}
    </>
  )
}

/** Active sessions: device, browser, IP and approximate location; sign out one or all others. */
export function SessionsCard({ policy }: { policy: Schemas['AccountSessionPolicy'] }) {
  const qc = useQueryClient()
  const sessions = $api.useQuery('get', '/api/v1/me/sessions')
  const [busy, setBusy] = useState<string>()
  const [confirmAll, setConfirmAll] = useState(false)
  const items = sessions.data?.items ?? []
  const others = items.filter((s) => !s.current)

  const signOut = async (s: Session) => {
    setBusy(s.id)
    try {
      await http.DELETE('/api/v1/me/sessions/{id}', { params: { path: { id: s.id } } })
      await qc.invalidateQueries({ queryKey: sessionsKey })
      toast({ kind: 'deleted', title: 'Session signed out', detail: `${s.device} · ${s.browser}` })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't sign it out", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }

  return (
    <section className="card" style={{ gridColumn: '1 / -1' }} aria-labelledby="h-sessions">
      <CardSection
        id="h-sessions"
        title="Active sessions"
        caption={`Sessions last ${policy.lifetime} and end after ${policy.idle} idle (instance policy)`}
        actions={
          <Button size="sm" disabledReason={sessions.data && others.length === 0 ? 'No other sessions are signed in' : undefined} onClick={() => setConfirmAll(true)}>
            Sign out all other sessions
          </Button>
        }
      />
      {sessions.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load your sessions" actions={<Button onClick={() => sessions.refetch()}>Retry</Button>}>
            {errorMessage(sessions.error)} Your sessions weren't changed.
          </ErrorPanel>
        </div>
      ) : sessions.isPending ? (
        <div className="pad col" style={{ gap: 8 }} aria-busy="true" aria-label="Loading sessions">
          <div className="h-3.5 w-[60%] rounded-[6px] bg-line-2" />
          <div className="h-3.5 w-[80%] rounded-[6px] bg-line-2" />
          <div className="h-3.5 w-[45%] rounded-[6px] bg-line-2" />
          <span className="k" role="status">
            Fetching your sessions…
          </span>
        </div>
      ) : (
        <Table density="compact" className="[&_td]:whitespace-normal" caption="Active sessions">
          <thead>
            <tr>
              <Th>Device</Th>
              <Th>Browser</Th>
              <Th>IP · location</Th>
              <Th>Last active</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {items.map((s) => (
              <tr key={s.id}>
                <Td>
                  <div className="col" style={{ gap: 0 }}>
                    <span className="font-semibold">
                      {s.device}{' '}
                      {s.current ? (
                        <Badge tone="ok" className="font-semibold">
                          This session
                        </Badge>
                      ) : null}
                    </span>
                    {s.os ? <span className="k">{s.os}</span> : null}
                  </div>
                </Td>
                <Td>{s.browser || '–'}</Td>
                <Td>{where(s)}</Td>
                <Td>{lastActive(s)}</Td>
                <Td>
                  {s.current ? (
                    <span className="k" aria-label="This session">
                      –
                    </span>
                  ) : (
                    <Button size="sm" loading={busy === s.id} onClick={() => signOut(s)} aria-label={`Sign out ${s.device}, ${s.browser}`}>
                      Sign out
                    </Button>
                  )}
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
      )}
      <SignOutOthersDialog open={confirmAll} others={others} onClose={() => setConfirmAll(false)} onDone={() => qc.invalidateQueries({ queryKey: sessionsKey })} />
    </section>
  )
}

function SignOutOthersDialog({ open, others, onClose, onDone }: { open: boolean; others: Session[]; onClose: () => void; onDone: () => Promise<void> }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const names = others.map((s) => `${s.device} (${s.browser || 'unknown browser'})`)
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title="Sign out all other sessions?"
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button
            variant="primary"
            loading={busy}
            onClick={async () => {
              setBusy(true)
              setError(undefined)
              try {
                const { data } = await http.POST('/api/v1/me/sessions/revoke-others')
                await onDone()
                toast({ kind: 'success', title: `${data?.revoked ?? 0} ${data?.revoked === 1 ? 'session' : 'sessions'} signed out`, detail: 'You stay signed in here.' })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            Sign out others
          </Button>
        </>
      }
    >
      <p className="text-ink-2">
        {others.length === 1 ? `${names[0]} is signed out now.` : `${others.length} sessions end now${names.length ? `: ${names.join(', ')}` : ''}.`} You stay signed in here.
      </p>
      {error ? (
        <Note tone="bad" label="Not signed out">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
