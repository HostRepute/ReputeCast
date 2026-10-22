import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, DialogOption, DialogRow } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, calendarDate, count } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe } from '@/lib/session'
import { useInvalidateUsers } from '../queries'
import { m } from './messages'

type User = Schemas['InstanceUser']

const first = (name: string) => name.split(' ')[0] ?? name

function Eyebrow() {
  const t = useT()
  return <span className="k">{t(m.dialogEyebrow)}</span>
}

/** Reset two-factor (InstanceUserDialogs): reason required, verify first, confirm with your password. */
export function ResetTwoFactorDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const t = useT()
  const id = useId()
  const { data: me } = useMe()
  const policy = $api.useQuery('get', '/api/v1/admin/policy')
  const reset = $api.useMutation('post', '/api/v1/admin/users/{userId}/two-factor-reset')
  const invalidate = useInvalidateUsers()
  const [reason, setReason] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<{ field?: 'reason' | 'password'; message: string }>()
  const hasPassword = me?.user.hasPassword ?? true
  const grace = policy.data?.graceDays ?? 7
  const name = first(user.name)
  const blocked = !reason.trim() ? t(m.reasonMissing) : hasPassword && !password ? t(m.passwordMissing) : undefined
  const submit = async () => {
    if (blocked) return
    setError(undefined)
    try {
      await reset.mutateAsync({ params: { path: { userId: user.id } }, body: { reason: reason.trim(), password: password || undefined } })
      await invalidate()
      toast({ kind: 'success', title: tt(m.twoFactorReset, { name: user.name }), detail: tt(m.twoFactorResetDetail, { name }) })
      onClose()
    } catch (e) {
      const msg = errorMessage(e)
      setError({ field: /password/i.test(msg) ? 'password' : /reason/i.test(msg) ? 'reason' : undefined, message: msg })
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !reset.isPending && onClose()}
      hideClose
      dismissible={false}
      eyebrow={<Eyebrow />}
      title={t(m.resetTitle, { name: user.name })}
      description={<span className="text-[13.5px]">{t(m.resetBody, { name, days: grace })}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={reset.isPending}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={reset.isPending} disabledReason={blocked}>
            {t(m.resetTwoFactor)}
          </Button>
        </>
      }
    >
      <label className="lbl" htmlFor={`${id}-reason`}>
        <span>
          <T msg={m.reasonLabel} values={{ k: (c) => <span className="k font-medium">{c}</span> }} />
        </span>
        <textarea
          id={`${id}-reason`}
          className="in h-auto py-2"
          rows={3}
          placeholder={t(m.reasonPlaceholder, { date: calendarDate('2026-10-12', 'monthDay') })}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          maxLength={500}
          aria-invalid={error?.field === 'reason' || undefined}
          aria-describedby={error?.field === 'reason' ? `${id}-err` : undefined}
        />
      </label>
      <Note tone="warn" label={t(m.verifyFirst)}>
        {t(m.verifyBody, { name })}
      </Note>
      {hasPassword ? (
        <label className="lbl" htmlFor={`${id}-pw`}>
          {t(m.passwordLabel)}
          <input
            id={`${id}-pw`}
            className="in"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={error?.field === 'password' || undefined}
            aria-describedby={error?.field === 'password' ? `${id}-err` : undefined}
          />
        </label>
      ) : (
        <Note tone="neutral">{t(m.ssoConfirms)}</Note>
      )}
      {error ? (
        <span className="err" id={`${id}-err`} role="alert">
          {error.message}
        </span>
      ) : null}
    </Dialog>
  )
}

/** Send a password reset (InstanceUserDialogs): the System stream, or the CLI command that prints the same link. */
export function PasswordResetDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const t = useT()
  const info = $api.useQuery('get', '/api/v1/admin/users/{userId}/password-reset', { params: { path: { userId: user.id } } })
  const send = $api.useMutation('post', '/api/v1/admin/users/{userId}/password-reset')
  const [error, setError] = useState<string>()
  const d = info.data
  const healthy = !d?.providerStatus || d.providerStatus === 'healthy' || d.providerStatus === 'warming_up'
  const command = d?.command ?? `reputecast user reset-password --email ${user.email}`
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !send.isPending && onClose()}
      hideClose
      eyebrow={<Eyebrow />}
      title={t(m.passwordResetTitle, { name: user.name })}
      description={<span className="text-[13.5px]">{t(m.passwordResetBody, { minutes: d?.minutesValid ?? 15 })}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={send.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            onClick={async () => {
              const ok = await copyText(command)
              toast(
                ok
                  ? { kind: 'success', title: tt(m.cliCopied), detail: tt(m.cliCopiedDetail) }
                  : { kind: 'error', title: tt(m.copyFailed), detail: tt(m.copyFailedDetail) },
              )
            }}
          >
            {t(m.copyCli)}
          </Button>
          <Button
            variant="primary"
            loading={send.isPending}
            disabledReason={!d ? t(m.checkingSystem) : d.deactivatedNotice ? d.deactivatedNotice : !d.system ? t(m.noSystemReason) : undefined}
            onClick={async () => {
              setError(undefined)
              try {
                const r = await send.mutateAsync({ params: { path: { userId: user.id } } })
                toast({ kind: 'success', title: tt(m.resetSent, { email: user.email }), detail: tt(m.resetSentDetail, { via: r.via, minutes: 15 }) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            {t(m.sendResetEmail)}
          </Button>
        </>
      }
    >
      {info.isPending ? (
        <Skeleton w="100%" h={44} r={8} />
      ) : d?.system ? (
        healthy ? (
          <Note tone="ok" label={t(m.systemStream)}>
            {t(m.systemHealthy, { hasProvider: d.provider ? 'yes' : 'no', provider: d.provider ?? '' })}
          </Note>
        ) : (
          <Note tone="warn" label={t(m.systemStream)}>
            {t(m.systemUnhealthy, { provider: d.provider ?? '', status: d.providerStatus ?? '', statusText: d.providerStatus?.replace('_', ' ') ?? '' })}
          </Note>
        )
      ) : (
        <Note tone="warn" label={t(m.noSystemLabel)}>
          {t(m.noSystemBody, { hasProvider: d?.provider ? 'yes' : 'no', provider: d?.provider ?? '' })}
        </Note>
      )}
      {d?.deactivatedNotice ? <Note tone="warn">{d.deactivatedNotice}</Note> : null}
      <div className="note n">
        <span className="col gap-1.5">
          <span>{t(m.cliNote)}</span>
          <pre className="code">{command}</pre>
        </span>
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/** Make instance admin (InstanceUserDialogs). */
export function MakeAdminDialog({ user, admins, onClose }: { user: User; admins: string[]; onClose: () => void }) {
  const t = useT()
  const set = $api.useMutation('put', '/api/v1/admin/users/{userId}/instance-admin')
  const invalidate = useInvalidateUsers()
  const [error, setError] = useState<string>()
  const name = first(user.name)
  const roles = user.workspaces.filter((w) => w.status === 'active').map((w) => t(m.roleIn, { role: roleName({ name: w.role }), workspace: w.name }))
  const twofa =
    user.twoFactor.state === 'idp'
      ? t(m.twoFAByIdp, { hasProvider: user.ssoProvider ? 'yes' : 'no', provider: user.ssoProvider ?? '' })
      : user.twoFactor.state === 'on'
        ? t(m.twoFAIsOn, { method: user.twoFactor.method === 'passkey' ? 'passkey' : 'app' })
        : t(m.twoFANotYet, { name })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !set.isPending && onClose()}
      hideClose
      eyebrow={<Eyebrow />}
      title={t(m.makeAdminTitle, { name: user.name })}
      footer={
        <>
          <Button onClick={onClose} disabled={set.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={set.isPending}
            onClick={async () => {
              setError(undefined)
              try {
                await set.mutateAsync({ params: { path: { userId: user.id } }, body: { admin: true } })
                await invalidate()
                toast({ kind: 'success', title: tt(m.madeAdmin, { name: user.name }) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            {t(m.makeAdmin)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(m.managesInstance)}</li>
        <li>{roles.length ? t(m.keepsRoles, { roles: roles.join(', ') }) : t(m.noWorkspaces)}</li>
        <li>{twofa}</li>
      </ul>
      <div className="note n">
        <span>{t(m.adminsNote, { count: admins.length, admins: andList(admins), name, position: count(admins.length + 1) })}</span>
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/** Remove instance admin (never the last one). */
export function RemoveAdminDialog({ user, admins, onClose }: { user: User; admins: string[]; onClose: () => void }) {
  const t = useT()
  const set = $api.useMutation('put', '/api/v1/admin/users/{userId}/instance-admin')
  const invalidate = useInvalidateUsers()
  const last = admins.length <= 1
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={t(m.removeBadge)}
      title={t(m.removeTitle, { name: user.name })}
      description={
        last
          ? t(m.removeLast, { name: user.name })
          : t(m.removeBody, { name: first(user.name), remaining: andList(admins.filter((a) => a !== user.name)) })
      }
      confirmLabel={t(m.removeAdmin)}
      confirmVariant="primary"
      onConfirm={async () => {
        if (last) throw new Error(tt(m.lastAdmin))
        await set.mutateAsync({ params: { path: { userId: user.id } }, body: { admin: false } })
        await invalidate()
        toast({ kind: 'success', title: tt(m.removedAdmin, { name: user.name }) })
      }}
    />
  )
}

/** Deactivate an account: consequences first, keys kept or revoked, a reason for the audit log. */
export function DeactivateUserDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const t = useT()
  const deactivate = $api.useMutation('post', '/api/v1/admin/users/{userId}/deactivate')
  const invalidate = useInvalidateUsers()
  const [revoke, setRevoke] = useState(true)
  const name = first(user.name)
  const blocked = user.soleOwnerOf.length > 0
  const active = user.workspaces.filter((w) => w.status === 'active')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={t(m.deactivateAccount)}
      title={t(m.deactivateTitle, { name: user.name })}
      description={t(m.deactivateBody, { name, sessions: user.sessions })}
      confirmLabel={t(m.deactivateAccount)}
      busyLabel={t(m.deactivating)}
      reason={{ label: t(m.deactivateReason), minLength: 3 }}
      onConfirm={async ({ reason }) => {
        if (blocked) throw new Error(tt(m.soleOwner, { name, workspaces: andList(user.soleOwnerOf) }))
        const r = await deactivate.mutateAsync({ params: { path: { userId: user.id } }, body: { reason: reason ?? '', revokeKeys: revoke } })
        await invalidate()
        toast({
          kind: 'success',
          title: tt(m.deactivatedToast, { name: user.name }),
          detail: tt(m.deactivatedDetail, { sessions: r.sessions, keys: r.keys }),
        })
      }}
    >
      <DialogRow tone="info" label={t(m.workspacesRow)} first>
        {active.length ? active.map((w) => `${w.name} (${roleName({ name: w.role })})`).join(', ') : t(common.none)}
      </DialogRow>
      {user.apiKeys > 0 ? (
        <DialogOption
          type="checkbox"
          checked={revoke}
          onChange={setRevoke}
          title={t(m.revokeKeys, { count: user.apiKeys, name })}
          description={t(m.revokeKeysHint)}
        />
      ) : null}
      {blocked ? (
        <Note tone="bad">
          <T msg={m.soleOwnerNote} values={{ name, workspaces: andList(user.soleOwnerOf), badge: (c) => <Badge tone="bad">{c}</Badge> }} />
        </Note>
      ) : null}
    </ConfirmDialog>
  )
}
