import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout, useInstanceMeta } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { clock, usDate } from '@/lib/format'
import { meQueryKey } from '@/lib/session'
import { AuthHeading, InlineAction, NotYou, OtpInput, orgName, PasskeyGlyph, SocialButtons, SocialMark } from './AuthParts'
import { passkeyErrorMessage, passkeySecondFactor, passkeySignIn } from './passkey'
import { RecoveryCodesPanel } from './RecoveryCodes'
import { m } from './signin.messages'

export interface SignInSearch {
  next?: string
  reason?: 'expired'
  error?: 'idp' | 'no_account' | 'deactivated'
  provider?: string
  code?: string
  ref?: string
  email?: string
  reset?: 'done'
  /** An SSO or social sign-in that still needs the person's own second factor. */
  challenge?: string
  methods?: string
  hint?: string
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

/**
 * Only same-origin relative paths are followed after sign-in. Browsers drop
 * tabs and newlines and read a backslash as a slash, so none of them may
 * appear ("/<tab>/host" and "/\host" both lead to "//host"), as on the server.
 */
function safeNext(next?: string) {
  // eslint-disable-next-line no-control-regex
  return next && next.startsWith('/') && !next.startsWith('//') && !/[\\\u0000-\u001f\u007f]/.test(next) ? next : undefined
}

/** A start link of an SSO or social provider that returns to `next`. */
function withNext(url: string, next?: string) {
  const n = safeNext(next)
  return n ? `${url}${url.includes('?') ? '&' : '?'}next=${encodeURIComponent(n)}` : url
}

/** The first step: the callbacks of SSO and social sign-in come back with a
 * two-factor challenge or a deactivated account. */
function initialStep(search: SignInSearch): Step {
  if (search.challenge && search.email) {
    const methods = (search.methods ?? '').split(',').filter(Boolean)
    const mode: TwoFactorMode = methods.includes('totp') ? 'totp' : methods.includes('passkey') ? 'passkey' : 'recovery'
    return { kind: 'two_factor', email: search.email, challenge: search.challenge, methods, mode, passkeyHint: search.hint }
  }
  if (search.error === 'deactivated' && search.email) return { kind: 'deactivated', email: search.email }
  if (search.email && !search.error) return { kind: 'password', email: search.email }
  return { kind: 'email' }
}

/** A5 §5.1–5.2: identifier-first sign-in, passkeys, two-factor and every error state. */
export function SignInPage({ search }: { search: SignInSearch }) {
  const [step, setStep] = useState<Step>(() => initialStep(search))
  const navigate = useNavigate()
  const qc = useQueryClient()
  // The challenge is single use and short-lived; keep it out of the history.
  useEffect(() => {
    if (search.challenge || search.error === 'deactivated') void navigate({ to: '/sign-in', search: safeNext(search.next) ? { next: search.next } : {}, replace: true })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

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
          next={search.next}
          onNext={setStep}
          onPasskey={finish}
          onProblem={(email, e) => special(email, e)}
        />
      )
    case 'password':
      return (
        <PasswordStep
          step={step}
          next={search.next}
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
      return <SsoStep step={step} next={search.next} onPassword={() => setStep({ kind: 'password', email: step.email })} onBack={back} />
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
  next,
  onNext,
  onPasskey,
  onProblem,
}: {
  initial?: string
  expired: boolean
  resetDone: boolean
  next?: string
  onNext: (s: Step) => void
  onPasskey: (r: Schemas['SignInResult']) => void
  onProblem: (email: string, e: unknown) => boolean
}) {
  const t = useT()
  const { data: meta } = useInstanceMeta()
  const [email, setEmail] = useState(initial ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const errId = useId()
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setError(t(m.emailInvalid))
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
    <AuthLayout title={t(m.title)} labelledBy="h-signin">
      <AuthHeading id="h-signin" title={t(m.title)} sub={t(m.emailSub)} />
      {expired ? (
        <Note tone="info" label={t(m.sessionEndedLabel)}>
          {t(m.sessionEnded)}
        </Note>
      ) : null}
      {resetDone ? (
        <Note tone="ok" label={t(m.passwordChangedLabel)}>
          {t(m.passwordChanged)}
        </Note>
      ) : null}
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <label className="lbl">
          {t(m.workEmail)}
          <input
            className={error ? 'in err' : 'in'}
            type="email"
            name="email"
            autoComplete="username"
            placeholder={t(m.emailPlaceholder)}
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
          {t(m.continue)}
        </Button>
      </form>
      {meta?.passkeys !== false ? <PasskeyButton label={t(m.passkeySignIn)} onDone={onPasskey} onProblem={(e) => onProblem('', e)} /> : null}
      {social.length > 0 ? (
        <>
          <div className="sep">{t(m.orContinueWith)}</div>
          <SocialButtons providers={social} href={(p) => withNext(`/api/v1/auth/social/${p}/start`, next)} />
        </>
      ) : null}
      <span className="foot">
        <T msg={m.foot} values={{ reset: (c) => <Link to="/reset-password">{c}</Link> }} />
      </span>
    </AuthLayout>
  )
}

function PasswordStep({
  step,
  next,
  onBack,
  onSubmit,
  onPasskey,
}: {
  step: Extract<Step, { kind: 'password' }>
  next?: string
  onBack: () => void
  onSubmit: (pw: string) => Promise<void>
  onPasskey: (r: Schemas['SignInResult']) => void
}) {
  const t = useT()
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
  // A domain with single sign-on (not enforced) offers it next to the password.
  const domain = step.email.split('@')[1]?.toLowerCase() ?? ''
  const sso = meta?.sso.find((p) => p.domains.includes(domain))
  return (
    <AuthLayout title={t(m.title)} labelledBy="h-password">
      <AuthHeading id="h-password" title={t(m.title)} sub={<NotYou email={step.email} onBack={onBack} />} />
      <form className="flex flex-col gap-4" onSubmit={submit}>
        <input type="email" name="email" autoComplete="username" value={step.email} readOnly hidden />
        <label className="lbl">
          {t(m.password)}
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
          {t(m.continue)}
        </Button>
      </form>
      <div className="links">
        <Link to="/reset-password" search={{ email: step.email }}>
          {t(m.forgotPassword)}
        </Link>
        {meta?.passkeys !== false ? (
          <button type="button" onClick={usePasskey}>
            {t(m.usePasskeyInstead)}
          </button>
        ) : null}
      </div>
      {sso ? (
        <>
          <div className="sep">{t(m.or)}</div>
          <a className="btn" href={withNext(`${sso.startUrl}?login_hint=${encodeURIComponent(step.email)}`, next)}>
            <SocialMark provider={ssoMark(sso.preset)} />
            {t(m.continueWith, { name: sso.name })}
          </a>
        </>
      ) : null}
      {step.error ? <span className="foot">{t(m.sameMessage)}</span> : null}
    </AuthLayout>
  )
}

/** The mark of an SSO provider: Google's and Microsoft's for their presets. */
function ssoMark(preset: string) {
  return preset === 'google' ? 'google' : preset === 'entra' ? 'microsoft' : 'sso'
}

function SsoStep({ step, next, onPassword, onBack }: { step: Extract<Step, { kind: 'sso' }>; next?: string; onPassword: () => void; onBack: () => void }) {
  const t = useT()
  const domain = step.email.split('@')[1] ?? ''
  const google = step.sso.preset === 'google'
  return (
    <AuthLayout title={t(m.continueWith, { name: step.sso.name })} labelledBy="h-sso">
      <AuthHeading id="h-sso" title={t(m.continueWith, { name: step.sso.name })} sub={<NotYou email={step.email} onBack={onBack} />} />
      <Note tone="info" label={t(m.ssoLabel)}>
        {t(m.ssoNote, { domain, name: step.sso.name })}
      </Note>
      <a className="btn pri" href={withNext(step.sso.startUrl, next)}>
        <SocialMark provider={ssoMark(step.sso.preset)} />
        {t(m.continueWith, { name: step.sso.name })}
      </a>
      <Note tone="neutral" label={t(m.ownerLabel)}>
        <T msg={m.breakGlass} values={{ action: (c) => <InlineAction onClick={onPassword}>{c}</InlineAction> }} />
      </Note>
      <span className="foot">{t(m.returnAfter, { name: google ? 'Google' : step.sso.name })}</span>
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
  const t = useT()
  const { data: meta } = useInstanceMeta()
  const { left, label } = useCountdown(step.until)
  const at = clock(step.until)
  return (
    <AuthLayout title={t(m.lockedTitle)} labelledBy="h-lock">
      <AuthHeading id="h-lock" title={left > 0 ? t(m.lockedTitle) : t(m.unlockedTitle)} sub={t(m.lockedSub, { email: step.email, count: step.failures })} />
      {left > 0 ? (
        <div className="lock" role="timer" aria-live="off">
          <span className="t">{label}</span>
          <span className="k">{t(m.lockedRemaining, { time: at })}</span>
        </div>
      ) : null}
      <p className="body">{t(m.lockedBody)}</p>
      <div className="tiles">
        <Link className="btn" to="/reset-password" search={{ email: step.email }}>
          {t(m.resetPassword)}
        </Link>
        {left > 0 ? meta?.passkeys !== false ? <PasskeyButton label={t(m.usePasskey)} onDone={onPasskey} /> : null : <Button onClick={onBack}>{t(m.signInButton)}</Button>}
      </div>
    </AuthLayout>
  )
}

function DeactivatedStep({ step, onBack }: { step: Extract<Step, { kind: 'deactivated' }>; onBack: () => void }) {
  const t = useT()
  return (
    <AuthLayout title={t(m.deactivatedTitle)} labelledBy="h-deact">
      <AuthHeading id="h-deact" title={t(m.deactivatedTitle)} sub={step.email} />
      <Note tone="warn" label={t(m.deactivatedLabel)}>
        {step.at ? t(m.deactivatedOn, { date: usDate(step.at) }) : t(m.deactivated)}
      </Note>
      <p className="body">{t(m.deactivatedMistake)}</p>
      <Button onClick={onBack}>{t(m.backToSignIn)}</Button>
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
  const t = useT()
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
      setLocalError(totp ? t(m.enterSixDigits) : t(m.enterRecovery))
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
          {t(m.usePasskeyInstead)}
        </button>
      ) : null}
      {step.mode !== 'totp' && has('totp') ? (
        <button type="button" onClick={() => onMode('totp')}>
          {t(m.useAuthenticator)}
        </button>
      ) : null}
      {step.mode !== 'recovery' && has('recovery') ? (
        <button type="button" onClick={() => onMode('recovery')}>
          {t(m.useRecovery)}
        </button>
      ) : null}
    </div>
  )

  if (step.mode === 'passkey') {
    return (
      <AuthLayout title={t(m.passkeyTitle)} labelledBy="h-2fa-pk">
        <AuthHeading id="h-2fa-pk" title={t(m.passkeyTitle)} sub={t(m.signingInAs, { email: step.email })} />
        <div className="lock">
          <PasskeyGlyph size={44} strokeWidth={1.4} color="#0F766E" />
          {step.passkeyHint ? <span className="pk-name">{step.passkeyHint}</span> : null}
          <span className="k">{busy ? t(m.browserAsking) : t(m.useButton)}</span>
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
          {t(m.usePasskeyButton)}
        </Button>
        {links}
      </AuthLayout>
    )
  }

  const totp = step.mode === 'totp'
  const signingInAs = t(m.signingInAs, { email: step.email })
  return (
    <AuthLayout title={totp ? t(m.checkAuthenticator) : t(m.useRecovery)} labelledBy="h-2fa">
      <AuthHeading id="h-2fa" title={totp ? t(m.checkAuthenticator) : t(m.useRecovery)} sub={totp && !error ? <NotYou email={signingInAs} onBack={onBack} /> : signingInAs} />
      <form className="flex flex-col gap-4" onSubmit={submit}>
        {totp ? (
          <div className="stack-8">
            <span className="otp-label" id="otp-label">
              {error ? t(m.codeLabelShort) : t(m.codeLabel)}
            </span>
            <OtpInput value={code} onChange={setCode} invalid={Boolean(error)} describedBy={error ? errId : undefined} labelledBy="otp-label" />
            {error ? (
              <span className="err" id={errId} role="alert">
                {error}
              </span>
            ) : (
              <span className="k">{t(m.pasteHint)}</span>
            )}
          </div>
        ) : (
          <label className="lbl">
            {t(m.recoveryCode)}
            <input
              className={error ? 'in err mono' : 'in mono'}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              // The code's format, the same in every language.
              placeholder={'xxxx-xxxx-xxxx'}
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
                {step.recoveryLeft !== undefined ? t(m.eachCodeOnceLeft, { left: step.recoveryLeft, total: step.recoveryTotal ?? 10 }) : t(m.eachCodeOnce)}
              </span>
            )}
          </label>
        )}
        <Button type="submit" variant="primary" loading={busy}>
          {t(m.verify)}
        </Button>
      </form>
      {links}
      {totp && !error ? <span className="foot">{t(m.lostDevice)}</span> : null}
      {!totp ? <span className="foot">{t(m.lostCodes)}</span> : null}
    </AuthLayout>
  )
}

function LastCodeStep({ step }: { step: Extract<Step, { kind: 'last_code' }> }) {
  const t = useT()
  const navigate = useNavigate()
  const [codes, setCodes] = useState<string[]>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  return (
    <AuthLayout title={t(m.lastCodeTitle)} labelledBy="h-last">
      <AuthHeading id="h-last" title={t(m.lastCodeTitle)} sub={step.email} />
      {codes ? (
        <RecoveryCodesPanel codes={codes} email={step.email} onFinish={() => navigate({ to: step.next })} finishLabel={t(m.continue)} />
      ) : (
        <>
          <Note tone="warn" label={t(m.lastCodeLeft, { total: step.total })} role="alert">
            {t(m.lastCodeBody)}
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
              {t(m.generate)}
            </Button>
            <Button onClick={() => navigate({ to: step.next })}>{t(m.remindLater)}</Button>
          </div>
          <span className="foot">{t(m.doItLater)}</span>
        </>
      )}
    </AuthLayout>
  )
}

function IdpError({ provider: given, code, reference, email }: { provider?: string; code?: string; reference?: string; email?: string }) {
  const t = useT()
  const provider = given ?? t(m.idpDefaultProvider)
  const { data: meta } = useInstanceMeta()
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()
  const retry = meta?.sso.find((s) => s.name === provider)
  const social = (['google', 'microsoft', 'github'] as const).find((k) => k === provider.toLowerCase() && meta?.social.includes(k))
  return (
    <AuthLayout title={t(m.idpTitle, { provider })} labelledBy="h-idp">
      <AuthHeading id="h-idp" title={t(m.idpTitle, { provider })} sub={email} />
      <Note tone="bad" label={t(m.idpLabel)}>
        <IdpSentence provider={provider} code={code} /> {t(m.nothingChanged)}
      </Note>
      {reference ? (
        <div className="refbox">
          <span className="k">
            <T msg={m.referenceId} values={{ reference, mono: (c) => <span className="mono text-ink">{c}</span> }} />
          </span>
          <Button
            size="sm"
            className="ml-auto"
            onClick={async () => {
              await navigator.clipboard.writeText(reference)
              setCopied(true)
            }}
          >
            {copied ? t(m.copied) : t(m.copy)}
          </Button>
        </div>
      ) : null}
      <div className="tiles">
        {retry || social ? (
          <a className="btn pri" href={retry ? retry.startUrl + (email ? `?login_hint=${encodeURIComponent(email)}` : '') : `/api/v1/auth/social/${social}/start`}>
            {t(m.tryAgain)}
          </a>
        ) : (
          <Button variant="primary" onClick={() => navigate({ to: '/sign-in', search: email ? { email } : {} })}>
            {t(m.tryAgain)}
          </Button>
        )}
        <Link className="btn" to="/sign-in" search={{}}>
          {t(m.anotherWay)}
        </Link>
      </div>
    </AuthLayout>
  )
}

/** The sentence for each code ReputeCast sets itself; others are the IdP's. */
const idpMessages: Record<string, MessageDescriptor> = {
  state_expired: m.idpStateExpired,
  state_mismatch: m.idpStateMismatch,
  nonce_mismatch: m.idpNonceMismatch,
  provider_disabled: m.idpDisabled,
  method_disabled: m.idpDisabled,
  not_in_org: m.idpNotInOrg,
  org_restricted: m.idpOrgRestricted,
  email_not_verified: m.idpEmailNotVerified,
  invite_wrong_account: m.idpInviteWrongAccount,
  identity_in_use: m.idpIdentityInUse,
  already_linked: m.idpAlreadyLinked,
}

/** What went wrong. */
function IdpSentence({ provider, code }: { provider: string; code?: string }) {
  const t = useT()
  const known = code ? idpMessages[code] : undefined
  if (known) return <>{t(known, { provider })}</>
  if (!code) return <>{t(m.idpReturnedError, { provider })}</>
  return <T msg={m.idpReturned} values={{ provider, code, mono: (c) => <span className="mono">{c}</span> }} />
}

function NoAccount({ email, onRetry }: { email: string; onRetry: (email: string) => Promise<void> }) {
  const t = useT()
  const { data: meta } = useInstanceMeta()
  const [value, setValue] = useState(email)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const errId = useId()
  const sso = meta?.sso.find((s) => s.domains.length > 0)
  const org = orgName(meta?.instanceName)
  return (
    <AuthLayout title={t(m.noAccountTitle, { email })} labelledBy="h-noacct">
      <AuthHeading id="h-noacct" title={t(m.noAccountTitle, { email })} sub={t(m.noAccountSub)} />
      <Note tone="neutral" label={t(m.inviteOnlyLabel)}>
        {t(m.inviteOnly)}
      </Note>
      <form
        className="flex flex-col gap-4"
        onSubmit={async (e) => {
          e.preventDefault()
          if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())) {
            setError(t(m.emailInvalid))
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
          {t(m.tryAnother)}
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
          {t(m.continue)}
        </Button>
      </form>
      {sso ? (
        <span className="foot">
          {org ? t(m.workFor, { org, domain: sso.domains[0], provider: sso.name }) : t(m.useSso, { domain: sso.domains[0], provider: sso.name })}
        </span>
      ) : null}
    </AuthLayout>
  )
}
