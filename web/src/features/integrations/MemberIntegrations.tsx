import { useQueries } from '@tanstack/react-query'
import { useState } from 'react'
import { http, type Schemas } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, entryName, stamp, Tile, useRefreshHub, type EntryKey } from './shared'

type Owned = Schemas['IntegrationOwned']
type Choice = 'take-over' | 'disconnect'

/**
 * The integrations whose keys the members being deactivated or removed own
 * (TeamDialogs): each is taken over by default, or disconnected.
 */
export function useMemberIntegrations(userIds: string[]) {
  const { slug } = useWorkspace()
  const refresh = useRefreshHub()
  const queries = useQueries({
    queries: userIds.map((userId) => ({
      queryKey: ['get', `${base}/owned-by/{userId}`, { params: { path: { workspace: slug, userId } } }],
      queryFn: async () => (await http.GET(`${base}/owned-by/{userId}`, { params: { path: { workspace: slug, userId } } })).data!,
    })),
  })
  const items: Owned[] = queries.flatMap((q) => q.data?.items ?? [])
  const canManage = queries.every((q) => q.data?.canManage ?? true)
  const loaded = queries.every((q) => q.data || q.isError)
  const [choices, setChoices] = useState<Record<string, Choice>>({})
  const choice = (id: string): Choice => choices[id] ?? 'take-over'
  const set = (id: string, c: Choice) => setChoices((prev) => ({ ...prev, [id]: c }))

  /** Takes over or disconnects each one, before the member goes. */
  async function apply(reason: string) {
    if (!canManage) return
    for (const it of items) {
      const path = { params: { path: { workspace: slug, id: it.connectionId } } }
      if (choice(it.connectionId) === 'disconnect') await http.POST(`${base}/connections/{id}/disconnect`, { ...path, body: { reason: reason || undefined } })
      else await http.POST(`${base}/connections/{id}/take-over`, path)
    }
    if (items.length) await refresh()
  }
  return { items, canManage, loaded, choice, set, apply }
}

export type MemberIntegrationsState = ReturnType<typeof useMemberIntegrations>

/** The fieldset: one row per integration with Take over (default) or Disconnect. */
export function MemberIntegrations({ state, name }: { state: MemberIntegrationsState; name?: string }) {
  const t = useT()
  if (!state.loaded || state.items.length === 0) return null
  const tools = formatList([...new Set(state.items.map((i) => t(entryName[i.kind as EntryKey] ?? m.title)))], 'disjunction')
  return (
    <fieldset className="m-0 flex flex-col gap-2.5 rounded-lg border border-line px-3.5 py-2.5">
      <legend className="px-1 text-[12px] font-semibold text-ink-2">
        {name ? t(m.memberLegend, { name, count: state.items.length }) : t(m.memberLegendMany, { count: state.items.length })}
      </legend>
      {state.items.map((it) => {
        const kind = it.kind as EntryKey
        const label = it.kind === 'wordpress' ? `${t(m.wordpress)} · ${it.label}` : t(entryName[kind] ?? m.title)
        const line =
          it.kind === 'wordpress'
            ? it.silentSince
              ? t(m.memberSiteSilent, { since: stamp(it.silentSince) })
              : t(m.memberSite)
            : t(m.memberPlatform, { kind: it.kind, key: it.keyName, count: it.workflows })
        const group = `int-${it.connectionId}`
        return (
          <div key={it.connectionId} className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
            <div className="int-member-who row min-w-0 flex-nowrap gap-2.5">
              <Tile k={kind} size="sm" />
              <span className="flex min-w-0 flex-col">
                <span className="font-semibold">{label}</span>
                <span className="k">{line}</span>
              </span>
            </div>
            {state.canManage ? (
              <div className="row gap-3 text-[13.5px]" role="radiogroup" aria-label={label}>
                <label className="row gap-1.5">
                  <input type="radio" name={group} checked={state.choice(it.connectionId) === 'take-over'} onChange={() => state.set(it.connectionId, 'take-over')} />
                  {t(m.memberTakeOver)} <span className="k">{t(m.memberDefault)}</span>
                </label>
                <label className="row gap-1.5">
                  <input type="radio" name={group} checked={state.choice(it.connectionId) === 'disconnect'} onChange={() => state.set(it.connectionId, 'disconnect')} />
                  {t(m.memberDisconnect)}
                </label>
              </div>
            ) : null}
          </div>
        )
      })}
      <span className="k">{state.canManage ? t(m.memberFoot, { tools }) : t(m.memberNoManage)}</span>
    </fieldset>
  )
}
