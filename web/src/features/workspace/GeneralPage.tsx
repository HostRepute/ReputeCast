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
import { AgencyGeneralCard } from '@/features/agency/GeneralCard'
import { LanguagesCard } from '@/features/languages/LanguagesCard'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useWorkspace, workspaceMeKey } from '@/lib/session'
import { DeleteWorkspaceDialog } from './DeleteWorkspaceDialog'
import { m } from './messages'
import { TransferDialog } from './TransferDialog'
import { timezoneOptions } from './timezones'

/**
 * Cards at least 460px wide; on phones the board lays them out two by two
 * (its narrow-screen rule: columns of at least 140px).
 */
const cardGrid = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(460px,100%),1fr))] max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]'

/** Settings › General (WorkspaceSettings, WorkspaceDialogs). */
export function GeneralPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/settings', { params: { path: { workspace: slug } } })
  return (
    <Page title={t(m.title)} subtitle={me.workspace.name} tabs={settingsTabs(slug)}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.nothingChanged)}
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
  const t = useT()
  const [transfer, setTransfer] = useState(false)
  const [del, setDel] = useState(false)
  const ownerOnly = t(m.ownerOnly)
  return (
    <div className={cardGrid}>
      {/* An agency's client: who manages it, and the client details for its staff (agency accounts). */}
      <AgencyGeneralCard />
      {/* Saved values come back from the server; remounting resets the form to them. */}
      <WorkspaceCard key={`${s.name}|${s.timezone}|${s.defaultSenderId ?? ''}`} s={s} />
      <SidebarSectionsCard s={s} />
      <LanguagesCard className="col-span-full" />
      <AccessibilityCard />
      <ApprovalsCard s={s} />
      <MembersCard s={s} />
      <RetentionCard key={`${s.retention.unconfirmedEraseDays}|${s.retention.eventDetailDays}`} s={s} />
      <section className="card" style={{ borderColor: '#FCA5A5' }} aria-labelledby="danger-h">
        <CardSection title={<span id="danger-h">{t(m.dangerTitle)}</span>} caption={t(m.dangerCaption)} />
        <div className="pad col gap-3.5">
          <div className="row justify-between rounded-lg border border-line px-3.5 py-3">
            <div className="col gap-0.5">
              <span className="font-semibold">{t(m.transferTitle)}</span>
              <span className="k">{t(m.transferBody)}</span>
            </div>
            <Button size="sm" disabledReason={s.canTransfer ? undefined : ownerOnly} onClick={() => setTransfer(true)}>
              {t(m.transferButton)}
            </Button>
          </div>
          <div className="row justify-between rounded-lg border border-bad-line bg-bad-tint px-3.5 py-3">
            <div className="col gap-0.5">
              <span className="font-semibold text-bad-fg">{t(m.deleteTitle)}</span>
              <span className="k">
                {t(m.deleteBody, {
                  contacts: s.danger.contacts,
                  contactsText: count(s.danger.contacts),
                  campaigns: s.danger.campaigns,
                  campaignsText: count(s.danger.campaigns),
                  automations: s.danger.automations,
                })}
              </span>
            </div>
            <Button size="sm" variant="danger" disabledReason={s.canDelete ? undefined : ownerOnly} onClick={() => setDel(true)}>
              {t(m.deleteButton)}
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
  const t = useT()
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
      setError({ field: 'name', message: t(m.nameRequired) })
      return
    }
    save.mutate(
      { params: { path: { workspace: slug } }, body: { name: name.trim(), timezone: tz, defaultSenderId: sender || undefined } },
      {
        onSuccess: () => {
          toast({ title: t(m.saved) })
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
      <CardSection title={<span id="ws-h">{t(m.workspaceTitle)}</span>} caption={t(m.workspaceCaption)} />
      <div className="pad col gap-3.5">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            {t(m.nameLabel)}
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
            {t(m.timezoneLabel)}
            <select className="in" value={tz} onChange={(e) => setTz(e.target.value)} disabled={!s.canManage}>
              {zones.map((z) => (
                <option key={z.value} value={z.value}>
                  {z.label}
                </option>
              ))}
            </select>
            <span className="k font-medium">{t(m.timezoneHint)}</span>
          </label>
        </div>
        <label className="lbl">
          {t(m.senderLabel)}
          <select className="in" value={sender} onChange={(e) => setSender(e.target.value)} disabled={!s.canManage || s.senders.length === 0}>
            {s.senders.length === 0 ? <option value="">{t(m.noSenders)}</option> : null}
            {s.senders.map((x) => (
              <option key={x.id} value={x.id}>
                {`${x.name} <${x.email}>`}
              </option>
            ))}
          </select>
          <span className="k font-medium">
            <T
              msg={s.senders.length === 0 ? m.senderHintEmpty : current && current.verifiedOn.length > 0 ? m.senderHintVerified : m.senderHint}
              values={{
                providers: current ? andList(current.verifiedOn) : '',
                link: (c) => (
                  <Link to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          </span>
        </label>
        <label className="lbl">
          {t(m.idLabel)}
          <input className="in mono max-w-[320px] bg-ground" value={s.slug} readOnly />
          <span className="k font-medium">{t(m.idHint)}</span>
        </label>
        {error && !error.field ? (
          <div className="note b" role="alert">
            <span>{error.message}</span>
          </div>
        ) : null}
        <div className="row justify-end">
          <Button variant="primary" size="sm" loading={save.isPending} disabledReason={reason} onClick={submit}>
            {t(common.save)}
          </Button>
        </div>
        <UnsavedChangesGuard dirty={s.canManage && dirty && !save.isPending} description={t(m.unsaved)} />
      </div>
    </section>
  )
}

const unconfirmedChoices = [0, 14, 30, 60, 90] as const
const detailChoices = [30, 90, 180, 365, 730, 0] as const

/** Data retention (review 2026-11 DP-4, DP-6): how long unconfirmed signups and open and click details are kept. */
function RetentionCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [unconfirmed, setUnconfirmed] = useState<number>(s.retention.unconfirmedEraseDays)
  const [detail, setDetail] = useState<number>(s.retention.eventDetailDays)
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/settings/retention')
  const dirty = unconfirmed !== s.retention.unconfirmedEraseDays || detail !== s.retention.eventDetailDays
  const submit = () => {
    setError(undefined)
    save.mutate(
      {
        params: { path: { workspace: slug } },
        body: {
          unconfirmedEraseDays: unconfirmed as Schemas['WorkspaceRetention']['unconfirmedEraseDays'],
          eventDetailDays: detail as Schemas['WorkspaceRetention']['eventDetailDays'],
        },
      },
      {
        onSuccess: () => {
          toast({ title: t(m.saved) })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/settings'] })
        },
        onError: (e) => setError(errorMessage(e)),
      },
    )
  }
  return (
    <section className="card" aria-labelledby="retention-h">
      <CardSection title={<span id="retention-h">{t(m.retentionTitle)}</span>} caption={t(m.retentionCaption)} />
      <div className="pad col gap-3.5">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            {t(m.unconfirmedLabel)}
            <select className="in" value={unconfirmed} onChange={(e) => setUnconfirmed(Number(e.target.value))} disabled={!s.canManage}>
              {unconfirmedChoices.map((d) => (
                <option key={d} value={d}>
                  {d === 0 ? t(m.retentionNever) : t(m.retentionDays, { n: d })}
                </option>
              ))}
            </select>
            <span className="k font-medium">{t(m.unconfirmedHint)}</span>
          </label>
          <label className="lbl">
            {t(m.detailLabel)}
            <select className="in" value={detail} onChange={(e) => setDetail(Number(e.target.value))} disabled={!s.canManage}>
              {detailChoices.map((d) => (
                <option key={d} value={d}>
                  {d === 0 ? t(m.retentionKeep) : d < 180 ? t(m.retentionDays, { n: d }) : t(m.retentionMonths, { n: Math.round(d / 30.4) })}
                </option>
              ))}
            </select>
            <span className="k font-medium">{t(m.detailHint)}</span>
          </label>
        </div>
        {error ? (
          <div className="note b" role="alert">
            <span>{error}</span>
          </div>
        ) : null}
        <div className="row justify-end">
          <Button variant="primary" size="sm" loading={save.isPending} disabledReason={s.canManage ? undefined : s.reason} onClick={submit}>
            {t(common.save)}
          </Button>
        </div>
        <UnsavedChangesGuard dirty={s.canManage && dirty && !save.isPending} description={t(m.retentionUnsaved)} />
      </div>
    </section>
  )
}

function ApprovalsCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = s.approvals
  return (
    <section className="card" aria-labelledby="appr-h">
      <CardSection title={<span id="appr-h">{t(m.approvalsTitle)}</span>} caption={t(m.approvalsCaption)} />
      <div className="pad col gap-3">
        <KeyValues
          items={[
            // The rules come from the server in words ("audience ≥ 10,000").
            [t(m.approvalNeeded), a.rules.length ? a.rules.join(' · ') : t(m.noRule)],
            [
              t(m.approvers),
              t(m.approversValue, { hasApprovers: a.approvers.length ? 'yes' : 'no', names: a.approvers.join(', '), count: a.approvalsRequired }),
            ],
            [t(m.automations), a.automations ? t(m.automationsOn) : t(m.automationsOff)],
            [t(m.bypass), a.bypass ? t(m.bypassOn) : t(common.off)],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/settings/approvals" params={{ workspace: slug }}>
            {t(m.editPolicy)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
            {t(m.inbox, { count: count(a.pending) })}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function MembersCard({ s }: { s: Schemas['WorkspaceSettings'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const mb = s.members
  // The built-in roles' names read in the app language; custom roles keep theirs.
  const builtin = mb.builtin.map((name) => roleName({ name, builtin: true })).join(', ')
  const roles = mb.custom.length ? t(m.rolesCustom, { builtin, custom: mb.custom.map((c) => `“${c}”`).join(', ') }) : builtin
  return (
    <section className="card" aria-labelledby="mem-h">
      <CardSection title={<span id="mem-h">{t(m.membersTitle)}</span>} caption={t(m.membersCaption)} />
      <div className="pad col gap-3">
        <KeyValues
          items={[
            [t(m.members), t(m.membersValue, { active: count(mb.active), invited: count(mb.invited), deactivated: count(mb.deactivated) })],
            [t(m.roles), roles],
            // The sign-in summary comes from the server ("Google Workspace SSO enforced for acme.io").
            [t(m.signIn), mb.signIn],
          ]}
        />
        <div className="row">
          <ButtonLink size="sm" to="/w/$workspace/settings/team" params={{ workspace: slug }}>
            {t(m.teamButton)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/settings/roles" params={{ workspace: slug }}>
            {t(m.rolesButton)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function GeneralSkeleton() {
  const t = useT()
  return (
    <div className={cardGrid} aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
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
