import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

const TOP = 8

/** Blacklist status: 84 sources via HostRepute, or the built-in 8. */
export function BlocklistCard({ b, connected, canConnect }: { b: Schemas['DeliverabilityBlocklistStatus']; connected: boolean; canConnect: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [all, setAll] = useState(false)
  const viaHostRepute = b.via === 'hostrepute'
  const rows = all ? b.results : b.results.slice(0, TOP)
  const title = viaHostRepute ? t(m.blocklistTitleVia, { sources: b.sources }) : t(m.blocklistTitleBuiltin, { sources: b.sources })
  const interval = t(m.interval, { interval: b.interval })
  const caption = viaHostRepute
    ? b.stale
      ? t(m.blocklistCaptionStale, { assets: b.assets.join(', '), interval })
      : t(m.blocklistCaption, { assets: b.assets.join(', '), interval })
    : t(m.blocklistCaptionBuiltin)
  return (
    <section className="card min-w-0 flex-[1_1_520px]" aria-labelledby="h-blocklists">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="h-blocklists">
            {title}
          </h2>
          <span className="k">{caption}</span>
        </div>
        <div className="flex gap-1.5">
          <Badge tone="ok">{t(m.cleanCount, { count: count(b.clean) })}</Badge>
          {b.listed ? <Badge tone="warn">{t(m.listedCount, { count: count(b.listed) })}</Badge> : null}
        </div>
      </div>
      {b.results.length ? (
        <Table density="dense" phoneWrap caption={title} className="dl-blocklists">
          <thead>
            <tr>
              <Th>{t(m.colSource)}</Th>
              <Th>{t(m.colChecks)}</Th>
              <Th>{t(m.colStatus)}</Th>
              {viaHostRepute ? <Th>{t(m.colSince)}</Th> : null}
              <Th>{t(m.colDelisting)}</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.source} className={r.listed ? 'bg-[#FFFBEB]' : undefined}>
                <Td className="font-semibold">{r.source}</Td>
                <Td className="mono">{r.assets.join(', ')}</Td>
                <Td>{r.listed ? <Badge tone={r.impact === 'high' ? 'bad' : 'warn'}>{t(m.listed)}</Badge> : <Badge tone="ok">{t(m.clean)}</Badge>}</Td>
                {viaHostRepute ? <Td className="dl-since">{r.since ? dayMonthDate(r.since) : '–'}</Td> : null}
                {r.listed && r.delistUrl ? (
                  <Td>
                    <ButtonAnchor size="sm" href={r.delistUrl} aria-label={t(m.delistingPageLabel, { source: r.source })}>
                      {t(m.delistingPage)}
                    </ButtonAnchor>
                  </Td>
                ) : (
                  <Td className="k">–</Td>
                )}
              </tr>
            ))}
          </tbody>
        </Table>
      ) : (
        <p className="px-4.5 py-4 text-ink-2">{t(m.noBlocklistCheck)}</p>
      )}
      {viaHostRepute ? (
        b.results.length > TOP ? (
          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
            <span className="k" role="status">
              {t(m.showingSorted, { shown: all ? count(b.results.length) : count(TOP), total: count(b.results.length) })}
            </span>
            <Button size="sm" onClick={() => setAll(!all)} aria-expanded={all}>
              {all ? t(m.showTop, { n: count(TOP) }) : t(m.showAll, { n: count(b.results.length) })}
            </Button>
          </div>
        ) : null
      ) : !connected ? (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-b-[10px] border-t border-line-2 bg-ground-3 px-4.5 py-3">
          <span className="k">{t(m.withHostRepute)}</span>
          {canConnect ? (
            <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
              {t(m.connectHostRepute)}
            </ButtonLink>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
