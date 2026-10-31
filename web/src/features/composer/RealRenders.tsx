import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, dayMonthDate } from '@/lib/format'
import { useCan } from '@/lib/session'
import { useComposer } from './context'
import { mr } from './renders.messages'

/*
 * Real renders (CO-13, ComposerPreview): the built-in previews say they're
 * approximations, and a workspace that connects its own Litmus or Mailgun
 * Inspect account gets screenshots from real email clients.
 */

type Service = 'litmus' | 'mailgun_inspect'

const SERVICES: Record<Service, string> = { litmus: 'Litmus', mailgun_inspect: 'Mailgun Inspect' }

/** "Litmus", "Mailgun Inspect". */
export function serviceName(s: string | undefined): string {
  return (s && SERVICES[s as Service]) || s || ''
}

const servicePath = '/api/v1/workspaces/{workspace}/rendering-service'
const rendersPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/renders'

/** The workspace's rendering service (connected or not). */
export function useRendering() {
  const c = useComposer()
  return $api.useQuery('get', servicePath, { params: { path: { workspace: c.slug } } }, { staleTime: 60_000 })
}

function InfoIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true" style={{ flex: 'none' }}>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v4M8 4.9v.2" />
    </svg>
  )
}

const bold = (chunks: ReactNode) => <b>{chunks}</b>

/**
 * The note above the previews: they're drawn in the browser; real renders
 * need (or come from) a rendering service. Connect for people who manage
 * integrations, Render in real clients for people who edit the campaign.
 */
export function ApproxNote({ onConnect, onRender }: { onConnect: () => void; onRender: () => void }) {
  const t = useT()
  const c = useComposer()
  const canManage = useCan('integrations:manage')
  const r = useRendering().data
  const service = serviceName(r?.service)
  return (
    <div className="honest" role="note">
      <InfoIcon />
      <span className="t">
        {!r?.connected ? (
          <T msg={mr.note} values={{ b: bold }} />
        ) : r.status === 'failing' ? (
          <T msg={mr.noteFailing} values={{ service, error: r.lastError ?? '', b: bold }} />
        ) : (
          <T msg={mr.noteConnected} values={{ service, b: bold }} />
        )}
      </span>
      {r?.connected ? (
        <>
          {canManage ? (
            <button type="button" className="btn sm" onClick={onConnect}>
              {t(mr.manage)}
            </button>
          ) : null}
          {c.canEdit ? (
            <button type="button" className="btn sm" onClick={onRender}>
              {t(mr.renderReal)}
            </button>
          ) : null}
        </>
      ) : canManage ? (
        <button type="button" className="btn sm" onClick={onConnect}>
          {t(mr.connect)}
        </button>
      ) : (
        <span className="k">{t(mr.whoConnects)}</span>
      )}
    </div>
  )
}

/** Connect Litmus or Mailgun Inspect with the workspace's own key; test, replace or disconnect it. */
export function ConnectRenderingDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const c = useComposer()
  const qc = useQueryClient()
  const id = useId()
  const r = useRendering().data
  const [service, setService] = useState<Service>('litmus')
  const [key, setKey] = useState('')
  const [region, setRegion] = useState<'us' | 'eu'>('us')
  const [busy, setBusy] = useState<'' | 'connect' | 'test'>('')
  const [error, setError] = useState('')
  const [replacing, setReplacing] = useState(false)
  const [confirmOff, setConfirmOff] = useState(false)
  const path = { params: { path: { workspace: c.slug } } }
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', servicePath] })
  const connect = async () => {
    setBusy('connect')
    setError('')
    try {
      await http.PUT(servicePath, { ...path, body: { service, apiKey: key.trim(), region: service === 'mailgun_inspect' ? region : undefined } })
      setKey('')
      setReplacing(false)
      await refresh()
      toast({ kind: 'success', title: t(mr.connectedToast, { service: serviceName(service) }) })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy('')
    }
  }
  const test = async () => {
    setBusy('test')
    setError('')
    try {
      const { data } = await http.POST(`${servicePath}/test`, path)
      await refresh()
      if (data?.status === 'connected') toast({ kind: 'success', title: t(mr.checkedOk, { service: serviceName(data.service), count: data.clients.length }) })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy('')
    }
  }
  const disconnect = async () => {
    const name = serviceName(r?.service)
    await http.DELETE(servicePath, path)
    await refresh()
    toast({ kind: 'info', title: t(mr.disconnected, { service: name }) })
  }
  const close = () => {
    setReplacing(false)
    setError('')
    setKey('')
    onClose()
  }
  const name = serviceName(r?.service)
  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()} title={t(mr.connectTitle)} description={t(mr.connectLead)}>
      {r?.connected && !replacing ? (
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1 rounded-lg border border-line" style={{ padding: 12 }}>
            <span className="font-semibold">
              {r.connectedBy
                ? t(mr.connectedLine, { service: name, hint: r.keyHint ?? '', name: r.connectedBy, date: dayMonthDate(r.connectedAt ?? new Date()) })
                : t(mr.connectedAnon, { service: name, hint: r.keyHint ?? '', date: dayMonthDate(r.connectedAt ?? new Date()) })}
            </span>
            <span className="k">{t(mr.clientsLine, { count: r.clients.length })}</span>
            {r.status === 'failing' ? (
              <span className="text-bad-fg" style={{ fontSize: 13 }}>
                {t(mr.failingLine, { error: r.lastError ?? '' })}
              </span>
            ) : null}
          </div>
          {error ? (
            <p role="alert" className="m-0 text-bad-fg">
              {error}
            </p>
          ) : null}
          <div className="flex flex-wrap items-center gap-2">
            <button type="button" className="btn" onClick={() => void test()} disabled={!!busy} aria-busy={busy === 'test' || undefined}>
              {busy === 'test' ? t(mr.testing) : t(mr.testAgain)}
            </button>
            <button type="button" className="btn" onClick={() => setReplacing(true)} disabled={!!busy}>
              {t(mr.replaceKey)}
            </button>
            <span className="flex-1" />
            <button type="button" className="btn danger" onClick={() => setConfirmOff(true)} disabled={!!busy}>
              {t(mr.disconnect)}
            </button>
          </div>
        </div>
      ) : (
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            if (key.trim().length >= 8) void connect()
          }}
        >
          <fieldset className="m-0 flex flex-wrap gap-x-4 gap-y-1 border-0 p-0">
            <legend className="mb-1 font-semibold">{t(mr.service)}</legend>
            {(Object.keys(SERVICES) as Service[]).map((s) => (
              <label key={s} className="tg">
                <input type="radio" name={`${id}-service`} value={s} checked={service === s} onChange={() => setService(s)} />
                {SERVICES[s]}
              </label>
            ))}
          </fieldset>
          <label className="font-semibold" htmlFor={`${id}-key`}>
            {t(mr.apiKey)}
          </label>
          <input
            id={`${id}-key`}
            className="in mono"
            type="password"
            autoComplete="off"
            spellCheck={false}
            value={key}
            onChange={(e) => setKey(e.target.value)}
            aria-describedby={`${id}-help`}
            aria-invalid={error ? true : undefined}
          />
          <span id={`${id}-help`} className="k">
            {service === 'litmus' ? t(mr.keyHelpLitmus) : t(mr.keyHelpMailgun)}
          </span>
          {service === 'mailgun_inspect' ? (
            <label className="flex flex-col gap-1">
              <span className="font-semibold">{t(mr.region)}</span>
              <select className="in" value={region} onChange={(e) => setRegion(e.target.value as 'us' | 'eu')} style={{ maxWidth: 160 }}>
                <option value="us">{t(mr.regionUs)}</option>
                <option value="eu">{t(mr.regionEu)}</option>
              </select>
            </label>
          ) : null}
          {error ? (
            <p role="alert" className="m-0 text-bad-fg">
              {error}
            </p>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            {replacing ? (
              <button type="button" className="btn" onClick={() => setReplacing(false)}>
                {t(common.cancel)}
              </button>
            ) : null}
            <button type="submit" className="btn pri" disabled={key.trim().length < 8 || !!busy} aria-busy={busy === 'connect' || undefined}>
              {busy === 'connect' ? t(mr.testing) : t(mr.testAndConnect)}
            </button>
          </div>
        </form>
      )}
      <ConfirmDialog
        open={confirmOff}
        onOpenChange={setConfirmOff}
        tone="bad"
        badge={t(mr.disconnectBadge)}
        title={t(mr.disconnectTitle, { service: name })}
        description={t(mr.disconnectBody)}
        confirmLabel={t(mr.disconnect)}
        busyLabel={t(mr.disconnecting)}
        onConfirm={disconnect}
      />
    </Dialog>
  )
}

/** Pick the clients and send the saved version for real renders. */
export function RenderDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const c = useComposer()
  const qc = useQueryClient()
  const r = useRendering().data
  const service = serviceName(r?.service)
  const clients = r?.clients ?? []
  const [picked, setPicked] = useState<string[] | null>(null)
  const chosen = picked ?? clients.filter((x) => x.default).map((x) => x.id)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const testing = !!c.abTest && c.abTest.kind !== 'send_time'
  const close = () => {
    setPicked(null)
    setError('')
    onClose()
  }
  const start = async () => {
    setBusy(true)
    setError('')
    try {
      if (c.hasUnsaved?.() && !(await c.save({ quiet: true }))) {
        setError(t(mr.saveFailed, { service }))
        return
      }
      await http.POST(rendersPath, {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: { clients: chosen, variant: testing ? (c.variant as 'A' | 'B' | 'C') : undefined },
      })
      await qc.invalidateQueries({ queryKey: ['get', rendersPath] })
      close()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()} title={t(mr.renderTitle)} description={t(mr.renderLead, { service })}>
      <form
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          if (chosen.length) void start()
        }}
      >
        {testing ? <p className="k m-0">{t(mr.renderVariant, { key: c.variant })}</p> : null}
        {c.hasUnsaved?.() ? <p className="k m-0">{t(mr.unsavedFirst)}</p> : null}
        <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
          <legend className="mb-1 font-semibold">{t(mr.clients)}</legend>
          {clients.length ? (
            <div className="flex flex-col gap-1 overflow-auto" style={{ maxHeight: 280 }}>
              {clients.map((x) => (
                <label key={x.id} className="tg">
                  <input
                    type="checkbox"
                    checked={chosen.includes(x.id)}
                    onChange={(e) => setPicked(e.target.checked ? [...chosen, x.id] : chosen.filter((id) => id !== x.id))}
                  />
                  {x.name}
                </label>
              ))}
            </div>
          ) : (
            <p className="k m-0">{t(mr.noClients, { service })}</p>
          )}
        </fieldset>
        {error ? (
          <p role="alert" className="m-0 text-bad-fg">
            {error}
          </p>
        ) : null}
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="btn" onClick={close}>
            {t(common.cancel)}
          </button>
          <button type="submit" className="btn pri" disabled={!chosen.length || chosen.length > 20 || busy} aria-busy={busy || undefined}>
            {busy ? t(mr.starting) : t(mr.startRender, { count: chosen.length })}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

type Render = Schemas['Render']

/** The latest real render of the campaign: one screenshot per client, refreshed while the service captures. */
export function RealRendersCard() {
  const t = useT()
  const c = useComposer()
  const q = $api.useQuery(
    'get',
    rendersPath,
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } },
    { refetchInterval: (query) => (query.state.data?.items.some((x: Render) => x.status === 'pending') ? 5000 : false) },
  )
  const latest = q.data?.items[0]
  if (!latest) return null
  const service = serviceName(latest.service)
  const settled = latest.results.filter((x) => x.status !== 'pending').length
  const current = c.currentVersion?.()
  return (
    <section className="card flex flex-col gap-2" style={{ padding: 12, gridColumn: '1 / -1' }} aria-label={t(mr.realRenders, { service })}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="m-0 text-[12px] font-semibold">{t(mr.realRenders, { service })}</h2>
        <span className="k">{t(mr.renderMeta, { version: latest.version, variant: latest.variant, time: clock(latest.requestedAt), name: latest.requestedBy })}</span>
      </div>
      {current && latest.version < current ? <span className="k">{t(mr.stale, { version: latest.version })}</span> : null}
      {latest.status === 'pending' ? <span className="k">{t(mr.progress, { done: settled, count: latest.results.length })}</span> : null}
      {latest.status === 'failed' && latest.error ? <span className="text-bad-fg">{latest.error}</span> : null}
      <ul className="m-0 grid gap-2.5 p-0" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(150px,1fr))', listStyle: 'none' }}>
        {latest.results.map((x) => (
          <li key={x.client} className="flex min-w-0 flex-col gap-1">
            {x.status === 'done' && x.imageUrl ? (
              <a
                href={x.imageUrl}
                target="_blank"
                rel="noreferrer noopener"
                aria-label={t(mr.openFull, { client: x.name })}
                className="block overflow-hidden rounded-md border border-line"
                style={{ height: 120, background: 'var(--ground-3)' }}
              >
                <img src={x.thumbUrl || x.imageUrl} alt="" loading="lazy" referrerPolicy="no-referrer" style={{ width: '100%', height: '100%', objectFit: 'cover', objectPosition: 'top' }} />
              </a>
            ) : (
              <div className="flex items-center justify-center rounded-md border border-line k text-center" style={{ height: 120, background: 'var(--ground-3)', padding: 8 }}>
                {x.status === 'pending' ? t(mr.capturing) : t(mr.notCaptured)}
              </div>
            )}
            <span className="truncate text-[12px] font-semibold" title={x.name}>
              {x.name}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
