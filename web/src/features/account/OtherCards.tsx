import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { meQueryKey, useWorkspace } from '@/lib/session'
import { useRefreshAccount } from './AccountPage'

type Account = Schemas['AccountOverview']
type Membership = Schemas['AccountWorkspace']

/** My workspaces: my role in each, and Leave workspace (not the last Owner). */
export function WorkspacesCard({ a }: { a: Account }) {
  const [leave, setLeave] = useState<Membership>()
  return (
    <section className="card" aria-labelledby="h-workspaces">
      <CardSection id="h-workspaces" title="My workspaces" caption="Leaving removes your access; your history in the audit log stays" />
      <Table density="compact" className="[&_td]:whitespace-normal" caption="My workspaces">
        <thead>
          <tr>
            <Th>Workspace</Th>
            <Th>My role</Th>
            <Th>Members</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {a.workspaces.map((w) => (
            <tr key={w.id}>
              <Td>
                <span className="font-semibold">{w.name}</span>
              </Td>
              <Td>
                <Badge tone={w.roleKey === 'owner' ? 'purple' : 'neutral'}>{w.roleName}</Badge>
              </Td>
              <Td>{count(w.members)}</Td>
              <Td>
                <Button size="sm" disabledReason={w.canLeave ? undefined : w.leaveReason} onClick={() => setLeave(w)} aria-label={`Leave ${w.name}`}>
                  Leave workspace
                </Button>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
      <LeaveDialog workspace={leave} onClose={() => setLeave(undefined)} />
    </section>
  )
}

function LeaveDialog({ workspace, onClose }: { workspace?: Membership; onClose: () => void }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const refresh = useRefreshAccount()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  return (
    <Dialog
      open={Boolean(workspace)}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title={`Leave ${workspace?.name ?? ''}?`}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>Stay</Button>
          <Button
            variant="danger-primary"
            loading={busy}
            onClick={async () => {
              if (!workspace) return
              setBusy(true)
              setError(undefined)
              try {
                await http.POST('/api/v1/workspaces/{workspace}/leave', { params: { path: { workspace: workspace.slug } } })
                toast({ kind: 'deleted', title: `You left ${workspace.name}` })
                onClose()
                if (workspace.slug === slug) {
                  qc.removeQueries({ queryKey: ['workspace-me', slug] })
                  await qc.invalidateQueries({ queryKey: meQueryKey })
                  void navigate({ to: '/' })
                } else await refresh()
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            Leave workspace
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-ink-2">
        <li>You lose access to its campaigns, contacts, automations and settings.</li>
        <li>API keys and MCP agents you created there are revoked.</li>
        <li>Your history stays in the audit log. An admin can invite you again.</li>
      </ul>
      {error ? (
        <Note tone="bad" label="Not left">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** Summary of the notification preferences, edited on the notifications screen. */
export function NotificationPreferencesCard({ a }: { a: Account }) {
  const { slug } = useWorkspace()
  const n = a.notifications
  return (
    <section className="card" aria-labelledby="h-notif-prefs" id="notifications">
      <CardSection id="h-notif-prefs" title="Notification preferences" caption="Per event type, in-app and/or email" />
      <div className="pad col">
        <KeyValues
          items={[
            ['In-app', n.inApp],
            ['Email', n.email],
            ['Quiet hours', n.quietHours],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/notifications" params={{ workspace: slug }} hash="preferences">
            Edit preferences
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}
