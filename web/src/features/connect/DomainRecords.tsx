import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { clockSeconds } from '@/lib/format'
import { copy, failed } from './parts'
import { Ticks } from './text'
import './connect.css'

export type DomainDetail = Schemas['SendingDomainDetail']
type DomainRecord = Schemas['SendingDomainRecord']

const domainPath = '/api/v1/workspaces/{workspace}/domains/{domainId}' as const

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
  const [cloudflare, setCloudflare] = useState(false)
  const all = detail.counts.total > 0 && detail.counts.verified === detail.counts.total
  const zone = detail.records.map((r) => r.zoneLine).join('\n')
  return (
    <div className="flex flex-wrap items-center gap-2.5">
      {children}
      {detail.dnsHost ? (
        <Badge tone="ok" dot>
          {`Nameservers: ${detail.dnsHost}`}
        </Badge>
      ) : null}
      {detail.cloudflare ? (
        <Button size="sm" onClick={() => setCloudflare(true)} disabledReason={reason}>
          Add all records via Cloudflare API
        </Button>
      ) : null}
      <Button size="sm" onClick={() => void copy(zone + '\n', 'the zone file')}>
        Copy as zone file
      </Button>
      {detail.checkedAt ? (
        <span className="k ml-auto flex items-center gap-1.5" role="status">
          {all ? null : <span className="spin sm" aria-hidden="true" />}
          {all ? `All records verified · last check ${clockSeconds(detail.checkedAt)}` : `Checking every 30 s · last check ${clockSeconds(detail.checkedAt)}`}
        </span>
      ) : null}
      <CloudflareDialog workspace={workspace} detail={detail} open={cloudflare} onOpenChange={setCloudflare} />
    </div>
  )
}

/** The records table with the counts and the DMARC tip. */
export function DomainRecordsCard({ detail, title }: { detail: DomainDetail; title?: string }) {
  const n = detail.counts
  return (
    <section className="card flex flex-col" aria-labelledby={`records-${detail.id}`}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <h3 className="h2" id={`records-${detail.id}`}>
          {title ?? `Records · ${n.verified} of ${n.total} verified`}
        </h3>
        <div className="flex gap-2">
          {n.verified ? <Badge tone="ok">{`${n.verified} verified`}</Badge> : null}
          {n.pending ? <Badge tone="warn">{`${n.pending} pending`}</Badge> : null}
          {n.missing ? <Badge tone="bad">{`${n.missing} missing`}</Badge> : null}
        </div>
      </div>
      <Table className="cn-tbl" caption={`DNS records for ${detail.domain}`}>
        <thead>
          <tr>
            <Th>Purpose</Th>
            <Th>Type</Th>
            <Th>Name</Th>
            <Th>Value</Th>
            <Th>Status</Th>
            <Th>
              <span className="vh">Copy</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {detail.records.map((r) => (
            <tr key={r.key} className={cn(r.status === 'pending' && 'bg-[#FFFBEB]', (r.status === 'missing' || r.status === 'wrong_value') && 'bg-bad-tint')}>
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
                {r.status === 'wrong_value' && r.found ? <div className="k mt-1">Found: <span className="mono">{r.found}</span></div> : null}
              </td>
              <td>
                <RecordBadge record={r} />
              </td>
              <td>
                <Button size="sm" onClick={() => void copy(r.value, `the ${r.label} value`)} aria-label={`Copy the ${r.label} value`}>
                  Copy
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
      {detail.dmarcTip ? (
        <div className="flex flex-wrap gap-4 border-t border-line-2 px-5 py-3 text-[12.5px] text-ink-2">
          <span>
            <strong>DMARC tip:</strong> <Ticks text={detail.dmarcTip} />
          </span>
        </div>
      ) : null}
    </section>
  )
}

function RecordBadge({ record: r }: { record: DomainRecord }) {
  switch (r.status) {
    case 'verified':
      return <Badge tone="ok">{r.statusText || 'Verified'}</Badge>
    case 'pending':
      return (
        <Badge tone="warn">
          <span className="spin sm" style={{ borderTopColor: '#92400E' }} aria-hidden="true" />
          {r.statusText || 'Pending'}
        </Badge>
      )
    case 'missing':
    case 'wrong_value':
      return <Badge tone="bad">{r.statusText || 'Missing'}</Badge>
  }
  return <Badge tone="neutral">{r.statusText || "Couldn't check"}</Badge>
}

/** BIMI and the custom tracking domain. */
export function DomainOptional({ workspace, detail, reason }: { workspace: string; detail: DomainDetail; reason?: string }) {
  const refresh = useDomainRefresh(workspace, detail.id)
  const add = $api.useMutation('patch', domainPath)
  const t = detail.tracking
  const addTracking = () =>
    add.mutate(
      { params: { path: { workspace, domainId: detail.id } }, body: { trackingDomain: t.suggested } },
      { onSuccess: () => void refresh(), onError: failed("Couldn't add the tracking domain", addTracking) },
    )
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-4" aria-labelledby={`optional-${detail.id}`}>
      <h3 className="h2" id={`optional-${detail.id}`}>
        Optional
      </h3>
      <div className="flex flex-wrap items-start gap-3">
        <div className="flex flex-[1_1_300px] flex-col gap-1">
          <span className="text-[13px] font-semibold">BIMI</span>
          <span className="k">
            {detail.bimi === 'pass'
              ? 'Published: your logo shows in Gmail, Yahoo and Apple Mail where the receiver supports it.'
              : 'Shows your logo in Gmail, Yahoo and Apple Mail. Needs DMARC at quarantine or reject and an SVG Tiny PS logo; a Verified Mark Certificate is optional for Gmail.'}
          </span>
        </div>
        <div className="flex flex-[1_1_300px] flex-col gap-1">
          <span className="text-[13px] font-semibold">Custom tracking domain</span>
          <span className="k">
            <span className="mono">{t.domain || t.suggested}</span> CNAME to <span className="mono">{t.target}</span>
            {t.domain ? trackingState(t.status) : ', so links and the open pixel use your domain.'}
          </span>
        </div>
        {t.domain ? null : (
          <Button size="sm" className="self-start" onClick={addTracking} loading={add.isPending} disabledReason={reason}>
            Add tracking domain
          </Button>
        )}
      </div>
    </section>
  )
}

function trackingState(status: Schemas['SendingDomainTracking']['status']): string {
  switch (status) {
    case 'verified':
      return ': verified, links and the open pixel use your domain.'
    case 'pending':
      return ': waiting for DNS.'
    case 'wrong_value':
      return ': points somewhere else.'
    case 'missing':
      return ': add the CNAME at your DNS host.'
  }
  return '.'
}

function CloudflareDialog({ workspace, detail, open, onOpenChange }: { workspace: string; detail: DomainDetail; open: boolean; onOpenChange: (o: boolean) => void }) {
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
        title: changed ? `Added ${changed} ${changed === 1 ? 'record' : 'records'} in Cloudflare` : 'Cloudflare already had every record',
        detail: r.failed.length ? `Couldn't add: ${r.failed.join(', ')}. Add them by hand.` : 'ReputeCast checks them again every 30 seconds.',
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
      title="Add the records via the Cloudflare API"
      description={`ReputeCast adds the missing records to the ${detail.domain} zone and fixes wrong ones. Existing records it doesn't need stay as they are.`}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="primary" onClick={() => void submit()} loading={busy} busyLabel="Adding…" disabledReason={token.trim().length < 20 ? 'Paste a Cloudflare API token first.' : undefined}>
            Add records
          </Button>
        </>
      }
    >
      <Field label="Cloudflare API token" hint="Needs Zone › DNS › Edit on this zone. Used once and never stored." error={error}>
        <Input type="password" className="mono" value={token} onChange={(e) => setToken(e.target.value)} autoComplete="off" />
      </Field>
    </Dialog>
  )
}
