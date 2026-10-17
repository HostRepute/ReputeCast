import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { useT } from '@/i18n'
import { dayMonthDate, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { FOLDER, plainLine, providerColor, Pv, resultLine, Sw, type Counts, type TestRow } from './shared'

type Bar = { test: TestRow; provider: TestRow['providers'][number] }

const W = 720
const LEFT = 52
const RIGHT = 712
const BASE = 240
const TOP = 54
const FONT = 'IBM Plex Sans,system-ui,sans-serif'

/** One bar per test and provider, oldest first: no line and no rate. */
export function TrendChart({ tests, providers, seeds }: { tests: TestRow[]; providers: { id: string; name: string }[]; seeds: number }) {
  const t = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const [only, setOnly] = useState('')
  const bars = useMemo(() => {
    const out: Bar[] = []
    for (const test of [...tests].reverse()) {
      if (test.status === 'sending' || test.status === 'waiting') continue
      for (const p of test.providers) if (!only || p.id === only) out.push({ test, provider: p })
    }
    return out
  }, [tests, only])
  return (
    <section className="card min-w-0 flex-[2_1_560px]">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.trendTitle)}</h2>
          <span className="k">{t(m.trendLead)}</span>
        </div>
        <div className="seg" role="group" aria-label={t(m.trendView)}>
          <button type="button" className={view === 'chart' ? 'on' : undefined} aria-pressed={view === 'chart'} onClick={() => setView('chart')}>
            {t(m.trendChart)}
          </button>
          <button type="button" className={view === 'table' ? 'on' : undefined} aria-pressed={view === 'table'} onClick={() => setView('table')}>
            {t(m.trendTable)}
          </button>
        </div>
      </div>
      <div className="pad flex flex-col gap-3.5">
        <div className="row gap-1.5!" role="group" aria-label={t(m.trendProviders)}>
          <button type="button" className="pl-fchip" aria-pressed={!only} onClick={() => setOnly('')}>
            {t(m.allProviders)}
          </button>
          {providers.map((p) => (
            <button key={p.id} type="button" className="pl-fchip" aria-pressed={only === p.id} onClick={() => setOnly(p.id)}>
              {p.name}
            </button>
          ))}
        </div>
        {bars.length === 0 ? <span className="k">{t(m.trendEmpty)}</span> : view === 'chart' ? <Chart bars={bars} seeds={seeds} /> : <TrendTable bars={bars} />}
        {view === 'chart' ? (
          <div className="pl-plg">
            <span>
              <Sw color={FOLDER.inbox} />
              {t(m.legendInbox)}
            </span>
            <span>
              <Sw color={FOLDER.tab} />
              {t(m.legendTab)}
            </span>
            <span>
              <Sw color={FOLDER.spam} />
              {t(m.legendSpam)}
            </span>
            <span>
              <Sw color={FOLDER.notArrived} />
              {t(m.legendNotArrived)}
            </span>
          </div>
        ) : null}
        <span className="k text-ink-2!">{t(m.trendNote, { seeds })}</span>
      </div>
    </section>
  )
}

type Seg = { value: number; fill: string; ink: string }

function segments(c: Counts): Seg[] {
  return [
    { value: c.inbox, fill: FOLDER.inbox, ink: '#FFFFFF' },
    { value: c.tab, fill: FOLDER.tab, ink: '#0B1220' },
    { value: c.spam, fill: FOLDER.spam, ink: '#FFFFFF' },
    { value: c.notArrived, fill: FOLDER.notArrived, ink: '#0B1220' },
  ].filter((s) => s.value > 0)
}

const f1 = (n: number) => n.toFixed(1)

function Chart({ bars, seeds }: { bars: Bar[]; seeds: number }) {
  const t = useT()
  const [hover, setHover] = useState<number>()
  const max = Math.max(1, seeds, ...bars.map((b) => b.provider.counts.copies))
  const unit = (BASE - TOP) / max
  const slot = (RIGHT - LEFT) / bars.length
  const bw = Math.min(39.6, slot * 0.7)
  const every = slot >= 44 ? 1 : Math.ceil(44 / slot)
  const mid = Math.round(max / 2)
  const label = t(m.trendChartLabel, {
    list: bars.map((b) => t(m.trendBar, { date: dayMonthDate(b.test.startedAt), provider: b.provider.name, result: plainLine(b.provider.counts) })).join('; '),
  })
  const tip = hover === undefined ? undefined : bars[hover]
  return (
    <div className="pl-tchart">
      <svg width="100%" viewBox={`0 0 ${W} 290`} role="img" aria-label={label} style={{ maxWidth: '100%' }} onMouseLeave={() => setHover(undefined)}>
        {[
          [0, '#C9CED6', '0'],
          [mid, '#EEF0F3', String(mid)],
          [max, '#EEF0F3', t(m.trendSeeds, { n: max })],
        ].map(([v, stroke, text]) => {
          const y = BASE - (v as number) * unit
          return (
            <g key={String(v)}>
              <line x1={LEFT} x2={RIGHT} y1={f1(y)} y2={f1(y)} stroke={stroke as string} />
              <text x={44} y={f1(y + 4)} textAnchor="end" fontSize={11} fill="#5B6472" fontFamily={FONT}>
                {text}
              </text>
            </g>
          )
        })}
        {bars.map((b, i) => {
          const x = LEFT + i * slot + (slot - bw) / 2
          const cx = x + bw / 2
          const segs = segments(b.provider.counts)
          let cur = BASE
          const nodes = segs.map((s, k) => {
            const full = s.value * unit
            const bottom = k === 0 ? cur : cur - 2
            const h = k === 0 ? full : full - 2
            const top = bottom - h
            cur -= full
            const last = k === segs.length - 1
            const r = Math.min(4, h / 2)
            const d = last
              ? `M${f1(x)},${f1(bottom)} v-${f1(h - r)} a${f1(r)},${f1(r)} 0 0 1 ${f1(r)},-${f1(r)} h${f1(bw - 2 * r)} a${f1(r)},${f1(r)} 0 0 1 ${f1(r)},${f1(r)} v${f1(h - r)} z`
              : `M${f1(x)},${f1(top)} h${f1(bw)} v${f1(h)} h-${f1(bw)} z`
            return (
              <g key={k}>
                <path d={d} fill={s.fill} />
                {h >= 13 ? (
                  <text x={f1(cx)} y={f1((top + bottom) / 2 + 4)} textAnchor="middle" fontSize={11} fontWeight={700} fill={s.ink} fontFamily={FONT}>
                    {s.value}
                  </text>
                ) : null}
              </g>
            )
          })
          const name = b.provider.short
          const tw = name.length * 6.2
          const showLabel = i % every === 0
          return (
            <g key={`${b.test.id}-${b.provider.id}`} onMouseEnter={() => setHover(i)}>
              <rect x={f1(LEFT + i * slot)} y={0} width={f1(slot)} height={290} fill="transparent" />
              {nodes}
              {showLabel ? (
                <>
                  <text x={f1(cx)} y={258} textAnchor="middle" fontSize={11} fill="#374151" fontFamily={FONT}>
                    {dayMonthDate(b.test.startedAt)}
                  </text>
                  <rect x={f1(cx + 6 - tw / 2 - 12)} y={265} width={8} height={8} rx={2} fill={providerColor(b.provider.adapter)} />
                  <text x={f1(cx + 6)} y={273} textAnchor="middle" fontSize={11} fill="#5B6472" fontFamily={FONT}>
                    {name}
                  </text>
                </>
              ) : null}
            </g>
          )
        })}
        {tip && hover !== undefined ? <Tip bar={tip} i={hover} slot={slot} bw={bw} unit={unit} /> : null}
      </svg>
    </div>
  )
}

/** The hovered bar's outline and its tooltip, drawn in the chart. */
function Tip({ bar, i, slot, bw, unit }: { bar: Bar; i: number; slot: number; bw: number; unit: number }) {
  const t = useT()
  const x = LEFT + i * slot + (slot - bw) / 2
  const cx = x + bw / 2
  const title = t(m.trendTip, { date: shortDate(bar.test.startedAt), content: bar.test.contentName, provider: bar.provider.short })
  const sub = plainLine(bar.provider.counts)
  const w = Math.max(title.length * 6.4, sub.length * 6.1) + 22
  const left = Math.min(Math.max(cx - w / 2, 0), W - w)
  const top = BASE - bar.provider.counts.copies * unit
  return (
    <g pointerEvents="none">
      <rect x={f1(x - 3)} y={f1(top - 3)} width={f1(bw + 6)} height={f1(BASE + 2 - (top - 3))} rx={5} fill="none" stroke="#0B1220" strokeWidth={1.5} />
      <rect x={f1(left)} y={2} width={f1(w)} height={44} rx={6} fill="#0B1220" />
      <path d={`M${f1(cx - 6)},46 l6,6 l6,-6 z`} fill="#0B1220" />
      <text x={f1(left + 11)} y={20} fontSize={11.5} fill="#FFFFFF" fontWeight={600} fontFamily={FONT}>
        {title}
      </text>
      <text x={f1(left + 11)} y={36} fontSize={11.5} fill="#C9CED6" fontWeight={400} fontFamily={FONT}>
        {sub}
      </text>
    </g>
  )
}

function TrendTable({ bars }: { bars: Bar[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="overflow-x-auto">
      <table className="tbl" aria-label={t(m.trendTableLabel)}>
        <thead>
          <tr>
            <th scope="col">{t(m.colDate)}</th>
            <th scope="col">{t(m.colTest)}</th>
            <th scope="col">{t(m.colProvider)}</th>
            <th scope="col">{t(m.colResult)}</th>
          </tr>
        </thead>
        <tbody>
          {bars.map((b) => (
            <tr key={`${b.test.id}-${b.provider.id}`}>
              <td className="nw">{shortDate(b.test.startedAt)}</td>
              <td>
                <Link to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: b.test.id }} className="font-semibold text-ink!">
                  {b.test.contentName}
                </Link>
              </td>
              <td>
                <Pv adapter={b.provider.adapter}>{b.provider.name}</Pv>
              </td>
              <td>{resultLine(b.provider.counts)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
