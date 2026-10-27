import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type FormEvent, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { aiBase } from './api'
import { estimateMicros } from './format'
import { m, ms } from './messages'
import { ConnectButton, HOSTREPUTE_ASSISTANT, Refusal, sendingText, sentText, Sparkle, SuggestFlow, useAssist, useOutcome, useSuggest } from './suggest'
import './ai.css'

type Summary = Schemas['AIReportSummary']
type Answer = Schemas['AIReportAnswer']
const keptPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/ai-summary' as const

/** The green tick of "Every number matches this report" (Report board). */
function Tick() {
  return (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="flex-none">
      <path d="m3 8.5 3.2 3L13 4.5" />
    </svg>
  )
}

/** Whether the numbers in a text are the report's: "Every number matches this report", or which aren't. */
function NumbersCheck({ ok, unmatched }: { ok: boolean; unmatched: string[] }) {
  const t = useT()
  return ok ? (
    <span className="ai-numcheck ok">
      <Tick />
      {t(ms.numbersMatchReport)}
    </span>
  ) : (
    <span className="ai-numcheck bad">{t(ms.numbersOffReport, { numbers: unmatched.join(', ') })}</span>
  )
}

/**
 * "Ask about this report" (review AN-14): a question answered by the
 * workspace's own model from the report's aggregate numbers, each number in
 * the answer checked against the report.
 */
function AskBox({ campaignId, model }: { campaignId: string; model: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const [question, setQuestion] = useState('')
  const [answer, setAnswer] = useState<Answer>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const q = question.trim()
    if (!q || busy) return
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST(`${aiBase}/report-question`, { params: { path: { workspace: slug } }, body: { campaignId, question: q } })
      setAnswer(data)
    } catch (err) {
      setAnswer(undefined)
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  return (
    <form className="ai-ask" onSubmit={(e) => void submit(e)}>
      <label htmlFor={`${id}-q`} className="text-[13px] font-semibold">
        {t(ms.askTitle)}
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <input
          id={`${id}-q`}
          className="in min-w-0 flex-[1_1_320px]"
          type="text"
          maxLength={500}
          value={question}
          placeholder={t(ms.askPlaceholder)}
          aria-describedby={`${id}-note`}
          onChange={(e) => setQuestion(e.target.value)}
        />
        <Button type="submit" loading={busy} disabledReason={question.trim() ? undefined : t(ms.askEmpty)}>
          {t(ms.ask)}
        </Button>
      </div>
      <span id={`${id}-note`} className="k">
        {t(ms.askNote, { model })}
      </span>
      <div aria-live="polite" className="contents">
        {answer ? (
          answer.refusal ? (
            <Refusal model={model} text={answer.refusal} />
          ) : (
            <div className="ai-answer">
              <p className="m-0 text-[14px] leading-relaxed">{answer.answer}</p>
              <NumbersCheck ok={answer.numbersMatch} unmatched={answer.unmatched} />
            </div>
          )
        ) : null}
        {error ? <span className="ai-numcheck bad">{error}</span> : null}
      </div>
    </form>
  )
}

/**
 * Summarize results on a campaign's report (AISegmentReport): the button
 * for people who hold ai:use, the suggestion card, and the summary someone
 * kept at the top of the report. Read-only report links never show it.
 */
export function useReportAI(campaignId: string, sent: boolean): { button: ReactNode; panel: ReactNode } {
  const { slug } = useWorkspace()
  const t = useT()
  const assist = useAssist()
  const qc = useQueryClient()
  const outcome = useOutcome()
  const path = { workspace: slug, campaignId }
  const kept = $api.useQuery('get', keptPath, { params: { path } }, { enabled: sent })
  const s = useSuggest<{ campaignId: string }, Summary>(async (body, signal) => (await http.POST(`${aiBase}/report-summary`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  if (!sent) return { button: null, panel: null }
  const off = assist.configured && !assist.featureOn('report_summary')
  const summarize = () => {
    setOpen(true)
    setEditing(null)
    if (assist.configured) void s.start({ campaignId })
  }
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', keptPath] })
  const close = (decided?: 'discarded') => {
    if (decided && s.state.phase === 'done') outcome(s.state.data.requestId, decided)
    s.cancel()
    setOpen(false)
    setEditing(null)
  }
  const keep = async (r: Summary, text: string, edited: boolean) => {
    if (!r.requestId) return
    setBusy(true)
    try {
      await http.PUT(keptPath, { params: { path }, body: { requestId: r.requestId, summary: text.trim(), edited } })
      await refresh()
      setOpen(false)
      setEditing(null)
      toast({ kind: 'success', title: t(ms.summaryKept), detail: t(ms.summaryKeptDetail) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    setBusy(true)
    try {
      await http.DELETE(keptPath, { params: { path } })
      await refresh()
      toast({ kind: 'deleted', title: t(ms.summaryRemoved) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  // With a kept summary the header has no button (the Report boards):
  // removing the summary brings it back.
  const button = assist.can && !kept.data?.kept ? (
    <Button className="ai-assist" onClick={summarize} disabledReason={off ? t(ms.summaryOff) : s.state.phase === 'working' ? t(m.writing) : undefined}>
      <Sparkle />
      {t(ms.summarize)}
    </Button>
  ) : null
  const prices = assist.status?.prices ?? { input: 0, output: 0 }
  const card = !open ? null : !assist.configured ? (
    <section className="card ai-describe ai-describe-invite" aria-label={t(ms.summarize)}>
      <h2 className="h2 m-0 flex items-center gap-2">
        <Sparkle size={16} />
        {t(ms.summarize)}
      </h2>
      <p className="m-0 text-[13px] leading-normal text-ink-2">{t(ms.summarizeInvite)}</p>
      <div>
        <ConnectButton assist={assist} />
      </div>
    </section>
  ) : (
    <SuggestFlow
      title={t(ms.summaryTitle)}
      assist={assist}
      s={s.state}
      sending={sendingText(t, ms.sentSummary, assist.status, estimateMicros(2400, 220, prices))}
      onClose={() => close(s.state.phase === 'done' ? 'discarded' : undefined)}
      onRetry={s.retry}
      done={(r) => {
        if (r.refusal) {
          return {
            body: <Refusal model={assist.model} text={r.refusal} />,
            actions: (
              <Button size="sm" onClick={() => close()}>
                {t(m.close)}
              </Button>
            ),
          }
        }
        if (editing !== null) {
          return {
            body: (
              <label className="lbl">
                <span className="vh">{t(ms.summaryLabel)}</span>
                <textarea className="in ai-describe-in" rows={5} value={editing} onChange={(e) => setEditing(e.target.value)} autoFocus />
              </label>
            ),
            actions: (
              <>
                <Button size="sm" variant="primary" loading={busy} onClick={() => void keep(r, editing, editing.trim() !== r.summary.trim())} disabledReason={editing.trim() ? undefined : t(ms.summaryLabel)}>
                  {t(ms.keep)}
                </Button>
                <Button size="sm" onClick={() => setEditing(null)}>
                  {t(m.cancel)}
                </Button>
              </>
            ),
            sent: sentText(t, ms.sentSummary, r.sent),
          }
        }
        return {
          body: <p className="m-0 text-[14px] leading-relaxed">{r.summary}</p>,
          checks: [
            { key: 'numbers', ok: r.numbersMatch, text: r.numbersMatch ? t(ms.numbersMatch) : t(ms.numbersOff, { numbers: r.unmatched.join(', ') }) },
            { key: 'causes', ok: r.noCauses, text: r.noCauses ? t(ms.noCauses) : t(ms.causes) },
          ],
          actions: (
            <>
              <Button size="sm" variant="primary" loading={busy} onClick={() => void keep(r, r.summary, false)}>
                {t(ms.keep)}
              </Button>
              <Button size="sm" onClick={() => setEditing(r.summary)}>
                {t(m.editSuggestion)}
              </Button>
              <Button size="sm" onClick={() => close('discarded')}>
                {t(ms.discard)}
              </Button>
              <Button size="sm" icon="refresh" onClick={s.retry}>
                {t(m.tryAgain)}
              </Button>
            </>
          ),
          sent: sentText(t, ms.sentSummary, r.sent),
        }
      }}
    />
  )
  const k = kept.data?.kept
  // Ask about this report: the workspace's model, for people who may use it.
  const canAsk = assist.can && assist.configured && !off
  const model = assist.model
  const keptCard = k ? (
    <section className="card ai-kept" aria-labelledby="ai-kept-h">
      <div className="ai-kept-h">
        <Sparkle />
        <h2 className="h2 m-0" id="ai-kept-h">
          {t(ms.keptBy, { name: k.keptBy })}
        </h2>
        <div className="ml-auto flex flex-wrap items-center gap-2.5">
          <span className="k">{t(ms.keptMeta, { date: shortDate(k.keptAt), time: clock(k.keptAt), model: k.modelName || k.model })}</span>
          {assist.can ? (
            <Button size="sm" loading={busy} onClick={() => void remove()}>
              {t(ms.removeSummary)}
            </Button>
          ) : null}
        </div>
      </div>
      <p className="m-0 max-w-[980px] text-[14px] leading-relaxed">{k.summary}</p>
      <NumbersCheck ok={k.numbersMatch} unmatched={k.unmatched} />
      {kept.data?.hostRepute ? (
        <div className="ai-kept-f">
          <a className="ai-kept-hr" href={HOSTREPUTE_ASSISTANT} target="_blank" rel="noreferrer">
            <Icon name="shield" size={14} />
            {t(ms.reputationAsk)}
            <span className="vh">{t(ms.newTab)}</span>
          </a>
        </div>
      ) : null}
      {canAsk ? <AskBox campaignId={campaignId} model={model} /> : null}
    </section>
  ) : canAsk && !card ? (
    <section className="card ai-kept" aria-label={t(ms.askTitle)}>
      <AskBox campaignId={campaignId} model={model} />
    </section>
  ) : null
  return {
    button,
    panel:
      card || keptCard ? (
        <div className="flex flex-col gap-4">
          {card}
          {keptCard}
        </div>
      ) : null,
  }
}
