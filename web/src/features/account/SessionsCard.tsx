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
import { formatList, t as tr, useT } from '@/i18n'
import { ago, usDate } from '@/lib/format'
import { m } from './messages'

type Session = Schemas['AccountSession']
const sessionsKey = ['get', '/api/v1/me/sessions'] as const

function lastActive(s: Session) {
  if (s.current) return tr(m.activeNow)
  const days = (Date.now() - new Date(s.lastActiveAt).getTime()) / 86_400_000
  return days >= 1 ? usDate(s.lastActiveAt) : ago(s.lastActiveAt)
}

function where(s: Session) {
  if (!s.ip) return <span className="k">{tr(m.unknown)}</span>
  return (
    <>
      <span className="mono">{s.ip}</span>
      {s.location ? ` · ${tr(m.approximate, { location: s.location })}` : null}
    </>
  )
}

/** Active sessions: device, browser, IP and approximate location; sign out one or all others. */
export function SessionsCard({ policy }: { policy: Schemas['AccountSessionPolicy'] }) {
  const t = useT()
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
      toast({ kind: 'deleted', title: t(m.sessionSignedOut), detail: `${s.device} · ${s.browser}` })
    } catch (e) {
      toast({ kind: 'error', title: t(m.signOutFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }

  return (
    <section className="card" style={{ gridColumn: '1 / -1' }} aria-labelledby="h-sessions">
      <CardSection
        id="h-sessions"
        title={t(m.activeSessions)}
        caption={t(m.sessionsCaption, { lifetime: policy.lifetime, idle: policy.idle })}
        actions={
          <Button size="sm" disabledReason={sessions.data && others.length === 0 ? t(m.noOtherSessions) : undefined} onClick={() => setConfirmAll(true)}>
            {t(m.signOutAllOthers)}
          </Button>
        }
      />
      {sessions.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.sessionsLoadFailed)} actions={<Button onClick={() => sessions.refetch()}>{t(m.retry)}</Button>}>
            {errorMessage(sessions.error)} {t(m.sessionsNotChanged)}
          </ErrorPanel>
        </div>
      ) : sessions.isPending ? (
        <div className="pad col" style={{ gap: 8 }} aria-busy="true" aria-label={t(m.loadingSessions)}>
          <div className="h-3.5 w-[60%] rounded-[6px] bg-line-2" />
          <div className="h-3.5 w-[80%] rounded-[6px] bg-line-2" />
          <div className="h-3.5 w-[45%] rounded-[6px] bg-line-2" />
          <span className="k" role="status">
            {t(m.fetchingSessions)}
          </span>
        </div>
      ) : (
        <Table density="compact" className="[&_td]:whitespace-normal" caption={t(m.activeSessions)}>
          <thead>
            <tr>
              <Th>{t(m.device)}</Th>
              <Th>{t(m.browser)}</Th>
              <Th>{t(m.ipLocation)}</Th>
              <Th>{t(m.lastActive)}</Th>
              <Th>
                <span className="vh">{t(m.actions)}</span>
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
                          {t(m.thisSession)}
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
                    <span className="k" aria-label={t(m.thisSession)}>
                      –
                    </span>
                  ) : (
                    <Button size="sm" loading={busy === s.id} onClick={() => signOut(s)} aria-label={t(m.signOutNamed, { device: s.device, browser: s.browser })}>
                      {t(m.signOut)}
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
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const names = formatList(others.map((s) => `${s.device} (${s.browser || t(m.unknownBrowser)})`))
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title={t(m.signOutOthersTitle)}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            loading={busy}
            onClick={async () => {
              setBusy(true)
              setError(undefined)
              try {
                const { data } = await http.POST('/api/v1/me/sessions/revoke-others')
                await onDone()
                toast({ kind: 'success', title: t(m.sessionsSignedOut, { count: data?.revoked ?? 0 }), detail: t(m.staySignedIn) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            {t(m.signOutOthers)}
          </Button>
        </>
      }
    >
      <p className="text-ink-2">{t(m.signOutOthersBody, { count: others.length, names })}</p>
      {error ? (
        <Note tone="bad" label={t(m.notSignedOut)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
