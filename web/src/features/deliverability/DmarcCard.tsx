import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { KeyValues } from '@/components/ui/Data'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count, dayMonthDate, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { md } from './dmarc.messages'
import { providerColor } from './words'

function resultTone(v: string): Tone {
  return v === 'aligned' || v === 'pass' ? 'ok' : 'bad'
}

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

/** DMARC reports · last 7 days, per sending source. */
export function DmarcCard({ d, className }: { d: Schemas['DeliverabilityDMARCSummary']; className?: string }) {
  const t = useT()
  const [investigate, setInvestigate] = useState<string | null>(null)
  const pass = d.passRate ?? 0
  const caption = d.via === 'hostrepute' && d.rua ? t(md.captionRua, { rua: d.rua }) : t(md.captionUploaded, { count: d.reports })
  const enforced = d.policy === 'quarantine' || d.policy === 'reject'
  return (
    <section className={className ?? 'card min-w-0 flex-[1_1_440px]'} aria-labelledby="h-dmarc">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="h-dmarc">
            {t(md.title, { days: d.days })}
          </h2>
          <span className="k">{caption}</span>
        </div>
        <Badge tone={pass >= 98 ? 'ok' : pass >= 90 ? 'warn' : 'bad'}>{t(md.passBadge, { rate: trimPct(d.passRate) })}</Badge>
      </div>
      <Table density="dense" phoneWrap caption={t(md.tableCaption)}>
        <thead>
          <tr>
            <Th>{t(md.colSource)}</Th>
            <Th align="right">{t(md.colMessages)}</Th>
            <Th>{'SPF'}</Th>
            <Th>{'DKIM'}</Th>
            <Th>{t(md.colResult)}</Th>
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
                {s.known ? null : <div className="k">{t(md.notYourProvider)}</div>}
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
              ? t(md.unknownEnforced, { domain: d.domain ?? '', policy: d.policy ?? '' })
              : t(md.unknownNotEnforced, { domain: d.domain ?? '', policy: d.policy || 'none' })}
          </span>
          <Button size="sm" onClick={() => setInvestigate(d.unknown!.ip ?? null)}>
            {t(md.investigate)}
          </Button>
        </div>
      ) : null}
      {investigate ? <InvestigateDialog ip={investigate} domain={d.domain ?? ''} onClose={() => setInvestigate(null)} /> : null}
    </section>
  )
}

/** "Investigate source": what the reports say about one sending IP. */
function InvestigateDialog({ ip, domain, onClose }: { ip: string; domain: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/deliverability/dmarc/sources/{ip}', { params: { path: { workspace: slug, ip } } })
  const s = q.data
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(md.sourceTitle, { ip })}
      description={s?.known ? t(md.sourceKnown, { provider: s.provider ?? '' }) : t(md.sourceUnknown, { domain })}
      footer={<Button onClick={onClose}>{t(common.close)}</Button>}
    >
      {q.isPending ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="70%" />
          <Skeleton w="50%" />
          <Skeleton w="60%" />
        </div>
      ) : q.isError || !s ? (
        <Note tone="bad">
          {errorMessage(q.error)} {t(md.nothingChanged)}
        </Note>
      ) : (
        <>
          <KeyValues
            items={[
              [t(md.messages7d), t(md.messagesInReports, { messages: count(s.messages), reports: s.reports })],
              [t(md.seen), t(md.seenRange, { from: dayMonthDate(s.firstSeen), to: dayMonthDate(s.lastSeen) })],
              [t(md.reportedBy), andList(s.reporters)],
              [t(md.reverseDns), <span className="mono">{s.ptr || t(md.none)}</span>],
              [t(md.fromDomains), <span className="mono">{s.headerFrom.join(', ') || '—'}</span>],
              ['SPF', `${s.spf}${s.spfDomains.length ? ` (${s.spfDomains.join(', ')})` : ''}`],
              ['DKIM', `${s.dkim}${s.dkimDomains.length ? ` (${s.dkimDomains.join(', ')})` : ''}`],
              [
                t(md.receiversDid),
                s.disposition === 'none' ? t(md.dispositionNone) : s.disposition === 'quarantine' ? t(md.dispositionQuarantine) : t(md.dispositionReject),
              ],
            ]}
          />
          {!s.known ? (
            <Note tone={s.disposition === 'none' ? 'warn' : 'info'}>
              {t(md.unknownAdvice, { domain, hasPolicy: s.policy ? 'yes' : 'no', policy: s.policy ?? '', delivered: s.disposition === 'none' ? 'yes' : 'no' })}
            </Note>
          ) : null}
        </>
      )}
    </Dialog>
  )
}

/** DMARC report summary without reports (HostRepute not receiving them). */
export function DmarcEmptyCard({ connected, rua }: { connected: boolean; rua?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [upload, setUpload] = useState(false)
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-4.5 py-4" aria-labelledby="h-dmarc-empty">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="h-dmarc-empty">
          {t(md.emptyTitle)}
        </h2>
        <Badge tone="neutral">{t(md.noReports)}</Badge>
      </div>
      <span className="k">{connected && rua ? <T msg={md.emptyRua} values={{ rua, mono }} /> : <T msg={md.emptyNoRua} values={{ mono }} />}</span>
      <div className="flex flex-wrap gap-2">
        {connected ? null : (
          <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            {t(md.useHostRepute)}
          </ButtonLink>
        )}
        <UploadButton open={upload} setOpen={setUpload} />
      </div>
    </section>
  )
}

/** "Upload report XML" and its dialog. */
export function UploadButton({ open, setOpen }: { open: boolean; setOpen: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('domains:manage')
  const qc = useQueryClient()
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  async function send() {
    if (!file) {
      setError(t(md.chooseFile))
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
      toast({ title: t(md.uploaded, { org: data?.orgName ?? '' }), detail: t(md.uploadedDetail, { messages: count(data?.messages ?? 0), domain: data?.domain ?? '' }) })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <Button size="sm" onClick={() => setOpen(true)} disabledReason={reason}>
        {t(md.uploadButton)}
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
        title={t(md.uploadTitle)}
        description={t(md.uploadDescription)}
        footer={
          <>
            <Button onClick={() => setOpen(false)}>{t(common.cancel)}</Button>
            <Button variant="primary" onClick={send} loading={busy}>
              {t(md.upload)}
            </Button>
          </>
        }
      >
        <div className="lbl">
          <label htmlFor="dmarc-file">{t(md.reportFile)}</label>
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
