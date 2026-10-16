import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { SidebarSectionsCard } from '@/components/shell/SidebarSectionsCard'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { KeyValues } from '@/components/ui/Data'
import { toast } from '@/components/ui/Toast'
import { PolicyCard as AccessibilityCard } from '@/features/accessibility/PolicyCard'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { andList, count } from '@/lib/format'
import { useWorkspace, workspaceMeKey } from '@/lib/session'
import { DeleteWorkspaceDialog } from './DeleteWorkspaceDialog'
import { TransferDialog } from './TransferDialog'
import { timezoneOptions } from './timezones'

/**
 * Cards at least 460px wide; on phones the board lays them out two by two
 * (its narrow-screen rule: columns of at least 140px).
 */
const cardGrid = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(460px,100%),1fr))] max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]'

/** Settings › General (WorkspaceSettings, WorkspaceDialogs). */
export function GeneralPage() {
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/settings', { params: { path: { workspace: slug } } })
  return (
    <Page title="Settings" subtitle={me.workspace.name} tabs={settingsTabs(slug)}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load the workspace settings" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : q.isPending ? (
        <GeneralSkeleton />
      ) : (
        <General s={q.data} />
      )}
    </Page>
  )
}

function General({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const [transfer, setTransfer] = useState(false)
  const [del, setDel] = useState(false)
  const ownerOnly = 'Only the Owner can do this.'
  return (
    <div className={cardGrid}>
      {/* Saved values come back from the server; remounting resets the form to them. */}
      <WorkspaceCard key={`${s.name}|${s.timezone}|${s.defaultSenderId ?? ''}`} s={s} />
      <SidebarSectionsCard s={s} />
      <AccessibilityCard />
      <ApprovalsCard s={s} />
      <MembersCard s={s} />
      <section className="card" style={{ borderColor: '#FCA5A5' }} aria-labelledby="danger-h">
        <CardSection title={<span id="danger-h">Danger zone</span>} caption="Owner only · both actions are logged" />
        <div className="pad col gap-3.5">
          <div className="row justify-between rounded-lg border border-line px-3.5 py-3">
            <div className="col gap-0.5">
              <span className="font-semibold">Transfer ownership</span>
              <span className="k">Make another member the Owner. You become an Admin. Only the Owner can do this.</span>
            </div>
            <Button size="sm" disabledReason={s.canTransfer ? undefined : ownerOnly} onClick={() => setTransfer(true)}>
              Transfer ownership
            </Button>
          </div>
          <div className="row justify-between rounded-lg border border-bad-line bg-bad-tint px-3.5 py-3">
            <div className="col gap-0.5">
              <span className="font-semibold text-bad-fg">Delete this workspace</span>
              <span className="k">
                Deletes {count(s.danger.contacts)} contacts, {count(s.danger.campaigns)} campaigns, {count(s.danger.automations)} automations, providers and routing.
                Suppressions are kept at instance level. You&apos;ll be asked to type the name and offered an export first.
              </span>
            </div>
            <Button size="sm" variant="danger" disabledReason={s.canDelete ? undefined : ownerOnly} onClick={() => setDel(true)}>
              Delete workspace
            </Button>
          </div>
        </div>
      </section>
      {transfer ? <TransferDialog settings={s} open onOpenChange={setTransfer} /> : null}
      {del ? <DeleteWorkspaceDialog settings={s} open onOpenChange={setDel} /> : null}
    </div>
  )
}

function WorkspaceCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [name, setName] = useState(s.name)
  const [tz, setTz] = useState(s.timezone)
  const [sender, setSender] = useState(s.defaultSenderId ?? '')
  const [error, setError] = useState<{ field?: string; message: string }>()
  const zones = useMemo(() => timezoneOptions(tz), [tz])
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/settings')
  const current = s.senders.find((x) => x.id === sender)
  const dirty = name !== s.name || tz !== s.timezone || sender !== (s.defaultSenderId ?? '')
  const reason = s.canManage ? undefined : s.reason
  const submit = () => {
    setError(undefined)
    if (!name.trim()) {
      setError({ field: 'name', message: 'Give the workspace a name.' })
      return
    }
    save.mutate(
      { params: { path: { workspace: slug } }, body: { name: name.trim(), timezone: tz, defaultSenderId: sender || undefined } },
      {
        onSuccess: () => {
          toast({ title: 'Workspace settings saved' })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/settings'] })
          void qc.invalidateQueries({ queryKey: workspaceMeKey(slug) })
          void qc.invalidateQueries({ queryKey: ['me'] })
        },
        onError: (e) => setError({ message: errorMessage(e) }),
      },
    )
  }
  return (
    <section className="card" aria-labelledby="ws-h">
      <CardSection title={<span id="ws-h">Workspace</span>} caption="Name, default sender and timezone" />
      <div className="pad col gap-3.5">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            Workspace name
            <input
              className="in"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={!s.canManage}
              maxLength={80}
              aria-invalid={error?.field === 'name' || undefined}
              aria-describedby={error?.field === 'name' ? 'ws-name-err' : undefined}
            />
            {error?.field === 'name' ? (
              <span className="err" id="ws-name-err" role="alert">
                {error.message}
              </span>
            ) : null}
          </label>
          <label className="lbl">
            Timezone
            <select className="in" value={tz} onChange={(e) => setTz(e.target.value)} disabled={!s.canManage}>
              {zones.map((z) => (
                <option key={z.value} value={z.value}>
                  {z.label}
                </option>
              ))}
            </select>
            <span className="k font-medium">Schedules and reports use this timezone.</span>
          </label>
        </div>
        <label className="lbl">
          Default sender
          <select className="in" value={sender} onChange={(e) => setSender(e.target.value)} disabled={!s.canManage || s.senders.length === 0}>
            {s.senders.length === 0 ? <option value="">No senders yet</option> : null}
            {s.senders.map((x) => (
              <option key={x.id} value={x.id}>
                {x.name} &lt;{x.email}&gt;
              </option>
            ))}
          </select>
          <span className="k font-medium">
            {s.senders.length === 0 ? 'Add a sending domain and sender in ' : 'New campaigns start with this sender. '}
            {current && current.verifiedOn.length > 0 ? `Verified on ${andList(current.verifiedOn)}; ` : ''}
            <Link to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
              senders &amp; domains
            </Link>
            .
          </span>
        </label>
        <label className="lbl">
          Workspace ID
          <input className="in mono max-w-[320px] bg-ground" value={s.slug} readOnly />
          <span className="k font-medium">Used in API URLs and the audit log.</span>
        </label>
        {error && !error.field ? (
          <div className="note b" role="alert">
            <span>{error.message}</span>
          </div>
        ) : null}
        <div className="row justify-end">
          <Button variant="primary" size="sm" loading={save.isPending} disabledReason={reason} onClick={submit}>
            Save
          </Button>
        </div>
        <UnsavedChangesGuard dirty={s.canManage && dirty && !save.isPending} description="Your changes to the workspace name, timezone or default sender aren't saved." />
      </div>
    </section>
  )
}

function ApprovalsCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const { slug } = useWorkspace()
  const a = s.approvals
  return (
    <section className="card" aria-labelledby="appr-h">
      <CardSection title={<span id="appr-h">Send approvals</span>} caption="Who must sign off before a risky send" />
      <div className="pad col gap-3">
        <KeyValues
          items={[
            ['Approval needed when', a.rules.length ? a.rules.join(' · ') : 'Never: no rule is on'],
            [
              'Approvers',
              `Members with campaigns:approve: ${a.approvers.join(', ') || 'nobody yet'} · ${a.approvalsRequired} ${a.approvalsRequired === 1 ? 'approval' : 'approvals'} required`,
            ],
            ['Automations', a.automations ? 'Publishing needs approval' : 'Publishing does not need approval'],
            ['Emergency bypass', a.bypass ? 'Owners, with a reason · logged, approvers notified' : 'Off'],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/settings/approvals" params={{ workspace: slug }}>
            Edit approval policy
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
            Approval inbox · {a.pending} pending
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function MembersCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const { slug } = useWorkspace()
  const m = s.members
  const roles = m.builtin.join(', ') + (m.custom.length ? ` + custom ${m.custom.map((c) => `“${c}”`).join(', ')}` : '')
  return (
    <section className="card" aria-labelledby="mem-h">
      <CardSection title={<span id="mem-h">Members and access</span>} caption="Summary · managed in Team and Roles" />
      <div className="pad col gap-3">
        <KeyValues
          items={[
            ['Members', `${m.active} active · ${m.invited} invited · ${m.deactivated} deactivated`],
            ['Roles', roles],
            ['Sign-in', m.signIn],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/settings/team" params={{ workspace: slug }}>
            Team
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
            Roles &amp; permissions
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function GeneralSkeleton() {
  return (
    <div className={cardGrid} aria-busy="true">
      <span className="vh" role="status">
        Loading the workspace settings
      </span>
      {[5, 3, 4, 3, 2].map((rows, i) => (
        <section key={i} className="card">
          <div className="sec">
            <div className="flex flex-col gap-1.5">
              <Skeleton w={120} h={14} />
              <Skeleton w={200} h={10} />
            </div>
          </div>
          <div className="pad col gap-3">
            {Array.from({ length: rows }, (_, j) => (
              <Skeleton key={j} h={j === 0 ? 36 : 12} w={j === 0 ? '100%' : `${70 - j * 8}%`} />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
