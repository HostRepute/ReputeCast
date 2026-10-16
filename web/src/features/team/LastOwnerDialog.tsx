import { useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { TransferDialog } from '../workspace/TransferDialog'
import { m } from './role.messages'

/**
 * The blocked change for the last Owner (TeamDialogs, left): a workspace
 * always keeps one Owner, so the way out is a transfer of ownership.
 */
export function LastOwnerDialog({ member, open, onOpenChange }: { member: Pick<Schemas['TeamMember'], 'name'>; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const settings = $api.useQuery('get', '/api/v1/workspaces/{workspace}/settings', { params: { path: { workspace: slug } } })
  const [transfer, setTransfer] = useState(false)
  const s = settings.data
  const suggest = s?.candidates.find((c) => c.eligible)?.name
  const first = member.name.split(' ')[0] ?? member.name
  if (transfer && s) return <TransferDialog settings={s} open onOpenChange={(o) => (o ? setTransfer(true) : onOpenChange(false))} />
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.lastOwnerEyebrow)}</span>}
      title={t(m.lastOwnerTitle, { name: member.name })}
      className="card border-bad-line"
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.close)}</Button>
          <Button
            variant="primary"
            disabledReason={s && !s.canTransfer ? t(m.onlyOwnerTransfers) : !s ? t(common.loading) : undefined}
            onClick={() => setTransfer(true)}
          >
            {t(m.transfer)}
          </Button>
        </>
      }
    >
      <div className="note b" role="alert">
        <Badge tone="bad">{t(m.cantDemote)}</Badge>
        <span>{t(m.keepsOwner)}</span>
      </div>
      <p className="m-0 text-[13.5px] text-ink-nav">
        {/* TeamDialogs names the Owner even when it's the person reading ("Linh then becomes"). */}
        {suggest ? t(m.transferFirstSuggest, { suggest, name: first }) : t(m.transferFirst, { name: first })}
      </p>
    </Dialog>
  )
}
