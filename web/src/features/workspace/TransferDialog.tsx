import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useWorkspace, workspaceMeKey } from '@/lib/session'

/** Transfer ownership (WorkspaceDialogs, left): pick an eligible member and re-authenticate. */
export function TransferDialog({
  settings,
  open,
  onOpenChange,
  initialMemberId,
}: {
  settings: Pick<Schemas['WorkspaceSettings'], 'name' | 'candidates' | 'hasPassword'>
  open: boolean
  onOpenChange: (o: boolean) => void
  initialMemberId?: string
}) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const eligible = settings.candidates.filter((c) => c.eligible)
  const [memberId, setMemberId] = useState(initialMemberId ?? eligible[0]?.memberId ?? '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<{ field?: string; message: string }>()
  const target = settings.candidates.find((c) => c.memberId === memberId)
  const transfer = $api.useMutation('post', '/api/v1/workspaces/{workspace}/transfer')
  const submit = () => {
    setError(undefined)
    if (settings.hasPassword && !password) {
      setError({ field: 'password', message: 'Enter your password to confirm.' })
      return
    }
    transfer.mutate(
      { params: { path: { workspace: slug } }, body: { memberId, password: password || undefined } },
      {
        onSuccess: () => {
          toast({ title: `${target?.name} is now the Owner`, detail: 'you are an Admin' })
          void qc.invalidateQueries()
          void qc.invalidateQueries({ queryKey: workspaceMeKey(slug) })
          onOpenChange(false)
        },
        onError: (e) => {
          const field = e instanceof ApiError ? e.problem.errors?.[0]?.location?.replace('body.', '') : undefined
          setError({ field, message: errorMessage(e) })
        },
      },
    )
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Dialog · Owner only</span>}
      title={`Transfer ownership of ${settings.name}`}
      description={
        <span className="text-[13.5px]">
          The new Owner gets every permission, including deleting the workspace and transferring it again. You become an Admin and keep your break-glass sign-in as an
          instance admin.
        </span>
      }
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="primary" loading={transfer.isPending} disabledReason={target?.eligible ? undefined : 'Pick a member with two-factor.'} onClick={submit}>
            Transfer ownership
          </Button>
        </>
      }
    >
      <label className="lbl">
        New owner
        <select className="in" value={memberId} onChange={(e) => setMemberId(e.target.value)}>
          {settings.candidates.length === 0 ? <option value="">No other active members yet</option> : null}
          {settings.candidates.map((c) => (
            <option key={c.memberId} value={c.memberId} disabled={!c.eligible}>
              {c.name} · {c.roleName} · {c.email}
              {c.eligible ? '' : ' (no two-factor)'}
            </option>
          ))}
        </select>
        <span className="k font-medium">Only active members with two-factor can become Owner.</span>
      </label>
      {target?.note ? (
        <div className="note w">
          <Badge tone="warn">2FA</Badge>
          <span>{target.note}</span>
        </div>
      ) : target?.reason ? (
        <div className="note b" role="alert">
          <Badge tone="bad">2FA</Badge>
          <span>{target.reason} Ask them to add a passkey or an authenticator app first.</span>
        </div>
      ) : null}
      {settings.hasPassword ? (
        <label className="lbl">
          Confirm with your password
          <input
            className="in"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={error?.field === 'password' || undefined}
            aria-describedby="transfer-pw-hint"
          />
          {error?.field === 'password' ? (
            <span className="err" role="alert">
              {error.message}
            </span>
          ) : null}
          <span className="k font-medium" id="transfer-pw-hint">
            Re-authentication is required for this action.
          </span>
        </label>
      ) : (
        <span className="k">You sign in through single sign-on: the transfer is accepted within 10 minutes of signing in.</span>
      )}
      {error && error.field !== 'password' ? (
        <div className="note b" role="alert">
          <span>{error.message}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
