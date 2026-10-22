import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { roleName } from '@/lib/roles'
import { useWorkspace, workspaceMeKey } from '@/lib/session'
import { m } from './messages'

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
  const t = useT()
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
      setError({ field: 'password', message: t(m.passwordRequired) })
      return
    }
    transfer.mutate(
      { params: { path: { workspace: slug } }, body: { memberId, password: password || undefined } },
      {
        onSuccess: () => {
          toast({ title: t(m.transferred, { name: target?.name ?? '' }), detail: t(m.transferredDetail) })
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
      eyebrow={<span className="k">{t(m.transferEyebrow)}</span>}
      title={t(m.transferDialogTitle, { name: settings.name })}
      description={<span className="text-[13.5px]">{t(m.transferDescription)}</span>}
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={transfer.isPending} disabledReason={target?.eligible ? undefined : t(m.transferPick)} onClick={submit}>
            {t(m.transferConfirm)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(m.newOwner)}
        <select className="in" value={memberId} onChange={(e) => setMemberId(e.target.value)}>
          {settings.candidates.length === 0 ? <option value="">{t(m.noCandidates)}</option> : null}
          {settings.candidates.map((c) => (
            <option key={c.memberId} value={c.memberId} disabled={!c.eligible}>
              {c.eligible
                ? `${c.name} · ${roleName({ name: c.roleName })} · ${c.email}`
                : t(m.candidateNoTwoFactor, { name: c.name, role: roleName({ name: c.roleName }), email: c.email })}
            </option>
          ))}
        </select>
        <span className="k font-medium">{t(m.transferHint)}</span>
      </label>
      {target?.note ? (
        <div className="note w">
          <Badge tone="warn">{'2FA'}</Badge>
          <span>{target.note}</span>
        </div>
      ) : target?.reason ? (
        <div className="note b" role="alert">
          <Badge tone="bad">{'2FA'}</Badge>
          <span>{t(m.askTwoFactor, { reason: target.reason })}</span>
        </div>
      ) : null}
      {settings.hasPassword ? (
        <label className="lbl">
          {t(m.passwordLabel)}
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
            {t(m.reauthHint)}
          </span>
        </label>
      ) : (
        <span className="k">{t(m.ssoNote, { minutes: 10 })}</span>
      )}
      {error && error.field !== 'password' ? (
        <div className="note b" role="alert">
          <span>{error.message}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
