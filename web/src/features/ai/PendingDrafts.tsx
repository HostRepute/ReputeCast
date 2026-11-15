import { useInfiniteQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Pager } from '@/components/ui/Pager'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ag } from './agent.messages'
import { aiBase } from './api'
import { md } from './draft.messages'

/** The server owns recovery, expiry and access; no content goes in browser storage. */
export function PendingDrafts({ kind, target, all = false, onOpen }: { kind: 'campaigns' | 'automations'; target?: string; all?: boolean; onOpen: (id: string) => Promise<void> }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [opening, setOpening] = useState('')
  const [error, setError] = useState('')
  const endpoint = `${aiBase}/drafts/${kind}` as const
  const q = useInfiniteQuery({
    queryKey: ['get', endpoint, { params: { path: { workspace: slug }, query: { target, all } } }],
    initialPageParam: '',
    queryFn: async ({ pageParam, signal }) => (await http.GET(endpoint, { params: { path: { workspace: slug }, query: { target, all, before: pageParam || undefined } }, signal })).data!,
    getNextPageParam: (page) => page.next || undefined,
  })
  const open = async (id: string) => {
    setError('')
    setOpening(id)
    try { await onOpen(id) } catch (err) { setError(errorMessage(err)); await q.refetch() } finally { setOpening('') }
  }
  const items = q.data?.pages.flatMap((page) => page.items) ?? []
  if (q.isPending) return <Skeleton h={60} />
  if (!items.length && !q.hasNextPage && !error && !q.error) return null
  const title = `${t(md.pending)} · ${t(ag.kind, { kind: kind === 'campaigns' ? 'campaign' : 'automation' })}`
  return <section aria-label={title} className="flex min-w-0 flex-col gap-3 rounded-md border border-line-2 p-3">
    <h3 className="m-0 text-sm">{title}</h3>
    <p className="k m-0">{t(md.recovery)}</p>
    {error || q.error ? <ErrorPanel title={error || errorMessage(q.error)} actions={<Button onClick={() => { setError(''); void q.refetch() }}>{t(common.retry)}</Button>} /> : null}
    {items.length ? <ul className="m-0 flex list-none flex-col gap-3 p-0">{items.map((item) => <li key={item.id} className="flex min-w-0 flex-wrap items-center justify-between gap-2">
      <div className="min-w-0 flex-1"><strong className="break-words">{item.name}</strong><p className="k m-0">{t(md.expires, { date: shortDate(item.expiresAt) })}</p></div>
      <Button disabled={!!opening || q.isError} loading={opening === item.id} onClick={() => void open(item.id)} aria-label={t(md.reviewNamed, { name: item.name })}>{t(md.review)}</Button>
    </li>)}</ul> : null}
    <Pager query={q} label={t(md.more)} size="md" />
  </section>
}
