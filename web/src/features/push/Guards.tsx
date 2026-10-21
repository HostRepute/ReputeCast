import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, count, inZone } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { m } from './composer.messages'

type Campaign = Schemas['PushCampaign']
type Zone = Schemas['PushZone']

const OVERNIGHT_MS = 6 * 3_600_000

/** "09:00" and "20:00" of "09:00–20:00". */
export function windowEnds(window: string): { start: string; end: string } {
  const [start = '09:00', end = '20:00'] = window.split(/[–-]/).map((s) => s.trim())
  return { start, end }
}

/** The instant of a wall-clock time in a time zone ("tomorrow 09:00 in America/New_York"). */
export function zonedInstant(y: number, mo: number, d: number, h: number, min: number, zone: string): Date {
  const guess = Date.UTC(y, mo - 1, d, h, min)
  const offset = (t: number) => {
    const p: Record<string, number> = {}
    for (const x of new Intl.DateTimeFormat('en-US', { timeZone: zone, hourCycle: 'h23', year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric' }).formatToParts(new Date(t)))
      if (x.type !== 'literal') p[x.type] = Number(x.value)
    return Date.UTC(p.year!, p.month! - 1, p.day!, p.hour! % 24, p.minute!) - t
  }
  const first = guess - offset(guess)
  return new Date(guess - offset(first))
}

/** Tomorrow at a time, in a time zone. */
export function tomorrowAt(time: string, zone: string, now = new Date()): Date {
  const p: Record<string, number> = {}
  for (const x of new Intl.DateTimeFormat('en-US', { timeZone: zone, year: 'numeric', month: 'numeric', day: 'numeric' }).formatToParts(now)) if (x.type !== 'literal') p[x.type] = Number(x.value)
  const next = new Date(Date.UTC(p.year!, p.month! - 1, p.day! + 1))
  const [h = 9, min = 0] = time.split(':').map(Number)
  return zonedInstant(next.getUTCFullYear(), next.getUTCMonth() + 1, next.getUTCDate(), h, min, zone)
}

function continent(zone: string): string {
  return zone.split('/')[0] ?? ''
}

/** "New York" for "America/New_York". */
function city(zone: string): string {
  return (zone.split('/').pop() ?? zone).replace(/_/g, ' ')
}

/**
 * The frequency cap and quiet hours as they apply to this audience now
 * (board PushStates › Composer): a line when nothing needs a decision, the
 * cap's warning with today's notifications, or the time zones that quiet
 * hours hold overnight with Hold or Skip.
 */
export function GuardNotes({ data, editable }: { data: Campaign; editable: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const g = data.guards
  const [skipCap, setSkipCap] = useState(false)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/send-settings')
  const { start, end } = windowEnds(g.window)
  const now = useNow(60_000).getTime()
  const overnight = g.quiet ? g.zones.filter((z) => !z.inside && z.until && new Date(z.until).getTime() - now > OVERNIGHT_MS) : []
  const heldOvernight = overnight.reduce((a, z) => a + z.subscribers, 0)
  const settings = (patch: Partial<Schemas['PushSendSettings']>, done?: string) =>
    save.mutate(
      {
        params: { path: { workspace: slug, campaignId: data.id } },
        body: { mode: data.schedule.mode, sendAt: data.schedule.sendAt, outside: data.schedule.outside, ...patch },
      },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}'] })
          if (done) toast({ kind: 'success', title: done })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const capLine = g.overCap ? t(m.capSome, { cap: g.cap, n: g.overCap }) : t(m.capNobody, { cap: g.cap })
  const quietLine = !g.quiet ? t(m.quietOff) : g.held ? t(m.quietHeld, { n: g.held, start }) : t(m.quietInside, { window: g.window })
  return (
    <>
      {g.overCap > 0 && !skipCap ? (
        <div className="card flex flex-col gap-3 p-4">
          <div className="note w" role="note">
            <WarnIcon />
            <div className="px-note-body">
              <b>{t(m.overCapTitle, { n: g.overCap, cap: g.cap })}</b>
              <div>
                {t(m.overCapText, { rest: Math.max(0, data.audience.final) })}
                {editable && data.access.canSend ? (
                  <div className="px-note-acts">
                    <Button
                      size="sm"
                      loading={save.isPending}
                      onClick={() => {
                        const at = tomorrowAt(start, data.schedule.zone)
                        settings({ mode: 'at', sendAt: at.toISOString() }, t(m.scheduledTomorrow, { time: start }))
                      }}
                    >
                      {t(m.scheduleTomorrow, { time: start })}
                    </Button>
                    <Button size="sm" onClick={() => setSkipCap(true)}>
                      {t(m.sendAndSkip)}
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
          {g.sentToday.length ? (
            <Table aria-label={t(m.sentToday)}>
              <thead>
                <tr>
                  <Th>{t(m.sentToday)}</Th>
                  <Th>{t(m.colKind)}</Th>
                  <Th align="right">{t(m.colTo)}</Th>
                  <Th>{t(m.colWhen)}</Th>
                </tr>
              </thead>
              <tbody>
                {g.sentToday.map((s, i) => (
                  <tr key={i}>
                    <Td>{s.name}</Td>
                    <Td>{s.kind === 'automation' ? t(m.kindAutomation, { name: s.by }) : t(m.kindCampaign, { name: s.by })}</Td>
                    <Td num align="right">
                      {count(s.to)}
                    </Td>
                    <Td className="k">{clock(s.at)}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          ) : null}
        </div>
      ) : null}
      {heldOvernight > 0 ? (
        <QuietCard data={data} zones={overnight} held={heldOvernight} start={start} end={end} editable={editable} onOutside={(outside) => settings({ outside })} busy={save.isPending} />
      ) : null}
      {g.overCap > 0 && !skipCap ? null : heldOvernight > 0 ? null : (
        <div className="note n bg-white" role="note">
          <InfoIcon />
          <div>{`${capLine} ${quietLine}`}</div>
        </div>
      )}
    </>
  )
}

function QuietCard({
  data,
  zones,
  held,
  start,
  end,
  editable,
  onOutside,
  busy,
}: {
  data: Campaign
  zones: Zone[]
  held: number
  start: string
  end: string
  editable: boolean
  onOutside: (o: 'hold' | 'skip') => void
  busy: boolean
}) {
  const t = useT()
  const g = data.guards
  const regions = new Set(zones.map((z) => continent(z.zone)))
  const region = regions.size === 1 ? [...regions][0]! : 'other'
  const evening = zones.every((z) => z.localTime >= end)
  const here = inZone(new Date(), data.schedule.zone)
  const top = g.zones.slice(0, 6)
  const rest = g.zones.slice(6).reduce((a, z) => a + z.subscribers, 0)
  const ttl = data.ttlMinutes % 60 === 0 ? t(m.hours, { n: data.ttlMinutes / 60 }) : t(m.minutes, { n: data.ttlMinutes })
  const outside = data.schedule.outside
  return (
    <div className="card flex flex-col gap-3 p-4">
      <div className="note i" role="note">
        <InfoIcon color="#1E40AF" />
        <div className="px-note-body">
          <b>{t(m.quietTitle, { n: held, region })}</b>
          <div>
            {evening
              ? t(m.quietTextEvening, { time: here.time, city: city(data.schedule.zone), n: held, region, end, start })
              : t(m.quietText, { time: here.time, city: city(data.schedule.zone), n: held, window: g.window, start })}
          </div>
        </div>
      </div>
      <Table aria-label={t(m.colZone)}>
        <thead>
          <tr>
            <Th>{t(m.colZone)}</Th>
            <Th>{t(m.colLocal)}</Th>
            <Th align="right">{t(m.colSubscribers)}</Th>
            <Th>{t(m.colWhat)}</Th>
          </tr>
        </thead>
        <tbody>
          {top.map((z) => (
            <tr key={z.zone}>
              <Td>{z.zone}</Td>
              <Td num>{z.localTime}</Td>
              <Td num align="right">
                {count(z.subscribers)}
              </Td>
              <Td>{z.inside ? <Badge tone="ok">{t(m.zoneNow)}</Badge> : <Badge tone="warn">{heldWord(t, z)}</Badge>}</Td>
            </tr>
          ))}
          {rest ? (
            <tr>
              <Td>{t(m.zoneOther)}</Td>
              <Td num>–</Td>
              <Td num align="right">
                {count(rest)}
              </Td>
              <Td>
                <Badge tone="neutral">{t(m.zoneOtherWhat)}</Badge>
              </Td>
            </tr>
          ) : null}
        </tbody>
      </Table>
      <div className="flex flex-wrap gap-2.5" role="radiogroup" aria-label={t(m.colWhat)}>
        <label className={outside === 'hold' ? 'pc-optc on' : 'pc-optc'}>
          <input type="radio" name="push-outside" checked={outside === 'hold'} disabled={!editable || busy} onChange={() => onOutside('hold')} />
          <span className="flex flex-col gap-0.5 font-semibold">
            {t(m.holdThem, { start })}
            <span className="k">{t(m.holdThemHint, { ttl })}</span>
          </span>
        </label>
        <label className={outside === 'skip' ? 'pc-optc on' : 'pc-optc'}>
          <input type="radio" name="push-outside" checked={outside === 'skip'} disabled={!editable || busy} onChange={() => onOutside('skip')} />
          <span className="flex flex-col gap-0.5 font-semibold">
            {t(m.skipThem)}
            <span className="k">{t(m.skipThemHint)}</span>
          </span>
        </label>
      </div>
    </div>
  )
}

/** "Held until 09:00 tomorrow", in the zone's own time. */
function heldWord(t: ReturnType<typeof useT>, z: Zone): string {
  if (!z.until) return t(m.zoneHeld, { time: '' })
  const at = inZone(z.until, z.zone)
  const today = inZone(new Date(), z.zone)
  return at.date !== today.date ? t(m.zoneHeldTomorrow, { time: at.time }) : t(m.zoneHeld, { time: at.time })
}

function InfoIcon({ color = '#4B5563' }: { color?: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

function WarnIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}
