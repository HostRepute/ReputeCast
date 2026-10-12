import { useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import { z } from 'zod'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { GitHubMark, GoogleMark, Icon, MicrosoftMark } from '@/components/icons/Icon'
import { AuthLayout, useInstanceMeta } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { usDate } from '@/lib/format'
import { meQueryKey } from '@/lib/session'

const search = z.object({
  next: z.string().optional(),
  reason: z.enum(['expired']).optional(),
  // Set by the SSO callback when the identity provider fails or no account matches.
  error: z.enum(['idp', 'no_account']).optional(),
  provider: z.string().optional(),
  code: z.string().optional(),
  ref: z.string().optional(),
  email: z.string().optional(),
})

export const Route = createFileRoute('/sign-in')({
  validateSearch: search,
  component: SignIn,
})

type Step =
  | { kind: 'email' }
  | { kind: 'password'; email: string; error?: string }
  | { kind: 'sso'; email: string; sso: Schemas['SSOStart'] }
  | { kind: 'locked'; email: string; until: string; failures: number }
  | { kind: 'deactivated'; email: string; at?: string }
  | { kind: 'two_factor'; email: string; challenge: string; methods: string[]; method: 'totp' | 'recovery'; error?: string }
  | { kind: 'last_code'; email: string; next: string }

/** Only same-origin relative paths are followed after sign-in. */
function safeNext(next?: string) {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : undefined
}

function SignIn() {
  const s = Route.useSearch()
  const [step, setStep] = useState<Step>(s.email ? { kind: 'password', email: s.email } : { kind: 'email' })
  const navigate = useNavigate()
  const qc = useQueryClient()

  const finish = async (res: Schemas['SignInResult']) => {
    await qc.invalidateQueries({ queryKey: meQueryKey })
    const next = safeNext(s.next) ?? res.next ?? '/'
    if (res.recoveryCodesLeft === 0) {
      setStep({ kind: 'last_code', email: res.email, next })
      return
    }
    void navigate({ to: next })
  }

  const onProblem = (email: string, e: unknown, fallback: Step) => {
    if (e instanceof ApiError) {
      const meta = (e.problem.meta ?? {}) as Record<string, unknown>
      switch (e.code) {
        case 'locked':
          setStep({ kind: 'locked', email, until: String(meta.lockedUntil), failures: Number(meta.failures ?? 0) })
          return
        case 'account_deactivated':
          setStep({ kind: 'deactivated', email, at: meta.deactivatedAt as string | undefined })
          return
      }
    }
    if (fallback.kind === 'password') setStep({ ...fallback, error: errorMessage(e) })
    else if (fallback.kind === 'two_factor') setStep({ ...fallback, error: errorMessage(e) })
  }

  if (s.error === 'idp') return <IdpError provider={s.provider} code={s.code} reference={s.ref} email={s.email} />
  if (s.error === 'no_account') return <NoAccount email={s.email ?? ''} onRetry={(email) => setStep({ kind: 'password', email })} />

  switch (step.kind) {
    case 'email':
      return <EmailStep expired={s.reason === 'expired'} onNext={setStep} />
    case 'password':
      return (
        <PasswordStep
          step={step}
          onBack={() => setStep({ kind: 'email' })}
          onSubmit={async (password, remember) => {
            try {
              const { data } = await http.POST('/api/v1/auth/password', { body: { email: step.email, password, remember } })
              if (!data) return
              if (data.status === 'two_factor') {
                const methods = data.methods ?? []
                setStep({ kind: 'two_factor', email: data.email, challenge: data.challenge!, methods, method: methods.includes('totp') ? 'totp' : 'recovery' })
              } else await finish(data)
            } catch (e) {
              if (e instanceof ApiError && e.code === 'sso_required') {
                const meta = (e.problem.meta ?? {}) as Record<string, string>
                setStep({ kind: 'sso', email: step.email, sso: { providerId: meta.providerId ?? '', name: meta.provider ?? 'SSO', preset: 'oidc', startUrl: `/api/v1/auth/sso/${meta.providerId}/start` } })
                return
              }
              onProblem(step.email, e, { ...step, kind: 'password' })
            }
          }}
        />
      )
    case 'sso':
      return <SsoStep step={step} onPassword={() => setStep({ kind: 'password', email: step.email })} onBack={() => setStep({ kind: 'email' })} />
    case 'locked':
      return <LockedStep step={step} onBack={() => setStep({ kind: 'email' })} />
    case 'deactivated':
      return <DeactivatedStep step={step} onBack={() => setStep({ kind: 'email' })} />
    case 'two_factor':
      return (
        <TwoFactorStep
          key={step.method}
          step={step}
          onBack={() => setStep({ kind: 'email' })}
          onMethod={(method) => setStep({ ...step, method, error: undefined })}
          onSubmit={async (code) => {
            try {
              const { data } = await http.POST('/api/v1/auth/two-factor', { body: { challenge: step.challenge, method: step.method, code } })
              if (data) await finish(data)
            } catch (e) {
              if (e instanceof ApiError && e.code === 'challenge_expired') {
                setStep({ kind: 'password', email: step.email, error: e.message })
                return
              }
              onProblem(step.email, e, step)
            }
          }}
        />
      )
    case 'last_code':
      return <LastCodeStep step={step} />
  }
}

function Heading({ id, title, sub }: { id: string; title: string; sub?: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <h1 id={id}>{title}</h1>
      {sub ? <span className="k">{sub}</span> : null}
    </div>
  )
}

function NotYou({ email, onBack }: { email: string; onBack: () => void }) {
  return (
    <>
      {email} ·{' '}
      <button type="button" className="prose-link cursor-pointer border-0 bg-transparent p-0 text-accent" onClick={onBack}>
        not you?
      </button>
    </>
  )
}

function EmailStep({ expired, onNext }: { expired: boolean; onNext: (s: Step) => void }) {
  const { data: meta } = useInstanceMeta()
  const [email, setEmail] = useState('')
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
      <Heading id="h-signin" title="Sign in" sub="Enter your work email. If your organization uses single sign-on, you go straight to it." />
      {expired ? (
        <Note tone="info" label="Session ended">
          Your session timed out. Sign in again to pick up where you left off.
        </Note>
      ) : null}
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <label className="lbl">
          Work email
          <input
            className="in"
            type="email"
            name="email"
            autoComplete="username webauthn"
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
      {social.length > 0 ? (
        <>
          <div className="sep">or continue with</div>
          <div className="soc">
            {social.includes('google') ? (
              <a className="btn" href="/api/v1/auth/social/google/start">
                <GoogleMark />
                Google
              </a>
            ) : null}
            {social.includes('microsoft') ? (
              <a className="btn" href="/api/v1/auth/social/microsoft/start">
                <MicrosoftMark />
                Microsoft
              </a>
            ) : null}
            {social.includes('github') ? (
              <a className="btn" href="/api/v1/auth/social/github/start">
                <GitHubMark />
                GitHub
              </a>
            ) : null}
          </div>
        </>
      ) : null}
      <span className="foot">
        <Link to="/reset-password">Forgot password?</Link> · No account? Ask an admin to invite you.
      </span>
    </AuthLayout>
  )
}

function PasswordStep({ step, onBack, onSubmit }: { step: Extract<Step, { kind: 'password' }>; onBack: () => void; onSubmit: (pw: string, remember: boolean) => Promise<void> }) {
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [busy, setBusy] = useState(false)
  const errId = useId()
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      await onSubmit(password, remember)
    } finally {
      setBusy(false)
    }
  }
  return (
    <AuthLayout title="Sign in" labelledBy="h-password">
      <Heading id="h-password" title="Sign in" sub={<NotYou email={step.email} onBack={onBack} />} />
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
            aria-describedby={step.error ? errId : undefined}
          />
          {step.error ? (
            <span className="err" id={errId} role="alert">
              {step.error}
            </span>
          ) : null}
        </label>
        <div className="chk">
          <input type="checkbox" id="remember" checked={remember} onChange={(e) => setRemember(e.target.checked)} />
          <label htmlFor="remember">Keep me signed in on this device</label>
        </div>
        <Button type="submit" variant="primary" loading={busy}>
          Continue
        </Button>
      </form>
      <div className="links">
        <Link to="/reset-password" search={{ email: step.email }}>
          Forgot password?
        </Link>
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
      <Heading id="h-sso" title={`Continue with ${step.sso.name}`} sub={<NotYou email={step.email} onBack={onBack} />} />
      <Note tone="info" label="SSO">
        {domain} signs in with {step.sso.name}. Password sign-in is turned off for this domain.
      </Note>
      <a className="btn pri" href={step.sso.startUrl}>
        {google ? <GoogleMark /> : <Icon name="key" />}
        Continue with {step.sso.name}
      </a>
      <Note tone="neutral" label="Owner">
        Owners keep a break-glass sign-in for emergencies.{' '}
        <button type="button" className="prose-link cursor-pointer border-0 bg-transparent p-0 text-accent" onClick={onPassword}>
          Sign in with password
        </button>{' '}
        (password and two-factor required; the sign-in is logged).
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

function LockedStep({ step, onBack }: { step: Extract<Step, { kind: 'locked' }>; onBack: () => void }) {
  const { left, label } = useCountdown(step.until)
  const at = new Date(step.until).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
  return (
    <AuthLayout title="Sign-in is locked" labelledBy="h-lock">
      <Heading id="h-lock" title={left > 0 ? 'Sign-in is locked' : 'You can sign in again'} sub={`${step.email} · ${step.failures} failed attempts`} />
      {left > 0 ? (
        <div className="lock" role="timer" aria-live="off">
          <span className="t">{label}</span>
          <span className="k">remaining · sign-in reopens at {at}</span>
        </div>
      ) : null}
      <p className="m-0 text-[13.5px] text-ink-nav">
        Lockouts protect your account. If this was you, reset your password or use a passkey. If it wasn't, tell an instance admin: the attempts are in the audit log with their IP
        addresses.
      </p>
      <Link className="btn" to="/reset-password" search={{ email: step.email }}>
        Reset password
      </Link>
      <Button onClick={onBack}>{left > 0 ? 'Back to sign in' : 'Sign in'}</Button>
    </AuthLayout>
  )
}

function DeactivatedStep({ step, onBack }: { step: Extract<Step, { kind: 'deactivated' }>; onBack: () => void }) {
  return (
    <AuthLayout title="This account was deactivated" labelledBy="h-deact">
      <Heading id="h-deact" title="This account was deactivated" sub={step.email} />
      <Note tone="warn" label="Deactivated">
        Your account was deactivated{step.at ? ` on ${usDate(step.at)}` : ''} by an instance admin. Your history stays in the audit log; you can't sign in.
      </Note>
      <p className="m-0 text-[13.5px] text-ink-nav">Think this is a mistake? Ask your instance admin to reactivate the account. Nothing was deleted.</p>
      <Button onClick={onBack}>Back to sign in</Button>
    </AuthLayout>
  )
}

function OtpInput({ value, onChange, invalid, describedBy }: { value: string; onChange: (v: string) => void; invalid?: boolean; describedBy?: string }) {
  const refs = useRef<(HTMLInputElement | null)[]>([])
  // Typing fast fires several events before React re-renders, so keep the
  // latest value in a ref instead of reading it from the render closure.
  const latest = useRef(value)
  useLayoutEffect(() => {
    latest.current = value
  }, [value])
  const emit = (v: string) => {
    latest.current = v
    onChange(v)
  }
  const digits = value.padEnd(6, ' ').slice(0, 6).split('')
  const set = (i: number, d: string) => {
    const next = latest.current.padEnd(6, ' ').slice(0, 6).split('')
    next[i] = d || ' '
    emit(next.join('').replace(/\s+$/, ''))
  }
  return (
    <div className="otp" role="group" aria-labelledby="otp-label">
      {digits.map((d, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el
          }}
          value={d.trim()}
          inputMode="numeric"
          autoComplete={i === 0 ? 'one-time-code' : 'off'}
          aria-label={`Digit ${i + 1}`}
          aria-invalid={invalid ? true : undefined}
          aria-describedby={describedBy}
          maxLength={6}
          autoFocus={i === 0}
          onFocus={(e) => e.currentTarget.select()}
          onPaste={(e) => {
            const text = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6)
            if (text) {
              e.preventDefault()
              emit(text)
              refs.current[Math.min(text.length, 5)]?.focus()
            }
          }}
          onKeyDown={(e) => {
            if (e.key === 'Backspace' && !e.currentTarget.value && i > 0) refs.current[i - 1]?.focus()
            if (e.key === 'ArrowLeft' && i > 0) refs.current[i - 1]?.focus()
            if (e.key === 'ArrowRight' && i < 5) refs.current[i + 1]?.focus()
          }}
          onChange={(e) => {
            const v = e.target.value.replace(/\D/g, '')
            if (v.length > 1) {
              // Autofill or typing over a filled box: spread the digits from here.
              const next = (latest.current.slice(0, i) + v).slice(0, 6)
              emit(next)
              refs.current[Math.min(next.length, 5)]?.focus()
              return
            }
            set(i, v)
            if (v && i < 5) refs.current[i + 1]?.focus()
          }}
        />
      ))}
    </div>
  )
}

function TwoFactorStep({
  step,
  onBack,
  onMethod,
  onSubmit,
}: {
  step: Extract<Step, { kind: 'two_factor' }>
  onBack: () => void
  onMethod: (m: 'totp' | 'recovery') => void
  onSubmit: (code: string) => Promise<void>
}) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [localError, setLocalError] = useState<string>()
  const errId = useId()
  const error = localError ?? step.error
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const totpMode = step.method === 'totp'
    if (totpMode ? code.length < 6 : code.trim().length < 12) {
      setLocalError(totpMode ? 'Enter all 6 digits.' : 'Enter the whole recovery code, like b4xk-9q2m-7hdn.')
      return
    }
    setLocalError(undefined)
    setBusy(true)
    try {
      await onSubmit(code)
    } finally {
      setBusy(false)
    }
  }
  const totp = step.method === 'totp'
  return (
    <AuthLayout title={totp ? 'Check your authenticator' : 'Use a recovery code'} labelledBy="h-2fa">
      <Heading id="h-2fa" title={totp ? 'Check your authenticator' : 'Use a recovery code'} sub={<NotYou email={`Signing in as ${step.email}`} onBack={onBack} />} />
      <form className="flex flex-col gap-4" onSubmit={submit}>
        {totp ? (
          <div className="flex flex-col gap-2">
            <span className="text-[13px] font-semibold text-ink-2" id="otp-label">
              6-digit code from your authenticator app
            </span>
            <OtpInput value={code} onChange={setCode} invalid={Boolean(error)} describedBy={error ? errId : undefined} />
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
              autoFocus
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? errId : undefined}
            />
            {error ? (
              <span className="err" id={errId} role="alert">
                {error}
              </span>
            ) : (
              <span className="k">Each code works once.</span>
            )}
          </label>
        )}
        <Button type="submit" variant="primary" loading={busy}>
          Verify and sign in
        </Button>
      </form>
      <div className="links">
        {totp && step.methods.includes('recovery') ? (
          <button type="button" onClick={() => onMethod('recovery')}>
            Use a recovery code
          </button>
        ) : null}
        {!totp && step.methods.includes('totp') ? (
          <button type="button" onClick={() => onMethod('totp')}>
            Use my authenticator app
          </button>
        ) : null}
      </div>
      <span className="foot">Lost your device? An instance admin can reset two-factor for you; the reset is logged with a reason.</span>
    </AuthLayout>
  )
}

function LastCodeStep({ step }: { step: Extract<Step, { kind: 'last_code' }> }) {
  const navigate = useNavigate()
  const [codes, setCodes] = useState<string[]>()
  const [busy, setBusy] = useState(false)
  return (
    <AuthLayout title="Signed in with your last recovery code" labelledBy="h-last">
      <Heading id="h-last" title="Signed in with your last recovery code" sub={step.email} />
      <Note tone="warn" label="0 of 10 left">
        That was your last recovery code. Generate a new set now so you aren't locked out if you lose your authenticator.
      </Note>
      {codes ? (
        <>
          <pre className="code grid grid-cols-2 gap-x-4 max-md:grid-cols-1">{codes.join('\n')}</pre>
          <Button variant="primary" onClick={() => navigate({ to: step.next })}>
            I saved them · continue
          </Button>
        </>
      ) : (
        <>
          <Button
            variant="primary"
            loading={busy}
            onClick={async () => {
              setBusy(true)
              try {
                const { data } = await http.POST('/api/v1/me/recovery-codes')
                setCodes(data?.codes)
              } finally {
                setBusy(false)
              }
            }}
          >
            Generate new codes
          </Button>
          <Button onClick={() => navigate({ to: step.next })}>Remind me later</Button>
          <span className="foot">You can also do this any time from Account &amp; security.</span>
        </>
      )}
    </AuthLayout>
  )
}

function IdpError({ provider = 'Your identity provider', code, reference, email }: { provider?: string; code?: string; reference?: string; email?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <AuthLayout title={`${provider} couldn't sign you in`} labelledBy="h-idp">
      <Heading id="h-idp" title={`${provider} couldn't sign you in`} sub={email} />
      <Note tone="bad" label="IdP error">
        {provider} returned {code ? <span className="mono">{code}</span> : 'an error'} while exchanging your sign-in. Nothing was changed on your account.
      </Note>
      {reference ? (
        <div className="flex items-center gap-2.5 rounded-lg border border-line px-3 py-2.5 text-[12.5px]">
          <span className="flex-1">
            Reference ID <b className="mono">{reference}</b> · an admin can look it up in the audit log
          </span>
          <Button
            size="sm"
            onClick={async () => {
              await navigator.clipboard.writeText(reference)
              setCopied(true)
            }}
          >
            {copied ? 'Copied' : 'Copy'}
          </Button>
        </div>
      ) : null}
      <Link className="btn pri" to="/sign-in">
        Try again
      </Link>
      <div className="links justify-center">
        <Link to="/sign-in">Sign in another way</Link>
      </div>
    </AuthLayout>
  )
}

function NoAccount({ email, onRetry }: { email: string; onRetry: (email: string) => void }) {
  const [value, setValue] = useState(email)
  return (
    <AuthLayout title={`No account for ${email}`} labelledBy="h-noacct">
      <Heading id="h-noacct" title={`No account for ${email}`} sub="Automatic sign-up is off for this domain" />
      <Note tone="neutral" label="Invite only">
        This instance only creates accounts for invited people, or for allowed email domains through single sign-on. Ask an admin to invite you.
      </Note>
      <form
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          onRetry(value)
        }}
      >
        <label className="lbl">
          Try another email
          <input className="in" type="email" value={value} onChange={(e) => setValue(e.target.value)} autoComplete="email" />
        </label>
        <Button type="submit" variant="primary">
          Continue
        </Button>
      </form>
    </AuthLayout>
  )
}
