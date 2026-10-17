import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
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
const keptPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/ai-summary' as const

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
  const button = assist.can ? (
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
  const keptCard = k ? (
    <section className="card ai-kept" aria-label={t(ms.keptBy, { name: k.keptBy })}>
      <div className="ai-kept-h">
        <Sparkle />
        <span className="font-semibold">{t(ms.keptBy, { name: k.keptBy })}</span>
        <span className="k ml-auto">{t(ms.keptMeta, { date: shortDate(k.keptAt), time: clock(k.keptAt), model: k.modelName || k.model })}</span>
      </div>
      <p className="m-0 text-[14px] leading-relaxed">{k.summary}</p>
      <div className="ai-kept-f">
        {kept.data?.hostRepute ? (
          <a className="ai-kept-hr" href={HOSTREPUTE_ASSISTANT} target="_blank" rel="noreferrer">
            <Icon name="shield" size={14} />
            {t(ms.reputationAsk)}
            <span className="vh">{t(ms.newTab)}</span>
          </a>
        ) : null}
        {assist.can ? (
          <Button size="sm" className="ml-auto" loading={busy} onClick={() => void remove()}>
            {t(ms.removeSummary)}
          </Button>
        ) : null}
      </div>
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
