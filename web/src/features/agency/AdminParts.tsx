import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Th, Td } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useAgencyMe, useRefreshAgency } from './api'
import { m } from './messages'
import { calendarDate, ClientTile } from './parts'

type Mode = Schemas['AgencyModeView']

/**
 * Instance admin › Instance & updates › Agency (AgencyStates): agency mode,
 * the agency's name and own workspace, the default template for new
 * clients, and which workspaces are clients. One agency per instance.
 */
export function AdminAgencyCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/agency')
  return (
    <section className="card" aria-labelledby="agency-mode-h">
      {q.isError ? (
        <>
          <CardSection id="agency-mode-h" title={t(m.adminTitle)} caption={t(m.adminCaption)} />
          <div className="pad">
            <ErrorPanel title={t(m.adminLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
              {errorMessage(q.error)} {t(m.nothingChanged)}
            </ErrorPanel>
          </div>
        </>
      ) : !q.data ? (
        <>
          <CardSection id="agency-mode-h" title={t(m.adminTitle)} caption={t(m.adminCaption)} />
          <div className="pad col gap-3" aria-busy="true">
            <Skeleton w="40%" h={18} />
            <Skeleton w="100%" h={36} r={8} />
            <Skeleton w="100%" h={36} r={8} />
          </div>
        </>
      ) : (
        <ModeForm key={JSON.stringify(q.data)} v={q.data} />
      )}
    </section>
  )
}

function ModeForm({ v }: { v: Mode }) {
  const t = useT()
  const qc = useQueryClient()
  const refreshAgency = useRefreshAgency()
  const [enabled, setEnabled] = useState(v.enabled)
  const [name, setName] = useState(v.name || v.instanceName)
  const [own, setOwn] = useState(v.agencyWorkspaceId ?? '')
  const [tpl, setTpl] = useState(v.defaultTemplateId ?? '')
  const [clients, setClients] = useState(() => new Set(v.workspaces.filter((w) => w.client).map((w) => w.id)))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [confirmOff, setConfirmOff] = useState(false)
  const candidates = v.workspaces.filter((w) => w.id !== own)
  // Rejects on failure: the turn-off dialog stays open with the error.
  const put = async (next: boolean) => {
    await http.PUT('/api/v1/admin/agency', {
      body: { enabled: next, name: name.trim(), agencyWorkspaceId: own || undefined, defaultTemplateId: tpl || undefined, clientIds: [...clients].filter((id) => id !== own) },
    })
    await qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/agency'] })
    await qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/users'] })
    await refreshAgency()
    toast({ kind: 'success', title: next === v.enabled ? t(m.adminSaved) : next ? t(m.adminTurnedOn) : t(m.adminTurnedOff) })
  }
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await put(enabled)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const toggle = (on: boolean) => {
    if (!on && v.enabled) setConfirmOff(true)
    else setEnabled(on)
  }
  const modeText = (
    <span className="flex flex-col gap-0.5">
      <span className="font-semibold">{t(m.adminMode)}</span>
      <span className="k">{v.enabled && v.enabledByName && v.enabledAt ? t(m.adminTurnedOnBy, { name: v.enabledByName, date: calendarDate(v.enabledAt) }) : t(m.adminModeHint)}</span>
    </span>
  )
  return (
    <>
      <CardSection
        id="agency-mode-h"
        title={t(m.adminTitle)}
        caption={t(m.adminCaption)}
        actions={
          v.enabled ? (
            <Badge tone="ok" dot>
              {t(m.adminOn)}
            </Badge>
          ) : (
            <Badge tone="neutral">{t(m.adminOff)}</Badge>
          )
        }
      />
      <div className="pad grid items-start gap-[22px]" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(300px,100%),1fr))' }}>
        <div className="flex flex-col gap-3">
          <label className="flex items-center justify-between gap-3">
            {modeText}
            <input type="checkbox" className="tg" role="switch" checked={enabled} onChange={(e) => toggle(e.target.checked)} />
          </label>
          <label className="lbl">
            {t(m.adminName)}
            <input className="in" type="text" value={name} maxLength={80} onChange={(e) => setName(e.target.value)} />
          </label>
          <label className="lbl">
            {t(m.adminOwnWorkspace)}
            <select className="in" value={own} onChange={(e) => setOwn(e.target.value)}>
              {v.workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
              <option value="">{t(m.adminNone)}</option>
            </select>
          </label>
          <label className="lbl">
            {t(m.adminDefaultTemplate)}
            <select className="in" value={tpl} onChange={(e) => setTpl(e.target.value)}>
              {v.templates.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
              <option value="">{t(m.adminNone)}</option>
            </select>
          </label>
        </div>
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="lbl mb-1.5 p-0">{t(m.adminClients, { n: candidates.filter((w) => clients.has(w.id)).length })}</legend>
          {candidates.map((w) => (
            <label key={w.id} className="flex items-center gap-2 py-1 text-[13px]">
              <input
                type="checkbox"
                checked={clients.has(w.id)}
                onChange={(e) => {
                  const next = new Set(clients)
                  if (e.target.checked) next.add(w.id)
                  else next.delete(w.id)
                  setClients(next)
                }}
              />
              <ClientTile ws={{ initials: w.initials, color: w.color }} size={20} />
              {w.name}
            </label>
          ))}
          <span className="k">{t(m.adminClientsNote)}</span>
        </fieldset>
      </div>
      {error ? (
        <div className="pad pt-0!">
          <div className="note b" role="alert">
            <span>{error}</span>
          </div>
        </div>
      ) : null}
      <div className="flex flex-wrap items-center gap-2.5 border-t border-line-2 px-[18px] py-3">
        <span className="k mr-auto">{t(m.adminOffNote)}</span>
        {v.enabled ? (
          <Button size="sm" variant="danger" onClick={() => setConfirmOff(true)}>
            {t(m.adminTurnOff)}
          </Button>
        ) : null}
        <Button size="sm" variant="primary" loading={busy} onClick={() => void save()}>
          {enabled && !v.enabled ? t(m.adminTurnOn) : t(m.save)}
        </Button>
      </div>
      <ConfirmDialog
        open={confirmOff}
        onOpenChange={setConfirmOff}
        badge={t(m.adminTurnOffBadge)}
        title={t(m.adminTurnOffTitle)}
        description={t(m.adminTurnOffBody, { agency: v.name || v.instanceName })}
        confirmLabel={t(m.adminTurnOff)}
        onConfirm={() => put(false)}
      />
    </>
  )
}

/** Instance admin › Audit log: the Agency category, while agency mode is on. */
export function AgencyAuditFilter({ value, onChange }: { value?: 'agency'; onChange: (v: 'agency' | undefined) => void }) {
  const t = useT()
  const on = useAgencyColumn()
  if (!on && !value) return null
  return (
    <select className="in w-[150px] max-md:w-full" aria-label={t(m.auditFilterLabel)} value={value ?? ''} onChange={(e) => onChange(e.target.value ? 'agency' : undefined)}>
      <option value="">{t(m.auditAnyAction)}</option>
      <option value="agency">{t(m.auditAgency)}</option>
    </select>
  )
}

/** Instance admin › Users: the Agency staff column, while agency mode is on. */
export function useAgencyColumn(): boolean {
  const { data } = useAgencyMe()
  return Boolean(data?.enabled)
}

export function AgencyStaffTh() {
  const t = useT()
  return <Th>{t(m.adminStaffColumn)}</Th>
}

export function AgencyStaffTd({ level }: { level?: string }) {
  const t = useT()
  return (
    <Td>
      {level === 'agency_manager' ? (
        <Badge tone="purple">{t(m.levelAgencyManager)}</Badge>
      ) : level === 'account_manager' ? (
        <Badge tone="info">{t(m.levelAccountManager)}</Badge>
      ) : (
        <span className="k">{t(m.dash)}</span>
      )}
    </Td>
  )
}
