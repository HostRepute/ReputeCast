import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { clockSeconds, count } from '@/lib/format'
import { md } from './domain.messages'
import { copy, failed } from './parts'
import { Ticks } from './text'
import './connect.css'

export type DomainDetail = Schemas['SendingDomainDetail']
type DomainRecord = Schemas['SendingDomainRecord']

const domainPath = '/api/v1/workspaces/{workspace}/domains/{domainId}' as const

const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>

/**
 * A sending domain's records, checked live: every 30 seconds while some
 * aren't verified yet. `provider` narrows them to one provider's records
 * (and the domain-wide SPF and DMARC).
 */
export function useDomainDetail(workspace: string, domainId: string | undefined, provider?: string) {
  return $api.useQuery(
    'get',
    domainPath,
    { params: { path: { workspace, domainId: domainId ?? '' }, query: { provider, refresh: true } } },
    {
      enabled: Boolean(domainId),
      refetchInterval: (q) => {
        const c = q.state.data?.counts
        return c && c.verified < c.total ? 30_000 : false
      },
    },
  )
}

/** Puts a fresh domain detail in the cache of every view of it. */
export function useDomainRefresh(workspace: string, domainId: string) {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['get', domainPath, { params: { path: { workspace, domainId } } }] })
}

/** "Nameservers: Cloudflare", Cloudflare, zone file, and the live check. */
export function DomainTools({
  workspace,
  detail,
  reason,
  children,
}: {
  workspace: string
  detail: DomainDetail
  reason?: string
  /** The domain input (setup) or nothing (Settings). */
  children?: React.ReactNode
}) {
  const t = useT()
  const [cloudflare, setCloudflare] = useState(false)
  const all = detail.counts.total > 0 && detail.counts.verified === detail.counts.total
  const zone = detail.records.map((r) => r.zoneLine).join('\n')
  return (
    <div className="flex flex-wrap items-center gap-2.5">
      {children}
      {detail.dnsHost ? (
        <Badge tone="ok" dot>
          {t(md.nameservers, { host: detail.dnsHost })}
        </Badge>
      ) : null}
      {detail.cloudflare ? (
        <Button size="sm" onClick={() => setCloudflare(true)} disabledReason={reason}>
          {t(md.addViaCloudflare)}
        </Button>
      ) : null}
      <Button size="sm" onClick={() => void copy(zone + '\n', 'zone')}>
        {t(md.copyZone)}
      </Button>
      {detail.checkedAt ? (
        <span className="k ml-auto flex items-center gap-1.5" role="status">
          {all ? null : <span className="spin sm" aria-hidden="true" />}
          {all ? t(md.allVerified, { time: clockSeconds(detail.checkedAt) }) : t(md.checking, { time: clockSeconds(detail.checkedAt) })}
        </span>
      ) : null}
      <CloudflareDialog workspace={workspace} detail={detail} open={cloudflare} onOpenChange={setCloudflare} />
    </div>
  )
}

/** The records table with the counts and the DMARC tip. */
export function DomainRecordsCard({ detail, title }: { detail: DomainDetail; title?: string }) {
  const t = useT()
  const n = detail.counts
  return (
    <section className="card flex flex-col" aria-labelledby={`records-${detail.id}`}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h3 className="h2" id={`records-${detail.id}`}>
          {title ?? t(md.recordsTitle, { verified: count(n.verified), total: count(n.total) })}
        </h3>
        <div className="flex gap-2">
          {n.verified ? <Badge tone="ok">{t(md.verifiedCount, { count: count(n.verified) })}</Badge> : null}
          {n.pending ? <Badge tone="warn">{t(md.pendingCount, { count: count(n.pending) })}</Badge> : null}
          {n.missing ? <Badge tone="bad">{t(md.missingCount, { count: count(n.missing) })}</Badge> : null}
        </div>
      </div>
      <Table className="cn-tbl" caption={t(md.caption, { domain: detail.domain })}>
        <thead>
          <tr>
            <Th>{t(md.colPurpose)}</Th>
            <Th>{t(md.colType)}</Th>
            <Th>{t(md.colName)}</Th>
            <Th>{t(md.colValue)}</Th>
            <Th>{t(md.colStatus)}</Th>
            <Th>
              <span className="vh">{t(md.colCopy)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {detail.records.map((r) => (
            <tr key={r.key} className={cn((r.status === 'pending' || stale(r)) && 'bg-[#FFFBEB]', (r.status === 'missing' || (r.status === 'wrong_value' && !stale(r))) && 'bg-bad-tint')}>
              <td className="font-semibold">
                {r.label}
                {r.note ? <div className="k">{r.note}</div> : null}
              </td>
              <td className="mono">{r.type}</td>
              <td>
                <span className="cn-val">{r.name}</span>
              </td>
              <td>
                <span className="cn-val">{r.value}</span>
                {r.status === 'wrong_value' && r.found && !stale(r) ? (
                  <div className="k mt-1">
                    <T msg={md.found} values={{ value: r.found, mono }} />
                  </div>
                ) : null}
              </td>
              <td>
                <RecordBadge record={r} />
              </td>
              <td>
                <Button size="sm" onClick={() => void copy(r.value, { record: r.label })} aria-label={t(md.copyValueLabel, { record: r.label })}>
                  {t(common.copy)}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
      {detail.dmarcTip ? (
        <div className="flex flex-wrap gap-4 border-t border-line-2 px-5 py-3 text-[12.5px] text-ink-2">
          <span>
            <T msg={md.dmarcTip} values={{ tip: <Ticks text={detail.dmarcTip} />, b: (chunks) => <strong>{chunks}</strong> }} />
          </span>
        </div>
      ) : null}
    </section>
  )
}

/**
 * A TXT record that still shows its old value (the SPF merge, a DMARC
 * change): resolvers serve the cached value until its TTL runs out, so it
 * is pending, not wrong.
 */
function stale(r: DomainRecord) {
  return r.status === 'wrong_value' && r.type.toUpperCase() === 'TXT'
}

function RecordBadge({ record: r }: { record: DomainRecord }) {
  const t = useT()
  if (stale(r)) {
    return (
      <Badge tone="warn">
        <span className="spin sm" style={{ borderTopColor: '#92400E' }} aria-hidden="true" />
        {r.statusText || t(md.statusPending)}
      </Badge>
    )
  }
  switch (r.status) {
    case 'verified':
      return <Badge tone="ok">{r.statusText || t(md.statusVerified)}</Badge>
    case 'pending':
      return (
        <Badge tone="warn">
          <span className="spin sm" style={{ borderTopColor: '#92400E' }} aria-hidden="true" />
          {r.statusText || t(md.statusPending)}
        </Badge>
      )
    case 'missing':
    case 'wrong_value':
      return <Badge tone="bad">{r.statusText || t(md.statusMissing)}</Badge>
  }
  return <Badge tone="neutral">{r.statusText || t(md.statusUnknown)}</Badge>
}

/** BIMI and the custom tracking domain. */
export function DomainOptional({ workspace, detail, reason }: { workspace: string; detail: DomainDetail; reason?: string }) {
  const t = useT()
  const refresh = useDomainRefresh(workspace, detail.id)
  const add = $api.useMutation('patch', domainPath)
  const tr = detail.tracking
  const addTracking = () =>
    add.mutate(
      { params: { path: { workspace, domainId: detail.id } }, body: { trackingDomain: tr.suggested } },
      { onSuccess: () => void refresh(), onError: failed(t(md.addTrackingFailed), addTracking) },
    )
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-4" aria-labelledby={`optional-${detail.id}`}>
      <h3 className="h2" id={`optional-${detail.id}`}>
        {t(md.optionalTitle)}
      </h3>
      <div className="flex flex-wrap items-start gap-3">
        <div className="flex flex-[1_1_300px] flex-col gap-1">
          <span className="text-[13px] font-semibold">{'BIMI'}</span>
          <span className="k">{detail.bimi === 'pass' ? t(md.bimiPublished) : t(md.bimiInfo)}</span>
        </div>
        <div className="flex flex-[1_1_300px] flex-col gap-1">
          <span className="text-[13px] font-semibold">{t(md.trackingTitle)}</span>
          <span className="k">
            <T msg={md.trackingLine} values={{ state: tr.domain ? tr.status : 'suggested', domain: tr.domain || tr.suggested, target: tr.target, mono }} />
          </span>
        </div>
        {tr.domain ? null : (
          <Button size="sm" className="self-start" onClick={addTracking} loading={add.isPending} disabledReason={reason}>
            {t(md.addTracking)}
          </Button>
        )}
      </div>
    </section>
  )
}

function CloudflareDialog({ workspace, detail, open, onOpenChange }: { workspace: string; detail: DomainDetail; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const qc = useQueryClient()
  const [token, setToken] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/domains/{domainId}/cloudflare', {
        params: { path: { workspace, domainId: detail.id } },
        body: { token: token.trim() },
      })
      const r = data!.result
      const changed = r.created.length + r.updated.length
      toast({
        title: changed ? t(md.cfAdded, { count: changed }) : t(md.cfNothing),
        detail: r.failed.length ? t(md.cfFailedSome, { records: r.failed.join(', ') }) : t(md.cfChecks),
        kind: r.failed.length ? 'info' : 'success',
      })
      await qc.invalidateQueries({ queryKey: ['get', domainPath] })
      setToken('')
      onOpenChange(false)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) setToken('')
        onOpenChange(o)
      }}
      size="sm"
      title={t(md.cfTitle)}
      description={t(md.cfDescription, { domain: detail.domain })}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={() => void submit()} loading={busy} busyLabel={t(md.adding)} disabledReason={token.trim().length < 20 ? t(md.pasteToken) : undefined}>
            {t(md.addRecords)}
          </Button>
        </>
      }
    >
      <Field label={t(md.tokenLabel)} hint={t(md.tokenHint, { permission: 'Zone › DNS › Edit' })} error={error}>
        <Input type="password" className="mono" value={token} onChange={(e) => setToken(e.target.value)} autoComplete="off" />
      </Field>
    </Dialog>
  )
}
