import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { RoleBadge } from '@/features/access/parts'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useAgencyMe, useRefreshAgency } from './api'
import { m as lm } from './library.messages'
import { m } from './messages'
import { calendarDate, ClientTile, PersonChip } from './parts'
import { rateText } from './rate'
import { useWorkspaceAgency } from './WorkspaceParts'

/**
 * Settings › General in a workspace the agency manages (AgencyStates,
 * "Client details in Settings › General"). Client people see who manages
 * the workspace, their contact and their role; the agency's staff also see
 * the client details and can save the setup as a workspace template.
 * Nothing on other workspaces.
 */
export function AgencyGeneralCard() {
  const t = useT()
  const { me } = useWorkspace()
  const { data } = useWorkspaceAgency()
  const [saving, setSaving] = useState(false)
  if (!data?.managed || data.kind !== 'client') return null
  const agency = data.agency ?? ''
  const contact = data.contact
  const staff = data.viewerIsStaff
  const roleKey = me.workspace.roleKey
  const roleLine = roleKey === 'client_approver' ? t(m.roleLineClientApprover) : roleKey === 'client_viewer' ? t(m.roleLineClientViewer) : ''
  const since = data.clientSince ? calendarDate(data.clientSince) : ''
  return (
    <section className="card col-span-full" aria-labelledby="ag-general-h">
      <CardSection
        id="ag-general-h"
        title={staff ? t(m.generalStaffTitle) : t(m.generalClientTitle)}
        caption={staff ? t(m.generalStaffCaption) : t(m.generalClientCaption, { workspace: me.workspace.name })}
        actions={
          staff && data.canSaveTemplate ? (
            <Button size="sm" onClick={() => setSaving(true)}>
              {t(m.saveAsTemplate)}
            </Button>
          ) : undefined
        }
      />
      <div className="pad col gap-3.5">
        <div className="note n items-center! flex-nowrap!">
          <ClientTile ws={{ initials: data.agencyInitials ?? '', color: '#0B1220' }} size={28} style={{ borderRadius: 7, fontSize: 11 }} />
          <div className="flex min-w-0 flex-auto flex-col gap-0.5">
            <span className="font-bold">{t(m.managedBy, { agency })}</span>
            <span>{contact ? t(m.managedLine, { name: contact.name, email: contact.email, agency }) : t(m.managedLineNoContact, { agency })}</span>
          </div>
        </div>
        {staff ? (
          <KeyValues
            items={[
              [t(m.clientSince), data.template ? t(m.clientSinceFrom, { date: since, template: data.template }) : since],
              [t(m.colManager), contact ? <PersonChip name={contact.name} color={contact.avatarColor} /> : t(m.noManager)],
              ...(data.rate ? ([[t(m.rebillRate), t(m.rateStaffOnly, { rate: rateText(t, data.rate) })]] as [string, string][]) : []),
            ]}
          />
        ) : (
          <>
            <KeyValues
              items={[
                [
                  t(m.yourRole),
                  <>
                    <RoleBadge role={{ key: roleKey, name: me.workspace.roleName, builtin: Boolean(roleLine) }} />
                    {roleLine ? <span className="k ml-1.5">{roleLine}</span> : null}
                  </>,
                ],
              ]}
            />
            <span className="k">{contact ? t(m.onlyAdminsAsk, { name: contact.name }) : t(m.onlyAdmins)}</span>
          </>
        )}
      </div>
      {saving ? <SaveTemplateDialog onClose={() => setSaving(false)} /> : null}
    </section>
  )
}

/** "Save as workspace template": this client's setup becomes a template in the agency's library. */
function SaveTemplateDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const navigate = useNavigate()
  const refresh = useRefreshAgency()
  const { slug, me } = useWorkspace()
  const { data: agency } = useAgencyMe()
  const [name, setName] = useState(me.workspace.name)
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/agency/template', {
        params: { path: { workspace: slug } },
        body: { name: name.trim(), description: description.trim() || undefined },
      })
      await refresh()
      const home = agency?.home
      toast({
        kind: 'success',
        title: t(lm.templateSaved, { name: data?.name ?? name.trim() }),
        action: home ? { label: t(m.openLibrary), onClick: () => void navigate({ to: '/w/$workspace/agency/library', params: { workspace: home } }) } : undefined,
      })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(m.saveAsTemplate)}
      description={t(m.saveAsTemplateBody, { workspace: me.workspace.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!name.trim()} onClick={save}>
            {t(lm.saveTemplate)}
          </Button>
        </>
      }
    >
      <Field label={t(lm.templateName)} error={error}>
        <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} autoFocus />
      </Field>
      <Field label={t(m.templateDescription)} optional>
        <Textarea value={description} onChange={(e) => setDescription(e.target.value)} maxLength={300} rows={2} />
      </Field>
    </Dialog>
  )
}
