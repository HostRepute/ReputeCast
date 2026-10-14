import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { streamName, type Stream } from './common'
import { providerUses, streams, toDraft, toInput, withoutProvider, type RoutingConfig } from './routing-model'

/** "System mail" reads better than "System" in the sentence. */
const streamMail: Record<Stream, string> = { broadcast: 'Broadcasts', automation: 'Automations', system: 'System mail' }

/**
 * Removing a provider (Dialogs board, "Remove provider"): what it carries,
 * a replacement for each route, and what stays. Confirming saves the
 * routing with the replacements, then disconnects the provider.
 */
export function RemoveProviderDialog({ providerId, name, open, onOpenChange, onRemoved }: { providerId: string; name: string; open: boolean; onOpenChange: (open: boolean) => void; onRemoved: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } } }, { enabled: open })
  return (
    <RemoveBody
      key={q.dataUpdatedAt}
      cfg={q.data}
      error={q.isError ? errorMessage(q.error) : undefined}
      providerId={providerId}
      name={name}
      open={open}
      onOpenChange={onOpenChange}
      onRemoved={onRemoved}
    />
  )
}

function RemoveBody({
  cfg,
  error,
  providerId: id,
  name,
  open,
  onOpenChange,
  onRemoved,
}: {
  cfg?: RoutingConfig
  error?: string
  providerId: string
  name: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onRemoved: () => void
}) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const draft = cfg ? toDraft(cfg) : undefined
  const uses = draft ? providerUses(draft, id) : []
  const others = (cfg?.providers ?? []).filter((p) => p.id !== id)
  /** Who can take over a stream: Broadcasts only go to providers whose terms allow bulk mail. */
  const candidates = (s: Stream, exclude?: string) => others.filter((p) => (s !== 'broadcast' || p.bulkAllowed) && p.id !== exclude)
  const [picked, setPicked] = useState<Record<string, string>>(() => {
    const out: Record<string, string> = {}
    if (!draft) return out
    for (const u of providerUses(draft, id)) {
      if (u.kind === 'primary') out[`primary:${u.stream}`] = candidates(u.stream)[0]?.id ?? ''
    }
    for (const u of providerUses(draft, id)) {
      if (u.kind !== 'failover') continue
      const primary = draft[u.stream].splits[0]?.providerId === id ? out[`primary:${u.stream}`] : draft[u.stream].splits[0]?.providerId
      out[`failover:${u.stream}`] = candidates(u.stream, primary)[0]?.id ?? ''
    }
    return out
  })
  const primaryOf = (s: Stream) => (draft && draft[s].splits[0]?.providerId === id ? picked[`primary:${s}`] : draft?.[s].splits[0]?.providerId) ?? ''
  const nameOf = (pid: string) => others.find((p) => p.id === pid)?.name ?? 'the primary'
  const stuck = uses.filter((u) => u.kind === 'primary' && !picked[`primary:${u.stream}`])

  const roles = uses.map((u) =>
    u.kind === 'primary'
      ? `the primary route for ${streamMail[u.stream]}`
      : u.kind === 'share'
        ? `${u.weight}% of ${streamName[u.stream]}`
        : u.kind === 'failover'
          ? `the failover for ${streamName[u.stream]}`
          : `a ${streamName[u.stream]} condition`,
  )
  const description = !cfg
    ? `Checking which routes use ${name}…`
    : uses.length === 0
      ? `No route uses ${name}, so nothing else changes.`
      : `${name} is ${andList(roles)}. ${
          uses.some((u) => u.kind === 'primary' || u.kind === 'failover') ? 'Removing it leaves those routes with no provider until you pick another.' : 'Its share goes back to the primary.'
        }`

  const remove = async () => {
    if (!draft) throw new Error(error ?? `Still checking which routes use ${name}. Try again in a moment.`)
    if (stuck.length) throw new Error(`Connect another provider that can carry ${andList(stuck.map((u) => streamName[u.stream]))} first.`)
    if (uses.length) {
      const next = { ...draft }
      for (const s of streams) next[s] = withoutProvider(draft[s], id, picked[`primary:${s}`] ?? '', picked[`failover:${s}`] ?? '')
      await http.PUT('/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } }, body: toInput(next) })
    }
    await http.DELETE('/api/v1/workspaces/{workspace}/providers/{providerId}', { params: { path: { workspace: slug, providerId: id } } })
    await Promise.all([
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/providers'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/routing'] }),
    ])
    toast({ kind: 'deleted', title: `${name} removed`, detail: uses.length ? 'Its routes now use the providers you picked; its credentials were deleted.' : 'Its credentials were deleted.' })
    onRemoved()
  }

  const select = (key: string, options: { id: string; name: string }[], none?: string) => (
    <select aria-label={key.startsWith('primary') ? 'Replacement provider' : 'Replacement failover'} className="in w-[170px] flex-none max-md:w-full" value={picked[key] ?? ''} onChange={(e) => setPicked((p) => ({ ...p, [key]: e.target.value }))}>
      {options.map((p) => (
        <option key={p.id} value={p.id}>
          {p.name}
        </option>
      ))}
      {none ? <option value="">{none}</option> : null}
    </select>
  )

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="bad"
      badge="Remove provider"
      title={`Remove ${name} from this workspace?`}
      description={description}
      confirmLabel={uses.length ? 'Reassign routes and remove' : 'Remove provider'}
      busyLabel="Removing…"
      onConfirm={remove}
    >
      {error ? (
        <Note tone="bad">{error}</Note>
      ) : !cfg ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="80%" />
          <Skeleton w="60%" />
        </div>
      ) : (
        <>
          {uses.map((u, i) => (
            <DialogRow
              key={`${u.kind}:${u.stream}:${i}`}
              label={streamName[u.stream]}
              first={i === 0}
              aside={
                u.kind === 'primary'
                  ? select(`primary:${u.stream}`, candidates(u.stream))
                  : u.kind === 'failover'
                    ? select(`failover:${u.stream}`, candidates(u.stream, primaryOf(u.stream)), 'None, pause and ask')
                    : undefined
              }
            >
              {u.kind === 'primary'
                ? `Primary: ${name} → ${picked[`primary:${u.stream}`] ? 'choose a replacement' : `no other provider can carry ${streamName[u.stream]}`}`
                : u.kind === 'failover'
                  ? `Failover: ${name} → choose a replacement`
                  : u.kind === 'share'
                    ? `Split: ${name} ${u.weight}% → goes to ${nameOf(primaryOf(u.stream))}`
                    : `Condition: ${u.values} → ${name} only, removed`}
            </DialogRow>
          ))}
          <DialogRow tone="neutral" label="Keeps" first={uses.length === 0}>
            The {count(cfg.shared.suppressions)} suppressions in the shared list stay, and so do its past events and reports.
          </DialogRow>
          {stuck.length ? <Note tone="warn">Connect another provider that can carry {andList(stuck.map((u) => streamName[u.stream]))} first.</Note> : null}
        </>
      )}
    </ConfirmDialog>
  )
}
