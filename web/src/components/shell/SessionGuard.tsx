import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { t, useT } from '@/i18n'
import { clock as clockTime, spanWords } from '@/lib/format'
import {
  extendSession,
  getSessionState,
  loadSessionTiming,
  markSessionEnded,
  registerSessionGuard,
  sessionRenewed,
  useSessionClock,
  useSessionState,
  type EndReason,
} from '@/lib/sessionClock'
import { meQueryKey, type Me } from '@/lib/session'
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { toast } from '../ui/Toast'
import { m } from './messages'
import { SessionEndingNotice } from './SystemNotices'

type SSOStart = Schemas['SSOStart']

async function signOutEverywhere(navigate: ReturnType<typeof useNavigate>, clear: () => void) {
  try {
    await http.POST('/api/v1/auth/sign-out')
  } catch {
    // the session may already be gone
  }
  clear()
  void navigate({ to: '/sign-in' })
}

/**
 * Keeps signed-in screens safe from session timeouts (SystemStates board):
 * two minutes before the session ends it offers "Extend 12 hours"; when the
 * session has ended, it asks for the password in a dialog over the page, so
 * unsaved work stays where it is.
 */
export function SessionGuard() {
  const clock = useSessionClock()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [extending, setExtending] = useState(false)
  const work = useSessionState().work

  useEffect(() => {
    const off = registerSessionGuard()
    void loadSessionTiming()
    return off
  }, [])

  // The local clock ran out: the server has ended the session too.
  useEffect(() => {
    if (clock.phase === 'ended' && !getSessionState().ended) markSessionEnded(clock.endsBy ?? 'idle')
  }, [clock.phase, clock.endsBy])

  const extend = async () => {
    setExtending(true)
    try {
      const timing = await extendSession()
      toast({ kind: 'success', title: t(m.sessionExtended), detail: t(m.sessionExtendedDetail, { time: clockTime(timing.endsAt) }) })
    } catch (e) {
      if (!(e instanceof ApiError && e.status === 401)) toast({ kind: 'error', title: t(m.sessionExtendFailed), detail: errorMessage(e), retry: () => void extend() })
    } finally {
      setExtending(false)
    }
  }

  return (
    <>
      {clock.phase === 'warning' ? (
        <SessionEndingNotice
          clock={clock}
          extending={extending}
          onExtend={() => void extend()}
          onSignOut={() => void signOutEverywhere(navigate, () => qc.clear())}
          workNote={work?.whileEnding}
        />
      ) : null}
      <SignInAgainDialog open={clock.phase === 'ended'} reason={clock.ended ?? 'idle'} workNote={work?.afterEnded} />
    </>
  )
}

function endedSentence(reason: EndReason): string {
  const timing = getSessionState().timing
  if (reason === 'idle' && timing?.idleTimeoutSeconds) return t(m.sessionEndedIdle, { span: spanWords(timing.idleTimeoutSeconds * 1000) })
  if (reason === 'lifetime' && timing) {
    const days = Math.round((Date.parse(timing.expiresAt) - Date.parse(timing.signedInAt)) / 86_400_000)
    return t(m.sessionEndedLifetime, { days })
  }
  return t(m.sessionEndedOther)
}

/** "Sign in again to keep working": the page stays mounted behind it. */
export function SignInAgainDialog({ open, reason, workNote }: { open: boolean; reason: EndReason; workNote?: string }) {
  const id = useId()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const email = qc.getQueryData<Me | null>(meQueryKey)?.user.email
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [challenge, setChallenge] = useState<string>()
  const [method, setMethod] = useState<'totp' | 'recovery'>('totp')
  const [sso, setSso] = useState<SSOStart | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const tr = useT()

  useEffect(() => {
    if (!open || !email) return
    let live = true
    http
      .POST('/api/v1/auth/identify', { body: { email } })
      .then(({ data }) => live && setSso(data?.next === 'sso' ? (data.sso ?? null) : null))
      .catch(() => {})
    return () => {
      live = false
    }
  }, [open, email])

  const reset = () => {
    setPassword('')
    setCode('')
    setChallenge(undefined)
    setMethod('totp')
    setError(undefined)
  }

  const done = async () => {
    reset()
    await sessionRenewed()
    await qc.invalidateQueries()
    toast({ kind: 'success', title: t(m.sessionSignedInAgain), detail: t(m.sessionSignedInAgainDetail) })
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!email || busy) return
    setBusy(true)
    setError(undefined)
    try {
      if (!challenge) {
        const { data } = await http.POST('/api/v1/auth/password', { body: { email, password, remember: false } })
        if (data?.status === 'two_factor' && data.challenge) {
          setChallenge(data.challenge)
          setPassword('')
        } else {
          await done()
        }
      } else {
        await http.POST('/api/v1/auth/two-factor', { body: { challenge, method, code: code.replace(/\s+/g, '') } })
        await done()
      }
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const fieldId = `${id}-field`
  const errId = `${id}-err`
  return (
    <Dialog
      open={open}
      onOpenChange={() => {}}
      layout="session"
      dismissible={false}
      eyebrow={<Badge tone="info">{tr(m.sessionEndedBadge)}</Badge>}
      title={tr(m.sessionSignInAgainTitle)}
      description={`${endedSentence(reason)} ${workNote ?? tr(m.sessionWorkKept)}`}
    >
      <form className="flex flex-col gap-3" onSubmit={(e) => void submit(e)} noValidate>
        {!challenge ? (
          <div className="lblc">
            <label htmlFor={fieldId}>{email ? tr(m.sessionPasswordFor, { email }) : tr(m.sessionPasswordForAccount)}</label>
            <input
              id={fieldId}
              className="in"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errId : undefined}
              required
            />
          </div>
        ) : (
          <div className="lblc">
            <label htmlFor={fieldId}>{method === 'totp' ? tr(m.sessionTotpLabel) : tr(m.sessionRecoveryLabel)}</label>
            <input
              id={fieldId}
              className="in"
              inputMode={method === 'totp' ? 'numeric' : 'text'}
              autoComplete="one-time-code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errId : undefined}
              autoFocus
              required
            />
            <button type="button" className="self-start border-0 bg-transparent p-0 text-[12.5px] font-medium text-accent underline" onClick={() => setMethod(method === 'totp' ? 'recovery' : 'totp')}>
              {method === 'totp' ? tr(m.sessionUseRecovery) : tr(m.sessionUseTotp)}
            </button>
          </div>
        )}
        {error ? (
          <span className="err" id={errId} role="alert">
            {error}
          </span>
        ) : null}
        <div className="flex flex-wrap gap-2">
          <Button type="submit" variant="primary" loading={busy} disabledReason={!email ? tr(m.sessionReload) : undefined}>
            {challenge ? tr(m.sessionVerify) : tr(m.sessionSignIn)}
          </Button>
          {sso ? (
            <a className="btn" href={sso.startUrl}>
              {tr(m.sessionUseSso)}
            </a>
          ) : null}
          <Button className="ml-auto" onClick={() => void signOutEverywhere(navigate, () => qc.clear())}>
            {tr(m.sessionSignOut)}
          </Button>
        </div>
      </form>
      <span className="k">{tr(m.sessionWarningNote)}</span>
    </Dialog>
  )
}
