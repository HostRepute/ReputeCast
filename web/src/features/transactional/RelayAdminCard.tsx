import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { dayMonthYear } from '@/lib/format'
import { mr } from './elsewhere.messages'

type Relay = Schemas['SMTPRelaySettings']

/**
 * Instance admin › Instance & updates: the SMTP relay section
 * (InstanceSettings): turn on or off, hostname, ports, certificate and the
 * status checks.
 */
export function RelayAdminCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/smtp-relay')
  return (
    <section className="card" style={{ gridColumn: '1 / -1' }} aria-labelledby="relay-h">
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(mr.error)} actions={<Button onClick={() => void q.refetch()}>{t(mr.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <div className="pad flex flex-col gap-3" aria-busy="true">
          <Skeleton w="30%" h={16} />
          <Skeleton w="100%" h={90} />
        </div>
      ) : (
        <RelayForm key={`${q.data.enabled}|${q.data.hostname}|${q.data.starttlsPort}|${q.data.tlsPort}|${q.data.certSource}`} r={q.data} />
      )}
    </section>
  )
}

function RelayForm({ r }: { r: Relay }) {
  const t = useT()
  const qc = useQueryClient()
  const [hostname, setHostname] = useState(r.hostname || r.suggestedHostname)
  const [starttls, setStarttls] = useState(String(r.starttlsPort || 587))
  const [tls, setTls] = useState(String(r.tlsPort || 465))
  const [source, setSource] = useState<'acme' | 'files'>(r.certSource === 'files' ? 'files' : 'acme')
  const [certFile, setCertFile] = useState('')
  const [keyFile, setKeyFile] = useState('')
  const save = $api.useMutation('put', '/api/v1/admin/smtp-relay')
  const check = $api.useMutation('post', '/api/v1/admin/smtp-relay/check')
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/smtp-relay'] })
  const dirty = hostname !== r.hostname || Number(starttls) !== r.starttlsPort || Number(tls) !== r.tlsPort || source !== r.certSource || Boolean(certFile || keyFile)
  const unreachable = r.enabled && Boolean(r.unreachableSince)
  const put = (enabled: boolean, done: string, detail?: string) =>
    save.mutate(
      {
        body: {
          enabled,
          hostname: hostname.trim(),
          starttlsPort: Number(starttls) || 587,
          tlsPort: Number(tls) || 465,
          certSource: source,
          certFile: certFile || undefined,
          keyFile: keyFile || undefined,
        },
      },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: done, detail })
          void refresh()
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const caption = t(mr.caption)
  const checks = r.status.checks ?? []
  return (
    <>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="relay-h">
            {t(mr.title)}
          </h2>
          <span className="k">
            {r.enabled && r.enabledAt
              ? r.enabledBy
                ? t(mr.turnedOn, { caption, name: r.enabledBy, date: dayMonthYear(r.enabledAt) })
                : t(mr.turnedOnNoName, { caption, date: dayMonthYear(r.enabledAt) })
              : caption}
          </span>
        </div>
        {unreachable ? (
          <Badge tone="bad" dot>
            {t(mr.unreachable)}
          </Badge>
        ) : r.enabled ? (
          <Badge tone="ok" dot>
            {t(mr.on)}
          </Badge>
        ) : (
          <Badge tone="neutral">{t(mr.off)}</Badge>
        )}
      </div>
      <div className="pad grid items-start gap-5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(360px, 100%), 1fr))' }}>
        <div className="flex flex-col gap-3.5">
          {r.enabled ? null : <p className="m-0 text-[13px]">{t(mr.offBody)}</p>}
          <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(200px, 100%), 1fr))' }}>
            <label className="lbl">
              {t(mr.hostname)}
              <input className="in mono" value={hostname} maxLength={253} onChange={(e) => setHostname(e.target.value)} />
            </label>
            <div className="lbl" role="group" aria-labelledby="relay-ports">
              <span id="relay-ports">{t(mr.ports)}</span>
              <span className="flex items-center gap-1.5">
                <input className="in mono w-[72px]" inputMode="numeric" aria-label={t(mr.starttlsPort)} value={starttls} onChange={(e) => setStarttls(e.target.value)} />
                <span className="k">{t(mr.starttls)} ·</span>
                <input className="in mono w-[72px]" inputMode="numeric" aria-label={t(mr.tlsPort)} value={tls} onChange={(e) => setTls(e.target.value)} />
                <span className="k">{t(mr.tls)}</span>
              </span>
            </div>
          </div>
          <label className="lbl">
            {t(mr.certificate)}
            <select className="in" value={source} onChange={(e) => setSource(e.target.value as 'acme' | 'files')}>
              <option value="acme">{r.certSource === 'acme' && r.certificate?.notAfter ? t(mr.acmeRenews, { date: dayMonthYear(r.certificate.notAfter) }) : t(mr.acme)}</option>
              <option value="files">{t(mr.files)}</option>
            </select>
          </label>
          {source === 'files' ? (
            <div className="flex flex-col gap-2.5">
              <label className="lbl">
                {t(mr.certFile)}
                <textarea className="in mono" rows={3} value={certFile} onChange={(e) => setCertFile(e.target.value)} spellCheck={false} />
              </label>
              <label className="lbl">
                {t(mr.keyFile)}
                <textarea className="in mono" rows={3} value={keyFile} onChange={(e) => setKeyFile(e.target.value)} spellCheck={false} autoComplete="off" />
              </label>
              <span className="k">{t(mr.keyHint)}</span>
            </div>
          ) : null}
          {r.renewError ? <Note tone="bad">{t(mr.renewFailed, { error: r.renewError })}</Note> : null}
          <span className="k">{t(mr.help)}</span>
        </div>
        <div className="flex flex-col gap-2.5">
          {/* A list only once there are checks: "Not checked yet" isn't a list item. */}
          <div
            className="flex flex-col rounded-lg border border-line px-3"
            style={{ paddingTop: 2, paddingBottom: 2 }}
            aria-label={checks.length ? t(mr.checks) : undefined}
            role={checks.length ? 'list' : undefined}
          >
            {checks.length ? (
              checks.map((c, i) => (
                <div key={c.key} role="listitem" className="flex items-center gap-2.5 py-2 text-[13px]" style={i ? { borderTop: '1px solid #EEF0F3' } : undefined}>
                  <span className="flex" style={{ color: c.ok ? '#15803D' : '#B91C1C' }}>
                    <Icon name={c.ok ? 'check' : 'alertCircle'} size={14} />
                  </span>
                  <span className="flex-1">{c.detail}</span>
                  <Badge tone={c.ok ? 'ok' : 'bad'}>{c.ok ? t(mr.pass) : t(mr.fail)}</Badge>
                </div>
              ))
            ) : (
              <span className="k py-2">{t(mr.notChecked)}</span>
            )}
          </div>
          {r.status.checkedAt ? <span className="k self-end">{t(mr.checkedAt, { date: dayMonthYear(r.status.checkedAt) })}</span> : null}
          <div className="flex flex-wrap justify-end gap-2">
            {r.enabled ? (
              <>
                {dirty ? (
                  <Button size="sm" variant="primary" loading={save.isPending} onClick={() => put(true, t(mr.saved))}>
                    {t(mr.save)}
                  </Button>
                ) : null}
                <Button
                  size="sm"
                  loading={check.isPending}
                  onClick={() =>
                    check.mutate(
                      {},
                      {
                        onSuccess: () => void refresh(),
                        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                      },
                    )
                  }
                >
                  {t(mr.runChecks)}
                </Button>
                <Button size="sm" variant="danger" loading={save.isPending} onClick={() => put(false, t(mr.turnedOffToast), t(mr.turnedOffDetail))}>
                  {t(mr.turnOff)}
                </Button>
              </>
            ) : (
              <Button size="sm" variant="primary" loading={save.isPending} disabledReason={hostname.trim() ? undefined : t(mr.hostname)} onClick={() => put(true, t(mr.turnedOnToast))}>
                {t(mr.turnOn)}
              </Button>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
