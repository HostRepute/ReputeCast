import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type FormEvent, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useI18n, useT } from '@/i18n'
import { setPageLanguage } from '@/i18n/sync'
import { monthDay, usDate } from '@/lib/format'
import { meQueryKey } from '@/lib/session'
import { AuthHeading, PasskeyGlyph, SocialButtons, SocialMark } from './AuthParts'
import { m } from './invite.messages'
import { passkeyErrorMessage, registerPasskey } from './passkey'
import { RecoveryCodesPanel } from './RecoveryCodes'
import { TotpSetupForm } from './TotpSetup'

type View = Schemas['InvitationView']

const bold = (c: ReactNode) => <b>{c}</b>

function daysUntil(d: string) {
  return Math.max(0, Math.ceil((new Date(d).getTime() - Date.now()) / 86_400_000))
}

function expiresIn(d: string) {
  const days = daysUntil(d)
  if (days >= 1) return tt(m.expiresInDays, { days })
  return tt(m.expiresInHours, { hours: Math.max(1, Math.round((new Date(d).getTime() - Date.now()) / 3_600_000)) })
}

/**
 * A5 §5.4: accept an invitation, with the expired, revoked and wrong-account
 * states. The page reads in the invitation's language unless the visitor
 * picked another in the menu; the server writes the role's description in
 * the language asked for, so the invitation is fetched again when it changes.
 */
export function InvitePage({ token }: { token: string }) {
  const qc = useQueryClient()
  const { locale } = useI18n()
  const inv = useQuery({
    queryKey: ['invitation', token, locale],
    queryFn: async () => (await http.GET('/api/v1/invitations/{token}', { params: { path: { token } } })).data!,
    retry: false,
    // While the page switches language, keep showing the invitation in the previous one.
    placeholderData: () => qc.getQueriesData<View>({ queryKey: ['invitation', token] }).find(([, d]) => d)?.[1],
  })
  const [joined, setJoined] = useState<Schemas['InvitationAccepted']>()
  const language = inv.data?.language
  useEffect(() => {
    if (language) void setPageLanguage(language)
  }, [language])

  if (joined) return <ProtectAccount joined={joined} />
  if (inv.isPending) return <InviteSkeleton />
  if (inv.isError) return <InviteError error={inv.error} onRetry={() => inv.refetch()} />
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

function InviteError({ error, onRetry }: { error: Error; onRetry: () => void }) {
  const t = useT()
  const notFound = error instanceof ApiError && error.status === 404
  const title = notFound ? t(m.invalidTitle) : t(m.loadFailedTitle)
  return (
    <AuthLayout title={title} labelledBy="h-inv-err">
      <AuthHeading id="h-inv-err" title={title} />
      <Note tone={notFound ? 'neutral' : 'bad'} label={notFound ? undefined : t(m.errorLabel)}>
        {notFound ? t(m.invalidBody) : `${errorMessage(error)} ${t(m.nothingCreated)}`}
      </Note>
      {notFound ? null : (
        <Button variant="primary" onClick={onRetry}>
          {t(m.tryAgain)}
        </Button>
      )}
      <Link className="btn" to="/sign-in">
        {t(m.backToSignIn)}
      </Link>
    </AuthLayout>
  )
}

function InviteSkeleton() {
  const t = useT()
  return (
    <AuthLayout title={t(m.docTitle)} labelledBy="h-inv-load">
      <div className="flex flex-col gap-4" aria-busy="true">
        <h1 id="h-inv-load" className="vh">
          {t(m.loading)}
        </h1>
        <Skeleton w="85%" h={22} />
        <Skeleton w="60%" h={12} />
        <Skeleton w="100%" h={56} r={8} />
        <Skeleton w="100%" h={40} r={8} />
        <Skeleton w="100%" h={40} r={8} />
        <span className="vh" role="status">
          {t(m.loadingStatus)}
        </span>
      </div>
    </AuthLayout>
  )
}

function Meta({ v }: { v: View }) {
  return <T msg={v.scope ? m.metaRoleScope : m.metaRole} values={{ role: v.roleName, scope: v.scope ?? '', expires: expiresIn(v.expiresAt), b: bold }} />
}

function Accept({ v, token, onJoined }: { v: View; token: string; onJoined: (j: Schemas['InvitationAccepted']) => void }) {
  const t = useT()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; password?: string; form?: string }>({})
  const nameErr = useId()
  const pwErr = useId()
  const pwHint = useId()
  const title = t(m.title, { inviter: v.inviterName, workspace: v.workspaceName })
  const signedIn = Boolean(v.signedInAs)
  const min = v.passwordPolicy.minLength

  const join = async (e?: FormEvent) => {
    e?.preventDefault()
    const next: typeof errors = {}
    if (!signedIn) {
      if (!name.trim()) next.name = t(m.nameRequired)
      if ([...password].length < min) next.password = t(m.tooShort, { min })
    }
    setErrors(next)
    if (next.name || next.password) return
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/invitations/{token}/accept', { params: { path: { token } }, body: signedIn ? {} : { name: name.trim(), password } })
      await qc.invalidateQueries({ queryKey: meQueryKey })
      if (!data) return
      if (data.hasSecondFactor) {
        toast({ kind: 'success', title: t(m.joinedToast, { workspace: data.workspaceName }), detail: t(m.joinedDetail, { role: data.roleName }) })
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
      <AuthHeading id="h-inv" title={title} sub={<Meta v={v} />} />
      <Note tone="neutral">{v.description}</Note>
      {v.message ? (
        <Note tone="info" label={t(m.fromInviter, { name: v.inviterName })}>
          {v.message}
        </Note>
      ) : null}
      {errors.form ? (
        <Note tone="bad" label={t(m.notJoinedLabel)}>
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
            {t(m.join, { workspace: v.workspaceName })}
          </Button>
        ) : (
          <Link className="btn pri" to="/sign-in" search={{ email: v.email, next: `/invite/${token}` }}>
            {t(m.signInToAccept)}
          </Link>
        )}
        <span className="foot">{signedIn ? t(m.signedInAs, { email: v.signedInAs ?? '' }) : t(m.accountExists, { email: v.email })}</span>
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
            {t(m.yourName)}
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
            {t(m.email)}
            <input className="in" type="email" autoComplete="username" value={v.email} readOnly />
          </label>
          <label className="lbl">
            {t(m.createPassword)}
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
              {v.passwordPolicy.breachedCheck ? t(m.hintBreached, { min }) : t(m.hintPassphrase, { min })}
            </span>
          </label>
          <Button type="submit" variant="primary" loading={busy}>
            {t(m.join, { workspace: v.workspaceName })}
          </Button>
        </form>
      ) : null}
      {v.sso ? (
        <a className={v.passwordAllowed ? 'btn' : 'btn pri'} href={v.sso.startUrl}>
          <SocialMark provider={v.sso.preset === 'google' ? 'google' : 'sso'} />
          {t(m.continueWith, { name: v.sso.name })}
        </a>
      ) : null}
      {v.social.length > 0 ? (
        <>
          <div className="sep">{t(m.orJoinWith)}</div>
          <SocialButtons providers={v.social} href={socialHref} />
        </>
      ) : null}
      <span className="foot">{t(m.nextPasskey)}</span>
    </AuthLayout>
  )
}

/** After joining: offer a passkey and two-factor; show the grace period when required. */
function ProtectAccount({ joined }: { joined: Schemas['InvitationAccepted'] }) {
  const t = useT()
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
        cancelLabel={t(m.back)}
      />
    )
  } else if (mode === 'codes') {
    body = (
      <RecoveryCodesPanel
        codes={codes}
        email={joined.email}
        busy={busy}
        finishLabel={t(m.continueTo, { workspace: joined.workspaceName })}
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
          <Note tone="warn" label={t(m.requiredLabel)}>
            <T
              msg={tf.graceUntil ? m.requiredGrace : m.requiredGraceNoDate}
              values={{ reason: tf.reason ?? '', days: tf.graceDays, date: tf.graceUntil ? usDate(tf.graceUntil) : '', b: bold }}
            />
          </Note>
        ) : (
          <Note tone="info" label={t(m.recommendedLabel)}>
            {t(m.recommended)}
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
              toast({ kind: 'success', title: t(m.passkeyAdded), detail: pk.name })
              void go()
            } catch (e) {
              setError(passkeyErrorMessage(e, 'add'))
            } finally {
              setBusy(false)
            }
          }}
        >
          {busy ? null : <PasskeyGlyph />}
          {t(m.addPasskey)}
        </Button>
        <Button onClick={() => setMode('totp')}>{t(m.setupAuthenticator)}</Button>
        <Button className="quiet" onClick={() => go()}>
          {tf.required ? t(m.skipLeft, { days: graceDays }) : t(m.skip)}
        </Button>
      </>
    )
  }
  return (
    <AuthLayout title={t(m.protectTitle)} labelledBy="h-inv-2fa">
      <AuthHeading
        id="h-inv-2fa"
        title={mode === 'codes' ? t(m.saveCodesTitle) : mode === 'totp' ? t(m.setupAuthenticator) : t(m.protectTitle)}
        sub={t(m.protectSub, { email: joined.email, role: joined.roleName, workspace: joined.workspaceName })}
      />
      {body}
    </AuthLayout>
  )
}

/** Asks the inviter for a new invitation (once a day per invitation). */
function useRequestNewInvite(token: string) {
  const [sent, setSent] = useState<Schemas['InvitationRequestResult']>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const ask = async () => {
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
  }
  return { sent, error, setError, busy, ask }
}

function Expired({ v, token }: { v: View; token: string }) {
  const t = useT()
  const req = useRequestNewInvite(token)
  const title = t(m.expiredTitle)
  return (
    <AuthLayout title={title} labelledBy="h-inv-exp">
      <AuthHeading id="h-inv-exp" title={title} sub={t(m.expiredSub, { email: v.email, sent: monthDay(v.sentAt), expired: monthDay(v.expiresAt) })} />
      <Note tone="neutral">{t(m.expiredNote)}</Note>
      {req.sent ? (
        <Note tone="ok" label={t(m.sentLabel)}>
          {req.sent.status === 'already_sent' ? t(m.alreadyAsked, { inviter: req.sent.inviter }) : t(m.requestSent, { inviter: req.sent.inviter })}
        </Note>
      ) : (
        <>
          {req.error ? (
            <span className="err" role="alert">
              {req.error}
            </span>
          ) : null}
          <Button variant="primary" loading={req.busy} onClick={req.ask}>
            {t(m.askInviter, { inviter: v.inviterUnknown ? t(m.anAdmin) : v.inviterName })}
          </Button>
        </>
      )}
      <span className="foot">
        <T msg={m.expiredFoot} values={{ link: (c) => <Link to="/sign-in">{c}</Link> }} />
      </span>
    </AuthLayout>
  )
}

function Revoked({ v }: { v: View }) {
  const t = useT()
  const title = t(m.revokedTitle)
  return (
    <AuthLayout title={title} labelledBy="h-inv-rev">
      <AuthHeading id="h-inv-rev" title={title} sub={v.email} />
      <Note tone="bad" label={t(m.revokedLabel)}>
        {v.revokedAt ? t(m.revokedOn, { workspace: v.workspaceName, date: monthDay(v.revokedAt) }) : t(m.revoked, { workspace: v.workspaceName })}
      </Note>
      <Link className="btn" to="/sign-in">
        {t(m.backToSignIn)}
      </Link>
    </AuthLayout>
  )
}

function Accepted({ v }: { v: View }) {
  const t = useT()
  const title = t(m.usedTitle)
  return (
    <AuthLayout title={title} labelledBy="h-inv-used">
      <AuthHeading id="h-inv-used" title={title} sub={v.email} />
      <Note tone="neutral">
        {v.acceptedAt ? t(m.joinedOn, { email: v.email, workspace: v.workspaceName, date: monthDay(v.acceptedAt) }) : t(m.joined, { email: v.email, workspace: v.workspaceName })}
      </Note>
      {v.signedInAs === v.email ? (
        <Link className="btn pri" to="/w/$workspace" params={{ workspace: v.workspaceSlug }}>
          {t(m.open, { workspace: v.workspaceName })}
        </Link>
      ) : (
        <Link className="btn pri" to="/sign-in" search={{ email: v.email }}>
          {t(m.signIn)}
        </Link>
      )}
    </AuthLayout>
  )
}

function WrongAccount({ v, token, onSignedOut }: { v: View; token: string; onSignedOut: () => Promise<void> }) {
  const t = useT()
  const qc = useQueryClient()
  const [signingOut, setSigningOut] = useState(false)
  const req = useRequestNewInvite(token)
  const title = t(m.wrongTitle)
  const current = v.signedInAs ?? ''
  return (
    <AuthLayout title={title} labelledBy="h-inv-diff">
      <AuthHeading id="h-inv-diff" title={title} sub={<T msg={m.wrongSub} values={{ email: v.email, current, b: bold }} />} />
      <p className="body">{t(m.wrongBody, { email: v.email, inviter: v.inviterUnknown ? t(m.anAdmin) : v.inviterName, current })}</p>
      {req.sent ? (
        <Note tone="ok" label={t(m.sentLabel)}>
          {req.sent.status === 'already_sent' ? t(m.alreadyAsked, { inviter: req.sent.inviter }) : t(m.requestSentFor, { inviter: req.sent.inviter, current })}
        </Note>
      ) : null}
      {req.error ? (
        <span className="err" role="alert">
          {req.error}
        </span>
      ) : null}
      <div className="tiles">
        <Button
          variant="primary"
          loading={signingOut}
          onClick={async () => {
            setSigningOut(true)
            try {
              await http.POST('/api/v1/auth/sign-out')
              qc.clear()
              await onSignedOut()
            } catch (e) {
              req.setError(errorMessage(e))
            } finally {
              setSigningOut(false)
            }
          }}
        >
          {t(m.signOutContinue)}
        </Button>
        {req.sent ? null : (
          <Button loading={req.busy} onClick={req.ask}>
            {t(m.askInstead)}
          </Button>
        )}
      </div>
    </AuthLayout>
  )
}
