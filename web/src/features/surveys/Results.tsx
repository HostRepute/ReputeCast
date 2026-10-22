import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, count, pct, shortDate } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { saveBlob } from '@/features/audience/download'
import { m } from './messages'
import { typeLabel } from './types'
import '@/features/polls/report.css'

type Survey = Schemas['Survey']
type QResult = Schemas['SurveyQuestionResult']

const b = (chunks: ReactNode[]) => <b>{chunks}</b>

function Tile({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card flex flex-col gap-[3px] px-4 py-3.5">
      <span className="k">{label}</span>
      <span className="n text-[22px]">{value}</span>
      <span className="k">{sub}</span>
    </div>
  )
}

function Bars({ r }: { r: QResult }) {
  return (
    <div className="flex flex-col gap-3">
      {r.counts.map((c) => (
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
            <i style={{ width: `${Math.min(100, c.percent)}%` }} />
          </span>
        </div>
      ))}
    </div>
  )
}

function NPSResult({ r }: { r: QResult }) {
  const t = useT()
  const n = r.nps
  if (!n) return null
  const color = { p: 'var(--chart-tier-1)', pa: 'var(--chart-tier-2)', d: 'var(--chart-tier-3)' }
  return (
    <div className="flex flex-col gap-3">
      <span className="k">{t(m.nps)}</span>
      <div className="flex flex-wrap items-baseline gap-x-4">
        <span className="pr-nps n">{n.score}</span>
        <span className="k">{t(m.npsLine, { promoters: pct(n.promoterPercent), detractors: pct(n.detractorPercent) })}</span>
      </div>
      <div className="pr-stack" role="img" aria-label={t(m.npsBar, { promoters: count(n.promoters), passives: count(n.passives), detractors: count(n.detractors) })}>
        {n.promoterPercent > 0 ? <i style={{ flexGrow: n.promoterPercent, background: color.p }} /> : null}
        {n.passivePercent > 0 ? <i style={{ flexGrow: n.passivePercent, background: color.pa }} /> : null}
        {n.detractorPercent > 0 ? <i style={{ flexGrow: n.detractorPercent, background: color.d }} /> : null}
      </div>
      <div className="pr-legend pr-legend-col">
        <span>
          <i style={{ background: color.p }} />
          <T msg={m.promotersItem} values={{ count: count(n.promoters), percent: pct(n.promoterPercent), b }} />
        </span>
        <span>
          <i style={{ background: color.pa }} />
          <T msg={m.passivesItem} values={{ count: count(n.passives), percent: pct(n.passivePercent), b }} />
        </span>
        <span>
          <i style={{ background: color.d }} />
          <T msg={m.detractorsItem} values={{ count: count(n.detractors), percent: pct(n.detractorPercent), b }} />
        </span>
      </div>
    </div>
  )
}

const PAGE = 4

/** A text question's answers, newest first, with search. */
function TextAnswers({ s, r }: { s: Survey; r: QResult }) {
  const t = useT()
  const { slug } = useWorkspace()
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
  const res = $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys/{surveyId}/questions/{key}/answers', {
    params: { path: { workspace: slug, surveyId: s.id, key: r.key }, query: { search: search || undefined, offset, limit: PAGE } },
  })
  const items = res.data?.items ?? []
  const total = res.data?.total ?? 0
  return (
    <div className="flex flex-col">
      <input className="in sv-tsearch" type="search" aria-label={t(m.searchAnswers, { count: r.answered })} placeholder={t(m.searchAnswers, { count: r.answered })} value={q} onChange={(e) => setQ(e.target.value)} />
      {items.length ? (
        <ul className="sv-texts">
          {items.map((x, i) => (
            <li key={`${offset}-${i}`}>
              <span>{t(m.quote, { text: x.text })}</span>
              <span className="k">{x.contact || t(m.aContact)}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="k" style={{ padding: '10px 0' }}>
          {res.isPending ? '…' : search ? t(m.noTextMatch, { query: search }) : t(m.noTextAnswers)}
        </p>
      )}
      {total > 0 ? (
        <div className="sv-page">
          <span className="k">{t(m.showingNewest, { shown: count(Math.min(total, offset + items.length)), total: count(total) })}</span>
          {total > PAGE ? (
            <span className="flex gap-1.5">
              <Button size="sm" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>
                {t(m.previous)}
              </Button>
              <Button size="sm" disabled={offset + PAGE >= total} onClick={() => setOffset(offset + PAGE)}>
                {t(m.next)}
              </Button>
            </span>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function QuestionCard({ s, r, pollOnly }: { s: Survey; r: QResult; pollOnly: number }) {
  const t = useT()
  const text = r.type === 'short' || r.type === 'long'
  const sub = text
    ? t(m.textQuestion, { type: typeLabel(t, r.type), required: r.required ? t(m.required) : t(m.optional), count: r.answered })
    : r.savedTo
      ? t(m.savedTo, { type: typeLabel(t, r.type), field: r.savedTo })
      : r.type === 'multi'
        ? t(m.answeredResponses, { type: typeLabel(t, r.type), count: r.answered })
        : t(m.answeredCount, { type: typeLabel(t, r.type), count: r.answered })
  return (
    <section className={`card sv-rq${text ? ' wide' : ''}`} aria-labelledby={`sv-rq-${r.key}`}>
      <div className="sv-rq-h">
        <h3 className="m-0 text-[14px] font-semibold" id={`sv-rq-${r.key}`}>
          {r.number}. {r.text}
        </h3>
        <span className="k">{sub}</span>
      </div>
      <div className="sv-rq-b">{text ? <TextAnswers s={s} r={r} /> : r.type === 'scale' && r.nps ? <NPSResult r={r} /> : <Bars r={r} />}</div>
      {r.number === 1 && pollOnly > 0 ? <p className="k sv-rq-f">{t(m.pollOnlyNote, { count: pollOnly })}</p> : null}
    </section>
  )
}

/** Completed responses: contact, completed at and the answers. */
function Responses({ s }: { s: Survey }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [offset, setOffset] = useState(0)
  const [busy, setBusy] = useState<'results' | 'answers'>()
  const canExport = useCan('subscribers:export')
  const res = $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys/{surveyId}/responses', {
    params: { path: { workspace: slug, surveyId: s.id }, query: { offset, limit: 10 } },
  })
  const items = res.data?.items ?? []
  const total = res.data?.total ?? 0
  const download = async (what: 'results' | 'answers') => {
    setBusy(what)
    const path = { workspace: slug, surveyId: s.id }
    try {
      const { data } =
        what === 'results'
          ? await http.GET('/api/v1/workspaces/{workspace}/surveys/{surveyId}/results.csv', { params: { path }, parseAs: 'blob' })
          : await http.GET('/api/v1/workspaces/{workspace}/surveys/{surveyId}/answers.csv', { params: { path }, parseAs: 'blob' })
      if (data) saveBlob(data as Blob, `${s.slug || 'survey'}-${what}.csv`)
    } catch (e) {
      toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <section className="card min-w-0" aria-labelledby="sv-resp-h">
      <div className="sv-list-h">
        <h3 className="m-0 text-[15px] font-semibold" id="sv-resp-h">
          {t(m.responsesTitle, { count: count(s.completed) })}
        </h3>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" loading={busy === 'results'} onClick={() => void download('results')}>
            {t(m.exportResults)}
          </Button>
          <Button size="sm" loading={busy === 'answers'} disabledReason={canExport ? undefined : deniedReason('subscribers:export', [], t(m.exportAnswersAction))} onClick={() => void download('answers')}>
            {t(m.exportAnswers)}
          </Button>
        </div>
      </div>
      {items.length ? (
        <div className="overflow-x-auto">
          <table className="tbl sv-tbl" aria-label={t(m.responsesTitle, { count: count(s.completed) })}>
            <thead>
              <tr>
                <th scope="col">{t(m.colContact)}</th>
                <th scope="col">{t(m.colCompleted)}</th>
                <th scope="col">{t(m.colAnswers)}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((r) => (
                <tr key={r.id}>
                  <td className="font-semibold nowrap">{r.nonContact ? <span className="k">{t(m.notAContact)}</span> : r.contact || t(m.aContact)}</td>
                  <td className="nowrap">{r.completedAt ? `${shortDate(r.completedAt)} ${clock(r.completedAt)}` : '—'}</td>
                  <td className="sv-answers">{r.answers.map((a) => a || t(m.noAnswer)).join(' · ')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="k" style={{ padding: '4px 20px 18px' }}>
          {res.isPending ? '…' : t(m.noResponses)}
        </p>
      )}
      {total > 0 ? (
        <div className="sv-page sv-page-b">
          <span className="k">{t(m.showingResponses, { shown: count(Math.min(total, offset + items.length)), total: count(total) })}</span>
          <span className="flex gap-1.5">
            <Button size="sm" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - 10))}>
              {t(m.previous)}
            </Button>
            <Button size="sm" disabled={offset + 10 >= total} onClick={() => setOffset(offset + 10)}>
              {t(m.next)}
            </Button>
          </span>
        </div>
      ) : null}
    </section>
  )
}

/** The Results tab: started and completed, a chart per question, the responses and the exports. */
export function SurveyResultsTab({ s }: { s: Survey }) {
  const t = useT()
  const { slug } = useWorkspace()
  const res = $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys/{surveyId}/results', { params: { path: { workspace: slug, surveyId: s.id } } })
  if (res.isError) {
    return (
      <section className="card sv-error" role="alert">
        <h2 className="m-0 text-[17px] font-semibold">{t(m.errorTitle)}</h2>
        <p className="m-0 text-ink-2">{t(m.errorBody, { detail: errorMessage(res.error) })}</p>
        <Button variant="primary" onClick={() => void res.refetch()}>
          {t(m.retry)}
        </Button>
      </section>
    )
  }
  if (res.isPending) {
    return (
      <section className="card sv-loading" aria-busy="true" aria-label={t(m.loading)}>
        {[0, 1, 2].map((i) => (
          <span key={i} className="sk" style={{ height: 24 }} />
        ))}
      </section>
    )
  }
  const r = res.data
  const tz = s.settings.timezone || ''
  return (
    <div className="flex flex-col gap-5">
      <div className="sv-tiles">
        <Tile label={t(m.started)} value={count(r.started)} sub={r.recipients ? t(m.startedOf, { count: r.recipients }) : t(m.startedNone)} />
        <Tile label={t(m.completed)} value={count(r.completed)} sub={t(m.completedOf, { percent: pct(r.completionPercent) })} />
        <Tile label={t(m.pollOnly)} value={count(r.pollOnly)} sub={t(m.pollOnlyHint)} />
        <Tile
          label={t(m.openUntilTile)}
          value={s.closesAt ? shortDate(s.closesAt) : '—'}
          sub={s.closesAt ? t(m.openUntilTime, { time: clock(s.closesAt), tz: tz.replace(/^.*\//, '').replace(/_/g, ' ') }) : t(m.noClose)}
        />
      </div>
      <div className="sv-rqs">
        {r.questions.map((q) => (
          <QuestionCard key={q.key} s={s} r={q} pollOnly={r.pollOnly} />
        ))}
      </div>
      <Responses s={s} />
    </div>
  )
}
