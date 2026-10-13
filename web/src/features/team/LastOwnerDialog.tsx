import { useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { useWorkspace } from '@/lib/session'
import { TransferDialog } from '../workspace/TransferDialog'

/**
 * The blocked change for the last Owner (TeamDialogs, left): a workspace
 * always keeps one Owner, so the way out is a transfer of ownership.
 */
export function LastOwnerDialog({ member, open, onOpenChange }: { member: Pick<Schemas['TeamMember'], 'name'>; open: boolean; onOpenChange: (o: boolean) => void }) {
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
      eyebrow={<span className="k">Dialog · change role · blocked</span>}
      title={`${member.name} is the last Owner`}
      className="card border-bad-line"
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Close</Button>
          <Button
            variant="primary"
            disabledReason={s && !s.canTransfer ? 'Only the Owner can transfer ownership.' : !s ? 'Loading…' : undefined}
            onClick={() => setTransfer(true)}
          >
            Transfer ownership
          </Button>
        </>
      }
    >
      <div className="note b" role="alert">
        <Badge tone="bad">Can&apos;t demote</Badge>
        <span>A workspace always keeps at least one Owner, so this role can&apos;t be changed to Admin and the member can&apos;t be removed or deactivated.</span>
      </div>
      <p className="m-0 text-[13.5px] text-ink-nav">
        {/* TeamDialogs names the Owner even when it's the person reading ("Linh then becomes"). */}
        To change it, first transfer ownership to another member with two-factor enabled{suggest ? `, for example ${suggest}` : ''}. {first} then becomes an Admin
        automatically.
      </p>
    </Dialog>
  )
}
