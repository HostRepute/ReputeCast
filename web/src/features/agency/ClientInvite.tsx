import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useId, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { AuthHeading, PasskeyGlyph, SocialButtons } from '@/features/auth/AuthParts'
import { m as im } from '@/features/auth/invite.messages'
import { passkeyErrorMessage, registerPasskey } from '@/features/auth/passkey'
import { useT } from '@/i18n'
import { roleName } from '@/lib/roles'
import { meQueryKey } from '@/lib/session'
import { m } from './messages'
import { dayMonth } from './parts'

type View = Schemas['InvitationView']

/**
 * Whether the invitation page shows its client variant (AgencyStates): a
 * new account for a client of the agency, with nothing that forces another
 * way in (a domain's SSO).
 */
export function isClientInvite(v: View): boolean {
  return Boolean(v.agency && !v.signedInAs && !v.accountExists && !v.sso && (v.agency.passkey || v.passwordAllowed))
}

/**
 * The invitation a client's person opens (AgencyStates): the agency's logo
 * and address (the instance's branding), the role in one line, and a
 * passkey first; a password stays one click away.
 */
export function ClientInviteAccept({ v, token, onJoined }: { v: View; token: string; onJoined: (j: Schemas['InvitationAccepted']) => void }) {
  const t = useT()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const agency = v.agency!
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [withPassword, setWithPassword] = useState(!agency.passkey)
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; password?: string; form?: string }>({})
  const nameErr = useId()
  const pwErr = useId()
  const pwHint = useId()
  const min = v.passwordPolicy.minLength
  const title = t(m.inviteTitle, { inviter: v.inviterName, workspace: v.workspaceName, role: roleName({ name: v.roleName }) })

  const joined = (data: Schemas['InvitationAccepted']) => {
    if (data.hasSecondFactor) {
      toast({ kind: 'success', title: t(im.joinedToast, { workspace: data.workspaceName }), detail: t(im.joinedDetail, { role: roleName({ name: data.roleName }) }) })
      void navigate({ to: '/w/$workspace', params: { workspace: data.workspace } })
    } else onJoined(data)
  }
  const accept = async (body: Schemas['InvitationAcceptInput']) => {
    const { data } = await http.POST('/api/v1/invitations/{token}/accept', { params: { path: { token } }, body })
    await qc.invalidateQueries({ queryKey: meQueryKey })
    return data!
  }
  const failed = (err: unknown) => {
    if (err instanceof ApiError) {
      const field = err.problem.errors?.find((x) => x.location === 'body.password' || x.location === 'body.name')
      if (field) {
        setErrors(field.location === 'body.name' ? { name: field.message } : { password: field.message })
        return
      }
    }
    setErrors({ form: errorMessage(err) })
  }

  const join = async (e: FormEvent) => {
    e.preventDefault()
    const next: typeof errors = {}
    if (!name.trim()) next.name = t(im.nameRequired)
    if (withPassword && [...password].length < min) next.password = t(im.tooShort, { min })
    setErrors(next)
    if (next.name || next.password) return
    setBusy(true)
    try {
      if (withPassword) {
        joined(await accept({ name: name.trim(), password }))
        return
      }
      const data = await accept({ name: name.trim(), passkey: true })
      try {
        await registerPasskey()
        joined({ ...data, hasSecondFactor: true })
      } catch (err) {
        // The account exists and is signed in, without a passkey yet:
        // Account & security adds one.
        toast({ kind: 'error', title: t(m.invitePasskeyLater, { workspace: data.workspaceName }), detail: passkeyErrorMessage(err, 'add') })
        void navigate({ to: '/w/$workspace/account', params: { workspace: data.workspace } })
      }
    } catch (err) {
      failed(err)
    } finally {
      setBusy(false)
    }
  }

  const socialHref = (p: string) => `/api/v1/auth/social/${p}/start?invite=${encodeURIComponent(token)}`
  return (
    <AuthLayout title={title} labelledBy="h-cinv">
      <AuthHeading id="h-cinv" title={title} />
      {agency.roleLine ? <p className="m-0 text-ink-2">{agency.roleLine}</p> : null}
      <span className="k">{t(m.inviteMeta, { email: v.email, date: dayMonth(v.expiresAt) })}</span>
      {v.message ? (
        <Note tone="info" label={t(im.fromInviter, { name: v.inviterName })}>
          {v.message}
        </Note>
      ) : null}
      {errors.form ? (
        <Note tone="bad" label={t(im.notJoinedLabel)}>
          {errors.form}
        </Note>
      ) : null}
      <form className="flex flex-col gap-4" onSubmit={join} noValidate>
        <label className="lbl">
          {t(im.yourName)}
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
          {t(im.email)}
          <input className="in" type="email" autoComplete="username" value={v.email} readOnly />
        </label>
        {withPassword ? (
          <label className="lbl">
            {t(im.createPassword)}
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
              {v.passwordPolicy.breachedCheck ? t(im.hintBreached, { min }) : t(im.hintPassphrase, { min })}
            </span>
          </label>
        ) : null}
        <Button type="submit" variant="primary" loading={busy} className="min-h-[42px]">
          {withPassword ? (
            t(im.join, { workspace: v.workspaceName })
          ) : (
            <>
              <PasskeyGlyph size={15} />
              {t(m.inviteCreatePasskey)}
            </>
          )}
        </Button>
        {agency.passkey && v.passwordAllowed ? (
          <Button
            className="min-h-[42px]"
            onClick={() => {
              setErrors({})
              setWithPassword(!withPassword)
            }}
          >
            {withPassword ? t(m.inviteUsePasskey) : t(m.inviteUsePassword)}
          </Button>
        ) : null}
      </form>
      {v.social.length > 0 ? (
        <>
          <div className="sep">{t(im.orJoinWith)}</div>
          <SocialButtons providers={v.social} href={socialHref} />
        </>
      ) : null}
      <span className="k text-center">{t(m.inviteManaged, { workspace: v.workspaceName, agency: agency.name })}</span>
    </AuthLayout>
  )
}
