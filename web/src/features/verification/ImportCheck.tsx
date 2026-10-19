import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { t as tr, useT } from '@/i18n'
import { clock, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, money, Notice, useJob, useRefreshVerification } from './shared'

type Preview = Schemas['ImportPreview']
type RiskyGroup = Preview['riskyGroups'][number]

/**
 * The import preview's optional "Verify the new addresses": the cost, the
 * notice and the job, or why it can't run. The import never needs it.
 */
export function ImportVerifyCard({ importId, rows, onSkip }: { importId: string; rows: number; onSkip?: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefreshVerification()
  const target = { kind: 'import' as const, importId }
  const q = $api.useQuery('post', `${base}/estimate`, { params: { path: { workspace: slug } }, body: { target, skipRecent: false } })
  const [busy, setBusy] = useState(false)
  const [jobId, setJobId] = useState<string>()
  const job = useJob(jobId ?? '', Boolean(jobId))
  const running = Boolean(jobId) && (!job.data || job.data.status === 'running' || job.data.status === 'queued' || job.data.status === 'paused')
  // When the check finishes, the preview reads its answers.
  useEffect(() => {
    if (jobId && job.data && !running) void qc.invalidateQueries({ predicate: (x) => String((x.queryKey as unknown[])[1] ?? '').includes('/imports/') })
  }, [jobId, job.data, running, qc])
  const e = q.data
  if (!e || e.toCheck === 0) return null
  if (!e.connected) {
    return (
      <section className="card flex min-w-0 flex-col gap-2.5 px-[18px] py-4">
        <span className="font-bold">{t(m.importVerifyTitlePlain)}</span>
        <span className="k text-ink-nav!">
          {e.manageHelpers.length ? t(m.importNoService, { rows, helpers: orList(e.manageHelpers) }) : t(m.importNoServiceNobody, { rows })}
        </span>
        <Link to="/w/$workspace/settings/integrations/verification" params={{ workspace: slug }} className="pl-lnk">
          {t(m.title)}
        </Link>
      </section>
    )
  }
  const name = e.serviceName ?? ''
  const start = async () => {
    setBusy(true)
    const res = await http.POST(`${base}/jobs`, { params: { path: { workspace: slug } }, body: { target, origin: 'import', skipRecent: false } })
    setBusy(false)
    if (!res.data) {
      toast({ kind: 'error', title: errorMessage(res.error) })
      return
    }
    setJobId(res.data.id)
    void refresh()
  }
  if (!e.canVerify) {
    const why = e.verifyHelpers.length ? t(m.importCantVerify, { name, helpers: orList(e.verifyHelpers) }) : t(m.importCantVerifyNobody, { name })
    return (
      <section className="card flex min-w-0 flex-col gap-3 px-[18px] py-4">
        <div className="row justify-between gap-2!">
          <span className="font-bold">{t(m.importVerifyTitle, { name })}</span>
          <span className="badge mut">{t(m.optional)}</span>
        </div>
        <span className="k text-ink-nav!">{t(m.importCostShort, { count: e.toCheck, cost: money(e.cost), balance: e.balance ?? 0 })}</span>
        <div>
          <Button size="sm" disabledReason={why}>
            {t(m.verifyN, { count: e.toCheck })}
          </Button>
        </div>
      </section>
    )
  }
  return (
    <section className="card min-w-0" style={{ borderColor: '#99F6E4' }}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="h2">{t(m.importVerifyTitle, { name })}</h3>
          <span className="k">{t(m.importOptionalLead)}</span>
        </div>
        <span className="badge mut">{t(m.optional)}</span>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="vf-cost">
          <span className="big">{t(m.importCostLine, { count: e.toCheck, price: money(e.price, true), cost: money(e.cost) })}</span>
          {e.balance != null && e.after != null ? <span className="k text-ink-nav!">{t(m.balanceShort, { balance: e.balance, after: e.after })}</span> : null}
        </div>
        <Notice head={e.regionLabel ? t(m.noticeCount, { count: e.toCheck, name, region: e.regionLabel }) : t(m.noticeCountNoRegion, { count: e.toCheck, name })} name={name} />
        <span className="k text-ink-nav!">{t(m.importHappens)}</span>
        {running ? (
          <div className="flex items-center gap-2.5 text-[13px]" role="status">
            <span className="vf-spin" aria-hidden="true" />
            {t(m.importChecking, { name, checked: job.data?.checked ?? 0, total: job.data?.total ?? e.toCheck })}
          </div>
        ) : (
          <div className="row gap-2!">
            <Button variant="primary" onClick={() => void start()} loading={busy} busyLabel={t(m.starting)} disabledReason={busy || e.enough ? undefined : t(m.notEnough, { name, balance: e.balance ?? 0, needed: e.toCheck })}>
              {t(m.verifyN, { count: e.toCheck })}
            </Button>
            {onSkip ? <Button onClick={onSkip}>{t(m.importWithout)}</Button> : null}
          </div>
        )}
      </div>
    </section>
  )
}

/** The tiles' notes after a check: "403 fewer after the check", "46 not an address · 403 by ZeroBounce". */
export function useCheckedNotes(p: Preview) {
  const t = useT()
  const v = p.verification
  if (!v) return undefined
  return {
    willImport: t(m.importFewer, { count: v.fewer }),
    invalid: t(m.importInvalidSplit, { own: p.invalid - v.invalid, more: v.invalid, name: v.serviceName }),
    risky: t(m.importRiskySplit, { own: p.risky - v.risky, more: v.risky, name: v.serviceName }),
    header: v.checkedAt ? t(m.importCheckedLine, { name: v.serviceName, count: v.checked, time: clock(v.checkedAt), cost: money(v.cost) }) : undefined,
  }
}

/** Who found a risky group: ReputeCast, the service, or both. */
export function FoundBy({ g, service }: { g: RiskyGroup; service: string }) {
  const t = useT()
  const chip = (
    <span className="badge mut h-5! text-[11px]!" key="s">
      {service}
    </span>
  )
  if (g.agree > 0 && g.agree === g.own && g.more === 0) return <span className="k">{t(m.foundAgree, { name: service })}</span>
  if (g.own === 0 && g.more > 0) return chip
  if (g.more > 0)
    return (
      <span className="k whitespace-normal">
        {t(m.foundSplit, { own: g.own, more: g.more })} {chip}
      </span>
    )
  return <span className="k">{t(m.foundOwn)}</span>
}

/** An invalid group the service rejected: "rejected by ZeroBounce · 403". */
export function serviceInvalidLabel(t: ReturnType<typeof useT>, service: string, rows: number): string {
  return t(m.invalidByService, { name: service, count: rows })
}

/** The import summary's reasons a verification service adds. */
export const serviceReasonKeys = ['invalid_service', 'verify_catch_all', 'verify_disposable', 'verify_role', 'verify_abuse', 'verify_unknown'] as const

/** A service reason's row in the import summary: what it is, what happened. */
export function serviceReasonRow(key: string): [string, string] | null {
  switch (key) {
    case 'invalid_service':
      return [tr(m.reportInvalid), tr(m.reportInvalidBody)]
    case 'verify_catch_all':
      return [tr(m.reasonCatchAll), tr(m.reportTagged, { tag: 'verify:catch-all' })]
    case 'verify_disposable':
      return [tr(m.reasonDisposable), tr(m.reportTagged, { tag: 'verify:disposable' })]
    case 'verify_role':
      return [tr(m.reasonRole), tr(m.reportTagged, { tag: 'verify:role' })]
    case 'verify_abuse':
      return [tr(m.reasonAbuse), tr(m.reportTagged, { tag: 'verify:abuse' })]
    case 'verify_unknown':
      return [tr(m.resultUnknown), tr(m.reportTagged, { tag: 'verify:unknown' })]
  }
  return null
}

