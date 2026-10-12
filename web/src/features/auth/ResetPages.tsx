import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { clock } from '@/lib/format'
import { AuthHeading, SocialMark } from './AuthParts'

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** "Locked out of Google too?": Google Workspace is Google; others keep their name. */
function shortProvider(sso: Schemas['SSOStart']) {
  return sso.preset === 'google' ? 'Google' : sso.preset === 'entra' ? 'Microsoft' : sso.name
}

/**
 * A5 §5.3: request a reset link. The answer is the same for every email,
 * except that SSO domains are told their IdP manages the password and an
 * instance without a System-stream provider shows the server command.
 */
export function ResetRequestPage({ email: initial, expired }: { email?: string; expired?: boolean }) {
  const [email, setEmail] = useState(initial ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [result, setResult] = useState<Schemas['PasswordResetRequest']>()
  const errId = useId()
  const input = useRef<HTMLInputElement>(null)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!EMAIL.test(email.trim())) {
      setError('Enter your work email, like you@company.com.')
      return
    }
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/auth/password-reset', { body: { email: email.trim() } })
      setResult(data)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  if (result?.status === 'email_unavailable') {
    return (
      <AuthLayout title="Reset your password" labelledBy="h-reset-cli">
        <AuthHeading id="h-reset-cli" title="Reset your password" sub={result.email} />
        <Note tone="warn" label="Email unavailable" role="status">
          No sending provider is connected to the System stream yet, so the reset email can't go out. An instance admin can reset it from the server:
        </Note>
        <pre className="code">{result.command}</pre>
        <p className="m-0 text-[13px] text-ink-nav">
          The command prints a one-time link that works for {result.expiresInMinutes} minutes. The same applies to two-factor resets:{' '}
          <span className="mono">reputecast user reset-2fa --email …</span>
        </p>
        <Link className="btn" to="/sign-in">
          Back to sign in
        </Link>
      </AuthLayout>
    )
  }

  if (result?.status === 'sso' && result.sso) {
    const sso = result.sso
    const domain = result.email.split('@')[1] ?? ''
    return (
      <AuthLayout title={`Your password is managed by ${sso.name}`} labelledBy="h-reset-sso">
        <AuthHeading id="h-reset-sso" title={`Your password is managed by ${sso.name}`} sub={result.email} />
        <Note tone="info" label="SSO">
          {domain} accounts sign in through {sso.name}. ReputeCast doesn't store a password for this account, so there is nothing to reset here.
        </Note>
        <a className="btn pri" href={sso.startUrl}>
          <SocialMark provider={sso.preset === 'google' ? 'google' : 'sso'} />
          Continue with {sso.name}
        </a>
        <span className="foot">
          Locked out of {shortProvider(sso)} too? Contact your {sso.name} admin.
        </span>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Reset your password" labelledBy="h-reset">
      {/* This card's board spaces its rows 14px apart (the others use 16px). */}
      <div className="-mt-0.5 flex flex-col gap-3.5">
        <h1 id="h-reset">Reset your password</h1>
        <form className="flex flex-col gap-3.5" onSubmit={submit} noValidate>
          <label className="lbl">
            Work email
            <input
              ref={input}
              className={error ? 'in err' : 'in'}
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
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
            Email me a reset link
          </Button>
        </form>
        {result?.status === 'sent' ? (
          <Note tone="ok" label="Sent">
            If {result.email} belongs to a member, a link is on its way. It works once and expires in {result.expiresInMinutes} minutes. Check Promotions or spam if it's slow.
          </Note>
        ) : null}
        {expired && !result ? (
          <Note tone="bad" label="Expired" role="alert">
            This reset link is older than 15 minutes or was already used.{' '}
            <a
              href="#email"
              onClick={(e) => {
                e.preventDefault()
                input.current?.focus()
              }}
            >
              Request a new one
            </a>
            ; your current password still works until you change it.
          </Note>
        ) : null}
        <span className="foot">
          <Link to="/sign-in">Back to sign in</Link> · Workspaces that use SSO reset passwords with their identity provider.
        </span>
      </div>
    </AuthLayout>
  )
}

/** Set a new password with a one-time link (15 minutes). */
export function ResetTokenPage({ token }: { token: string }) {
  const navigate = useNavigate()
  const info = useQuery({
    queryKey: ['password-reset', token],
    queryFn: async () => (await http.GET('/api/v1/auth/password-reset/{token}', { params: { path: { token } } })).data!,
    retry: false,
    staleTime: Infinity,
  })
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ password?: string; confirm?: string; form?: string }>({})
  const pwHint = useId()
  const pwErr = useId()
  const cfErr = useId()

  if (info.isPending) {
    return (
      <AuthLayout title="Set a new password" labelledBy="h-newpw">
        <div className="flex flex-col gap-3" aria-busy="true">
          <h1 id="h-newpw" className="vh">
            Set a new password
          </h1>
          <Skeleton w="60%" h={22} />
          <Skeleton w="40%" h={12} />
          <Skeleton w="100%" h={40} r={8} />
          <Skeleton w="100%" h={40} r={8} />
          <span className="vh" role="status">
            Checking the reset link…
          </span>
        </div>
      </AuthLayout>
    )
  }
  if (info.isError) {
    if (info.error instanceof ApiError && (info.error.code === 'reset_expired' || info.error.status === 404)) return <ResetRequestPage expired />
    return (
      <AuthLayout title="Set a new password" labelledBy="h-newpw">
        <AuthHeading id="h-newpw" title="Set a new password" />
        <Note tone="bad" label="Error">
          {errorMessage(info.error)} The link still works until it expires.
        </Note>
        <Button variant="primary" onClick={() => info.refetch()}>
          Try again
        </Button>
      </AuthLayout>
    )
  }

  const data = info.data
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const next: typeof errors = {}
    if ([...password].length < data.minLength) next.password = `Use at least ${data.minLength} characters. Passphrases are fine.`
    if (confirm !== password) next.confirm = "The two passwords don't match."
    setErrors(next)
    if (next.password || next.confirm) return
    setBusy(true)
    try {
      const { data: done } = await http.POST('/api/v1/auth/password-reset/{token}', { params: { path: { token } }, body: { password } })
      void navigate({ to: '/sign-in', search: { email: done?.email, reset: 'done' } })
    } catch (err) {
      if (err instanceof ApiError && err.code === 'reset_expired') {
        void info.refetch()
        return
      }
      const field = err instanceof ApiError ? err.problem.errors?.find((x) => x.location === 'body.password') : undefined
      setErrors(field ? { password: field.message } : { form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout title="Set a new password" labelledBy="h-newpw">
      <AuthHeading id="h-newpw" title="Set a new password" sub={data.email} />
      {errors.form ? (
        <Note tone="bad" label="Not saved">
          {errors.form}
        </Note>
      ) : null}
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <input type="email" name="email" autoComplete="username" value={data.email} readOnly hidden />
        <label className="lbl">
          New password
          <input
            className={errors.password ? 'in err' : 'in'}
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoFocus
            aria-invalid={errors.password ? true : undefined}
            aria-describedby={errors.password ? `${pwErr} ${pwHint}` : pwHint}
          />
          {errors.password ? (
            <span className="err" id={pwErr} role="alert">
              {errors.password}
            </span>
          ) : null}
          <span className="k" id={pwHint}>
            At least {data.minLength} characters. {data.breachedCheck ? 'Checked against known breaches.' : 'Passphrases are fine.'}
          </span>
        </label>
        <label className="lbl">
          Confirm new password
          <input
            className={errors.confirm ? 'in err' : 'in'}
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            aria-invalid={errors.confirm ? true : undefined}
            aria-describedby={errors.confirm ? cfErr : undefined}
          />
          {errors.confirm ? (
            <span className="err" id={cfErr} role="alert">
              {errors.confirm}
            </span>
          ) : null}
        </label>
        <Button type="submit" variant="primary" loading={busy}>
          Save new password
        </Button>
      </form>
      <span className="foot">
        The link works once, until {clock(data.expiresAt)}. Saving signs out every session; two-factor still applies at sign-in.
      </span>
    </AuthLayout>
  )
}
