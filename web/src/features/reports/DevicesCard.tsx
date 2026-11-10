import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Th } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { m } from './messages'
import { TableCard } from './TableCard'
import { shareOf } from './words'

type Breakdown = Schemas['ReportBreakdown']
type Row = Schemas['ReportDimRow']
type View = 'clients' | 'systems' | 'devices'

const deviceWord = { mobile: m.deviceMobile, tablet: m.deviceTablet, desktop: m.deviceDesktop } as const

/** The share of human opens: a bar (decorative, the number says it) and the number. */
export function ShareCell({ share }: { share: number }) {
  return (
    <td>
      <span className="flex items-center gap-2.5">
        <span className="bar min-w-[48px] max-w-[240px] flex-[1_1_120px] max-md:hidden" aria-hidden="true">
          <i style={{ width: `${Math.min(share, 100)}%`, background: 'var(--chart-human)' }} />
        </span>
        <span className="n min-w-[52px] whitespace-nowrap text-right">{shareOf(share)}</span>
      </span>
    </td>
  )
}

/**
 * "Mail clients and devices" (Report board, M57 G-12): the people whose
 * first human engagement came from each mail app, system or device, one
 * table switched by a segmented control.
 */
export function DevicesCard({ b }: { b: Breakdown }) {
  const t = useT()
  const [view, setView] = useState<View>('clients')
  const rows = b[view]
  const empty = b.clients.length === 0
  const col = { clients: t(m.colClient), systems: t(m.colSystem), devices: t(m.colDevice) }[view]
  // Product names as they are; the rest and devices in the app's language.
  const label = (r: Row) => {
    if (r.key === 'other') return t(m.dimOther)
    if (r.key === 'unknown') return t(m.dimUnknown)
    if (view === 'devices' && r.key in deviceWord) return t(deviceWord[r.key as keyof typeof deviceWord])
    return r.label
  }
  const unknown = b.systems.some((r) => r.key === 'unknown') || b.devices.some((r) => r.key === 'unknown')
  const notes = empty
    ? undefined
    : [
        <span key="once">{t(m.devicesOnce)}</span>,
        b.privacyOpens > 0 ? (
          <span key="privacy">{t(m.devicesPrivacy, { n: b.privacyOpens, count: count(b.privacyOpens) })}</span>
        ) : null,
        unknown ? <span key="unknown">{t(m.devicesUnknown)}</span> : null,
      ]
  return (
    <TableCard
      id="rep-dev-h"
      title={t(m.devicesTitle)}
      caption={t(m.devicesCaption)}
      footer={notes}
      actions={
        empty ? undefined : (
          <Segmented
            className="rep-seg"
            value={view}
            onChange={setView}
            label={t(m.groupBy)}
            items={[
              { value: 'clients', label: t(m.viewClients) },
              { value: 'systems', label: t(m.viewSystems) },
              { value: 'devices', label: t(m.viewDevices) },
            ]}
          />
        )
      }
    >
      {empty ? (
        <p className="k m-0 px-4.5 py-3.5">{t(m.noHumanOpens)}</p>
      ) : (
        <table className="tbl rep-tbl">
          <caption className="vh">{t(m.devicesTable, { view })}</caption>
          <thead>
            <tr>
              <Th>{col}</Th>
              <Th align="right">{t(m.humanOpens)}</Th>
              <Th>{t(m.colShareOpens)}</Th>
              <Th align="right">{t(m.humanClicks)}</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.key}>
                <th scope="row">{label(r)}</th>
                <td className="r n">{count(r.humanOpens)}</td>
                <ShareCell share={r.share} />
                <td className="r n">{count(r.clicks)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </TableCard>
  )
}

/**
 * Where the three breakdown cards stand for a campaign that has none: sent
 * before ReputeCast recorded them, or imported with its final totals
 * (ReportStates; the hourly chart's unavailable state).
 */
export function BreakdownUnavailable({ imported }: { imported: boolean }) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-col gap-3 px-5 py-4.5" aria-labelledby="rep-bd-h">
      <h2 className="h2" id="rep-bd-h">
        {t(m.breakdownTitle)}
      </h2>
      <div className="flex min-h-[120px] flex-col items-center justify-center gap-1.5 rounded-lg bg-ground-3 px-6 py-5 text-center">
        <span className="font-semibold">{t(m.noBreakdown)}</span>
        <span className="k max-w-[520px]">{imported ? t(m.noBreakdownImported) : t(m.noBreakdownOld)}</span>
      </div>
    </section>
  )
}
