import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { KeyValues } from '@/components/ui/Data'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { andList, count, dayMonthDate, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { providerColor } from './words'

function resultTone(v: string): Tone {
  return v === 'aligned' || v === 'pass' ? 'ok' : 'bad'
}

/** DMARC reports · last 7 days, per sending source. */
export function DmarcCard({ d, className }: { d: Schemas['DMARCSummary']; className?: string }) {
  const [investigate, setInvestigate] = useState<string | null>(null)
  const pass = d.passRate ?? 0
  const caption = d.via === 'hostrepute' && d.rua ? `aggregate reports to ${d.rua}` : `${count(d.reports)} uploaded aggregate ${d.reports === 1 ? 'report' : 'reports'}`
  const enforced = d.policy === 'quarantine' || d.policy === 'reject'
  return (
    <section className={className ?? 'card min-w-0 flex-[1_1_440px]'} aria-labelledby="h-dmarc">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="h-dmarc">
            DMARC reports · last {d.days} days
          </h2>
          <span className="k">{caption}</span>
        </div>
        <Badge tone={pass >= 98 ? 'ok' : pass >= 90 ? 'warn' : 'bad'}>{trimPct(d.passRate)} pass</Badge>
      </div>
      <Table density="dense" phoneWrap caption="DMARC results by sending source">
        <thead>
          <tr>
            <Th>Source</Th>
            <Th align="right">Messages</Th>
            <Th>SPF</Th>
            <Th>DKIM</Th>
            <Th>Result</Th>
          </tr>
        </thead>
        <tbody>
          {d.sources.map((s) => (
            <tr key={s.label} className={s.known ? undefined : 'bg-bad-tint'}>
              <Td>
                <span className="flex items-center gap-2">
                  <span className="dot" style={{ color: s.known ? providerColor(s.adapter) : '#B91C1C' }} aria-hidden="true" />
                  {s.label}
                </span>
                {s.known ? null : <div className="k">not one of your providers</div>}
              </Td>
              <Td align="right" num>
                {count(s.messages)}
              </Td>
              <Td>
                <Badge tone={resultTone(s.spf)}>{s.spf}</Badge>
              </Td>
              <Td>
                <Badge tone={resultTone(s.dkim)}>{s.dkim}</Badge>
              </Td>
              <Td>
                <Badge tone={resultTone(s.result)}>{s.result}</Badge>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
      {d.unknown ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
          <span className="k">
            {enforced
              ? `Someone is sending as ${d.domain} from an unknown host; p=${d.policy} kept it out of inboxes.`
              : `Someone is sending as ${d.domain} from an unknown host; with p=${d.policy || 'none'}, receivers may still deliver it.`}
          </span>
          <Button size="sm" onClick={() => setInvestigate(d.unknown!.ip ?? null)}>
            Investigate source
          </Button>
        </div>
      ) : null}
      {investigate ? <InvestigateDialog ip={investigate} domain={d.domain ?? ''} onClose={() => setInvestigate(null)} /> : null}
    </section>
  )
}

/** "Investigate source": what the reports say about one sending IP. */
function InvestigateDialog({ ip, domain, onClose }: { ip: string; domain: string; onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/deliverability/dmarc/sources/{ip}', { params: { path: { workspace: slug, ip } } })
  const s = q.data
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`Sending source ${ip}`}
      description={s?.known ? `${s.provider} sends from this IP.` : `This IP isn't one of your providers. It sent mail as ${domain} in the last 7 days.`}
      footer={<Button onClick={onClose}>Close</Button>}
    >
      {q.isPending ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="70%" />
          <Skeleton w="50%" />
          <Skeleton w="60%" />
        </div>
      ) : q.isError || !s ? (
        <Note tone="bad">{errorMessage(q.error)} Nothing was changed.</Note>
      ) : (
        <>
          <KeyValues
            items={[
              ['Messages, 7 d', `${count(s.messages)} in ${s.reports} ${s.reports === 1 ? 'report' : 'reports'}`],
              ['Seen', `${dayMonthDate(s.firstSeen)} to ${dayMonthDate(s.lastSeen)}`],
              ['Reported by', andList(s.reporters)],
              ['Reverse DNS', <span className="mono">{s.ptr || 'none'}</span>],
              ['From domains', <span className="mono">{s.headerFrom.join(', ') || '—'}</span>],
              ['SPF', `${s.spf}${s.spfDomains.length ? ` (${s.spfDomains.join(', ')})` : ''}`],
              ['DKIM', `${s.dkim}${s.dkimDomains.length ? ` (${s.dkimDomains.join(', ')})` : ''}`],
              ['What receivers did', s.disposition === 'none' ? 'delivered (no DMARC action)' : s.disposition === 'quarantine' ? 'quarantined (spam folder)' : 'rejected'],
            ]}
          />
          {!s.known ? (
            <Note tone={s.disposition === 'none' ? 'warn' : 'info'}>
              If this is a service you use (a CRM, a help desk, an invoicing tool), add it to the SPF record of {domain} and turn on DKIM signing there. If it isn't, nothing
              to do: {s.policy ? `p=${s.policy}` : 'your DMARC policy'} {s.disposition === 'none' ? 'should be raised to quarantine so receivers stop delivering it' : 'keeps it out of inboxes'}.
            </Note>
          ) : null}
        </>
      )}
    </Dialog>
  )
}

/** DMARC report summary without reports (HostRepute not receiving them). */
export function DmarcEmptyCard({ connected, rua }: { connected: boolean; rua?: string }) {
  const { slug } = useWorkspace()
  const [upload, setUpload] = useState(false)
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-4.5 py-4" aria-labelledby="h-dmarc-empty">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="h-dmarc-empty">
          DMARC report summary
        </h2>
        <Badge tone="neutral">No reports yet</Badge>
      </div>
      {connected && rua ? (
        <span className="k">
          Aggregate reports need a mailbox to land in. Point <span className="mono">rua=</span> at <span className="mono">{rua}</span> and HostRepute parses them for you, or paste
          the XML reports here.
        </span>
      ) : (
        <span className="k">
          Aggregate reports need a mailbox to land in. Point <span className="mono">rua=</span> at HostRepute (parsed for you) or at any mailbox you own and paste the XML reports
          here.
        </span>
      )}
      <div className="flex flex-wrap gap-2">
        {connected ? null : (
          <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            Use HostRepute as rua
          </ButtonLink>
        )}
        <UploadButton open={upload} setOpen={setUpload} />
      </div>
    </section>
  )
}

/** "Upload report XML" and its dialog. */
export function UploadButton({ open, setOpen }: { open: boolean; setOpen: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('domains:manage')
  const qc = useQueryClient()
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  async function send() {
    if (!file) {
      setError('Choose a report file first.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/deliverability/dmarc/reports', {
        params: { path: { workspace: slug } },
        body: file as unknown as string,
        bodySerializer: (b: unknown) => b as BodyInit,
        headers: { 'Content-Type': 'application/octet-stream' },
      })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/deliverability'] })
      setOpen(false)
      setFile(null)
      toast({ title: `Report from ${data?.orgName} added`, detail: `${count(data?.messages ?? 0)} messages for ${data?.domain}` })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <Button size="sm" onClick={() => setOpen(true)} disabledReason={reason}>
        Upload report XML
      </Button>
      <Dialog
        open={open}
        onOpenChange={(o) => {
          setOpen(o)
          if (!o) {
            setFile(null)
            setError(null)
          }
        }}
        title="Upload a DMARC aggregate report"
        description="The .xml file a mailbox provider sent to your rua= address, or the .xml.gz or .zip it came in. Only the report's counts are stored."
        footer={
          <>
            <Button onClick={() => setOpen(false)}>Cancel</Button>
            <Button variant="primary" onClick={send} loading={busy}>
              Upload
            </Button>
          </>
        }
      >
        <div className="lbl">
          <label htmlFor="dmarc-file">Report file</label>
          <input
            ref={input}
            id="dmarc-file"
            type="file"
            accept=".xml,.gz,.zip,application/xml,text/xml,application/gzip,application/zip"
            className="in pt-1.5"
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? 'dmarc-file-err' : undefined}
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
          {error ? (
            <span className="err" id="dmarc-file-err" role="alert">
              {error}
            </span>
          ) : null}
        </div>
      </Dialog>
    </>
  )
}
