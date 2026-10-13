import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { count, monthDay } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'

/** Delete the workspace (WorkspaceDialogs, right): consequences, export first, typed name. */
export function DeleteWorkspaceDialog({
  settings,
  open,
  onOpenChange,
}: {
  settings: Pick<Schemas['WorkspaceSettings'], 'name' | 'danger'>
  open: boolean
  onOpenChange: (o: boolean) => void
}) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const canExport = useCan('subscribers:export')
  const [typed, setTyped] = useState('')
  const del = $api.useMutation('post', '/api/v1/workspaces/{workspace}/delete')
  const d = settings.danger
  const match = typed === settings.name
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Dialog · Owner only · typed confirmation</span>}
      title={<span className="text-bad-fg">Delete {settings.name}?</span>}
      description={
        <span className="text-[13.5px]">
          This permanently deletes the workspace after a {d.restoreDays}-day hold during which an instance admin can restore it. Scheduled sends stop immediately.
        </span>
      }
      className="card border-bad-line"
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            variant="danger-primary"
            loading={del.isPending}
            disabledReason={match ? undefined : 'Type the workspace name to enable'}
            onClick={() =>
              del.mutate(
                { params: { path: { workspace: slug } }, body: { confirm: typed } },
                {
                  onSuccess: (r) => {
                    toast({ kind: 'deleted', title: `${settings.name} deleted`, detail: `an instance admin can restore it until ${monthDay(r.restoreUntil)}` })
                    void qc.invalidateQueries({ queryKey: ['me'] })
                    void navigate({ to: '/' })
                  },
                  onError: (e) => toast({ kind: 'error', title: "Couldn't delete the workspace", detail: errorMessage(e) }),
                },
              )
            }
          >
            Delete workspace
          </Button>
        </>
      }
    >
      <ul className="m-0 flex flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>
          {count(d.contacts)} contacts, {d.lists} {d.lists === 1 ? 'list' : 'lists'}, {d.segments} {d.segments === 1 ? 'segment' : 'segments'} and consent history
        </li>
        <li>
          {count(d.campaigns)} campaigns with reports, {d.automations} {d.automations === 1 ? 'automation' : 'automations'}, {d.templates}{' '}
          {d.templates === 1 ? 'template' : 'templates'}
        </li>
        <li>
          {d.providers} sending {d.providers === 1 ? 'provider' : 'providers'}, routing rules and warm-up plans (provider accounts themselves are untouched)
        </li>
        <li>API keys and MCP agents of this workspace stop working</li>
        <li>
          <b>Kept:</b> the shared suppression list, the audit log and the members&apos; accounts
        </li>
      </ul>
      <div className="note i">
        <Badge tone="info">Recommended</Badge>
        <span className="col gap-1.5">
          <span>Export everything first: contacts with consent, campaigns as MJML, reports as CSV, audit log as JSON.</span>
          <span>
            {canExport ? (
              <ButtonAnchor size="sm" href={`/api/v1/workspaces/${slug}/export`} newTab={false} download="">
                Export everything first
              </ButtonAnchor>
            ) : (
              <Button size="sm" disabledReason="Exporting contacts needs the subscribers:export permission.">
                Export everything first
              </Button>
            )}
          </span>
        </span>
      </div>
      <label className="lbl">
        <span>
          Type <b>{settings.name}</b> to confirm
        </span>
        <input className="in" type="text" value={typed} onChange={(e) => setTyped(e.target.value)} autoComplete="off" spellCheck={false} />
        <span className="k font-medium">Must match exactly.</span>
      </label>
    </Dialog>
  )
}
