import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout, useInstanceMeta } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { usDate } from '@/lib/format'
import { meQueryKey } from '@/lib/session'
import { AuthHeading, InlineAction, NotYou, OtpInput, orgName, PasskeyGlyph, SocialButtons, SocialMark } from './AuthParts'
import { passkeyErrorMessage, passkeySecondFactor, passkeySignIn } from './passkey'
import { RecoveryCodesPanel } from './RecoveryCodes'

export interface SignInSearch {
  next?: string
  reason?: 'expired'
  error?: 'idp' | 'no_account'
  provider?: string
  code?: string
  ref?: string
  email?: string
  reset?: 'done'
}

type TwoFactorMode = 'totp' | 'passkey' | 'recovery'

type Step =
  | { kind: 'email'; email?: string }
  | { kind: 'password'; email: string; error?: string }
  | { kind: 'sso'; email: string; sso: Schemas['SSOStart'] }
  | { kind: 'locked'; email: string; until: string; failures: number }
  | { kind: 'deactivated'; email: string; at?: string }
  | {
      kind: 'two_factor'
      email: string
      challenge: string
      methods: string[]
      mode: TwoFactorMode
      passkeyHint?: string
      recoveryLeft?: number
      recoveryTotal?: number
      error?: string
    }
  | { kind: 'last_code'; email: string; next: string; total: number }

/** Only same-origin relative paths are followed after sign-in. */
function safeNext(next?: string) {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : undefined
}

/** A5 §5.1–5.2: identifier-first sign-in, passkeys, two-factor and every error state. */
export function SignInPage({ search }: { search: SignInSearch }) {
  const [step, setStep] = useState<Step>(search.email && !search.error ? { kind: 'password', email: search.email } : { kind: 'email' })
  const navigate = useNavigate()
  const qc = useQueryClient()

  const finish = async (res: Schemas['SignInResult']) => {
    await qc.invalidateQueries({ queryKey: meQueryKey })
    const next = safeNext(search.next) ?? res.next ?? '/'
    if (res.recoveryCodesLeft === 0) {
      setStep({ kind: 'last_code', email: res.email, next, total: res.recoveryCodesTotal ?? 10 })
      return
    }
    void navigate({ to: next })
  }

  const afterPassword = async (res: Schemas['SignInResult']) => {
    if (res.status === 'two_factor') {
      const methods = res.methods ?? []
      const mode: TwoFactorMode = methods.includes('totp') ? 'totp' : methods.includes('passkey') ? 'passkey' : 'recovery'
      setStep({
        kind: 'two_factor',
        email: res.email,
        challenge: res.challenge!,
        methods,
        mode,
        passkeyHint: res.passkeyHint,
        recoveryLeft: res.recoveryCodesLeft,
        recoveryTotal: res.recoveryCodesTotal,
      })
    } else await finish(res)
  }

  /** Locked and deactivated accounts get their own screens; other problems stay on the step. */
  const special = (email: string, e: unknown) => {
    if (!(e instanceof ApiError)) return false
    const meta = (e.problem.meta ?? {}) as Record<string, unknown>
    switch (e.code) {
      case 'locked':
        setStep({ kind: 'locked', email, until: String(meta.lockedUntil), failures: Number(meta.failures ?? 0) })
        return true
      case 'account_deactivated':
        setStep({ kind: 'deactivated', email, at: meta.deactivatedAt as string | undefined })
        return true
    }
    return false
  }

  const back = () => setStep({ kind: 'email' })

  /** Runs the identifier-first step for an email and leaves the error screens. */
  const identify = async (email: string) => {
    const { data } = await http.POST('/api/v1/auth/identify', { body: { email } })
    if (data?.next === 'sso' && data.sso) setStep({ kind: 'sso', email: data.email, sso: data.sso })
    else setStep({ kind: 'password', email: data?.email ?? email })
    void navigate({ to: '/sign-in', search: safeNext(search.next) ? { next: search.next } : {} })
  }

  if (search.error === 'idp') return <IdpError provider={search.provider} code={search.code} reference={search.ref} email={search.email} />
  if (search.error === 'no_account') return <NoAccount email={search.email ?? ''} onRetry={identify} />

  switch (step.kind) {
    case 'email':
      return (
        <EmailStep
          initial={step.email}
          expired={search.reason === 'expired'}
          resetDone={search.reset === 'done'}
          onNext={setStep}
          onPasskey={finish}
          onProblem={(email, e) => special(email, e)}
        />
      )
    case 'password':
      return (
        <PasswordStep
          step={step}
          onBack={back}
          onPasskey={finish}
          onSubmit={async (password) => {
            try {
              const { data } = await http.POST('/api/v1/auth/password', { body: { email: step.email, password, remember: true } })
              if (data) await afterPassword(data)
            } catch (e) {
              if (e instanceof ApiError && e.code === 'sso_required') {
                const meta = (e.problem.meta ?? {}) as Record<string, string>
                setStep({
                  kind: 'sso',
                  email: step.email,
                  sso: { providerId: meta.providerId ?? '', name: meta.provider ?? 'SSO', preset: meta.preset ?? 'oidc', startUrl: `/api/v1/auth/sso/${meta.providerId}/start?login_hint=${encodeURIComponent(step.email)}` },
                })
                return
              }
              if (!special(step.email, e)) setStep({ ...step, error: errorMessage(e) })
            }
          }}
        />
      )
    case 'sso':
      return <SsoStep step={step} onPassword={() => setStep({ kind: 'password', email: step.email })} onBack={back} />
    case 'locked':
      return <LockedStep step={step} onBack={back} onPasskey={finish} />
    case 'deactivated':
      return <DeactivatedStep step={step} onBack={back} />
    case 'two_factor':
      return (
        <TwoFactorStep
          key={step.mode}
          step={step}
          onBack={back}
          onMode={(mode) => setStep({ ...step, mode, error: undefined })}
          onDone={finish}
          onExpired={(message) => setStep({ kind: 'password', email: step.email, error: message })}
          onError={(error) => setStep({ ...step, error })}
        />
      )
    case 'last_code':
      return <LastCodeStep step={step} />
  }
}

function PasskeyButton({ label, onDone, onProblem, primary, className }: { label: string; onDone: (r: Schemas['SignInResult']) => void; onProblem?: (e: unknown) => boolean; primary?: boolean; className?: string }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const errId = useId()
  return (
    <>
      <Button
        variant={primary ? 'primary' : 'default'}
        className={className}
        loading={busy}
        aria-describedby={error ? errId : undefined}
        onClick={async () => {
          setBusy(true)
          setError(undefined)
          try {
            onDone(await passkeySignIn(true))
          } catch (e) {
            if (!onProblem?.(e)) setError(passkeyErrorMessage(e, 'sign-in'))
          } finally {
            setBusy(false)
          }
        }}
      >
        {busy ? null : <PasskeyGlyph />}
        {label}
      </Button>
      {error ? (
        <span className="err" id={errId} role="alert">
          {error}
        </span>
      ) : null}
    </>
  )
}

function EmailStep({
  initial,
  expired,
  resetDone,
  onNext,
  onPasskey,
  onProblem,
}: {
  initial?: string
  expired: boolean
  resetDone: boolean
  onNext: (s: Step) => void
  onPasskey: (r: Schemas['SignInResult']) => void
  onProblem: (email: string, e: unknown) => boolean
}) {
  const { data: meta } = useInstanceMeta()
  const [email, setEmail] = useState(initial ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const errId = useId()
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setError('Enter your work email, like you@company.com.')
      return
    }
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/auth/identify', { body: { email: email.trim() } })
      if (!data) return
      if (data.next === 'sso' && data.sso) onNext({ kind: 'sso', email: data.email, sso: data.sso })
      else onNext({ kind: 'password', email: data.email })
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  const social = meta?.social ?? []
  return (
    <AuthLayout title="Sign in" labelledBy="h-signin">
      <AuthHeading id="h-signin" title="Sign in" sub="Enter your work email. If your organization uses single sign-on, you go straight to it." />
      {expired ? (
        <Note tone="info" label="Session ended">
          Your session timed out. Sign in again to pick up where you left off.
        </Note>
      ) : null}
      {resetDone ? (
        <Note tone="ok" label="Password changed">
          Sign in with your new password. Every other session was signed out.
        </Note>
      ) : null}
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <label className="lbl">
          Work email
          <input
            className={error ? 'in err' : 'in'}
            type="email"
            name="email"
            autoComplete="username"
            placeholder="you@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoFocus
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? errId : undefined}
          />
          {error ? (
            <span className="err" id={errId} role="alert">
              {error}
            </span>
          ) : null}
        </label>
        <Button type="submit" variant="primary" loading={busy}>
          Continue
        </Button>
      </form>
      {meta?.passkeys !== false ? <PasskeyButton label="Sign in with a passkey" onDone={onPasskey} onProblem={(e) => onProblem('', e)} /> : null}
      {social.length > 0 ? (
        <>
          <div className="sep">or continue with</div>
          <SocialButtons providers={social} href={(p) => `/api/v1/auth/social/${p}/start`} />
        </>
      ) : null}
      <span className="foot">
        <Link to="/reset-password">Forgot password?</Link> · No account? Ask an admin to invite you.
      </span>
    </AuthLayout>
  )
}

function PasswordStep({
  step,
  onBack,
  onSubmit,
  onPasskey,
}: {
  step: Extract<Step, { kind: 'password' }>
  onBack: () => void
  onSubmit: (pw: string) => Promise<void>
  onPasskey: (r: Schemas['SignInResult']) => void
}) {
  const { data: meta } = useInstanceMeta()
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [pkError, setPkError] = useState<string>()
  const errId = useId()
  const error = step.error ?? pkError
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setPkError(undefined)
    try {
      await onSubmit(password)
    } finally {
      setBusy(false)
    }
  }
  const usePasskey = async () => {
    setPkError(undefined)
    try {
      onPasskey(await passkeySignIn(true))
    } catch (e) {
      setPkError(passkeyErrorMessage(e, 'sign-in'))
    }
  }
  return (
    <AuthLayout title="Sign in" labelledBy="h-password">
      <AuthHeading id="h-password" title="Sign in" sub={<NotYou email={step.email} onBack={onBack} />} />
      <form className="flex flex-col gap-4" onSubmit={submit}>
        <input type="email" name="email" autoComplete="username" value={step.email} readOnly hidden />
        <label className="lbl">
          Password
          <input
            className={step.error ? 'in err' : 'in'}
            type="password"
            name="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoFocus
            aria-invalid={step.error ? true : undefined}
            aria-describedby={error ? errId : undefined}
          />
          {error ? (
            <span className="err" id={errId} role="alert">
              {error}
            </span>
          ) : null}
        </label>
        <Button type="submit" variant="primary" loading={busy}>
          Continue
        </Button>
      </form>
      <div className="links">
        <Link to="/reset-password" search={{ email: step.email }}>
          Forgot password?
        </Link>
        {meta?.passkeys !== false ? (
          <button type="button" onClick={usePasskey}>
            Use a passkey instead
          </button>
        ) : null}
      </div>
      {step.error ? <span className="foot">The message is the same whether or not an account exists for this email.</span> : null}
    </AuthLayout>
  )
}

function SsoStep({ step, onPassword, onBack }: { step: Extract<Step, { kind: 'sso' }>; onPassword: () => void; onBack: () => void }) {
  const domain = step.email.split('@')[1] ?? ''
  const google = step.sso.preset === 'google'
  return (
    <AuthLayout title={`Continue with ${step.sso.name}`} labelledBy="h-sso">
      <AuthHeading id="h-sso" title={`Continue with ${step.sso.name}`} sub={<NotYou email={step.email} onBack={onBack} />} />
      <Note tone="info" label="SSO">
        {domain} signs in with {step.sso.name}. Password sign-in is turned off for this domain.
      </Note>
      <a className="btn pri" href={step.sso.startUrl}>
        <SocialMark provider={google ? 'google' : 'sso'} />
        Continue with {step.sso.name}
      </a>
      <Note tone="neutral" label="Owner">
        Owners keep a break-glass sign-in for emergencies. <InlineAction onClick={onPassword}>Sign in with password</InlineAction> (password and two-factor required; the sign-in
        is logged).
      </Note>
      <span className="foot">You will return here after {google ? 'Google' : step.sso.name} confirms your identity.</span>
    </AuthLayout>
  )
}

function useCountdown(until: string) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const h = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(h)
  }, [])
  const left = Math.max(0, new Date(until).getTime() - now)
  const m = Math.floor(left / 60000)
  const sec = Math.floor((left % 60000) / 1000)
  return { left, label: `${m}:${String(sec).padStart(2, '0')}` }
}

function LockedStep({ step, onBack, onPasskey }: { step: Extract<Step, { kind: 'locked' }>; onBack: () => void; onPasskey: (r: Schemas['SignInResult']) => void }) {
  const { data: meta } = useInstanceMeta()
  const { left, label } = useCountdown(step.until)
  const at = new Date(step.until).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
  return (
    <AuthLayout title="Sign-in is locked" labelledBy="h-lock">
      <AuthHeading id="h-lock" title={left > 0 ? 'Sign-in is locked' : 'You can sign in again'} sub={`${step.email} · ${step.failures} failed attempts`} />
      {left > 0 ? (
        <div className="lock" role="timer" aria-live="off">
          <span className="t">{label}</span>
          <span className="k">remaining · sign-in reopens at {at}</span>
        </div>
      ) : null}
      <p className="body">
        Lockouts protect your account. If this was you, reset your password or use a passkey. If it wasn't, tell an instance admin: the attempts are in the audit log with their IP
        addresses.
      </p>
      <div className="tiles">
        <Link className="btn" to="/reset-password" search={{ email: step.email }}>
          Reset password
        </Link>
        {left > 0 ? meta?.passkeys !== false ? <PasskeyButton label="Use a passkey" onDone={onPasskey} /> : null : <Button onClick={onBack}>Sign in</Button>}
      </div>
    </AuthLayout>
  )
}

function DeactivatedStep({ step, onBack }: { step: Extract<Step, { kind: 'deactivated' }>; onBack: () => void }) {
  return (
    <AuthLayout title="This account was deactivated" labelledBy="h-deact">
      <AuthHeading id="h-deact" title="This account was deactivated" sub={step.email} />
      <Note tone="warn" label="Deactivated">
        Your account was deactivated{step.at ? ` on ${usDate(step.at)}` : ''} by an instance admin. Your history stays in the audit log; you can't sign in.
      </Note>
      <p className="body">Think this is a mistake? Ask your instance admin to reactivate the account. Nothing was deleted.</p>
      <Button onClick={onBack}>Back to sign in</Button>
    </AuthLayout>
  )
}

function TwoFactorStep({
  step,
  onBack,
  onMode,
  onDone,
  onExpired,
  onError,
}: {
  step: Extract<Step, { kind: 'two_factor' }>
  onBack: () => void
  onMode: (m: TwoFactorMode) => void
  onDone: (r: Schemas['SignInResult']) => Promise<void>
  onExpired: (message: string) => void
  onError: (message: string) => void
}) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [localError, setLocalError] = useState<string>()
  const errId = useId()
  const error = localError ?? step.error
  const has = (m: string) => step.methods.includes(m)

  const fail = (e: unknown) => {
    if (e instanceof ApiError && e.code === 'challenge_expired') onExpired(e.message)
    else onError(errorMessage(e))
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const totp = step.mode === 'totp'
    if (totp ? code.length < 6 : code.trim().length < 12) {
      setLocalError(totp ? 'Enter all 6 digits.' : 'Enter the whole recovery code, like b4xk-9q2m-7hdn.')
      return
    }
    setLocalError(undefined)
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/auth/two-factor', { body: { challenge: step.challenge, method: step.mode as 'totp' | 'recovery', code } })
      if (data) await onDone(data)
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  // Passkey as the second factor: the browser prompt opens right away.
  const started = useRef(false)
  const confirmPasskey = async () => {
    setBusy(true)
    setLocalError(undefined)
    try {
      await onDone(await passkeySecondFactor(step.challenge))
    } catch (err) {
      if (err instanceof ApiError) fail(err)
      else setLocalError(passkeyErrorMessage(err, 'sign-in'))
    } finally {
      setBusy(false)
    }
  }
  useEffect(() => {
    if (step.mode === 'passkey' && !started.current) {
      started.current = true
      void confirmPasskey()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step.mode])

  const links = (
    <div className="links">
      {step.mode !== 'passkey' && has('passkey') ? (
        <button type="button" onClick={() => onMode('passkey')}>
          Use a passkey instead
        </button>
      ) : null}
      {step.mode !== 'totp' && has('totp') ? (
        <button type="button" onClick={() => onMode('totp')}>
          Use my authenticator app
        </button>
      ) : null}
      {step.mode !== 'recovery' && has('recovery') ? (
        <button type="button" onClick={() => onMode('recovery')}>
          Use a recovery code
        </button>
      ) : null}
    </div>
  )

  if (step.mode === 'passkey') {
    return (
      <AuthLayout title="Confirm with your passkey" labelledBy="h-2fa-pk">
        <AuthHeading id="h-2fa-pk" title="Confirm with your passkey" sub={`Signing in as ${step.email}`} />
        <div className="lock">
          <PasskeyGlyph size={44} strokeWidth={1.4} color="#0F766E" />
          {step.passkeyHint ? <span className="pk-name">{step.passkeyHint}</span> : null}
          <span className="k">{busy ? 'Your browser is asking for your passkey' : 'Use the button to confirm with your passkey'}</span>
        </div>
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        {/* While the browser's own passkey prompt is open the button keeps its
            glyph (TwoFactorStates); a second press would only be refused. */}
        <Button variant="primary" aria-busy={busy || undefined} onClick={busy ? undefined : confirmPasskey}>
          <PasskeyGlyph />
          Use passkey
        </Button>
        {links}
      </AuthLayout>
    )
  }

  const totp = step.mode === 'totp'
  return (
    <AuthLayout title={totp ? 'Check your authenticator' : 'Use a recovery code'} labelledBy="h-2fa">
      <AuthHeading
        id="h-2fa"
        title={totp ? 'Check your authenticator' : 'Use a recovery code'}
        sub={totp && !error ? <NotYou email={`Signing in as ${step.email}`} onBack={onBack} /> : `Signing in as ${step.email}`}
      />
      <form className="flex flex-col gap-4" onSubmit={submit}>
        {totp ? (
          <div className="stack-8">
            <span className="otp-label" id="otp-label">
              {error ? '6-digit code' : '6-digit code from your authenticator app'}
            </span>
            <OtpInput value={code} onChange={setCode} invalid={Boolean(error)} describedBy={error ? errId : undefined} labelledBy="otp-label" />
            {error ? (
              <span className="err" id={errId} role="alert">
                {error}
              </span>
            ) : (
              <span className="k">You can paste the whole code. It changes every 30 seconds.</span>
            )}
          </div>
        ) : (
          <label className="lbl">
            Recovery code
            <input
              className={error ? 'in err mono' : 'in mono'}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="xxxx-xxxx-xxxx"
              autoComplete="one-time-code"
              spellCheck={false}
              autoFocus
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errId : undefined}
            />
            {error ? (
              <span className="err" id={errId} role="alert">
                {error}
              </span>
            ) : (
              <span className="k">
                Each code works once.
                {step.recoveryLeft !== undefined ? ` ${step.recoveryLeft} of ${step.recoveryTotal ?? 10} remain.` : ''}
              </span>
            )}
          </label>
        )}
        <Button type="submit" variant="primary" loading={busy}>
          Verify and sign in
        </Button>
      </form>
      {links}
      {totp && !error ? <span className="foot">Lost your device? An instance admin can reset two-factor for you; the reset is logged with a reason.</span> : null}
      {!totp ? <span className="foot">Lost the codes and your device? An instance admin can reset two-factor for you; the reset is logged with a reason.</span> : null}
    </AuthLayout>
  )
}

function LastCodeStep({ step }: { step: Extract<Step, { kind: 'last_code' }> }) {
  const navigate = useNavigate()
  const [codes, setCodes] = useState<string[]>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  return (
    <AuthLayout title="Signed in with your last recovery code" labelledBy="h-last">
      <AuthHeading id="h-last" title="Signed in with your last recovery code" sub={step.email} />
      {codes ? (
        <RecoveryCodesPanel codes={codes} email={step.email} onFinish={() => navigate({ to: step.next })} finishLabel="Continue" />
      ) : (
        <>
          <Note tone="warn" label={`0 of ${step.total} left`} role="alert">
            That was your last recovery code. Generate a new set now so you aren't locked out if you lose your authenticator.
          </Note>
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
          <div className="tiles">
            <Button
              variant="primary"
              loading={busy}
              onClick={async () => {
                setBusy(true)
                setError(undefined)
                try {
                  const { data } = await http.POST('/api/v1/me/recovery-codes')
                  setCodes(data?.codes)
                } catch (e) {
                  setError(errorMessage(e))
                } finally {
                  setBusy(false)
                }
              }}
            >
              Generate new codes
            </Button>
            <Button onClick={() => navigate({ to: step.next })}>Remind me later</Button>
          </div>
          <span className="foot">You can also do this any time from Account &amp; security.</span>
        </>
      )}
    </AuthLayout>
  )
}

function IdpError({ provider = 'Your identity provider', code, reference, email }: { provider?: string; code?: string; reference?: string; email?: string }) {
  const { data: meta } = useInstanceMeta()
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()
  const retry = meta?.sso.find((s) => s.name === provider)
  return (
    <AuthLayout title={`${provider} couldn't sign you in`} labelledBy="h-idp">
      <AuthHeading id="h-idp" title={`${provider} couldn't sign you in`} sub={email} />
      <Note tone="bad" label="IdP error">
        {provider} returned {code ? <span className="mono">{code}</span> : 'an error'} while exchanging your sign-in. Nothing was changed on your account.
      </Note>
      {reference ? (
        <div className="refbox">
          <span className="k">
            Reference ID <span className="mono text-ink">{reference}</span> · an admin can look it up in the audit log
          </span>
          <Button
            size="sm"
            className="ml-auto"
            onClick={async () => {
              await navigator.clipboard.writeText(reference)
              setCopied(true)
            }}
          >
            {copied ? 'Copied' : 'Copy'}
          </Button>
        </div>
      ) : null}
      <div className="tiles">
        {retry ? (
          <a className="btn pri" href={retry.startUrl + (email ? `?login_hint=${encodeURIComponent(email)}` : '')}>
            Try again
          </a>
        ) : (
          <Button variant="primary" onClick={() => navigate({ to: '/sign-in', search: email ? { email } : {} })}>
            Try again
          </Button>
        )}
        <Link className="btn" to="/sign-in" search={{}}>
          Sign in another way
        </Link>
      </div>
    </AuthLayout>
  )
}

function NoAccount({ email, onRetry }: { email: string; onRetry: (email: string) => Promise<void> }) {
  const { data: meta } = useInstanceMeta()
  const [value, setValue] = useState(email)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const errId = useId()
  const sso = meta?.sso.find((s) => s.domains.length > 0)
  const org = orgName(meta?.instanceName)
  return (
    <AuthLayout title={`No account for ${email}`} labelledBy="h-noacct">
      <AuthHeading id="h-noacct" title={`No account for ${email}`} sub="Automatic sign-up is off for this domain" />
      <Note tone="neutral" label="Invite only">
        This instance only creates accounts for invited people, or for allowed email domains through single sign-on. Ask an admin to invite you.
      </Note>
      <form
        className="flex flex-col gap-4"
        onSubmit={async (e) => {
          e.preventDefault()
          if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())) {
            setError('Enter your work email, like you@company.com.')
            return
          }
          setBusy(true)
          setError(undefined)
          try {
            await onRetry(value.trim())
          } catch (err) {
            setError(errorMessage(err))
          } finally {
            setBusy(false)
          }
        }}
        noValidate
      >
        <label className="lbl">
          Try another email
          <input
            className={error ? 'in err' : 'in'}
            type="email"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            autoComplete="email"
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? errId : undefined}
          />
          {error ? (
            <span className="err" id={errId} role="alert">
              {error}
            </span>
          ) : null}
        </label>
        <Button type="submit" variant="primary" loading={busy}>
          Continue
        </Button>
      </form>
      {sso ? (
        <span className="foot">
          {org ? `Work for ${org}? ` : ''}Sign in with your {sso.domains[0]} {sso.name} account instead.
        </span>
      ) : null}
    </AuthLayout>
  )
}
