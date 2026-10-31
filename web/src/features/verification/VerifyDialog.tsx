import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, count, dayMonthDate, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, money, Notice, ResultBadge, tagValue, useRefreshVerification, type Estimate, type Origin, type ServiceKey, type Target } from './shared'

/** An engagement tier's name: "Sunset", "At risk". */
export function tierName(t: ReturnType<typeof useT>, key: string): string {
  switch (key) {
    case 'active':
      return t(m.tierActive)
    case 'cooling':
      return t(m.tierCooling)
    case 'at_risk':
      return t(m.tierAtRisk)
    case 'sunset':
      return t(m.tierSunset)
  }
  return key
}

/** "the Sunset tier", "the At risk and Sunset tiers": a job's label in a sentence. */
export function targetLabel(t: ReturnType<typeof useT>, target: Target, fallback: string): string {
  if (target.kind === 'tier' && target.tiers?.length) {
    const names = target.tiers.map((x) => tierName(t, x))
    return names.length === 1 ? t(m.jobLabelTier, { tier: names[0]! }) : t(m.jobLabelTiers, { tiers: andList(names) })
  }
  return fallback
}

/**
 * Verify addresses: what is checked, the service, the cost, the notice and
 * what happens to each result. Every entry point opens this same dialog.
 */
export function VerifyDialog({
  open,
  onOpenChange,
  target,
  origin,
  from,
  campaignId,
  onStarted,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  target: Target
  origin: Origin
  /** The eyebrow: "List health · Sunset tier · Marcus Chen". */
  from: string
  campaignId?: string
  /** Called with the new job's id instead of opening its page. */
  onStarted?: (jobId: string) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshVerification()
  const skipId = useId()
  const [service, setService] = useState<ServiceKey>()
  const [skipRecent, setSkipRecent] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const q = $api.useQuery('post', `${base}/estimate`, { params: { path: { workspace: slug } }, body: { target, service, skipRecent } }, { placeholderData: keepPreviousData })
  const e = q.data

  const start = async () => {
    if (!e?.service) return
    setBusy(true)
    setError(undefined)
    const res = await http.POST(`${base}/jobs`, { params: { path: { workspace: slug } }, body: { target, origin, service: e.service as ServiceKey, skipRecent, campaignId } })
    setBusy(false)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    onOpenChange(false)
    toast({ kind: 'progress', title: t(m.jobStarted, { label: targetLabel(t, target, res.data.label), name: res.data.serviceName }), detail: t(m.jobStartedDetail, { count: res.data.total }) })
    if (onStarted) onStarted(res.data.id)
    else void navigate({ to: '/w/$workspace/audience/verification/$jobId', params: { workspace: slug, jobId: res.data.id } })
  }

  if (e && !e.connected) {
    return (
      <Dialog
        open={open}
        onOpenChange={onOpenChange}
        eyebrow={<span className="k">{from}</span>}
        title={t(m.verifyTitle)}
        footer={
          <>
            <Button onClick={() => onOpenChange(false)}>{t(m.close)}</Button>
            {e.canManage ? (
              <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/verification" params={{ workspace: slug }}>
                {t(m.connectService)}
              </ButtonLink>
            ) : null}
          </>
        }
      >
        <NothingYet e={e} />
      </Dialog>
    )
  }

  const blocked = !e
    ? t(m.loading)
    : !e.canVerify
      ? e.verifyHelpers.length
        ? t(m.cantVerifyPlain, { helpers: orList(e.verifyHelpers) })
        : t(m.cantVerifyPlainNobody)
      : e.toCheck === 0
        ? t(m.nothingToCheck)
        : !e.enough
          ? t(m.notEnough, { name: e.serviceName ?? '', balance: e.balance ?? 0, needed: e.toCheck })
          : undefined
  const tierChips = target.kind === 'tier' ? (target.tiers ?? []) : []
  const what =
    e && target.kind === 'tier' && tierChips.length
      ? tierChips.length === 1
        ? t(m.checkedTier, { tier: tierName(t, tierChips[0]!), count: e.contacts })
        : t(m.checkedTiers, { tiers: andList(tierChips.map((x) => tierName(t, x))), count: e.contacts })
      : e
        ? t(m.checkedOther, { label: e.label, count: e.contacts })
        : ''
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{from}</span>}
      title={t(m.verifyTitle)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={() => void start()} loading={busy} busyLabel={t(m.starting)} disabledReason={busy ? undefined : blocked}>
            {t(m.verifyN, { count: e?.toCheck ?? 0 })}
          </Button>
        </>
      }
    >
      {!e ? (
        q.isError ? (
          <div className="note b" role="alert">
            <Icon name="alertCircle" size={16} className="mt-px flex-none" />
            <span>
              {t(m.loadEstimateError)} · {errorMessage(q.error)}
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-3" aria-busy="true">
            <Skeleton w="100%" h={56} r={8} />
            <Skeleton w="100%" h={36} r={8} />
            <Skeleton w="100%" h={64} r={8} />
          </div>
        )
      ) : (
        <>
          <div className="rounded-lg border border-line px-3.5 py-3">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="k">{t(m.whatIsChecked)}</span>
                <b className="font-semibold">{what}</b>
              </div>
              <span className="flex flex-wrap gap-1.5">
                {tierChips.map((x) => (
                  <span key={x} className="badge info">
                    {tierName(t, x)}
                  </span>
                ))}
              </span>
            </div>
          </div>
          <div className="flex min-w-0 flex-col gap-1.5">
            <label className="text-[12.5px] font-semibold" htmlFor={`${skipId}-svc`}>
              {t(m.service)}
            </label>
            <select id={`${skipId}-svc`} className="in" value={e.service ?? ''} onChange={(ev) => setService(ev.target.value as ServiceKey)}>
              {e.services.map((s) => (
                <option key={s.key} value={s.key}>
                  {t(m.serviceOption, { name: s.name, isDefault: s.default ? 'yes' : 'no', credits: s.balance != null ? count(s.balance) : 'none' })}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-start gap-2.5 text-[13px]">
            <input id={skipId} type="checkbox" className="mt-0.5" checked={skipRecent} onChange={(ev) => setSkipRecent(ev.target.checked)} aria-describedby={`${skipId}-h`} />
            <span className="flex flex-col gap-px">
              <label htmlFor={skipId}>{t(m.skipRecent)}</label>
              <span className="k" id={`${skipId}-h`}>
                {!skipRecent
                  ? ''
                  : e.skipped && e.skippedService && e.skippedOn
                    ? t(m.skipsSome, { count: e.skipped, service: e.skippedService, date: dayMonthDate(e.skippedOn) })
                    : e.skipped
                      ? t(m.skipsSomePlain, { count: e.skipped })
                      : tierChips.length === 1
                        ? t(m.skipsNoneTier, { tier: tierName(t, tierChips[0]!) })
                        : t(m.skipsNone)}
              </span>
            </span>
          </div>
          <div className="vf-cost">
            <span className="big">{t(m.costLine, { count: e.toCheck, price: money(e.price, true), cost: money(e.cost) })}</span>
            <span className="k text-ink-2!">
              {e.balance != null && e.after != null ? t(m.balanceLine, { balance: e.balance, after: e.after, name: e.serviceName ?? '' }) : t(m.balanceUnknown, { name: e.serviceName ?? '' })}
            </span>
          </div>
          {e.enough || e.balance == null ? null : (
            <div className="note w" role="status">
              <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
              <span>{t(m.notEnough, { name: e.serviceName ?? '', balance: e.balance, needed: e.toCheck })}</span>
            </div>
          )}
          <Notice
            head={e.regionLabel ? t(m.noticeCount, { count: e.toCheck, name: e.serviceName ?? '', region: e.regionLabel }) : t(m.noticeCountNoRegion, { count: e.toCheck, name: e.serviceName ?? '' })}
            name={e.serviceName ?? ''}
          />
          {origin === 'readiness' ? (
            <span className="text-[13px] text-ink-nav">{t(m.happensShort)}</span>
          ) : (
            <div className="flex flex-col gap-1.5">
              <span className="text-[12.5px] font-semibold">{t(m.whatHappens)}</span>
              <div className="vf-map">
                <div className="rr">
                  <ResultBadge result="valid" />
                  <span>{t(m.happensValid)}</span>
                </div>
                <div className="rr">
                  <ResultBadge result="invalid" />
                  <span>{t(m.happensInvalid)}</span>
                </div>
                <div className="rr">
                  <ResultBadge result="risky" />
                  <span>
                    <T msg={m.happensRisky} values={tagValue} />
                  </span>
                </div>
                <div className="rr">
                  <ResultBadge result="unknown" />
                  <span>
                    <T msg={m.happensUnknown} values={tagValue} />
                  </span>
                </div>
              </div>
            </div>
          )}
          {!e.canVerify ? (
            <div className="note n" role="note">
              <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
              <span>
                <T msg={e.verifyHelpers.length ? m.cantVerify : m.cantVerifyNobody} values={{ name: e.serviceName ?? '', helpers: orList(e.verifyHelpers) }} />
              </span>
            </div>
          ) : null}
          {error ? (
            <div className="note b" role="alert">
              <Icon name="alertCircle" size={16} className="mt-px flex-none" />
              <span>{error}</span>
            </div>
          ) : null}
        </>
      )}
    </Dialog>
  )
}

function NothingYet({ e }: { e: Estimate }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  return (
    <div className="note n" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
      <div className="flex min-w-0 flex-col gap-1">
        <b className="font-bold">{t(m.nothingYetTitle)}</b>
        <span>
          {e.canManage ? (
            <T
              msg={m.nothingYet}
              values={{
                workspace: me.workspace.name,
                link: (c) => (
                  <Link to="/w/$workspace/settings/integrations/verification" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          ) : e.manageHelpers.length ? (
            t(m.nothingYetAsk, { helpers: orList(e.manageHelpers) })
          ) : (
            t(m.nothingYetAskNobody)
          )}
        </span>
      </div>
    </div>
  )
}
