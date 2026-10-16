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
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { meQueryKey, useWorkspace } from '@/lib/session'
import { useRefreshAccount } from './AccountPage'
import { m } from './messages'

type Account = Schemas['AccountOverview']
type Membership = Schemas['AccountWorkspace']

/** My workspaces: my role in each, and Leave workspace (not the last Owner). */
export function WorkspacesCard({ a }: { a: Account }) {
  const t = useT()
  const [leave, setLeave] = useState<Membership>()
  return (
    <section className="card" aria-labelledby="h-workspaces">
      <CardSection id="h-workspaces" title={t(m.myWorkspaces)} caption={t(m.workspacesCaption)} />
      <Table density="compact" className="[&_td]:whitespace-normal" caption={t(m.myWorkspaces)}>
        <thead>
          <tr>
            <Th>{t(m.workspace)}</Th>
            <Th>{t(m.myRole)}</Th>
            <Th>{t(m.members)}</Th>
            <Th>
              <span className="vh">{t(m.actions)}</span>
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
                <Badge tone={w.roleKey === 'owner' ? 'purple' : 'neutral'}>{roleName({ key: w.roleKey, name: w.roleName })}</Badge>
              </Td>
              <Td>{count(w.members)}</Td>
              <Td>
                <Button size="sm" disabledReason={w.canLeave ? undefined : w.leaveReason} onClick={() => setLeave(w)} aria-label={t(m.leaveNamed, { name: w.name })}>
                  {t(m.leaveWorkspace)}
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
  const t = useT()
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
      title={t(m.leaveTitle, { name: workspace?.name ?? '' })}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>{t(m.stay)}</Button>
          <Button
            variant="danger-primary"
            loading={busy}
            onClick={async () => {
              if (!workspace) return
              setBusy(true)
              setError(undefined)
              try {
                await http.POST('/api/v1/workspaces/{workspace}/leave', { params: { path: { workspace: workspace.slug } } })
                toast({ kind: 'deleted', title: t(m.left, { name: workspace.name }) })
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
            {t(m.leaveWorkspace)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-ink-2">
        <li>{t(m.leaveAccess)}</li>
        <li>{t(m.leaveKeys)}</li>
        <li>{t(m.leaveHistory)}</li>
      </ul>
      {error ? (
        <Note tone="bad" label={t(m.notLeft)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** Summary of the notification preferences, edited on the notifications screen. */
export function NotificationPreferencesCard({ a }: { a: Account }) {
  const t = useT()
  const { slug } = useWorkspace()
  const n = a.notifications
  return (
    <section className="card" aria-labelledby="h-notif-prefs" id="notifications">
      {/* The summary comes from the server, in the member's language. */}
      <CardSection id="h-notif-prefs" title={t(m.notificationPreferences)} caption={t(m.notificationsCaption)} />
      <div className="pad col">
        <KeyValues
          items={[
            [t(m.inApp), n.inApp],
            [t(m.email), n.email],
            [t(m.quietHours), n.quietHours],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/notifications" params={{ workspace: slug }} hash="preferences">
            {t(m.editPreferences)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}
