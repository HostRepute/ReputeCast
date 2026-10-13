import { useQueries, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { andList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

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
  const one = members.length === 1
  const name = one ? members[0]!.name : `${members.length} members`
  const firstName = one ? first(members[0]!.name) : `${members.length} members`

  const submit = async () => {
    setBusy(true)
    try {
      for (const m of members) {
        await http.POST('/api/v1/workspaces/{workspace}/members/{memberId}/deactivate', {
          params: { path: { workspace: slug, memberId: m.id } },
          body: { revokeKeys: !keep, reason: reason.trim() || undefined },
        })
      }
      toast({ title: `${name} deactivated`, detail: `signed out of ${sessions} ${sessions === 1 ? 'session' : 'sessions'}` })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members/{memberId}'] })
      onDone?.()
      onOpenChange(false)
    } catch (e) {
      toast({ kind: 'error', title: `Couldn't deactivate ${name}`, detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Dialog · deactivate member</span>}
      title={`Deactivate ${name}?`}
      description={
        <span className="text-[13.5px]">
          {one
            ? `${firstName} keeps their history and can be reactivated later. Removing instead deletes the membership; the audit log still names them.`
            : 'They keep their history and can be reactivated later. The audit log still names them.'}
        </span>
      }
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="danger-primary" loading={busy} disabledReason={loaded ? undefined : 'Loading what changes…'} onClick={submit}>
            Deactivate {one ? firstName : `${members.length} members`}
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
          <li>
            Signed out everywhere, now ({sessions} {sessions === 1 ? 'session' : 'sessions'})
          </li>
          <li>Can&apos;t sign in to {me.workspace.name}; other workspaces are unaffected</li>
          {scheduled > 0 ? (
            <li>
              {one ? 'Their' : 'Their'} {scheduled} scheduled {scheduled === 1 ? 'campaign keeps its' : 'campaigns keep their'} schedule and run under the workspace
            </li>
          ) : null}
          {approvals > 0 ? (
            <li>
              {approvals} pending approval {approvals === 1 ? 'request stays' : 'requests stay'} open for approvers
            </li>
          ) : null}
        </ul>
      )}
      {loaded && keys.length > 0 ? (
        <fieldset className="m-0 flex flex-col gap-2 rounded-lg border border-line px-3.5 py-2.5">
          <legend className="px-1 text-[12px] font-semibold text-ink-2">
            API keys {one ? 'they' : 'they'} created · {keys.length}
          </legend>
          <label className="row gap-2 text-[13.5px]">
            <input type="radio" name="keys" checked={keep} onChange={() => setKeep(true)} />
            <span>
              Keep {andList(keys.map((k) => `“${k.name}”`))} working{' '}
              {keys.length === 1 ? <span className="k">({[...keys[0]!.permissions.slice(0, 2)].join(', ')})</span> : null}
            </span>
          </label>
          <label className="row gap-2 text-[13.5px]">
            <input type="radio" name="keys" checked={!keep} onChange={() => setKeep(false)} />
            {keys.length === 1 ? 'Revoke it now' : 'Revoke them now'}
          </label>
        </fieldset>
      ) : null}
      <label className="lbl">
        <span>
          Reason <span className="k font-medium">· recorded in the audit log</span>
        </span>
        <input className="in" type="text" value={reason} onChange={(e) => setReason(e.target.value)} maxLength={500} placeholder="Contract ended" />
      </label>
    </Dialog>
  )
}
