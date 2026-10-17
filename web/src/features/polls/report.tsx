import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useNow } from '@/lib/hooks'
import { ago, clock, count, duration, shortDate } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { saveBlob } from '@/features/audience/download'
import type { MapPoll } from '@/features/reports/ClickMap'
import { AnswerPagePreview } from './PagePreview'
import { m } from './report.messages'
import './report.css'

export type PollResults = Schemas['PollResults']

function pct(v: number): string {
  return `${v.toFixed(1)}%`
}

const b = (chunks: ReactNode[]) => <b>{chunks}</b>

/** A campaign's polls with their results; live polls refresh every minute. */
export function useCampaignPolls(campaignId: string, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/polls',
    { params: { path: { workspace: slug, campaignId } } },
    { enabled, refetchInterval: (q) => (q.state.data?.items.some((p) => p.live && p.status === 'open') ? 60_000 : false), retry: 1 },
  )
}

/** The click map's poll regions. */
export function useMapPolls(polls: PollResults[] | undefined): MapPoll[] | undefined {
  const t = useT()
  return useMemo(() => polls?.map((p) => ({ id: p.blockId, label: t(m.mapRegion, { count: p.answers }) })), [polls, t])
}

function totals(polls: PollResults[]) {
  return polls.reduce(
    (a, p) => ({
      answers: a.answers + p.answers,
      delivered: Math.max(a.delivered, p.delivered),
      nonContacts: a.nonContacts + p.nonContacts,
      automated: a.automated + p.automated.total,
      changed: a.changed + p.changed,
    }),
    { answers: 0, delivered: 0, nonContacts: 0, automated: 0, changed: 0 },
  )
}

/** "Poll answers" and "Automated visits ignored", after Delivered. */
export function PollTiles({ polls }: { polls: PollResults[] }) {
  const t = useT()
  if (!polls.length) return null
  const s = totals(polls)
  return (
    <>
      <div className="card flex flex-col gap-[3px] px-4 py-3.5">
        <span className="k">{t(m.tileAnswers)}</span>
        <span className="n text-[22px]">{count(s.answers)}</span>
        <span className="k">{t(m.tileAnswersSub, { percent: pct(s.delivered ? (s.answers / s.delivered) * 100 : 0), nonContacts: s.nonContacts })}</span>
      </div>
      <div className="card flex flex-col gap-[3px] px-4 py-3.5">
        <span className="k">{t(m.tileAutomated)}</span>
        <span className="n text-[22px]">{count(s.automated)}</span>
        <span className="k">{t(m.tileAutomatedSub)}</span>
      </div>
    </>
  )
}

function Check() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--ok-bar)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      <path d="M3 8.5l3 3 7-7" />
    </svg>
  )
}

/** "How answers are counted", beside the click map. */
export function HowAnswersCounted({ polls }: { polls: PollResults[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [preview, setPreview] = useState(false)
  const s = totals(polls)
  const first = polls[0]
  const page = polls[0]?.answerPage ?? ''
  const items: ReactNode[] = [
    t(m.howRecorded),
    <T key="a" msg={m.howAutomated} values={{ count: s.automated, b }} />,
    <T key="c" msg={m.howChanged} values={{ count: s.changed, b }} />,
    <T key="n" msg={m.howNonContacts} values={{ count: s.nonContacts, b }} />,
    t(m.howTests),
  ]
  return (
    <section className="card flex min-w-0 flex-[1.2_1_420px] flex-col" aria-labelledby="pr-how-h">
      <h2 className="h2 pr-how-h" id="pr-how-h">
        {t(m.howTitle)}
      </h2>
      <ul className="pr-how">
        {items.map((x, i) => (
          <li key={i}>
            <Check />
            <span>{x}</span>
          </li>
        ))}
      </ul>
      {page ? (
        <div className="pr-how-foot">
          <span className="k">{t(m.howPage, { url: page.replace(/^https?:\/\//, '') })}</span>
          <Button size="sm" onClick={() => setPreview(true)}>
            {t(m.openPage)}
          </Button>
          {preview && first ? <AnswerPagePreview slug={slug} config={first.config} campaign={first.campaign} onClose={() => setPreview(false)} /> : null}
        </div>
      ) : null}
    </section>
  )
}

/** The status badge: open with its closing date, or closed and how. */
function Status({ p }: { p: PollResults }) {
  const t = useT()
  if (p.status === 'closed') {
    const at = p.closedAt ? `${shortDate(p.closedAt)}, ${clock(p.closedAt)}` : ''
    return <span className="badge mut">{p.closedAsScheduled ? t(m.statusClosedScheduled, { date: at }) : p.closedBy ? t(m.statusClosedBy, { date: at, name: p.closedBy }) : t(m.statusClosed, { date: at })}</span>
  }
  return (
    <span className="badge ok">
      <span className="dot" aria-hidden="true" />
      {p.closesAt ? t(m.statusOpenCloses, { date: shortDate(p.closesAt) }) : t(m.statusOpen)}
    </span>
  )
}

/** "1,377 automated visits ignored", with the explanation on hover or focus. */
function AutomatedChip({ p }: { p: PollResults }) {
  const t = useT()
  const id = useId()
  const [open, setOpen] = useState(false)
  return (
    <span className="pr-auto">
      <button
        type="button"
        className={cn('chip pr-chip', open && 'on')}
        aria-describedby={open ? id : undefined}
        aria-expanded={open}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onKeyDown={(e) => e.key === 'Escape' && setOpen(false)}
      >
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true" focusable="false">
          <circle cx="8" cy="8" r="6.5" />
          <path d="M8 7.2v3.6M8 5.2v.1" />
        </svg>
        {t(m.automatedChip, { count: p.automated.total })}
      </button>
      {open ? (
        <span role="tooltip" id={id} className="pr-tip">
          {t(m.automatedWhy, { gateways: p.automated.gateways, prefetch: p.automated.prefetch })}
        </span>
      ) : null}
    </span>
  )
}

function slugOf(s: string) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'poll'
}

function Actions({ p, onChanged }: { p: PollResults; onChanged: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState<'close' | 'reopen' | 'results' | 'answers'>()
  const [confirm, setConfirm] = useState(false)
  const path = { workspace: slug, pollId: p.id }
  const setOpen = async (open: boolean) => {
    setBusy(open ? 'reopen' : 'close')
    try {
      if (open) await http.POST('/api/v1/workspaces/{workspace}/polls/{pollId}/reopen', { params: { path } })
      else await http.POST('/api/v1/workspaces/{workspace}/polls/{pollId}/close', { params: { path } })
      toast({ kind: 'success', title: t(open ? m.reopened : m.closed) })
      onChanged()
    } catch (e) {
      toast({ kind: 'error', title: t(m.failed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
      setConfirm(false)
    }
  }
  const download = async (what: 'results' | 'answers') => {
    setBusy(what)
    try {
      const { data } =
        what === 'results'
          ? await http.GET('/api/v1/workspaces/{workspace}/polls/{pollId}/results.csv', { params: { path }, parseAs: 'blob' })
          : await http.GET('/api/v1/workspaces/{workspace}/polls/{pollId}/answers.csv', { params: { path }, parseAs: 'blob' })
      if (data) saveBlob(data as Blob, `${slugOf(p.campaign)}-poll-${what}.csv`)
    } catch (e) {
      toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  const closeReason = p.canClose ? undefined : deniedReason('campaigns:edit', p.closeHelpers, t(m.closeAction))
  const exportReason = p.canExportAnswers ? undefined : deniedReason('subscribers:export', p.exportHelpers, t(m.exportAnswersAction))
  return (
    <div className="pr-acts rep-noprint">
      {p.status === 'open' ? (
        <Button size="sm" disabledReason={closeReason} loading={busy === 'close'} onClick={() => setConfirm(true)}>
          {t(m.closeNow)}
        </Button>
      ) : (
        <Button size="sm" disabledReason={closeReason} loading={busy === 'reopen'} onClick={() => void setOpen(true)}>
          {t(m.reopen)}
        </Button>
      )}
      <Button size="sm" loading={busy === 'results'} onClick={() => void download('results')}>
        {t(m.exportResults)}
      </Button>
      <Button size="sm" disabledReason={exportReason} loading={busy === 'answers'} onClick={() => void download('answers')}>
        {t(m.exportAnswers)}
      </Button>
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        tone="warn"
        badge={t(m.closeBadge)}
        confirmVariant="primary"
        title={t(m.closeTitle)}
        description={t(m.closeBody)}
        confirmLabel={t(m.closeConfirm)}
        onConfirm={() => setOpen(false)}
      />
    </div>
  )
}

const GROUP_COLOR = { promoter: 'var(--chart-tier-1)', passive: 'var(--chart-tier-2)', detractor: 'var(--chart-tier-3)' } as const

function groupOf(score: number): keyof typeof GROUP_COLOR {
  return score >= 9 ? 'promoter' : score >= 7 ? 'passive' : 'detractor'
}

/** The NPS, the bar of promoters, passives and detractors, and its legend. */
function NPS({ p }: { p: PollResults }) {
  const t = useT()
  const n = p.nps
  if (!n) return null
  const so = p.live && p.status === 'open'
  const segs = [
    { key: 'promoter' as const, v: n.promoterPercent, title: t(m.promotersTitle, { count: count(n.promoters) }) },
    { key: 'passive' as const, v: n.passivePercent, title: t(m.passivesTitle, { count: count(n.passives) }) },
    { key: 'detractor' as const, v: n.detractorPercent, title: t(m.detractorsTitle, { count: count(n.detractors) }) },
  ]
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <span className="k">{so ? t(m.npsSoFar) : t(m.nps)}</span>
      <div className="flex flex-wrap items-baseline gap-x-16 gap-y-1">
        <span className="pr-nps n">{n.score}</span>
        <span className="k">{so ? t(m.npsEarly) : t(m.npsLine, { promoters: pct(n.promoterPercent), detractors: pct(n.detractorPercent) })}</span>
      </div>
      <div
        className="pr-stack"
        role="img"
        aria-label={t(m.npsBar, { promoters: count(n.promoters), pp: pct(n.promoterPercent), passives: count(n.passives), pap: pct(n.passivePercent), detractors: count(n.detractors), dp: pct(n.detractorPercent) })}
      >
        {segs.map((s) => (s.v > 0 ? <i key={s.key} style={{ flexGrow: s.v, background: GROUP_COLOR[s.key] }} title={s.title} /> : null))}
      </div>
      <div className="pr-legend">
        <span>
          <i style={{ background: GROUP_COLOR.promoter }} />
          <T msg={m.promotersItem} values={{ count: count(n.promoters), percent: pct(n.promoterPercent), b }} />
        </span>
        <span>
          <i style={{ background: GROUP_COLOR.passive }} />
          <T msg={m.passivesItem} values={{ count: count(n.passives), percent: pct(n.passivePercent), b }} />
        </span>
        <span>
          <i style={{ background: GROUP_COLOR.detractor }} />
          <T msg={m.detractorsItem} values={{ count: count(n.detractors), percent: pct(n.detractorPercent), b }} />
        </span>
      </div>
    </div>
  )
}

/** "Answers by score": eleven bars, 0 to 10. */
function ScoreBars({ p }: { p: PollResults }) {
  const t = useT()
  const counts = Array.from({ length: 11 }, (_, i) => p.counts.find((c) => c.key === i)?.count ?? 0)
  const max = Math.max(1, ...counts)
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <span className="k">{t(m.byScore)}</span>
      <div className="pr-bars" role="img" aria-label={t(m.byScoreAria, { list: counts.map((c, i) => t(m.scoreItem, { score: i, count: count(c) })).join(', ') })}>
        {counts.map((c, i) => (
          <span key={i} className="pr-bar" title={t(m.scoreTitle, { score: i, count: c })}>
            <span className="pr-bar-col">
              <i style={{ height: `${Math.max(c ? 3 : 0, (c / max) * 100)}%`, background: GROUP_COLOR[groupOf(i)] }} />
            </span>
            <b>{i}</b>
            <span className="k">{count(c)}</span>
          </span>
        ))}
      </div>
    </div>
  )
}

/** A choice or emoji poll's answers, and the tags or field they set. */
function ChoiceResults({ p }: { p: PollResults }) {
  const t = useT()
  const tagged = p.counts.filter((c) => c.tag)
  return (
    <div className="pr-grid">
      <div className="flex min-w-0 flex-col gap-3">
        <span className="k">{t(m.answers)}</span>
        {p.counts.map((c) => (
          <div key={c.key} className="pr-choice">
            <div className="flex justify-between gap-3">
              <span>
                {c.emoji ? `${c.emoji} ` : ''}
                {c.label}
              </span>
              <span className="nowrap">
                <b>{count(c.count)}</b> <span className="k">{pct(c.percent)}</span>
              </span>
            </div>
            <span className="pr-track">
              <i style={{ width: `${c.percent}%` }} />
            </span>
          </div>
        ))}
      </div>
      {p.save === 'tag' && tagged.length ? (
        <div className="flex min-w-0 flex-col gap-3">
          <span className="k">{t(m.savedAsTags)}</span>
          <div className="flex flex-wrap gap-1.5">
            {tagged.map((c) => (
              <span key={c.key} className="pr-tag mono">
                {c.tag} · {count(c.count)}
              </span>
            ))}
          </div>
          <span className="k">{t(m.tagsNote, { tag: tagged[0]!.tag! })}</span>
        </div>
      ) : p.save === 'field' && p.field ? (
        <div className="flex min-w-0 flex-col gap-3">
          <span className="k">{t(m.savedToField)}</span>
          <span className="k">{t(m.fieldNote, { field: p.field })}</span>
        </div>
      ) : null}
    </div>
  )
}

type By = 'tiers' | 'lists' | 'languages'

/** "Breakdown" by engagement tier, list or language. */
function Breakdown({ p }: { p: PollResults }) {
  const t = useT()
  const [by, setBy] = useState<By>('tiers')
  const rows = p.breakdown[by]
  const scale = p.kind === 'scale'
  const byLabel = by === 'tiers' ? t(m.byTier) : by === 'lists' ? t(m.byList) : t(m.byLanguage)
  const col = by === 'tiers' ? t(m.colTier) : by === 'lists' ? t(m.colList) : t(m.colLanguage)
  // "By language: English NPS 27, Spanish 19, German 22; Vietnamese has 21 answers."
  const langs = p.breakdown.languages
  const scored = langs.filter((r) => !r.tooFew && r.nps !== undefined).map((r, i) => t(i ? m.langNpsShort : m.langNps, { name: r.label, nps: String(r.nps).replace('-', '−') }))
  const few = langs.filter((r) => r.tooFew || r.nps === undefined).map((r) => t(m.langFew, { name: r.label, count: r.answers }))
  const summary =
    by !== 'languages' && scale && langs.length > 1
      ? scored.length && few.length
        ? t(m.langSummaryFew, { list: scored.join(', '), few: formatList(few) })
        : t(m.langSummary, { list: scored.length ? scored.join(', ') : formatList(few) })
      : ''
  return (
    <div className="flex min-w-0 flex-col">
      <div className="pr-sub">
        <h3 className="m-0 text-[14px] font-semibold">{t(m.breakdown)}</h3>
        <div className="seg" role="group" aria-label={t(m.breakdownBy)}>
          {(['tiers', 'lists', 'languages'] as const).map((x) => (
            <button key={x} type="button" aria-pressed={by === x} onClick={() => setBy(x)}>
              {x === 'tiers' ? t(m.byTier) : x === 'lists' ? t(m.byList) : t(m.byLanguage)}
            </button>
          ))}
        </div>
      </div>
      {rows.length ? (
        <table className="tbl pr-tbl" aria-label={scale ? t(m.breakdownTable, { by: byLabel.toLowerCase() }) : t(m.answersTable, { by: byLabel.toLowerCase() })}>
          <thead>
            <tr>
              <th scope="col">{col}</th>
              <th scope="col" className="r">
                {t(m.colAnswers)}
              </th>
              <th scope="col" className="r">
                {t(m.colShare)}
              </th>
              <th scope="col" className="r">
                {scale ? t(m.colNps) : t(m.colTop)}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => {
              const top = [...r.counts].sort((a, z) => z.count - a.count)[0]
              return (
                <tr key={r.key}>
                  <td className="font-semibold">{r.label}</td>
                  <td className="r num">{count(r.answers)}</td>
                  <td className="r num">{pct(r.percent)}</td>
                  <td className={cn('r', !r.tooFew && 'num font-semibold')}>
                    {r.tooFew ? <span className="k">{t(m.tooFew)}</span> : scale ? (r.nps !== undefined ? String(r.nps).replace('-', '−') : '—') : (top?.label ?? '—')}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      ) : (
        <p className="k" style={{ padding: '0 20px' }}>
          {t(m.noBreakdown)}
        </p>
      )}
      <p className="k pr-note">
        {t(m.tooFewNote)}
        {summary ? ` ${summary}` : ''}
      </p>
    </div>
  )
}

/** Cumulative answers by hour: the first 72 hours, or the first 6 while live. */
function OverTime({ p }: { p: PollResults }) {
  const t = useT()
  const live = p.live && p.status === 'open'
  const hours = live ? 6 : 72
  // cumulative[i] counts the answers by the end of hour i; the line starts at 0.
  const pts = [0, ...p.overTime.cumulative].slice(0, hours + 1)
  const W = 520
  const H = 190
  const L = 44
  const R = 30
  const TOP = 14
  const BASE = 160
  const last = pts[pts.length - 1] ?? 0
  // Three gridlines up to the total (four while live, with room to grow).
  const parts = live ? 4 : 3
  const top = Math.max(1, live ? Math.max(last * 1.25, 4) : last)
  const step = niceStep(top / parts)
  const yMax = step * Math.max(parts, Math.ceil(top / step))
  const x = (h: number) => L + ((W - L - R) * h) / hours
  const y = (v: number) => BASE - ((BASE - TOP) * v) / yMax
  const line = pts.map((v, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)} ${y(v).toFixed(1)}`).join(' ')
  const area = pts.length ? `${line} L${x(pts.length - 1).toFixed(1)} ${BASE} L${x(0).toFixed(1)} ${BASE} Z` : ''
  const ticks = live ? [0, 1, 2, 3, 4, 5, 6] : [0, 24, 48, 72]
  const six = !live && pts.length > 6 ? pts[6]! : undefined
  const since = p.overTime.since ?? p.startedAt
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="pr-sub">
        <h3 className="m-0 text-[14px] font-semibold">{live ? t(m.soFarSix) : t(m.overTime)}</h3>
        {live ? null : <span className="k">{t(m.cumulative)}</span>}
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} className="pr-line" role="img" aria-label={t(m.overTimeAria, { count: count(last), hours: Math.max(0, pts.length - 1) })}>
        {Array.from({ length: Math.round(yMax / step) + 1 }, (_, i) => (
          <g key={i}>
            <line x1={L} x2={W - R} y1={y(step * i)} y2={y(step * i)} stroke="var(--line-2)" />
            <text x={L - 8} y={y(step * i) + 4} textAnchor="end" className="pr-ax">
              {count(step * i)}
            </text>
          </g>
        ))}
        {ticks.map((h) => (
          <text key={h} x={x(h)} y={BASE + 18} textAnchor={h === 0 ? 'start' : 'middle'} className="pr-ax">
            {h === 0 ? (since ? t(m.sentAt, { time: clock(since) }) : '0') : `+${h} h`}
          </text>
        ))}
        {area ? <path d={area} fill="var(--accent-tint)" /> : null}
        {line ? <path d={line} fill="none" stroke="var(--accent)" strokeWidth="2" /> : null}
        {six !== undefined ? (
          <g>
            <circle cx={x(6)} cy={y(six)} r="3.5" fill="var(--accent)" />
            <text x={x(6) + 8} y={y(six) + 14} className="pr-ax b">
              {t(m.bySix, { percent: `${Math.round(p.overTime.firstSixPercent)}%` })}
            </text>
          </g>
        ) : null}
        {pts.length ? (
          <g>
            <circle cx={x(pts.length - 1)} cy={y(last)} r="3.5" fill="var(--accent)" />
            <text x={x(pts.length - 1) + 7} y={y(last) + 4} className="pr-ax b">
              {count(last)}
            </text>
            {live ? (
              <text x={x(pts.length - 1) + 7} y={y(last) + 16} className="pr-ax">
                {t(m.now)}
              </text>
            ) : null}
          </g>
        ) : null}
      </svg>
      {live ? null : <span className="k">{t(m.overTimeNote, { percent: `${Math.round(p.overTime.firstSixPercent)}%`, after: p.overTime.after })}</span>}
    </div>
  )
}

function niceStep(v: number): number {
  if (v <= 0) return 1
  const e = 10 ** Math.floor(Math.log10(v))
  const f = v / e
  return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 2.5 ? 2.5 : f <= 4 ? 4 : f <= 5 ? 5 : f <= 8 ? 8 : 10) * e
}

type Group = 'all' | 'promoters' | 'passives' | 'detractors'
const PAGE = 10

/** "Comments": score, comment, contact and time, filtered and searchable. */
function Comments({ p }: { p: PollResults }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [group, setGroup] = useState<Group>('all')
  const [q, setQ] = useState('')
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)
  useEffect(() => {
    const h = window.setTimeout(() => {
      setSearch(q.trim())
      setOffset(0)
    }, 300)
    return () => window.clearTimeout(h)
  }, [q])
  const res = $api.useQuery('get', '/api/v1/workspaces/{workspace}/polls/{pollId}/comments', {
    params: { path: { workspace: slug, pollId: p.id }, query: { group, search: search || undefined, offset, limit: PAGE } },
  })
  const data = res.data
  const scale = p.kind === 'scale'
  const counts = data?.counts
  const groups: { key: Group; label: string; n?: number }[] = [
    { key: 'all', label: t(m.all), n: counts?.all },
    ...(scale
      ? [
          { key: 'promoters' as const, label: t(m.promoters), n: counts?.promoters },
          { key: 'passives' as const, label: t(m.passives), n: counts?.passives },
          { key: 'detractors' as const, label: t(m.detractors), n: counts?.detractors },
        ]
      : []),
  ]
  const items = data?.items ?? []
  return (
    <div className="flex min-w-0 flex-col">
      <div className="pr-sub">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <h3 className="m-0 text-[14px] font-semibold">{t(m.comments, { count: count(counts?.all ?? p.comments) })}</h3>
          <div className="flex flex-wrap gap-1.5" role="group" aria-label={t(m.filterComments)}>
            {groups.map((g) => (
              <button
                key={g.key}
                type="button"
                className={cn('chip', group === g.key && 'pr-on')}
                aria-pressed={group === g.key}
                onClick={() => {
                  setGroup(g.key)
                  setOffset(0)
                }}
              >
                {g.label} <span className="num">{g.n !== undefined ? count(g.n) : ''}</span>
              </button>
            ))}
          </div>
        </div>
        <input className="in pr-search" type="search" aria-label={t(m.searchComments)} placeholder={t(m.searchComments)} value={q} onChange={(e) => setQ(e.target.value)} />
      </div>
      {items.length ? (
        <table className="tbl pr-tbl pr-comments" aria-label={t(m.commentsTable)}>
          <thead>
            <tr>
              <th scope="col">{scale ? t(m.colScore) : t(m.colAnswer)}</th>
              <th scope="col">{t(m.colComment)}</th>
              <th scope="col">{t(m.colContact)}</th>
              <th scope="col">{t(m.colTime)}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <tr key={c.id}>
                <td>
                  {scale ? (
                    <span className="flex flex-col items-start gap-0.5">
                      <span className="pr-score">{c.answer}</span>
                      <span className="k">{c.group === 'promoter' ? t(m.promoter) : c.group === 'passive' ? t(m.passive) : t(m.detractor)}</span>
                    </span>
                  ) : (
                    c.label
                  )}
                </td>
                <td className="wrap">{t(m.quote, { text: c.comment })}</td>
                <td className={c.nonContact ? 'k' : 'font-semibold'}>{c.nonContact ? t(m.notAContact) : c.contact || t(m.aContact)}</td>
                <td className="nowrap">
                  {shortDate(c.at)} {clock(c.at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className="k" style={{ padding: '4px 20px 16px' }}>
          {res.isPending ? '…' : search ? t(m.noMatch, { query: search }) : t(m.noComments)}
        </p>
      )}
      {data && data.total > 0 ? (
        <div className="pr-page">
          <span className="k">{t(m.showing, { shown: count(Math.min(data.total, offset + items.length)), total: count(data.total) })}</span>
          <span className="flex gap-1.5">
            <Button size="sm" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>
              {t(m.previous)}
            </Button>
            <Button size="sm" disabled={offset + PAGE >= data.total} onClick={() => setOffset(offset + PAGE)}>
              {t(m.next)}
            </Button>
          </span>
        </div>
      ) : null}
    </div>
  )
}

/** The poll's section of the report (PollResults). */
export function PollSection({ p, n, total, onChanged }: { p: PollResults; n: number; total: number; onChanged: () => void }) {
  const t = useT()
  const live = p.live && p.status === 'open'
  const hid = useId()
  const now = useNow(60_000)
  const elapsed = p.startedAt ? duration((now.getTime() - new Date(p.startedAt).getTime()) / 60000) : ''
  const header = (
    <div className="pr-head">
      <div className="flex min-w-0 flex-col gap-1.5">
        <span className="pr-kicker">{total > 1 ? t(m.kickerOf, { n, total }) : t(m.kicker)}</span>
        <h2 className="pr-q" id={hid}>
          {p.question}
        </h2>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <Status p={p} />
          {live ? (
            <span className="pr-live">
              <span className="dot" aria-hidden="true" />
              {t(m.live)}
            </span>
          ) : null}
          <span>
            {live && p.answers ? (
              <T msg={m.answersSoFar} values={{ count: p.answers, elapsed, b }} />
            ) : (
              <T msg={p.status === 'closed' ? m.answersOfFinal : m.answersOf} values={{ count: p.answers, percent: pct(p.percent), b }} />
            )}
          </span>
          {p.automated.total ? <AutomatedChip p={p} /> : null}
        </div>
      </div>
      <Actions p={p} onChanged={onChanged} />
    </div>
  )
  if (!p.answers) {
    return (
      <section className="card pr" aria-labelledby={hid}>
        {header}
        <div className="pr-empty">
          <span className="pr-empty-ic" aria-hidden="true">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round">
              <path d="M9 6h11M9 12h11M9 18h11" />
              <circle cx="4.5" cy="6" r="1.5" />
              <circle cx="4.5" cy="12" r="1.5" />
              <circle cx="4.5" cy="18" r="1.5" />
            </svg>
          </span>
          <h3 className="m-0 text-[17px] font-semibold">{t(m.emptyTitle)}</h3>
          {p.startedAt ? <p className="m-0 max-w-[560px] text-ink-2">{t(m.emptyBody, { campaign: p.campaign, time: clock(p.startedAt), ago: ago(p.startedAt, now) })}</p> : null}
          <span className="k">{t(m.emptyPoll, { question: p.question, closes: p.closesAt ? t(m.closesOn, { date: shortDate(p.closesAt) }) : t(m.statusOpen) })}</span>
        </div>
      </section>
    )
  }
  return (
    <section className="card pr" aria-labelledby={hid}>
      {header}
      {p.status === 'closed' && p.closedAt ? (
        <div className="pr-closed">
          <div className="note n">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true" focusable="false">
              <circle cx="8" cy="8" r="6.5" />
              <path d="M8 7.2v3.6M8 5.2v.1" />
            </svg>
            <span>{t(m.closedNote, { date: shortDate(p.closedAt) })}</span>
          </div>
        </div>
      ) : null}
      {p.kind === 'scale' ? (
        <div className="pr-grid">
          <NPS p={p} />
          {live ? <OverTime p={p} /> : <ScoreBars p={p} />}
        </div>
      ) : (
        <ChoiceResults p={p} />
      )}
      {live ? null : (
        <div className="pr-grid pr-split">
          <Breakdown p={p} />
          <OverTime p={p} />
        </div>
      )}
      {p.comments || p.kind === 'scale' ? <Comments p={p} /> : null}
      <div className="pr-foot">
        <span className="k">{[t(m.footChanged, { count: p.changed }), t(m.footNonContacts, { count: p.nonContacts }), t(m.footTests)].join(' · ')}</span>
        <span className="k">{t(m.footExports)}</span>
      </div>
    </section>
  )
}

/** The poll results couldn't load: the rest of the report did. */
export function PollsError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const status = error instanceof ApiError ? error.status : undefined
  const ref = error instanceof ApiError ? error.problem.instance : undefined
  return (
    <section className="card pr-error" role="alert">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--bad-strong)" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true" focusable="false">
        <circle cx="12" cy="12" r="9.5" />
        <path d="M12 7.5v5.5M12 16.2v.1" />
      </svg>
      <h2 className="m-0 text-[17px] font-semibold">{t(m.errorTitle)}</h2>
      <p className="m-0 max-w-[520px] text-ink-2">{t(m.errorBody, { detail: status ? `${errorMessage(error)} (${status}).` : errorMessage(error) })}</p>
      <Button variant="primary" onClick={onRetry}>
        {t(m.retry)}
      </Button>
      {ref ? <span className="mono k">{t(m.request, { id: ref })}</span> : null}
    </section>
  )
}

/** Loading: the board's grey blocks. */
export function PollsLoading() {
  const t = useT()
  return (
    <section className="card pr-loading" aria-busy="true" aria-label={t(m.loading)}>
      <div className="flex gap-3">
        {[0, 1, 2].map((i) => (
          <span key={i} className="rep-sk" style={{ flex: 1, height: 88 }} />
        ))}
      </div>
      {[0, 1, 2, 3].map((i) => (
        <span key={i} className="rep-sk" style={{ height: 24 }} />
      ))}
    </section>
  )
}

/** The Poll sections, refreshed after closing or reopening. */
export function PollSections({ campaignId, polls }: { campaignId: string; polls: PollResults[] }) {
  const qc = useQueryClient()
  const { slug } = useWorkspace()
  const refresh = () => void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/polls', { params: { path: { workspace: slug, campaignId } } }] })
  return (
    <>
      {polls.map((p, i) => (
        <PollSection key={p.id} p={p} n={i + 1} total={polls.length} onChanged={refresh} />
      ))}
    </>
  )
}
