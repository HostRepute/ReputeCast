import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { OptionCard } from '@/components/ui/Field'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, dayMonthYear, shortDate } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'

type Policy = Schemas['AccessibilityPolicy']
type PolicyInput = Schemas['AccessibilityPolicyInput']
type Impact = Schemas['AccessibilityImpact']

/** The minimum text sizes a workspace can choose (WCAG has none; 12 px is the default). */
export const minTextSizes = [12, 13, 14, 15, 16] as const

const policyPath = '/api/v1/workspaces/{workspace}/accessibility/policy' as const

/**
 * Settings › General › Accessibility (5.5): what happens when serious issues
 * remain, and the minimum text size. Switching to Block shows what it
 * affects first; every change is audited and can be undone from the toast.
 */
export function PolicyCard() {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', policyPath, { params: { path: { workspace: slug } } })
  if (q.isError) {
    return (
      <section className="card col-span-full" aria-labelledby="a11y-h">
        <CardSection title={<span id="a11y-h">{t(m.policyTitle)}</span>} caption={t(m.policyCaption)} />
        <div className="pad">
          <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      </section>
    )
  }
  if (q.isPending) {
    return (
      <section className="card col-span-full" aria-busy="true" aria-labelledby="a11y-h">
        <CardSection title={<span id="a11y-h">{t(m.policyTitle)}</span>} caption={t(m.policyCaption)} />
        <div className="pad col gap-3">
          <span className="vh" role="status">
            {t(m.loading)}
          </span>
          <Skeleton h={64} />
          <Skeleton h={64} />
          <Skeleton w="40%" h={12} />
        </div>
      </section>
    )
  }
  // Saved values come back from the server; remounting resets the form to them.
  return <PolicyForm key={`${q.data.onSerious}|${q.data.minTextPx}|${q.data.updatedAt ?? ''}`} policy={q.data} />
}

function PolicyForm({ policy }: { policy: Policy }) {
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const qc = useQueryClient()
  const t = useT()
  const [onSerious, setOnSerious] = useState<PolicyInput['onSerious']>(policy.onSerious)
  const [minTextPx, setMinTextPx] = useState(policy.minTextPx)
  const [impactOpen, setImpactOpen] = useState(false)
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', policyPath)
  const dirty = onSerious !== policy.onSerious || minTextPx !== policy.minTextPx
  const readOnly = !policy.canManage
  const reason = readOnly ? t(m.readOnlyReason) : undefined

  const put = (body: PolicyInput, after?: () => void) =>
    save.mutateAsync({ params: { path: { workspace: slug } }, body }).then((saved) => {
      qc.setQueryData($api.queryOptions('get', policyPath, { params: { path: { workspace: slug } } }).queryKey, saved)
      after?.()
      return saved
    })

  const apply = async (body: PolicyInput) => {
    const before: PolicyInput = { onSerious: policy.onSerious, minTextPx: policy.minTextPx }
    const saved = await put(body)
    const time = clock(saved.updatedAt ?? new Date())
    const changed = before.onSerious !== body.onSerious || before.minTextPx !== body.minTextPx
    const title =
      before.onSerious === body.onSerious
        ? t(m.saved, { time })
        : body.onSerious === 'block'
          ? t(m.savedBlock, { time })
          : t(m.savedWarn, { time })
    toast({
      kind: 'success',
      title,
      // Undo puts the policy back as it was, also recorded in the audit log.
      undo: changed
        ? () => {
            put(before).catch((e: unknown) => toast({ kind: 'error', title: t(m.undoFailed), detail: errorMessage(e) }))
          }
        : undefined,
    })
  }

  const submit = () => {
    if (reason) return
    setError(undefined)
    // Switching to Block shows what it affects first.
    if (onSerious === 'block' && policy.onSerious !== 'block') {
      setImpactOpen(true)
      return
    }
    apply({ onSerious, minTextPx }).catch((e: unknown) => setError(errorMessage(e)))
  }

  const changed = policy.updatedAt
    ? shortDate(policy.updatedAt) === shortDate(new Date())
      ? t(m.changedAt, { name: policy.updatedBy || '—', time: clock(policy.updatedAt) })
      : t(m.changedOn, { name: policy.updatedBy || '—', date: dayMonthYear(policy.updatedAt) })
    : t(m.neverChanged)

  return (
    <section className="card col-span-full min-w-0" aria-labelledby="a11y-h">
      <CardSection title={<span id="a11y-h">{t(m.policyTitle)}</span>} caption={t(m.policyCaption)} />
      <div className="pad col gap-4">
        {readOnly ? (
          <Note tone="neutral">
            <b className="font-semibold">{t(m.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}</b>
            <br />
            {policy.helpers.length ? t(m.readOnlyDetail, { helpers: formatList(policy.helpers, 'disjunction') }) : t(m.readOnlyNobody)}
          </Note>
        ) : null}
        <div className="grid items-start gap-7 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(400px,100%),1fr))]">
          <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
            <legend className="lbl mb-2 p-0">{t(m.whenSerious)}</legend>
            <div className="flex flex-wrap gap-2.5">
              <OptionCard
                name="a11y-policy"
                value="warn"
                checked={onSerious === 'warn'}
                onChange={() => setOnSerious('warn')}
                disabled={readOnly}
                className="flex-[1_1_260px] px-3 py-2.5"
                title={
                  <>
                    {t(m.warn)} <span className="k font-medium">{t(m.warnDefault)}</span>
                  </>
                }
                description={<span className="text-ink-2">{t(m.warnDetail)}</span>}
              />
              <OptionCard
                name="a11y-policy"
                value="block"
                checked={onSerious === 'block'}
                onChange={() => setOnSerious('block')}
                disabled={readOnly}
                className="flex-[1_1_260px] px-3 py-2.5"
                title={t(m.block)}
                description={<span className="text-ink-2">{t(m.blockDetail)}</span>}
              />
            </div>
          </fieldset>
          <div className="flex min-w-0 flex-col gap-3.5">
            <div className="flex flex-col gap-1">
              <span className="lbl">{t(m.seriousTitle)}</span>
              <span className="text-ink-2">{t(m.seriousDetail)}</span>
            </div>
            <label className="lbl max-w-[260px]">
              {t(m.minSize)}
              <select className="in" value={minTextPx} disabled={readOnly} onChange={(e) => setMinTextPx(Number(e.target.value))}>
                {minTextSizes.map((px) => (
                  <option key={px} value={px}>
                    {t(m.minSizeOption, { px })}
                  </option>
                ))}
              </select>
              <span className="k font-medium">{t(m.minSizeHint)}</span>
            </label>
          </div>
        </div>
        {error ? <Note tone="bad">{error}</Note> : null}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 px-[18px] py-3">
        <span className="k">{changed}</span>
        <Button variant="primary" size="sm" loading={save.isPending && !impactOpen} disabledReason={reason} onClick={submit}>
          {t(common.save)}
        </Button>
      </div>
      {impactOpen ? (
        <ImpactDialog
          input={{ onSerious, minTextPx }}
          onOpenChange={setImpactOpen}
          onConfirm={async () => {
            await apply({ onSerious, minTextPx })
            setImpactOpen(false)
          }}
        />
      ) : null}
      <UnsavedChangesGuard dirty={!readOnly && dirty && !save.isPending} description={t(m.unsaved)} />
    </section>
  )
}

/** "Block sends with serious accessibility issues?": what Block would stop now. */
function ImpactDialog({ input, onOpenChange, onConfirm }: { input: PolicyInput; onOpenChange: (open: boolean) => void; onConfirm: () => Promise<void> }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('post', '/api/v1/workspaces/{workspace}/accessibility/policy/impact', { params: { path: { workspace: slug } }, body: input })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const confirm = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await onConfirm()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => !busy && onOpenChange(open)}
      hideClose
      eyebrow={<span className="k">{t(m.impactEyebrow)}</span>}
      title={t(m.impactTitle)}
      description={t(m.impactBody)}
      footer={
        <>
          <span className="k mr-auto">{t(m.audited)}</span>
          <Button onClick={() => onOpenChange(false)} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={busy} disabledReason={q.isPending ? t(m.impactChecking) : undefined} onClick={() => void confirm()}>
            {t(m.switchToBlock)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-2">
        <h3 className="m-0 text-[13.5px] font-semibold">{t(m.impactNow)}</h3>
        {q.isPending ? (
          <div className="col gap-2" aria-busy="true">
            <span className="k" role="status">
              {t(m.impactChecking)}
            </span>
            <Skeleton h={52} />
          </div>
        ) : q.isError ? (
          <Note tone="warn">{t(m.impactFailed, { error: errorMessage(q.error) })}</Note>
        ) : (
          <ImpactList impact={q.data} />
        )}
      </div>
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

function ImpactList({ impact }: { impact: Impact }) {
  const { slug } = useWorkspace()
  const t = useT()
  const rows = [
    ...impact.campaigns.map((c) => ({
      key: `c:${c.id}`,
      name: c.name,
      detail:
        c.status === 'scheduled'
          ? t(m.impactCampaignScheduled, { when: c.scheduledAt ? `${shortDate(c.scheduledAt)} ${clock(c.scheduledAt)}` : '', count: c.serious, summary: c.summary })
          : c.status === 'approval'
            ? t(m.impactCampaignApproval, { count: c.serious, summary: c.summary })
            : c.status === 'approved'
              ? t(m.impactCampaignApproved, { count: c.serious, summary: c.summary })
              : t(m.impactCampaignDraft, { count: c.serious, summary: c.summary }),
      link: (
        <Link className="btn sm" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: c.id }} aria-label={t(m.openLabel, { name: c.name })}>
          {t(m.open)}
        </Link>
      ),
    })),
    ...impact.flows.map((f) => {
      const name = t(m.impactFlowName, { name: f.name, version: f.version, step: f.step })
      return {
        key: `f:${f.automationId}:${f.step}`,
        name,
        detail:
          f.liveVersion !== undefined
            ? t(m.impactFlowLive, { count: f.serious, summary: f.summary, live: f.liveVersion, version: f.version })
            : t(m.impactFlowNew, { count: f.serious, summary: f.summary, version: f.version }),
        link: (
          <Link className="btn sm" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: f.automationId }} aria-label={t(m.openLabel, { name })}>
            {t(m.open)}
          </Link>
        ),
      }
    }),
  ]
  return (
    <>
      {rows.length ? (
        <ul className="m-0 flex list-none flex-col rounded-lg border border-line p-0">
          {rows.map((r, i) => (
            <li key={r.key} className={i > 0 ? 'flex items-center gap-3 border-t border-line-2 px-3 py-2.5' : 'flex items-center gap-3 px-3 py-2.5'}>
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="font-semibold">{r.name}</span>
                <span className="k">{r.detail}</span>
              </span>
              {r.link}
            </li>
          ))}
        </ul>
      ) : (
        <span className="k">{t(m.impactNone)}</span>
      )}
      <span className="k">{t(m.impactHeld, { count: impact.held })}</span>
    </>
  )
}
