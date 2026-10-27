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
import { ago, andList, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { ClientSignoffRule } from '@/features/agency/ClientSignoffRule'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { m as txApproval } from '@/features/transactional-templates/approval.messages'
import { getI18n, T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { mp } from './policy.messages'
import './approvals.css'
import { im } from '@/features/digests/issueMessages'
import { me as wm } from '@/features/website/elsewhere.messages'
import { ChannelCostRule } from '@/features/sms/PolicyRule'

type Policy = Schemas['ApprovalPolicy']
type View = Schemas['ApprovalPolicyView']
const path = '/api/v1/workspaces/{workspace}/approval-policy' as const

/** A5 §5.12 Send approvals policy (Settings › Approvals). */
export function PolicyPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', path, { params: { path: { workspace: slug } } })
  return (
    <Page title={t(mp.settings)} subtitle={me.workspace.name} tabs={settingsTabs(slug)} documentTitle={t(mp.documentTitle)}>
      {q.isError ? (
        <ErrorPanel title={t(mp.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(mp.nothingChanged)}
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
      setNError(t(mp.numberRange, { min: count(1), max: count(100_000_000) }))
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
      toast({ kind: 'success', title: t(mp.saved), detail: t(mp.savedDetail) })
      await qc.invalidateQueries({ queryKey: ['get', path] })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] })
      return true
    } catch (e) {
      toast({ kind: 'error', title: t(mp.saveFailed), detail: errorMessage(e) })
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
  // Role names in the plural ("Editors today: …") in English; other languages name the role as it is.
  const english = getI18n().locale === 'en'
  const roles = m.lacksSendRoles.map((r) => roleName({ name: r })).map((r) => (english && !r.endsWith('s') ? `${r}s` : r))
  const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>
  const bypassText = (
    <>
      <span className="font-semibold">{t(mp.bypassSwitch)}</span>
      <span className="k">
        <T msg={mp.bypassLogged} values={{ mono }} />
      </span>
    </>
  )
  const twoReason = view.approvers.length < 2 ? t(mp.twoNeeds, { count: view.approvers.length }) : undefined
  return (
    <>
      {read ? (
        <div className="note n" role="status">
          <Badge tone="neutral">{t(mp.readOnly)}</Badge>
          <span>
            {t(mp.readOnlyLead)} {view.reason || t(mp.readOnlyReason)}
          </span>
        </div>
      ) : null}
      <div className="ap-policy">
        <div className="col gap-5!">
          <section className="card" aria-labelledby="ap-when">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="ap-when">
                  {t(mp.whenTitle)}
                </h2>
                <span className="k">{t(mp.whenCaption)}</span>
              </div>
            </div>
            <div className="pad col gap-2.5!">
              <span className="k">{t(mp.whenIntro)}</span>
              {view.clientSignoff ? (
                <ClientSignoffRule info={view.clientSignoff} value={p.clientSignoff ?? { enabled: false, automations: false }} disabled={read} onChange={(v) => set({ clientSignoff: v })} />
              ) : null}
              <Rule
                on={p.audienceMin.enabled}
                disabled={read}
                onChange={(v) => set({ audienceMin: { ...p.audienceMin, enabled: v } })}
                title={t(mp.audienceTitle)}
                detail={t(mp.audienceDetail)}
                extra={
                  <span className="ap-rule-extra row flex-nowrap! gap-1.5!">
                    <input
                      className="in mono w-[110px] max-md:min-w-0 max-md:flex-[1_1_120px]"
                      inputMode="numeric"
                      aria-label={t(mp.minimumRecipients)}
                      aria-invalid={nError ? true : undefined}
                      aria-describedby={nError ? 'ap-n-err' : undefined}
                      value={nText}
                      readOnly={read}
                      onChange={(e) => setNText(e.target.value)}
                      onBlur={() => void commitN()}
                    />
                    <span className="k">{t(mp.recipients)}</span>
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
                title={t(mp.warmingTitle, { days: count(p.newProviderDays) })}
                detail={m.warming ? t(mp.matchesToday, { what: m.warming }) : t(mp.warmingNone)}
              />
              <Rule
                on={p.readyWithWarnings}
                disabled={read}
                onChange={(v) => set({ readyWithWarnings: v })}
                title={t(mp.warningsTitle)}
                detail={t(mp.warningsDetail)}
              />
              <Rule
                on={p.authorLacksSend}
                disabled={read}
                onChange={(v) => set({ authorLacksSend: v })}
                title={t(mp.lacksSendTitle)}
                detail={t(mp.lacksSendDetail, {
                  who: m.lacksSend.length ? (roles.length ? 'roles' : 'members') : 'nobody',
                  roles: andList(roles),
                  names: andList(m.lacksSend),
                })}
              />
              <Rule
                on={p.approvalLists}
                disabled={read}
                onChange={(v) => set({ approvalLists: v })}
                title={t(mp.listsTitle)}
                detail={
                  <T
                    msg={mp.listsDetail}
                    values={{
                      marked: m.approvalLists.length ? 'yes' : 'no',
                      lists: andList(m.approvalLists),
                      link: (chunks) => (
                        <Link to="/w/$workspace/audience/health" params={{ workspace: slug }}>
                          {chunks}
                        </Link>
                      ),
                    }}
                  />
                }
              />
              {/* Texts (SMS, X21): what an SMS or WhatsApp campaign costs. */}
              <ChannelCostRule value={p.channelCost} disabled={read} onChange={(v) => set({ channelCost: v })} />
              <div className="border-t border-line-2 pt-2.5">
                <Rule
                  on={p.automationPublish}
                  disabled={read}
                  onChange={(v) => set({ automationPublish: v })}
                  title={t(mp.automationTitle)}
                  detail={t(mp.automationDetail)}
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
                    <T msg={wm.policyPostNote} values={{ perm: t(mp.postPublishing) }} />
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
                  {t(mp.approversTitle)}
                </h2>
              </div>
            </div>
            <div className="pad col gap-3!">
              <div className="note n">
                <span>
                  <T
                    msg={mp.approversNote}
                    values={{
                      has: view.approvers.length ? 'yes' : 'no',
                      list: view.approvers.map((a, i) => (
                        <span key={a.id}>
                          {i > 0 ? ', ' : ''}
                          {a.role ? (
                            <T msg={mp.approverWithRole} values={{ name: a.name, role: roleName({ name: a.role }), b: (chunks) => <b>{chunks}</b> }} />
                          ) : (
                            <T msg={mp.approverName} values={{ name: a.name, b: (chunks) => <b>{chunks}</b> }} />
                          )}
                        </span>
                      )),
                    }}
                  />
                </span>
              </div>
              <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
                <legend className="mb-1 text-[12px] font-semibold text-ink-2">{t(mp.requiredLegend)}</legend>
                <Choice
                  checked={p.approvalsRequired === 1}
                  disabled={read}
                  onChange={() => set({ approvalsRequired: 1 })}
                  title={t(mp.oneApproval)}
                  detail={t(mp.oneApprovalDetail)}
                />
                <Choice
                  checked={p.approvalsRequired === 2}
                  disabled={read || Boolean(twoReason && p.approvalsRequired !== 2)}
                  onChange={() => set({ approvalsRequired: 2 })}
                  title={t(mp.twoApprovals)}
                  detail={twoReason ?? t(mp.twoApprovalsDetail, { count: view.approvers.length })}
                />
              </fieldset>
              <dl className="kv">
                <dt>{t(mp.snapshot)}</dt>
                <dd>{t(mp.snapshotDetail)}</dd>
                <dt>{t(mp.recheck)}</dt>
                <dd>{t(mp.recheckDetail)}</dd>
              </dl>
            </div>
          </section>
          <section className="card" aria-labelledby="ap-bypass">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="ap-bypass">
                  {t(mp.bypassTitle)}
                </h2>
                <span className="k">{t(mp.bypassCaption)}</span>
              </div>
            </div>
            <div className="pad col gap-2.5!">
              <label className="row flex-nowrap! gap-3!">
                <input type="checkbox" className="tg" role="switch" checked={p.emergencyBypass} disabled={read} onChange={(e) => set({ emergencyBypass: e.target.checked })} />
                <span className="col gap-px!">{bypassText}</span>
              </label>
              <dl className="kv">
                <dt>{t(mp.used)}</dt>
                <dd>{t(mp.usedCount, { count: view.bypassCount })}</dd>
              </dl>
            </div>
          </section>
        </div>
      </div>
      {read ? null : (
        <div className="row justify-end gap-2.5!">
          <Button onClick={discard}>{t(mp.discard)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {t(mp.savePolicy)}
          </Button>
        </div>
      )}
      <UnsavedChangesGuard dirty={!read && dirty && !busy} description={t(mp.unsaved)} onSave={save} />
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
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  return (
    <section className="card" aria-labelledby="ap-pending">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-pending">
            {t(mp.pendingTitle, { count: count(pending.length) })}
          </h2>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
          {t(mp.approvalInbox)}
        </ButtonLink>
      </div>
      <AccessTable caption={t(mp.pendingCaption)}>
        <thead>
          <tr>
            <Th>{t(mp.colCampaign)}</Th>
            <Th>{t(mp.colAuthor)}</Th>
            <Th>{t(mp.colAudience)}</Th>
            <Th>{t(mp.colRequested)}</Th>
            <Th>
              <span className="vh">{t(mp.colActions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {pending.length === 0 ? (
            <EmptyRow colSpan={5}>{t(mp.nothingPending)}</EmptyRow>
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
                    {t(mp.review)}
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

function PolicySkeleton() {
  const t = useT()
  return (
    <Loading label={t(mp.loading)} className="ap-policy">
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
