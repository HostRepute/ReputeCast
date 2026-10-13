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
import { andList } from '@/lib/format'
import { useMe } from '@/lib/session'
import { useInvalidateUsers } from '../queries'

type User = Schemas['InstanceUser']

const first = (name: string) => name.split(' ')[0] ?? name
const ordinals = ['first', 'second', 'third', 'fourth', 'fifth', 'sixth', 'seventh', 'eighth', 'ninth', 'tenth']

function Eyebrow() {
  return <span className="k">Dialog · instance admin</span>
}

/** Reset two-factor (InstanceUserDialogs): reason required, verify first, confirm with your password. */
export function ResetTwoFactorDialog({ user, onClose }: { user: User; onClose: () => void }) {
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
  const blocked = !reason.trim() ? 'A reason is required' : hasPassword && !password ? 'Confirm with your password' : undefined
  const submit = async () => {
    if (blocked) return
    setError(undefined)
    try {
      await reset.mutateAsync({ params: { path: { userId: user.id } }, body: { reason: reason.trim(), password: password || undefined } })
      await invalidate()
      toast({ kind: 'success', title: `Two-factor reset for ${user.name}`, detail: `${name} was signed out and notified` })
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
      title={`Reset two-factor for ${user.name}?`}
      description={
        <span className="text-[13.5px]">
          Removes their authenticator and recovery codes. {name} signs in with a password only until they set two-factor up again; the {grace}-day grace period
          restarts. The reset is logged with your reason, and {name} is notified.
        </span>
      }
      footer={
        <>
          <Button onClick={onClose} disabled={reset.isPending}>
            Cancel
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={reset.isPending} disabledReason={blocked}>
            Reset two-factor
          </Button>
        </>
      }
    >
      <label className="lbl" htmlFor={`${id}-reason`}>
        <span>
          Reason <span className="k font-medium">· required, recorded in the audit log</span>
        </span>
        <textarea
          id={`${id}-reason`}
          className="in h-auto py-2"
          rows={3}
          placeholder="e.g. Lost phone, identity confirmed by video call on Oct 12"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          maxLength={500}
          aria-invalid={error?.field === 'reason' || undefined}
          aria-describedby={error?.field === 'reason' ? `${id}-err` : undefined}
        />
      </label>
      <Note tone="warn" label="Verify first">
        Confirm the request came from {name} (for example on a call), not from an email asking you to reset it.
      </Note>
      {hasPassword ? (
        <label className="lbl" htmlFor={`${id}-pw`}>
          Confirm with your password
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
        <Note tone="neutral">You sign in with single sign-on, so your session&apos;s two-factor confirms this reset.</Note>
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
      title={`Send a password reset to ${user.name}`}
      description={<span className="text-[13.5px]">A one-time link that works for {d?.minutesValid ?? 15} minutes, sent on the System stream.</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={send.isPending}>
            Cancel
          </Button>
          <Button
            onClick={async () => {
              const ok = await copyText(command)
              toast(ok ? { kind: 'success', title: 'CLI command copied', detail: 'run it on the server; it prints the link' } : { kind: 'error', title: "Couldn't copy", detail: 'select the command and copy it' })
            }}
          >
            Copy CLI command
          </Button>
          <Button
            variant="primary"
            loading={send.isPending}
            disabledReason={
              !d ? 'Checking the System stream…' : d.deactivatedNotice ? d.deactivatedNotice : !d.system ? 'No System-stream provider can send email yet. Use the CLI command.' : undefined
            }
            onClick={async () => {
              setError(undefined)
              try {
                const r = await send.mutateAsync({ params: { path: { userId: user.id } } })
                toast({ kind: 'success', title: `Reset link sent to ${user.email}`, detail: `via ${r.via} · works for 15 minutes` })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            Send reset email
          </Button>
        </>
      }
    >
      {info.isPending ? (
        <Skeleton w="100%" h={44} r={8} />
      ) : d?.system ? (
        healthy ? (
          <Note tone="ok" label="System stream">
            {d.provider ?? 'The System-stream provider'} is healthy · the email goes out immediately.
          </Note>
        ) : (
          <Note tone="warn" label="System stream">
            {d.provider} is {d.providerStatus?.replace('_', ' ')} · the email may be delayed. The CLI command works either way.
          </Note>
        )
      ) : (
        <Note tone="warn" label="No System provider">
          No sending provider is connected to the System stream yet{d?.provider ? ` (${d.provider} is routed but can't send system mail)` : ''}, so the email can&apos;t
          go out. Run the command below on the server instead.
        </Note>
      )}
      {d?.deactivatedNotice ? <Note tone="warn">{d.deactivatedNotice}</Note> : null}
      <div className="note n">
        <span className="col gap-1.5">
          <span>No System provider, or the user can&apos;t receive email? Run this on the server instead; it prints the same link:</span>
          <pre className="code">{command}</pre>
        </span>
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/** Make instance admin (InstanceUserDialogs). */
export function MakeAdminDialog({ user, admins, onClose }: { user: User; admins: string[]; onClose: () => void }) {
  const set = $api.useMutation('put', '/api/v1/admin/users/{userId}/instance-admin')
  const invalidate = useInvalidateUsers()
  const [error, setError] = useState<string>()
  const name = first(user.name)
  const roles = user.workspaces.filter((w) => w.status === 'active').map((w) => `${w.role} in ${w.name}`)
  const twofa =
    user.twoFactor.state === 'idp'
      ? `Two-factor is already enforced for them by ${user.ssoProvider ?? 'their identity provider'}`
      : user.twoFactor.state === 'on'
        ? `Two-factor is on (${user.twoFactor.method === 'passkey' ? 'passkey' : 'authenticator app'})`
        : `${name} doesn't use two-factor yet: ask them to add a passkey or an authenticator first`
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !set.isPending && onClose()}
      hideClose
      eyebrow={<Eyebrow />}
      title={`Make ${user.name} an instance admin?`}
      footer={
        <>
          <Button onClick={onClose} disabled={set.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            loading={set.isPending}
            onClick={async () => {
              setError(undefined)
              try {
                await set.mutateAsync({ params: { path: { userId: user.id } }, body: { admin: true } })
                await invalidate()
                toast({ kind: 'success', title: `${user.name} is an instance admin` })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            Make instance admin
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-[#374151]">
        <li>Manages users, sign-in methods, SSO, security policies and workspaces for the whole instance</li>
        <li>{roles.length ? `Does not change their workspace roles (${roles.join(', ')})` : 'Does not add them to any workspace'}</li>
        <li>{twofa}</li>
      </ul>
      <div className="note n">
        <span>
          Instance admins: {admins.length ? andList(admins) : 'none'}. {name} would be the {ordinals[admins.length] ?? `${admins.length + 1}th`}
          {admins.length === 1 ? ', which also gives you a backup for break-glass access.' : '.'}
        </span>
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/** Remove instance admin (never the last one). */
export function RemoveAdminDialog({ user, admins, onClose }: { user: User; admins: string[]; onClose: () => void }) {
  const set = $api.useMutation('put', '/api/v1/admin/users/{userId}/instance-admin')
  const invalidate = useInvalidateUsers()
  const last = admins.length <= 1
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge="Instance admin"
      title={`Remove ${user.name} as instance admin?`}
      description={
        last
          ? `${user.name} is the only instance admin. Make someone else an instance admin first, so the instance is never left without one.`
          : `${first(user.name)} keeps their workspace roles but can no longer open Instance admin. Remaining: ${andList(admins.filter((a) => a !== user.name))}.`
      }
      confirmLabel="Remove instance admin"
      confirmVariant="primary"
      onConfirm={async () => {
        if (last) throw new Error('The last instance admin can’t be removed.')
        await set.mutateAsync({ params: { path: { userId: user.id } }, body: { admin: false } })
        await invalidate()
        toast({ kind: 'success', title: `${user.name} is no longer an instance admin` })
      }}
    />
  )
}

/** Deactivate an account: consequences first, keys kept or revoked, a reason for the audit log. */
export function DeactivateUserDialog({ user, onClose }: { user: User; onClose: () => void }) {
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
      badge="Deactivate account"
      title={`Deactivate ${user.name}?`}
      description={`Signs ${name} out of ${user.sessions} ${user.sessions === 1 ? 'session' : 'sessions'} and every workspace now. Their history, campaigns and audit entries stay; you can reactivate the account later.`}
      confirmLabel="Deactivate account"
      busyLabel="Deactivating…"
      reason={{ label: 'Reason · recorded in the audit log', minLength: 3 }}
      onConfirm={async ({ reason }) => {
        if (blocked) throw new Error(`${name} is the only Owner of ${andList(user.soleOwnerOf)}. Transfer ownership first.`)
        const r = await deactivate.mutateAsync({ params: { path: { userId: user.id } }, body: { reason: reason ?? '', revokeKeys: revoke } })
        await invalidate()
        toast({
          kind: 'success',
          title: `${user.name} deactivated`,
          detail: `${r.sessions} ${r.sessions === 1 ? 'session' : 'sessions'} ended${r.keys ? ` · ${r.keys} API ${r.keys === 1 ? 'key' : 'keys'} revoked` : ''}`,
        })
      }}
    >
      <DialogRow tone="info" label="Workspaces" first>
        {active.length ? active.map((w) => `${w.name} (${w.role})`).join(', ') : 'None'}
      </DialogRow>
      {user.apiKeys > 0 ? (
        <DialogOption
          type="checkbox"
          checked={revoke}
          onChange={setRevoke}
          title={`Revoke the ${user.apiKeys} API ${user.apiKeys === 1 ? 'key' : 'keys'} and agents ${name} created`}
          description="Leave unchecked to keep integrations running; they stop at the next permission change anyway."
        />
      ) : null}
      {blocked ? (
        <Note tone="bad">
          {name} is the only Owner of {andList(user.soleOwnerOf)}. <Badge tone="bad">Blocked</Badge> Transfer ownership in the workspace&apos;s settings first.
        </Note>
      ) : null}
    </ConfirmDialog>
  )
}
