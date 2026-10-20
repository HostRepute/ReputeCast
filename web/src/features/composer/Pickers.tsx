import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { useComposer, type Campaign } from './context'
import { ToolIcon } from './icons'
import { WhoOnList, type Who } from '@/features/earn/composer/WhoOnList'

function useUpdateCampaign() {
  const { slug, campaign } = useComposer()
  const qc = useQueryClient()
  const m = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}')
  return (body: { audience?: { lists: string[]; segments: string[]; excludeLists: string[]; paid?: Who; paidTier?: string }; senderId?: string }, done?: () => void) =>
    m.mutate(
      { params: { path: { workspace: slug, campaignId: campaign.id } }, body },
      {
        onSuccess: (d) => {
          qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: campaign.id } } }], d as Campaign)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
          done?.()
        },
        onError: (e) => toast({ kind: 'error', title: "Couldn't change the campaign", detail: errorMessage(e) }),
      },
    )
}

/** Audience picker with the live recipient count. */
export function AudiencePicker() {
  const { campaign, context, slug, canEdit } = useComposer()
  const [open, setOpen] = useState(false)
  const [lists, setLists] = useState<string[]>(campaign.audience.lists)
  const [segments, setSegments] = useState<string[]>(campaign.audience.segments)
  const [exclude, setExclude] = useState<string[]>(campaign.audience.excludeLists)
  // Who on the lists (Earn): everyone, paid members only or free subscribers only.
  const [who, setWho] = useState<Who>(campaign.audience.paid ?? '')
  const [paidTier, setPaidTier] = useState(campaign.audience.paidTier ?? '')
  const [live, setLive] = useState<{ count: number; approximate: boolean } | null>(null)
  const update = useUpdateCampaign()

  const onOpenChange = (o: boolean) => {
    if (o) {
      setLists(campaign.audience.lists)
      setSegments(campaign.audience.segments)
      setExclude(campaign.audience.excludeLists)
      setWho(campaign.audience.paid ?? '')
      setPaidTier(campaign.audience.paidTier ?? '')
    }
    setOpen(o)
  }
  const nothing = !lists.length && !segments.length

  useEffect(() => {
    if (!open || nothing) return
    const h = window.setTimeout(() => {
      void http
        .POST('/api/v1/workspaces/{workspace}/campaigns/audience-count', { params: { path: { workspace: slug } }, body: { lists, segments, excludeLists: exclude, paid: who, paidTier: who === 'paid' ? paidTier : '' } })
        .then(({ data }) => data && setLive({ count: data.count, approximate: data.approximate }))
        .catch(() => setLive(null))
    }, 250)
    return () => window.clearTimeout(h)
  }, [open, nothing, lists, segments, exclude, who, paidTier, slug])

  const label = campaign.audienceLabel ? `${campaign.audienceLabel} · ${count(campaign.audienceCount ?? 0)}` : 'Choose audience'
  const toggle = (arr: string[], id: string) => (arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id])
  const trigger = campaign.audienceLabel ? (
    <button type="button" className="btn" aria-label={`Audience: ${campaign.audienceLabel}, ${count(campaign.audienceCount ?? 0)} recipients`}>
      <ToolIcon name="audience" size={14} strokeWidth={1.6} />
      {label}
    </button>
  ) : (
    <button type="button" className="btn" style={{ borderStyle: 'dashed' }}>
      Choose audience
    </button>
  )
  if (!canEdit) return trigger
  return (
    <Popover trigger={trigger} open={open} onOpenChange={onOpenChange} align="start" label="Audience" className="w-[340px] max-w-[calc(100vw-24px)]">
      <div className="flex flex-col gap-1 p-1">
        <span className="mh">Send to</span>
        {context.lists.map((l) => (
          <label key={l.id} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground">
            <input type="checkbox" checked={lists.includes(l.id)} onChange={() => setLists(toggle(lists, l.id))} />
            <span className="flex-1">{l.name}</span>
            <span className="k tabular">{count(l.count ?? 0)}</span>
          </label>
        ))}
        {context.segments.length ? <span className="mh">Segments</span> : null}
        {context.segments.map((s) => (
          <label key={s.id} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground">
            <input type="checkbox" checked={segments.includes(s.id)} onChange={() => setSegments(toggle(segments, s.id))} />
            <span className="flex-1">{s.name}</span>
            <span className="k tabular">{s.count != null ? count(s.count) : '–'}</span>
          </label>
        ))}
        {context.lists.length > 1 ? <span className="mh">Leave out</span> : null}
        {context.lists.length > 1
          ? context.lists
              .filter((l) => !lists.includes(l.id))
              .map((l) => (
                <label key={l.id} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground">
                  <input type="checkbox" checked={exclude.includes(l.id)} onChange={() => setExclude(toggle(exclude, l.id))} />
                  <span className="flex-1">People on {l.name}</span>
                </label>
              ))
          : null}
        <WhoOnList
          lists={lists}
          who={who}
          tier={paidTier}
          onChange={(w, tier) => {
            setWho(w)
            setPaidTier(tier)
          }}
        />
        <div className="mt-1 flex items-center gap-2 border-t border-line-2 px-2.5 pt-2.5 pb-1">
          <span className="k flex-1" role="status" aria-live="polite">
            {nothing ? 'Pick a list or a segment' : live ? `${live.approximate ? 'About ' : ''}${count(live.count)} recipients after suppressions` : 'Counting…'}
          </span>
          <button
            type="button"
            className="btn pri sm"
            onClick={() => update({ audience: { lists, segments, excludeLists: exclude, paid: who, paidTier: who === 'paid' ? paidTier : '' } }, () => setOpen(false))}
            disabled={!lists.length && !segments.length}
          >
            Apply
          </button>
        </div>
      </div>
    </Popover>
  )
}

/** Sender identity ("From") picker. */
export function SenderPicker() {
  const { campaign, context, canEdit } = useComposer()
  const [open, setOpen] = useState(false)
  const update = useUpdateCampaign()
  const s = campaign.sender
  const trigger = (
    <button type="button" className="btn" style={s ? undefined : { borderStyle: 'dashed' }}>
      {s ? `From: ${s.name} <${s.email}>` : 'Choose sender'}
    </button>
  )
  if (!canEdit) return trigger
  return (
    <Popover trigger={trigger} open={open} onOpenChange={setOpen} align="start" label="Sender" className="w-[360px] max-w-[calc(100vw-24px)]">
      <span className="mh">Send from</span>
      {context.senders.length === 0 ? (
        <p className="k px-2.5 py-2">No senders yet. Add one in Settings › Senders and domains.</p>
      ) : (
        context.senders.map((x) => (
          <button
            key={x.id}
            type="button"
            role="menuitemradio"
            aria-checked={s?.id === x.id}
            onClick={() => update({ senderId: x.id }, () => setOpen(false))}
          >
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="font-semibold">{x.name}</span>
              <span className="k">{x.email}</span>
            </span>
            {s?.id === x.id ? <span className="k">Current</span> : x.verified ? null : <span className="k">domain not verified</span>}
          </button>
        ))
      )}
    </Popover>
  )
}
