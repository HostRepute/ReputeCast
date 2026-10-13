import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Table, Td, Th } from '@/components/ui/Table'
import { count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

const TOP = 8

/** Blacklist status: 84 sources via HostRepute, or the built-in 8. */
export function BlocklistCard({ b, connected, canConnect }: { b: Schemas['DeliverabilityBlocklistStatus']; connected: boolean; canConnect: boolean }) {
  const { slug } = useWorkspace()
  const [all, setAll] = useState(false)
  const viaHostRepute = b.via === 'hostrepute'
  const rows = all ? b.results : b.results.slice(0, TOP)
  const title = viaHostRepute ? `Blacklist status · ${b.sources} sources via HostRepute` : `Blacklist status · built-in, ${b.sources} public sources`
  const caption = viaHostRepute ? `${b.assets.join(', ')} · ${b.interval}${b.stale ? ' · stale' : ''}` : 'checked once a day · delisting links included'
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
          <Badge tone="ok">{count(b.clean)} clean</Badge>
          {b.listed ? <Badge tone="warn">{count(b.listed)} listed</Badge> : null}
        </div>
      </div>
      {b.results.length ? (
        <Table density="dense" phoneWrap caption={title}>
          <thead>
            <tr>
              <Th>Source</Th>
              <Th>Checks</Th>
              <Th>Status</Th>
              {viaHostRepute ? <Th>Since</Th> : null}
              <Th>Delisting</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.source} className={r.listed ? 'bg-[#FFFBEB]' : undefined}>
                <Td className="font-semibold">{r.source}</Td>
                <Td className="mono">{r.assets.join(', ')}</Td>
                <Td>{r.listed ? <Badge tone={r.impact === 'high' ? 'bad' : 'warn'}>Listed</Badge> : <Badge tone="ok">Clean</Badge>}</Td>
                {viaHostRepute ? <Td>{r.since ? dayMonthDate(r.since) : '–'}</Td> : null}
                {r.listed && r.delistUrl ? (
                  <Td>
                    <ButtonAnchor size="sm" href={r.delistUrl} aria-label={`Delisting page of ${r.source} (opens in a new tab)`}>
                      Delisting page
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
        <p className="px-4.5 py-4 text-ink-2">No blocklist check has run yet. Run the checks to see where your domains and IPs stand.</p>
      )}
      {viaHostRepute ? (
        b.results.length > TOP ? (
          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
            <span className="k" role="status">
              Showing {all ? count(b.results.length) : TOP} of {count(b.results.length)} · sorted by impact
            </span>
            <Button size="sm" onClick={() => setAll(!all)} aria-expanded={all}>
              {all ? `Show top ${TOP}` : `Show all ${count(b.results.length)}`}
            </Button>
          </div>
        ) : null
      ) : !connected ? (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-b-[10px] border-t border-line-2 bg-ground-3 px-4.5 py-3">
          <span className="k">With HostRepute: 84 sources every hour, listing history, and one-click delisting requests. Free to connect.</span>
          {canConnect ? (
            <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
              Connect HostRepute
            </ButtonLink>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
