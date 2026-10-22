import { useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState } from 'react'
import { $api } from '@/api/client'
import { cn } from '@/lib/cn'
import { shortDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './messages'

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const h = window.setTimeout(() => setV(value), ms)
    return () => window.clearTimeout(h)
  }, [value, ms])
  return v
}

type Hit = { key: string; group: string; title: string; detail: string; go: () => void }

/**
 * "Search campaigns, contacts, events": results from the campaigns list and
 * the provider event log as you type (⌘K focuses it). Enter opens the
 * campaigns list filtered by the text.
 */
export function OverviewSearch() {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const id = useId()
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const box = useRef<HTMLDivElement>(null)
  const term = useDebounced(q.trim(), 200)
  const ready = term.length >= 2
  const canCampaigns = useCan('campaigns:view')
  const canEvents = useCan('providers:view')
  const path = { workspace: slug }
  const campaigns = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns', { params: { path, query: { q: term, pageSize: 5 } } }, { enabled: ready && canCampaigns })
  const events = $api.useQuery('get', '/api/v1/workspaces/{workspace}/provider-events', { params: { path, query: { q: term, pageSize: 5 } } }, { enabled: ready && canEvents })
  const hits: Hit[] = []
  for (const c of campaigns.data?.items ?? []) {
    const report = ['sent', 'cancelled', 'failed'].includes(c.status)
    const live = c.status === 'sending' || c.status === 'paused'
    hits.push({
      key: `c-${c.id}`,
      group: t(m.groupCampaigns),
      title: c.name,
      detail: [c.audience, c.startedAt ? shortDate(c.startedAt) : c.scheduledAt ? t(m.scheduledOn, { date: shortDate(c.scheduledAt) }) : t(m.campaignStatus, { status: c.status })]
        .filter(Boolean)
        .join(' · '),
      go: () =>
        void navigate({
          to: live ? '/w/$workspace/campaigns/$campaignId/live' : report ? '/w/$workspace/campaigns/$campaignId/report' : '/w/$workspace/campaigns/$campaignId/edit',
          params: { workspace: slug, campaignId: c.id },
        }),
    })
  }
  for (const e of events.data?.items ?? []) {
    hits.push({
      key: `e-${e.id}`,
      group: t(m.groupEvents),
      title: `${e.recipient} · ${t(m.eventType, { type: e.type, label: e.type.replace('_', ' ') })}`,
      detail: [e.provider, e.source, e.code].filter(Boolean).join(' · '),
      go: () => void navigate({ to: '/w/$workspace/providers/events', params: { workspace: slug }, search: { q: term } }),
    })
  }
  const loading = ready && (campaigns.isFetching || events.isFetching)
  const show = open && ready
  useEffect(() => {
    const close = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [])
  const submit = () => {
    const hit = hits[active]
    if (hit) hit.go()
    else if (term) void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: { q: term } })
    setOpen(false)
  }
  let lastGroup = ''
  return (
    <div ref={box} className="relative max-md:w-full">
      <label className="vh" htmlFor={`${id}-q`}>
        {t(common.search)}
      </label>
      <input
        id={`${id}-q`}
        type="search"
        data-shortcut="search"
        className="in w-[260px] max-md:w-full"
        placeholder={t(m.searchPlaceholder)}
        aria-label={t(m.searchPlaceholder)}
        role="combobox"
        aria-expanded={show}
        aria-controls={`${id}-list`}
        aria-autocomplete="list"
        aria-activedescendant={show && hits[active] ? `${id}-${hits[active]!.key}` : undefined}
        value={q}
        onChange={(e) => {
          setQ(e.target.value)
          setActive(0)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') setOpen(false)
          else if (e.key === 'ArrowDown') {
            e.preventDefault()
            setActive((a) => Math.min(hits.length - 1, a + 1))
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setActive((a) => Math.max(0, a - 1))
          } else if (e.key === 'Enter') {
            e.preventDefault()
            submit()
          }
        }}
      />
      {show ? (
        <div id={`${id}-list`} role="listbox" aria-label={t(m.searchResults)} className="menu right-0 top-[42px] w-[360px] max-md:left-0 max-md:w-auto">
          {hits.map((h, i) => {
            const head = h.group !== lastGroup ? h.group : ''
            lastGroup = h.group
            return (
              <div key={h.key} className="contents">
                {head ? <div className="mh" role="presentation">{head}</div> : null}
                <div
                  id={`${id}-${h.key}`}
                  role="option"
                  aria-selected={i === active}
                  tabIndex={-1}
                  className={cn('flex cursor-pointer flex-col rounded-lg px-2.5 py-1.5', i === active && 'bg-ground')}
                  onMouseEnter={() => setActive(i)}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => {
                    h.go()
                    setOpen(false)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      h.go()
                      setOpen(false)
                    }
                  }}
                >
                  <span className="text-[13px] font-medium">{h.title}</span>
                  {h.detail ? <span className="k">{h.detail}</span> : null}
                </div>
              </div>
            )
          })}
          {!hits.length ? (
            <div className="k px-2.5 py-2" role="status">
              {loading ? t(m.searching) : t(m.searchNothing, { term })}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
