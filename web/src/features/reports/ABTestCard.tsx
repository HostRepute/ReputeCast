import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { clock, count, decimal, pct, shortDate } from '@/lib/format'
import { m } from './messages'
import { abExplanation, pct1, pct2, variantLine, type CampaignReport } from './words'

type AB = Schemas['ReportABTest']
type Variant = Schemas['ReportABVariant']
type TT = ReturnType<typeof useT>

/**
 * The card's badge: "Testing · decides at 12:00", "No clear winner",
 * "Winner B · 98% confidence · decided on clicks". A confidence is only
 * claimed for a significant decision; a test decided before decisions were
 * kept has none to claim.
 */
export function abHeadline(t: TT, x: AB): { tone: Tone; text: string } {
  if (x.state === 'testing') return { tone: 'info', text: x.decidedAt ? t(m.abTestingAt, { time: clock(x.decidedAt) }) : t(m.abTesting) }
  if (x.state === 'no_winner') return { tone: 'neutral', text: t(m.abNoWinner) }
  const variant = x.winner ?? ''
  if (x.outcome === 'manual') return { tone: 'ok', text: t(m.abChosenBy, { variant }) }
  if (x.outcome === 'unrecorded') return { tone: 'neutral', text: t(m.abChosenUnrecorded, { variant }) }
  if (!x.significant) return { tone: 'warn', text: t(m.abChosen, { variant }) }
  if (x.confidence === undefined) return { tone: 'ok', text: t(m.abWinner, { variant }) }
  return { tone: 'ok', text: t(m.abWinnerOn, { variant, confidence: pct(Math.min(99, Math.round(x.confidence)), 0), metric: x.metric }) }
}

/** The decision rates, the leader first: "B 3.4%, A 2.6%". */
function decisionRates(x: AB): string {
  return [...x.variants]
    .filter((v) => v.decisionRate !== undefined)
    .sort((a, b) => (b.decisionRate ?? 0) - (a.decisionRate ?? 0))
    .map((v) => `${v.key} ${pct1(v.decisionRate)}`)
    .join(', ')
}

/** What the bars show: the final numbers, or when they arrive. */
function barsLine(t: TT, x: AB, final: boolean, finalAt?: string): string | undefined {
  const max = pct(x.barMax, 0)
  if (final) return t(m.abBarsFinal, { metric: x.metric, max })
  return finalAt ? t(m.abBarsComing, { day: shortDate(finalAt), time: clock(finalAt), max }) : undefined
}

/** "How the winner was picked", as the board words it. */
export function abHow(t: TT, x: AB, final: boolean, finalAt?: string): string[] {
  if (x.outcome === 'unrecorded') {
    // Decided before decisions were kept: what decided it isn't known.
    const rest = x.remaining > 0 && x.rolloutAt
    const out = [rest ? t(m.abUnrecorded, { count: count(x.remaining), winner: x.winner ?? x.default, time: clock(x.rolloutAt!) }) : t(m.abUnrecordedNoRest)]
    const bars = barsLine(t, x, final, finalAt)
    return bars ? [...out, bars] : out
  }
  if (x.state === 'testing' || x.remaining <= 0 || x.decidedAfter === undefined) return abExplanation(x)
  const hours = decimal(x.decidedAfter, Number.isInteger(x.decidedAfter) ? 0 : 1)
  const values = {
    metric: x.metric,
    hours,
    rates: decisionRates(x),
    min: count(x.minSample),
    count: count(x.remaining),
    winner: x.winner ?? x.default,
    time: x.rolloutAt ? clock(x.rolloutAt) : '',
    threshold: pct(x.threshold, 0),
  }
  const out: string[] = []
  if (x.outcome === 'manual') out.push(t(m.abPicked, values))
  else out.push(t(x.state === 'no_winner' ? m.abZTestNoWinner : m.abZTest, values))
  const bars = barsLine(t, x, final, finalAt)
  if (final) out.push(`${t(x.metric === 'clicks' ? m.abOpensCouldnt : m.abPrivacyLeftOut)} ${bars}`)
  else if (bars) out.push(bars)
  return out
}

/** "B was 0.4 pts ahead at 61% confidence, so everyone else got A (your default)." */
export function abLead(t: TT, x: AB): string | null {
  if (x.state !== 'no_winner') return null
  const winner = x.winner ?? x.default
  if (x.outcome === 'too_small') return t(m.abLeadSmall, { min: count(x.minSample), winner })
  if (!x.leader || x.lift === undefined || x.confidence === undefined) return null
  return t(m.abLead, { leader: x.leader, lift: decimal(Math.abs(x.lift), 1), confidence: pct(Math.min(99, Math.round(x.confidence)), 0), winner })
}

function VariantPane({ v, x, final }: { v: Variant; x: AB; final: boolean }) {
  const t = useT()
  // A test decided before decisions were kept has no numbers as of its decision.
  const unrecorded = x.outcome === 'unrecorded'
  const decided = x.state !== 'testing' && x.decidedAt !== undefined && !unrecorded
  // Before the results are final the bars show the decision, as of when it was made.
  const atDecision = !final && decided
  const value = atDecision ? v.decisionRate : v.metricRate
  const metric = t(x.metric === 'clicks' ? m.abMetricClicks : m.abMetricOpens)
  const time = x.decidedAt ? clock(x.decidedAt) : ''
  const winner = x.state === 'decided' && v.winner
  const badge = winner
    ? { tone: 'ok' as Tone, text: t(unrecorded ? m.abVariantChosen : m.abVariantWinner, { variant: v.key }) }
    : x.state === 'no_winner' && v.default
      ? { tone: 'neutral' as Tone, text: t(m.abYourDefault, { variant: v.key }) }
      : { tone: 'neutral' as Tone, text: v.key }
  const sub =
    x.metric === 'clicks'
      ? final
        ? t(m.abSubOpens, { rate: pct1(v.humanOpenRate), complaints: pct2(v.complaintRate) })
        : t(m.abSubOpensSoFar, { rate: pct1(v.humanOpenRate) })
      : final
        ? t(m.abSubClicks, { rate: pct1(v.clickRate), complaints: pct2(v.complaintRate) })
        : t(m.abSubClicksSoFar, { rate: pct1(v.clickRate) })
  const color = winner ? 'var(--accent)' : x.state === 'decided' ? '#8B94A3' : 'var(--ink)'
  const width = x.barMax > 0 && value !== undefined ? Math.min(100, (value / x.barMax) * 100) : 0
  return (
    <div className={winner ? 'rep-pane win' : 'rep-pane'}>
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <Badge tone={badge.tone}>{badge.text}</Badge>
        <span className="k">{variantLine(v)}</span>
      </div>
      <span className="font-semibold">{v.label}</span>
      <div className="flex items-baseline justify-between gap-2.5 text-[13px]">
        <span>{atDecision ? t(m.abMetricAt, { metric, time }) : metric}</span>
        <b className="n">{pct1(value)}</b>
      </div>
      <div
        className="bar rep-abbar"
        role="img"
        aria-label={t(m.abBarLabel, { variant: v.key, rate: pct1(value), metric: x.metric, max: pct(x.barMax, 0) })}
      >
        <i style={{ width: `${width}%`, background: color }} />
      </div>
      <span className="k">{sub}</span>
    </div>
  )
}

/** "A/B test": the decision with its confidence and sample, honest when there is no clear winner (Report, ReportStates). */
export function ABTestCard({ r }: { r: CampaignReport }) {
  const t = useT()
  const x = r.abTest!
  const final = r.campaign.final
  const badge = abHeadline(t, x)
  const lead = abLead(t, x)
  const how = abHow(t, x, final, r.campaign.finalAt)
  return (
    <section className="card flex flex-col" aria-labelledby="rep-ab-h">
      <div className="sec">
        <h2 className="h2" id="rep-ab-h">
          {t(m.abTitle, { label: x.label })}
        </h2>
        <Badge tone={badge.tone} dot>
          {badge.text}
        </Badge>
      </div>
      {lead ? <p className="m-0 border-b border-line-2 px-4.5 py-3 text-[14px]">{lead}</p> : null}
      <div className="rep-panes">
        {x.variants.map((v) => (
          <VariantPane key={v.key} v={v} x={x} final={final} />
        ))}
        <div className="rep-pane gap-1.5 text-[13px]">
          <span className="font-semibold">{x.state === 'testing' ? t(m.abHowIs) : x.state === 'no_winner' ? t(m.abHowDecided) : t(m.abHowWas)}</span>
          {how.map((l) => (
            <span key={l} className="k">
              {l}
            </span>
          ))}
        </div>
      </div>
    </section>
  )
}
