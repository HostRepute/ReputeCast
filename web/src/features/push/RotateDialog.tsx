import { useId, useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useInvalidatePush } from './common'
import { m } from './messages'

function AlertIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

/**
 * Rotating the VAPID keys (board PushStates, "Rotating the keys"): every
 * subscription ends, so the count is typed to confirm.
 */
export function RotateDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const id = useId()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  const [typed, setTyped] = useState('')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/keys/rotation', { params: { path: { workspace: slug } } }, { enabled: open })
  const rotate = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/keys/rotate')
  const r = q.data
  const n = r?.subscriptions ?? 0
  const blocked = !r ? t(m.rotateLoading) : typed.trim() !== r.confirm ? t(m.rotateType, { n: r.confirm }) : undefined
  const close = (next: boolean) => {
    if (rotate.isPending) return
    if (!next) setTyped('')
    onOpenChange(next)
  }
  const confirm = () =>
    r &&
    rotate.mutate(
      { params: { path: { workspace: slug } }, body: { confirm: typed.trim() } },
      {
        onSuccess: () => {
          void invalidate()
          toast({ kind: 'success', title: t(m.rotated, { count: n }) })
          close(false)
        },
      },
    )
  return (
    <Dialog
      open={open}
      onOpenChange={close}
      dismissible={false}
      hideClose
      eyebrow={<span className="k">{t(m.rotateEyebrow)}</span>}
      title={t(m.rotateTitle)}
      footer={
        <>
          <Button onClick={() => close(false)} disabled={rotate.isPending}>
            {t(m.cancel)}
          </Button>
          <Button variant="danger-primary" onClick={confirm} loading={rotate.isPending} busyLabel={t(m.rotating)} disabledReason={blocked}>
            {n ? t(m.rotateConfirm, { count: n }) : t(m.rotateConfirmNone)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <div className="note b" role="alert">
          <AlertIcon />
          <div className="px-note-body">
            <b>{r ? t(m.rotateEnd, { count: n }) : <Skeleton w={180} h={13} />}</b>
            <div>{t(m.rotateEndText)}</div>
          </div>
        </div>
        {r ? (
          <dl className="kv">
            {r.sites.map((s) => (
              <div key={s.host} className="contents">
                <dt>{s.host}</dt>
                <dd>{t(m.rotateSite, { count: s.subscriptions })}</dd>
              </div>
            ))}
            {r.scheduled.map((name, i) => (
              <div key={name + i} className="contents">
                <dt>{i === 0 ? t(m.rotateScheduled) : <span className="vh">{t(m.rotateScheduled)}</span>}</dt>
                <dd>{t(m.rotateReachNobody, { name })}</dd>
              </div>
            ))}
          </dl>
        ) : q.isError ? (
          <span className="err" role="alert">
            {errorMessage(q.error)}
          </span>
        ) : (
          <Skeleton w="70%" h={12} />
        )}
        {r ? (
          <div className="lbl">
            <label htmlFor={`${id}-typed`}>{t(m.rotateType, { n: r.confirm })}</label>
            <input
              id={`${id}-typed`}
              className="in"
              value={typed}
              placeholder={r.confirm}
              onChange={(e) => setTyped(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              inputMode="numeric"
            />
          </div>
        ) : null}
        {rotate.isError ? (
          <span className="err" role="alert">
            {errorMessage(rotate.error)}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}
