import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ago, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { m as txApproval } from '@/features/transactional-templates/approval.messages'
import { T, useT } from '@/i18n'
import './approvals.css'
import { im } from '@/features/digests/issueMessages'
import { me as wm } from '@/features/website/elsewhere.messages'
import { ChannelCostRule } from '@/features/sms/PolicyRule'

type Policy = Schemas['ApprovalPolicy']
type View = Schemas['ApprovalPolicyView']
const path = '/api/v1/workspaces/{workspace}/approval-policy' as const

/** A5 §5.12 Send approvals policy (Settings › Approvals). */
export function PolicyPage() {
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', path, { params: { path: { workspace: slug } } })
  return (
    <Page title="Settings" subtitle={me.workspace.name} tabs={settingsTabs(slug)} documentTitle="Approvals · Settings">
      {q.isError ? (
        <ErrorPanel title="Couldn't load the approval policy" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : q.isPending ? (
        <PolicySkeleton />
      ) : (
        <PolicyForm key={JSON.stringify(q.data.policy)} view={q.data} />
      )}
    </Page>
  )
}

function PolicyForm({ view }: { view: View }) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const [p, setP] = useState<Policy>(view.policy)
  const [nText, setNText] = useState(count(view.policy.audienceMin.n))
  const [nError, setNError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const read = !view.canManage
  const dirty = JSON.stringify(p) !== JSON.stringify(view.policy) || nText.replace(/\D/g, '') !== String(p.audienceMin.n)
  const set = (patch: Partial<Policy>) => setP((v) => ({ ...v, ...patch }))
  const commitN = () => {
    const n = Number(nText.replace(/\D/g, ''))
    if (!nText.trim() || !Number.isFinite(n) || n < 1 || n > 100_000_000) {
      setNError('Enter a number of recipients between 1 and 100,000,000.')
      return null
    }
    setNError(undefined)
    setNText(count(n))
    set({ audienceMin: { ...p.audienceMin, n } })
    return n
  }
  const save = async () => {
    const n = commitN()
    if (n === null) return false
    setBusy(true)
    try {
      const body = { ...p, audienceMin: { ...p.audienceMin, n } }
      await http.PUT(path, { params: { path: { workspace: slug } }, body })
      toast({ kind: 'success', title: 'Approval policy saved', detail: 'It applies to the next request; open requests keep the rules they matched.' })
      await qc.invalidateQueries({ queryKey: ['get', path] })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] })
      return true
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't save the approval policy", detail: errorMessage(e) })
      return false
    } finally {
      setBusy(false)
    }
  }
  const discard = () => {
    setP(view.policy)
    setNText(count(view.policy.audienceMin.n))
    setNError(undefined)
  }
  const m = view.matches
  const roles = m.lacksSendRoles.map((r) => (r.endsWith('s') ? r : `${r}s`))
  const bypassText = (
    <>
      <span className="font-semibold">Owners can send without approval by entering a reason</span>
      <span className="k">
        The bypass is logged as <span className="mono">campaign.approval_bypassed</span> and every approver is notified immediately.
      </span>
    </>
  )
  const twoReason =
    view.approvers.length < 2 ? `Needs two members with campaigns:approve; today there ${view.approvers.length === 1 ? 'is 1' : `are ${view.approvers.length}`}.` : undefined
  return (
    <>
      {read ? (
        <div className="note n" role="status">
          <Badge tone="neutral">Read-only</Badge>
          <span>You can view the approval policy. {view.reason || 'Changing it needs the workspace:manage permission.'}</span>
        </div>
      ) : null}
      <div className="ap-policy">
        <div className="col gap-5!">
          <section className="card" aria-labelledby="ap-when">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="ap-when">
                  When is approval required?
                </h2>
                <span className="k">Per workspace</span>
              </div>
            </div>
            <div className="pad col gap-2.5!">
              <span className="k">Approval is required when any enabled rule matches. The author sees which rules matched and can add a note.</span>
              <Rule
                on={p.audienceMin.enabled}
                disabled={read}
                onChange={(v) => set({ audienceMin: { ...p.audienceMin, enabled: v } })}
                title="A Broadcast audience of at least"
                detail="Counted after exclusions, at request time and again at send time."
                extra={
                  <span className="row flex-nowrap! gap-1.5!">
                    <input
                      className="in mono w-[110px] max-md:min-w-0 max-md:flex-[1_1_120px]"
                      inputMode="numeric"
                      aria-label="Minimum recipients"
                      aria-invalid={nError ? true : undefined}
                      aria-describedby={nError ? 'ap-n-err' : undefined}
                      value={nText}
                      readOnly={read}
                      onChange={(e) => setNText(e.target.value)}
                      onBlur={() => void commitN()}
                    />
                    <span className="k">recipients</span>
                  </span>
                }
              />
              {nError ? (
                <span className="err" id="ap-n-err" role="alert">
                  {nError}
                </span>
              ) : null}
              <Rule
                on={p.warmingProvider}
                disabled={read}
                onChange={(v) => set({ warmingProvider: v })}
                title={`The route includes a Warming-up provider, or one connected in the last ${p.newProviderDays} days`}
                detail={m.warming ? `Matches today: ${m.warming}` : 'Matches nothing today: no provider in the Broadcasts route is warming up or new.'}
              />
              <Rule
                on={p.readyWithWarnings}
                disabled={read}
                onChange={(v) => set({ readyWithWarnings: v })}
                title="Send Readiness is “Ready with warnings”"
                detail="Blocked sends never start; this rule adds a human check for warnings."
              />
              <Rule
                on={p.authorLacksSend}
                disabled={read}
                onChange={(v) => set({ authorLacksSend: v })}
                title="The author lacks the send permission"
                detail={`Members with campaigns:edit but not campaigns:send can still prepare a send and request approval. ${
                  m.lacksSend.length ? `${roles.length ? listAnd(roles) : 'Members'} today: ${listAnd(m.lacksSend)}.` : 'Nobody is in that position today.'
                }`}
              />
              <Rule
                on={p.approvalLists}
                disabled={read}
                onChange={(v) => set({ approvalLists: v })}
                title="The audience includes a list marked “Requires approval”"
                detail={
                  <>
                    {m.approvalLists.length ? `Marked today: ${listAnd(m.approvalLists)}. Mark more in ` : 'No list is marked yet. Mark lists in '}
                    <Link to="/w/$workspace/audience/health" params={{ workspace: slug }}>
                      Audience → Lists
                    </Link>
                    .
                  </>
                }
              />
              {/* Texts (SMS, X21): what an SMS or WhatsApp campaign costs. */}
              <ChannelCostRule value={p.channelCost} disabled={read} onChange={(v) => set({ channelCost: v })} />
              <div className="border-t border-line-2 pt-2.5">
                <Rule
                  on={p.automationPublish}
                  disabled={read}
                  onChange={(v) => set({ automationPublish: v })}
                  title="Publishing an automation also requires approval"
                  detail="Applies to automations:publish. Off: publishing is logged only."
                />
              </div>
              <div className="border-t border-line-2 pt-2.5">
                <Rule
                  on={p.transactionalPublish}
                  disabled={read}
                  onChange={(v) => set({ transactionalPublish: v })}
                  title={t(txApproval.ruleTitle)}
                  detail={t(txApproval.ruleDetail)}
                />
              </div>
              {/* The website's posts and pages (features/website). */}
              <div className="border-t border-line-2 pt-2.5">
                <Rule on={p.postPublish} disabled={read} onChange={(v) => set({ postPublish: v })} title={t(wm.policyPost)} detail={t(wm.policyPostText)} />
                <div className="note n mt-2">
                  <div>
                    <T msg={wm.policyPostNote} values={{ perm: <code className="mono">website:publish</code> }} />
                  </div>
                </div>
              </div>
              <span className="k">{t(im.rulesEach)}</span>
            </div>
          </section>
          {/* In the wide column, so its table fits the card (ApprovalPolicy). */}
          <PendingCard pending={view.pending} />
        </div>
        <div className="col gap-5!">
          <section className="card" aria-labelledby="ap-approvers">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="ap-approvers">
                  Approvers
                </h2>
              </div>
            </div>
            <div className="pad col gap-3!">
              <div className="note n">
                <span>
                  Approvers are members with <span className="mono">campaigns:approve</span>.{' '}
                  {view.approvers.length ? (
                    <>
                      Today:{' '}
                      {view.approvers.map((a, i) => (
                        <span key={a.id}>
                          {i > 0 ? ', ' : ''}
                          <b>{a.name}</b>
                          {a.role ? ` (${a.role})` : ''}
                        </span>
                      ))}
                      .
                    </>
                  ) : (
                    'Nobody holds it today: give it to a role in Settings › Roles.'
                  )}{' '}
                  Authors can never approve their own campaign.
                </span>
              </div>
              <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
                <legend className="mb-1 text-[12px] font-semibold text-ink-2">Approvals required</legend>
                <Choice
                  checked={p.approvalsRequired === 1}
                  disabled={read}
                  onChange={() => set({ approvalsRequired: 1 })}
                  title="1 approval"
                  detail="Any approver other than the author"
                />
                <Choice
                  checked={p.approvalsRequired === 2}
                  disabled={read || Boolean(twoReason && p.approvalsRequired !== 2)}
                  onChange={() => set({ approvalsRequired: 2 })}
                  title="2 approvals"
                  detail={twoReason ?? `Two different approvers; with ${view.approvers.length} approvers today one of them must not be the author`}
                />
              </fieldset>
              <dl className="kv">
                <dt>Snapshot</dt>
                <dd>
                  An approval covers content and subject, the audience definition, the sending route and the schedule. Changing any of them, or the audience count moving by more
                  than 10%, invalidates it; the approver sees exactly what changed.
                </dd>
                <dt>Recheck</dt>
                <dd>Send Readiness runs again when the send starts. A Blocked result stops the send and notifies author and approvers.</dd>
              </dl>
            </div>
          </section>
          <section className="card" aria-labelledby="ap-bypass">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="ap-bypass">
                  Emergency bypass
                </h2>
                <span className="k">Owners only</span>
              </div>
            </div>
            <div className="pad col gap-2.5!">
              <label className="row flex-nowrap! gap-3!">
                <input type="checkbox" className="tg" role="switch" checked={p.emergencyBypass} disabled={read} onChange={(e) => set({ emergencyBypass: e.target.checked })} />
                <span className="col gap-px!">{bypassText}</span>
              </label>
              <dl className="kv">
                <dt>Used</dt>
                <dd>
                  {view.bypassCount} {view.bypassCount === 1 ? 'time' : 'times'} in the last 90 days
                </dd>
              </dl>
            </div>
          </section>
        </div>
      </div>
      {read ? null : (
        <div className="row justify-end gap-2.5!">
          <Button onClick={discard}>Discard</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            Save policy
          </Button>
        </div>
      )}
      <UnsavedChangesGuard dirty={!read && dirty && !busy} description="Your changes to the approval policy aren't saved. Requests keep following the saved policy." onSave={save} />
    </>
  )
}

function Rule({
  on,
  disabled,
  onChange,
  title,
  detail,
  extra,
}: {
  on: boolean
  disabled: boolean
  onChange: (v: boolean) => void
  title: string
  detail: ReactNode
  extra?: ReactNode
}) {
  const text = (
    <>
      <span className="font-semibold">{title}</span>
      <span className="k whitespace-normal">{detail}</span>
    </>
  )
  return (
    <label className="ap-rule">
      <input type="checkbox" className="tg" role="switch" checked={on} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      <span className="col min-w-0 flex-1 gap-px!">{text}</span>
      {extra}
    </label>
  )
}

function Choice({ checked, disabled, onChange, title, detail }: { checked: boolean; disabled: boolean; onChange: () => void; title: string; detail: string }) {
  const text = (
    <>
      <span className="font-semibold">{title}</span>
      <span className="k">{detail}</span>
    </>
  )
  return (
    <label className="ap-opt">
      <input type="radio" name="approvals-required" checked={checked} disabled={disabled} onChange={onChange} />
      <span className="col gap-px!">{text}</span>
    </label>
  )
}

function PendingCard({ pending }: { pending: Schemas['ApprovalPendingItem'][] }) {
  const { slug } = useWorkspace()
  const now = useNow()
  return (
    <section className="card" aria-labelledby="ap-pending">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-pending">
            Pending now · {pending.length}
          </h2>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
          Approval inbox
        </ButtonLink>
      </div>
      <AccessTable caption="Pending approval requests">
        <thead>
          <tr>
            <Th>Campaign</Th>
            <Th>Author</Th>
            <Th>Audience</Th>
            <Th>Requested</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {pending.length === 0 ? (
            <EmptyRow colSpan={5}>Nothing is waiting for approval.</EmptyRow>
          ) : (
            pending.map((it) => (
              <tr key={it.id}>
                <Td>
                  <span className="font-semibold">{it.name}</span>
                </Td>
                <Td>
                  <PersonCell name={it.author.name} sub={it.author.email} color={it.author.color} />
                </Td>
                <Td>{count(it.audience)}</Td>
                <Td>{ago(it.requestedAt, now)}</Td>
                <Td>
                  <ButtonLink size="sm" variant="primary" to="/w/$workspace/approvals/$approvalId" params={{ workspace: slug, approvalId: it.id }}>
                    Review
                  </ButtonLink>
                </Td>
              </tr>
            ))
          )}
        </tbody>
      </AccessTable>
    </section>
  )
}

function listAnd(words: string[]): string {
  return words.length <= 1 ? (words[0] ?? '') : `${words.slice(0, -1).join(', ')} and ${words[words.length - 1]}`
}

function PolicySkeleton() {
  return (
    <Loading label="Loading the approval policy" className="ap-policy">
      <div className="card">
        <div className="sec">
          <Skeleton w={220} h={14} />
        </div>
        <div className="pad col gap-2.5!">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="ap-rule">
              <Skeleton w={40} h={24} r={999} />
              <div className="col flex-1 gap-1.5!">
                <Skeleton w="55%" h={12} />
                <Skeleton w="80%" h={10} />
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="col gap-5!">
        {[3, 2, 2].map((n, i) => (
          <div key={i} className="card">
            <div className="sec">
              <Skeleton w={120} h={14} />
            </div>
            <div className="pad col gap-2.5!">
              {Array.from({ length: n }, (_, j) => (
                <Skeleton key={j} w={`${80 - j * 15}%`} h={12} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </Loading>
  )
}
