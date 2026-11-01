import { useNavigate } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { weekdayDate, clock } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { mf } from './followup.messages'
import { useRefreshCampaigns } from './manage'

export type FollowUpKind = 'non_openers' | 'clickers'

/** The suggested delays: the original's time plus a day or two. */
const DELAYS = [48, 24]

/** The report header's "Follow up ▾": resend to non-openers or send to clickers (CA-4). */
export function FollowUpMenu({ campaign, size = 'sm' }: { campaign: { id: string; name: string; subject?: string }; size?: 'sm' | 'md' }) {
  const tr = useT()
  const reason = usePermissionReason('campaigns:edit')
  const [kind, setKind] = useState<FollowUpKind>()
  if (reason) {
    return (
      <Button size={size === 'sm' ? 'sm' : undefined} disabledReason={reason}>
        {tr(mf.menu)}
      </Button>
    )
  }
  return (
    <>
      <Menu
        label={tr(mf.menu)}
        trigger={
          <button type="button" className={size === 'sm' ? 'btn sm' : 'btn'} aria-haspopup="menu">
            {tr(mf.menu)} <span aria-hidden="true">▾</span>
          </button>
        }
      >
        <MenuItem onSelect={() => setKind('non_openers')}>{tr(mf.nonOpeners)}</MenuItem>
        <MenuItem onSelect={() => setKind('clickers')}>{tr(mf.clickers)}</MenuItem>
      </Menu>
      {kind ? <FollowUpDialog campaign={campaign} kind={kind} onClose={() => setKind(undefined)} /> : null}
    </>
  )
}

/**
 * Makes the follow-up draft: the same content with a subject of its own, to
 * a segment of the report's bucket that stays up to date until it goes out.
 */
export function FollowUpDialog({ campaign, kind: initial, onClose }: { campaign: { id: string; name: string; subject?: string }; kind: FollowUpKind; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshCampaigns()
  const id = useId()
  const [kind, setKind] = useState(initial)
  const counts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/follow-up', {
    params: { path: { workspace: slug, campaignId: campaign.id } },
  })
  // The suggested subject waits for the original's, unless the person typed one.
  const [typed, setSubject] = useState<string>()
  const subject = typed ?? tr(kind === 'clickers' ? mf.subjectClickers : mf.subjectNonOpeners, { subject: counts.data?.subject || campaign.subject || campaign.name })
  const [delay, setDelay] = useState<string>()
  // When the dialog opened: the delays still ahead are offered.
  const [openedAt] = useState(() => Date.now())
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string>()
  const sentAt = counts.data?.sentAt ? new Date(counts.data.sentAt) : undefined
  // Only delays still ahead are offered; otherwise Review & send picks the time.
  const delays = sentAt ? DELAYS.filter((h) => sentAt.getTime() + h * 3600_000 > openedAt + 15 * 60_000) : []
  const chosen = delay ?? (delays[0] !== undefined ? String(delays[0]) : '0')
  const reach = kind === 'clickers' ? counts.data?.clickers : counts.data?.nonOpeners
  const create = async () => {
    if (!subject.trim()) {
      setErr(tr(mf.subjectRequired))
      return
    }
    setBusy(true)
    setErr(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/follow-up', {
        params: { path: { workspace: slug, campaignId: campaign.id } },
        body: { kind, subject: subject.trim(), delayHours: Number(chosen) },
      })
      await refresh()
      toast({ kind: 'success', title: t(mf.created, { name: data?.name ?? '' }) })
      onClose()
      if (data) void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: data.id } })
    } catch (e) {
      setErr(errorMessage(e))
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(kind === 'clickers' ? mf.titleClickers : mf.titleNonOpeners)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={counts.isPending} onClick={() => void create()}>
            {busy ? tr(mf.creating) : tr(mf.create)}
          </Button>
        </>
      }
    >
      <form
        noValidate
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void create()
        }}
      >
        {counts.isPending ? (
          <div className="flex flex-col gap-1.5" aria-busy="true">
            <span className="vh" role="status">
              {tr(mf.loading)}
            </span>
            <Skeleton w={300} h={12} />
            <Skeleton w={220} h={12} />
          </div>
        ) : counts.isError ? (
          <Note tone="bad">{errorMessage(counts.error)}</Note>
        ) : (
          <p className="m-0">
            <T
              msg={kind === 'clickers' ? mf.bodyClickers : mf.bodyNonOpeners}
              values={{ count: reach ?? 0, name: campaign.name, b: (chunks) => <b>{chunks}</b> }}
            />
            {kind === 'non_openers' && counts.data.machineOnly > 0 ? <> {tr(mf.machineOnly, { count: counts.data.machineOnly })}</> : null}
          </p>
        )}
        {reach === 0 ? <Note tone="info">{tr(mf.nobody)}</Note> : null}
        <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
          <legend className="lbl mb-1.5 p-0">{tr(mf.who)}</legend>
          {(['non_openers', 'clickers'] as const).map((k) => (
            <label key={k} className={`flex cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2 text-[13px] ${kind === k ? 'border-accent bg-accent-tint' : 'border-line'}`}>
              <input type="radio" name={`${id}-kind`} checked={kind === k} onChange={() => setKind(k)} />
              {tr(k === 'clickers' ? mf.whoClickers : mf.whoNonOpeners)}
            </label>
          ))}
        </fieldset>
        <div className="field">
          <label className="lbl" htmlFor={`${id}-subject`}>
            {tr(mf.subject)}
          </label>
          <input
            id={`${id}-subject`}
            className="in w-full"
            value={subject}
            maxLength={998}
            aria-invalid={err && !subject.trim() ? true : undefined}
            onChange={(e) => setSubject(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="lbl" htmlFor={`${id}-when`}>
            {tr(mf.send)}
          </label>
          <select id={`${id}-when`} className="in" value={chosen} onChange={(e) => setDelay(e.target.value)}>
            {sentAt
              ? delays.map((h) => {
                  const at = new Date(sentAt.getTime() + h * 3600_000)
                  return (
                    <option key={h} value={String(h)}>
                      {tr(mf.after, { days: h / 24, when: `${weekdayDate(at)}, ${clock(at)}` })}
                    </option>
                  )
                })
              : null}
            <option value="0">{tr(mf.later)}</option>
          </select>
        </div>
        <span className="k">{tr(mf.note)}</span>
        {err ? <Note tone="bad">{err}</Note> : null}
      </form>
    </Dialog>
  )
}
