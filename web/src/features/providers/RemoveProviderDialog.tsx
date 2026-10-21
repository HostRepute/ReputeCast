import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { andList, count, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { streamName, type Stream } from './common'
import { mo } from './overview.messages'
import { providerUses, streams, toDraft, toInput, withoutProvider, type RoutingConfig } from './routing-model'

/**
 * Removing a provider (Dialogs board, "Remove provider"). Confirming saves
 * the routing with the replacements, then disconnects the provider.
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
  const t = useT()
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
  const nameOf = (pid: string) => others.find((p) => p.id === pid)?.name ?? t(mo.thePrimary)
  const stuck = uses.filter((u) => u.kind === 'primary' && !picked[`primary:${u.stream}`])
  const stuckStreams = andList(stuck.map((u) => streamName[u.stream]))

  const roles = uses.map((u) =>
    u.kind === 'primary'
      ? t(mo.rolePrimaryRoute, { stream: u.stream })
      : u.kind === 'share'
        ? t(mo.roleShareOf, { share: pct(u.weight, 0), stream: streamName[u.stream] })
        : u.kind === 'failover'
          ? t(mo.roleFailoverFor, { stream: streamName[u.stream] })
          : t(mo.roleCondition, { stream: streamName[u.stream] }),
  )
  const description = !cfg
    ? t(mo.removeChecking, { name })
    : uses.length === 0
      ? t(mo.removeNoUses, { name })
      : t(mo.removeUses, { name, roles: andList(roles), orphans: uses.some((u) => u.kind === 'primary' || u.kind === 'failover') ? 'yes' : 'no' })

  const remove = async () => {
    if (!draft) throw new Error(error ?? t(mo.stillChecking, { name }))
    if (stuck.length) throw new Error(t(mo.connectAnother, { streams: stuckStreams }))
    if (uses.length) {
      const next = { ...draft }
      for (const s of streams) next[s] = withoutProvider(draft[s], id, picked[`primary:${s}`] ?? '', picked[`failover:${s}`] ?? '')
      await http.PUT('/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } }, body: toInput(next) })
    }
    const { data } = await http.DELETE('/api/v1/workspaces/{workspace}/providers/{providerId}', { params: { path: { workspace: slug, providerId: id } } })
    await Promise.all([
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/providers'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/routing'] }),
    ])
    const routes = uses.length ? t(mo.removedRoutes) : t(mo.removedCredentials)
    const hook = data?.webhook
    // A webhook left at the provider stays on screen until dismissed: it has to be removed there by hand.
    toast({
      kind: hook === 'failed' || hook === 'manual' ? 'info' : 'deleted',
      title: t(mo.removedTitle, { name }),
      detail:
        hook === 'removed'
          ? `${routes} ${t(mo.removedWebhook, { name })}`
          : hook === 'manual'
            ? `${routes} ${t(mo.removedWebhookManual, { url: data?.url ?? '', name })}`
            : hook === 'failed'
              ? `${routes} ${t(mo.removedWebhookFailed, { name, detail: data?.detail ?? '', url: data?.url ?? '' })}`
              : routes,
      duration: hook === 'failed' || hook === 'manual' ? 0 : undefined,
    })
    onRemoved()
  }

  const select = (key: string, options: { id: string; name: string }[], none?: string) => (
    <select
      aria-label={key.startsWith('primary') ? t(mo.replacementProvider) : t(mo.replacementFailover)}
      className="in w-[170px] flex-none max-md:w-full"
      value={picked[key] ?? ''}
      onChange={(e) => setPicked((p) => ({ ...p, [key]: e.target.value }))}
    >
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
      badge={t(mo.removeProvider)}
      title={t(mo.removeTitle, { name })}
      description={description}
      confirmLabel={uses.length ? t(mo.reassignAndRemove) : t(mo.removeProvider)}
      busyLabel={t(mo.removing)}
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
                    ? select(`failover:${u.stream}`, candidates(u.stream, primaryOf(u.stream)), t(mo.nonePauseAsk))
                    : undefined
              }
            >
              {u.kind === 'primary'
                ? picked[`primary:${u.stream}`]
                  ? t(mo.rowPrimary, { name })
                  : t(mo.rowPrimaryNone, { name, stream: streamName[u.stream] })
                : u.kind === 'failover'
                  ? t(mo.rowFailover, { name })
                  : u.kind === 'share'
                    ? t(mo.rowSplit, { name, share: pct(u.weight, 0), target: nameOf(primaryOf(u.stream)) })
                    : t(mo.rowCondition, { values: u.values, name })}
            </DialogRow>
          ))}
          <DialogRow tone="neutral" label={t(mo.keeps)} first={uses.length === 0}>
            {t(mo.keepsBody, { count: count(cfg.shared.suppressions) })}
            {cfg.providers.find((p) => p.id === id)?.removesWebhook ? ` ${t(mo.keepsWebhooks)}` : ''}
          </DialogRow>
          {stuck.length ? <Note tone="warn">{t(mo.connectAnother, { streams: stuckStreams })}</Note> : null}
        </>
      )}
    </ConfirmDialog>
  )
}
