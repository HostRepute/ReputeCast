import { useQueries, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { MemberIntegrations, useMemberIntegrations } from '@/features/integrations/MemberIntegrations'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m as md } from './deactivate.messages'

type Member = Pick<Schemas['TeamMember'], 'id' | 'name'>

const first = (name: string) => name.split(' ')[0] ?? name

/**
 * Deactivate one or several members (TeamDialogs): consequences first, then
 * keep or revoke the API keys they created, and a reason for the audit log.
 */
export function DeactivateDialog({
  members,
  open,
  onOpenChange,
  onDone,
}: {
  members: Member[]
  open: boolean
  onOpenChange: (o: boolean) => void
  onDone?: () => void
}) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const details = useQueries({
    queries: members.map((m) => ({
      queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}', { params: { path: { workspace: slug, memberId: m.id } } }],
      queryFn: async () => (await http.GET('/api/v1/workspaces/{workspace}/members/{memberId}', { params: { path: { workspace: slug, memberId: m.id } } })).data!,
    })),
  })
  const [keep, setKeep] = useState(true)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const loaded = details.every((d) => d.data)
  const data = details.map((d) => d.data).filter(Boolean) as Schemas['TeamMemberDetail'][]
  const sessions = data.reduce((n, d) => n + d.sessions, 0)
  const scheduled = data.reduce((n, d) => n + d.scheduledCampaigns, 0)
  const approvals = data.reduce((n, d) => n + d.approvalsRequested.filter((a) => a.status === 'pending').length, 0)
  const keys = data.flatMap((d) => d.apiKeys.filter((k) => k.status === 'active'))
  const integrations = useMemberIntegrations(data.map((d) => d.member.userId))
  const one = members.length === 1
  // One member is named (their first name in the description and the button); several are counted.
  const name = one ? members[0]!.name : undefined
  const firstName = name !== undefined ? first(name) : undefined
  const keyNames = andList(keys.map((k) => `“${k.name}”`))

  const submit = async () => {
    setBusy(true)
    try {
      await integrations.apply(reason.trim())
      for (const m of members) {
        await http.POST('/api/v1/workspaces/{workspace}/members/{memberId}/deactivate', {
          params: { path: { workspace: slug, memberId: m.id } },
          body: { revokeKeys: !keep, reason: reason.trim() || undefined },
        })
      }
      toast({
        title: name !== undefined ? t(md.doneOne, { name }) : t(md.doneMany, { count: members.length }),
        detail: t(md.doneDetail, { count: sessions }),
      })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}'] })
      onDone?.()
      onOpenChange(false)
    } catch (e) {
      toast({ kind: 'error', title: name !== undefined ? t(md.failedOne, { name }) : t(md.failedMany, { count: members.length }), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(md.eyebrow)}</span>}
      title={name !== undefined ? t(md.titleOne, { name }) : t(md.titleMany, { count: members.length })}
      description={<span className="text-[13.5px]">{firstName !== undefined ? t(md.descriptionOne, { name: firstName }) : t(md.descriptionMany)}</span>}
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="danger-primary" loading={busy} disabledReason={loaded && integrations.loaded ? undefined : t(md.loadingChanges)} onClick={submit}>
            {firstName !== undefined ? t(md.confirmOne, { name: firstName }) : t(md.confirmMany, { count: members.length })}
          </Button>
        </>
      }
    >
      {!loaded ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="80%" />
          <Skeleton w="70%" />
          <Skeleton w="60%" />
        </div>
      ) : (
        <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
          <li>{t(md.signedOutEverywhere, { count: sessions })}</li>
          <li>{t(md.cantSignIn, { workspace: me.workspace.name })}</li>
          {scheduled > 0 ? <li>{t(md.scheduled, { count: scheduled })}</li> : null}
          {approvals > 0 ? <li>{t(md.approvals, { count: approvals })}</li> : null}
        </ul>
      )}
      {loaded && keys.length > 0 ? (
        <fieldset className="m-0 flex flex-col gap-2 rounded-lg border border-line px-3.5 py-2.5">
          <legend className="px-1 text-[12px] font-semibold text-ink-2">{t(md.keysLegend, { count: count(keys.length) })}</legend>
          <label className="row gap-2 text-[13.5px]">
            <input type="radio" name="keys" checked={keep} onChange={() => setKeep(true)} />
            <span>
              {keys.length === 1 ? (
                <T
                  msg={md.keepKey}
                  values={{ keys: keyNames, permissions: [...keys[0]!.permissions.slice(0, 2)].join(', '), k: (c) => <span className="k">{c}</span> }}
                />
              ) : (
                t(md.keepKeys, { keys: keyNames })
              )}
            </span>
          </label>
          <label className="row gap-2 text-[13.5px]">
            <input type="radio" name="keys" checked={!keep} onChange={() => setKeep(false)} />
            {t(md.revoke, { count: keys.length })}
          </label>
        </fieldset>
      ) : null}
      <MemberIntegrations state={integrations} name={firstName} />
      <label className="lbl">
        <span>
          <T msg={md.reason} values={{ k: (c) => <span className="k font-medium">{c}</span> }} />
        </span>
        <input className="in" type="text" value={reason} onChange={(e) => setReason(e.target.value)} maxLength={500} placeholder={t(md.reasonPlaceholder)} />
      </label>
    </Dialog>
  )
}
