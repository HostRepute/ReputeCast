import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useState, type FormEvent, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { meQueryKey } from '@/lib/session'
import { AuthHeading, PasskeyGlyph, SocialButtons, SocialMark } from './AuthParts'
import { passkeyErrorMessage, registerPasskey } from './passkey'
import { RecoveryCodesPanel } from './RecoveryCodes'
import { TotpSetupForm } from './TotpSetup'

type View = Schemas['InvitationView']

const monthDay = (d: string) => new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
const fullDate = (d: string) => new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })

function daysUntil(d: string) {
  return Math.max(0, Math.ceil((new Date(d).getTime() - Date.now()) / 86_400_000))
}

function expiresIn(d: string) {
  const days = daysUntil(d)
  if (days >= 1) return `expires in ${days} ${days === 1 ? 'day' : 'days'}`
  const h = Math.max(1, Math.round((new Date(d).getTime() - Date.now()) / 3_600_000))
  return `expires in ${h} h`
}

/** A5 §5.4: accept an invitation, with the expired, revoked and wrong-account states. */
export function InvitePage({ token }: { token: string }) {
  const qc = useQueryClient()
  const inv = useQuery({
    queryKey: ['invitation', token],
    queryFn: async () => (await http.GET('/api/v1/invitations/{token}', { params: { path: { token } } })).data!,
    retry: false,
  })
  const [joined, setJoined] = useState<Schemas['InvitationAccepted']>()

  if (joined) return <ProtectAccount joined={joined} />
  if (inv.isPending) return <InviteSkeleton />
  if (inv.isError) {
    const notFound = inv.error instanceof ApiError && inv.error.status === 404
    return (
      <AuthLayout title={notFound ? "This invitation link isn't valid" : "The invitation couldn't load"} labelledBy="h-inv-err">
        <AuthHeading id="h-inv-err" title={notFound ? "This invitation link isn't valid" : "The invitation couldn't load"} />
        <Note tone={notFound ? 'neutral' : 'bad'} label={notFound ? undefined : 'Error'}>
          {notFound ? 'Check that you copied the whole link from the email, or ask the person who invited you for a new one.' : `${errorMessage(inv.error)} Nothing was created.`}
        </Note>
        {notFound ? null : (
          <Button variant="primary" onClick={() => inv.refetch()}>
            Try again
          </Button>
        )}
        <Link className="btn" to="/sign-in">
          Back to sign in
        </Link>
      </AuthLayout>
    )
  }
  const v = inv.data
  const refresh = async () => {
    await qc.invalidateQueries({ queryKey: meQueryKey })
    await inv.refetch()
  }
  switch (v.status) {
    case 'expired':
      return <Expired v={v} token={token} />
    case 'revoked':
      return <Revoked v={v} />
    case 'accepted':
      return <Accepted v={v} />
  }
  if (v.signedInAs && v.signedInAs !== v.email) return <WrongAccount v={v} token={token} onSignedOut={refresh} />
  return <Accept v={v} token={token} onJoined={setJoined} />
}

function InviteSkeleton() {
  return (
    <AuthLayout title="Invitation" labelledBy="h-inv-load">
      <div className="flex flex-col gap-4" aria-busy="true">
        <h1 id="h-inv-load" className="vh">
          Loading the invitation
        </h1>
        <Skeleton w="85%" h={22} />
        <Skeleton w="60%" h={12} />
        <Skeleton w="100%" h={56} r={8} />
        <Skeleton w="100%" h={40} r={8} />
        <Skeleton w="100%" h={40} r={8} />
        <span className="vh" role="status">
          Loading the invitation…
        </span>
      </div>
    </AuthLayout>
  )
}

function Title({ v }: { v: View }) {
  return (
    <>
      {v.inviterName} invited you to {v.workspaceName}
    </>
  )
}

function Meta({ v }: { v: View }) {
  return (
    <>
      Role <b>{v.roleName}</b>
      {v.scope ? (
        <>
          {' '}
          · scope <b>{v.scope}</b>
        </>
      ) : null}{' '}
      · {expiresIn(v.expiresAt)}
    </>
  )
}

function Accept({ v, token, onJoined }: { v: View; token: string; onJoined: (j: Schemas['InvitationAccepted']) => void }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; password?: string; form?: string }>({})
  const nameErr = useId()
  const pwErr = useId()
  const pwHint = useId()
  const title = `${v.inviterName} invited you to ${v.workspaceName}`
  const signedIn = Boolean(v.signedInAs)

  const join = async (e?: FormEvent) => {
    e?.preventDefault()
    const next: typeof errors = {}
    if (!signedIn) {
      if (!name.trim()) next.name = 'Enter your name as the team should see it.'
      if ([...password].length < v.passwordPolicy.minLength) next.password = `Use at least ${v.passwordPolicy.minLength} characters. Passphrases are fine.`
    }
    setErrors(next)
    if (next.name || next.password) return
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/invitations/{token}/accept', { params: { path: { token } }, body: signedIn ? {} : { name: name.trim(), password } })
      await qc.invalidateQueries({ queryKey: meQueryKey })
      if (!data) return
      if (data.hasSecondFactor) {
        toast({ kind: 'success', title: `You joined ${data.workspaceName}`, detail: `as ${data.roleName}` })
        void navigate({ to: '/w/$workspace', params: { workspace: data.workspace } })
      } else onJoined(data)
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === 'invite_sign_in') {
          void navigate({ to: '/sign-in', search: { email: v.email, next: `/invite/${token}` } })
          return
        }
        const field = err.problem.errors?.find((x) => x.location === 'body.password' || x.location === 'body.name')
        if (field) {
          setErrors(field.location === 'body.name' ? { name: field.message } : { password: field.message })
          return
        }
      }
      setErrors({ form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }

  const header = (
    <>
      <AuthHeading id="h-inv" title={<Title v={v} />} sub={<Meta v={v} />} />
      <Note tone="neutral">{v.description}</Note>
      {v.message ? (
        <Note tone="info" label={`From ${v.inviterName}`}>
          {v.message}
        </Note>
      ) : null}
      {errors.form ? (
        <Note tone="bad" label="Not joined">
          {errors.form}
        </Note>
      ) : null}
    </>
  )

  // An account already uses this email: sign in to accept, or accept now.
  if (signedIn || v.accountExists) {
    return (
      <AuthLayout title={title} labelledBy="h-inv">
        {header}
        {signedIn ? (
          <Button variant="primary" loading={busy} onClick={() => join()}>
            Join {v.workspaceName}
          </Button>
        ) : (
          <Link className="btn pri" to="/sign-in" search={{ email: v.email, next: `/invite/${token}` }}>
            Sign in to accept
          </Link>
        )}
        <span className="foot">{signedIn ? `You're signed in as ${v.signedInAs}.` : `${v.email} already has an account here. Sign in and the invitation opens again.`}</span>
      </AuthLayout>
    )
  }

  const socialHref = (p: string) => `/api/v1/auth/social/${p}/start?invite=${encodeURIComponent(token)}`
  return (
    <AuthLayout title={title} labelledBy="h-inv">
      {header}
      {v.passwordAllowed ? (
        <form className="flex flex-col gap-4" onSubmit={join} noValidate>
          <label className="lbl">
            Your name
            <input
              className={errors.name ? 'in err' : 'in'}
              type="text"
              autoComplete="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
              aria-invalid={errors.name ? true : undefined}
              aria-describedby={errors.name ? nameErr : undefined}
            />
            {errors.name ? (
              <span className="err" id={nameErr} role="alert">
                {errors.name}
              </span>
            ) : null}
          </label>
          <label className="lbl">
            Email
            <input className="in" type="email" autoComplete="username" value={v.email} readOnly />
          </label>
          <label className="lbl">
            Create a password
            <input
              className={errors.password ? 'in err' : 'in'}
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={errors.password ? true : undefined}
              aria-describedby={errors.password ? `${pwErr} ${pwHint}` : pwHint}
            />
            {errors.password ? (
              <span className="err" id={pwErr} role="alert">
                {errors.password}
              </span>
            ) : null}
            <span className="k font-medium" id={pwHint}>
              At least {v.passwordPolicy.minLength} characters. {v.passwordPolicy.breachedCheck ? 'Checked against known breaches.' : 'Passphrases are fine.'}
            </span>
          </label>
          <Button type="submit" variant="primary" loading={busy}>
            Join {v.workspaceName}
          </Button>
        </form>
      ) : null}
      {v.sso ? (
        <a className={v.passwordAllowed ? 'btn' : 'btn pri'} href={v.sso.startUrl}>
          <SocialMark provider={v.sso.preset === 'google' ? 'google' : 'sso'} />
          Continue with {v.sso.name}
        </a>
      ) : null}
      {v.social.length > 0 ? (
        <>
          <div className="sep">or join with</div>
          <SocialButtons providers={v.social} href={socialHref} />
        </>
      ) : null}
      <span className="foot">Next you can add a passkey and set up two-factor.</span>
    </AuthLayout>
  )
}

/** After joining: offer a passkey and two-factor; show the grace period when required. */
function ProtectAccount({ joined }: { joined: Schemas['InvitationAccepted'] }) {
  const navigate = useNavigate()
  const [mode, setMode] = useState<'choose' | 'totp' | 'codes'>('choose')
  const [codes, setCodes] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const go = () => navigate({ to: '/w/$workspace', params: { workspace: joined.workspace } })
  const tf = joined.twoFactor
  const graceDays = tf.graceUntil ? daysUntil(tf.graceUntil) : tf.graceDays

  let body: ReactNode
  if (mode === 'totp') {
    body = (
      <TotpSetupForm
        onConfirmed={(c) => {
          setCodes(c)
          setMode('codes')
        }}
        onCancel={() => setMode('choose')}
        cancelLabel="Back"
      />
    )
  } else if (mode === 'codes') {
    body = (
      <RecoveryCodesPanel
        codes={codes}
        email={joined.email}
        busy={busy}
        finishLabel={`Continue to ${joined.workspaceName}`}
        onFinish={async () => {
          setBusy(true)
          try {
            await http.POST('/api/v1/me/recovery-codes/saved')
          } finally {
            setBusy(false)
          }
          void go()
        }}
      />
    )
  } else {
    body = (
      <>
        {tf.required ? (
          <Note tone="warn" label="Required">
            {tf.reason} Grace period: {tf.graceDays} days{tf.graceUntil ? (
              <>
                , ends <b>{fullDate(tf.graceUntil)}</b>
              </>
            ) : null}
            . After that you'll have to set it up before continuing.
          </Note>
        ) : (
          <Note tone="info" label="Recommended">
            Add a passkey or an authenticator app so a stolen password isn't enough to get in. You can also do it later from Account &amp; security.
          </Note>
        )}
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        <Button
          variant="primary"
          loading={busy}
          onClick={async () => {
            setBusy(true)
            setError(undefined)
            try {
              const pk = await registerPasskey()
              toast({ kind: 'success', title: 'Passkey added', detail: pk.name })
              void go()
            } catch (e) {
              setError(passkeyErrorMessage(e, 'add'))
            } finally {
              setBusy(false)
            }
          }}
        >
          {busy ? null : <PasskeyGlyph />}
          Add a passkey (recommended)
        </Button>
        <Button onClick={() => setMode('totp')}>Set up an authenticator app</Button>
        <Button className="quiet" onClick={() => go()}>
          {tf.required ? `Skip for now · ${graceDays} ${graceDays === 1 ? 'day' : 'days'} left` : 'Skip for now'}
        </Button>
      </>
    )
  }
  return (
    <AuthLayout title="Protect your new account" labelledBy="h-inv-2fa">
      <AuthHeading id="h-inv-2fa" title={mode === 'codes' ? 'Save your recovery codes' : mode === 'totp' ? 'Set up an authenticator app' : 'Protect your new account'} sub={`${joined.email} · ${joined.roleName} in ${joined.workspaceName}`} />
      {body}
    </AuthLayout>
  )
}

function Expired({ v, token }: { v: View; token: string }) {
  const [sent, setSent] = useState<Schemas['InvitationRequestResult']>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const inviter = v.inviterName.startsWith('An admin of') ? 'an admin' : v.inviterName
  return (
    <AuthLayout title="This invitation has expired" labelledBy="h-inv-exp">
      <AuthHeading id="h-inv-exp" title="This invitation has expired" sub={`${v.email} · sent ${monthDay(v.sentAt)}, expired ${monthDay(v.expiresAt)}`} />
      <Note tone="neutral">Invitations work for 7 days by default. Nothing was created for you.</Note>
      {sent ? (
        <Note tone="ok" label="Sent">
          {sent.status === 'already_sent' ? `${sent.inviter} was already asked today.` : `${sent.inviter} got your request and can send a new invitation.`}
        </Note>
      ) : (
        <>
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
          <Button
            variant="primary"
            loading={busy}
            onClick={async () => {
              setBusy(true)
              setError(undefined)
              try {
                const { data } = await http.POST('/api/v1/invitations/{token}/request', { params: { path: { token } } })
                setSent(data)
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            Ask {inviter} for a new invite
          </Button>
        </>
      )}
      <span className="foot">
        Sends a short note to the inviter. Already have an account? <Link to="/sign-in">Sign in</Link>.
      </span>
    </AuthLayout>
  )
}

function Revoked({ v }: { v: View }) {
  return (
    <AuthLayout title="This invitation was revoked" labelledBy="h-inv-rev">
      <AuthHeading id="h-inv-rev" title="This invitation was revoked" sub={v.email} />
      <Note tone="bad" label="Revoked">
        An admin of {v.workspaceName} withdrew this invitation{v.revokedAt ? ` on ${monthDay(v.revokedAt)}` : ''}. The link no longer works.
      </Note>
      <Link className="btn" to="/sign-in">
        Back to sign in
      </Link>
    </AuthLayout>
  )
}

function Accepted({ v }: { v: View }) {
  return (
    <AuthLayout title="This invitation was already used" labelledBy="h-inv-used">
      <AuthHeading id="h-inv-used" title="This invitation was already used" sub={v.email} />
      <Note tone="neutral">
        {v.email} joined {v.workspaceName}
        {v.acceptedAt ? ` on ${monthDay(v.acceptedAt)}` : ''}. Sign in to open it.
      </Note>
      {v.signedInAs === v.email ? (
        <Link className="btn pri" to="/w/$workspace" params={{ workspace: v.workspaceSlug }}>
          Open {v.workspaceName}
        </Link>
      ) : (
        <Link className="btn pri" to="/sign-in" search={{ email: v.email }}>
          Sign in
        </Link>
      )}
    </AuthLayout>
  )
}

function WrongAccount({ v, token, onSignedOut }: { v: View; token: string; onSignedOut: () => Promise<void> }) {
  const qc = useQueryClient()
  const [busy, setBusy] = useState<'out' | 'ask'>()
  const [sent, setSent] = useState<Schemas['InvitationRequestResult']>()
  const [error, setError] = useState<string>()
  return (
    <AuthLayout title="This invite isn't for the account you're using" labelledBy="h-inv-diff">
      <AuthHeading
        id="h-inv-diff"
        title="This invite isn't for the account you're using"
        sub={
          <>
            Invite for <b>{v.email}</b> · signed in as <b>{v.signedInAs}</b>
          </>
        }
      />
      <p className="body">
        Invitations are tied to the email they were sent to. Sign out and continue as {v.email}, or ask {v.inviterName} to invite {v.signedInAs} instead.
      </p>
      {sent ? (
        <Note tone="ok" label="Sent">
          {sent.status === 'already_sent' ? `${sent.inviter} was already asked today.` : `${sent.inviter} got your request to invite ${v.signedInAs}.`}
        </Note>
      ) : null}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
      <div className="tiles">
        <Button
          variant="primary"
          loading={busy === 'out'}
          onClick={async () => {
            setBusy('out')
            try {
              await http.POST('/api/v1/auth/sign-out')
              qc.clear()
              await onSignedOut()
            } catch (e) {
              setError(errorMessage(e))
            } finally {
              setBusy(undefined)
            }
          }}
        >
          Sign out and continue
        </Button>
        {sent ? null : (
          <Button
            loading={busy === 'ask'}
            onClick={async () => {
              setBusy('ask')
              setError(undefined)
              try {
                const { data } = await http.POST('/api/v1/invitations/{token}/request', { params: { path: { token } } })
                setSent(data)
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(undefined)
              }
            }}
          >
            Ask for an invite instead
          </Button>
        )}
      </div>
    </AuthLayout>
  )
}
