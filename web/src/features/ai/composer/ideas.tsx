import { Link } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { useVariantSubject } from '@/features/composer/SubjectBar'
import { versionLanguage } from '@/features/languages/composerParts'
import { estimateMicros, langName } from '../format'
import { m, ms } from '../messages'
import { AINote, tags } from '../parts'
import { Refusal, sendingText, sentText, Sparkle, useOutcome, Writing } from '../suggest'
import { useComposerAssist, type Ideas, type IdeasInput } from './state'

/** The text of the email, as the subject ideas send it (no images, no markup): for the estimate. */
function textLength(mjml: string): number {
  return mjml.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().length
}

/** What subject ideas send: the subject, the preheader, the email (the server keeps its text only). */
export function useIdeasInput(): () => IdeasInput {
  const c = useComposer()
  const [subject] = useVariantSubject()
  return () => ({ campaignId: c.campaign.id, subject, preheader: c.preheader, mjml: c.currentMjml(), language: versionLanguage(c.campaign) })
}

/**
 * "Ideas" beside the subject (AIComposer): opens the subject ideas, or the
 * invitation to connect a model (AIStates · not configured).
 */
export function IdeasButton() {
  const ca = useComposerAssist()
  const c = useComposer()
  const t = useT()
  const input = useIdeasInput()
  if (!ca || !ca.assist.can || !c.canEdit || c.kind === 'transactional') return null
  const a = ca.assist
  const off = a.configured && !a.featureOn('subject_ideas')
  const button = (
    <Button
      className="ai-assist ai-ideas-btn"
      icon={undefined}
      aria-label={t(ms.ideasLabel)}
      disabledReason={off ? t(ms.featureOff) : undefined}
      onClick={a.configured ? () => ca.openIdeas(input()) : undefined}
    >
      <Sparkle />
      {t(ms.ideas)}
    </Button>
  )
  if (a.configured || a.loading) return button
  // Not configured: the note under the subject, as a popover from the button.
  const helpers = a.status?.helpers ?? []
  return (
    <Popover trigger={button} label={t(ms.ideasLabel)} className="ai-pop" align="end">
      <AINote tone="info">
        {a.status?.canManage ? (
          <T
            msg={ms.ideasNeed}
            values={{
              ...tags,
              a: (x) => (
                <Link to="/w/$workspace/settings/integrations/ai" params={{ workspace: c.slug }} className="font-semibold">
                  {x}
                </Link>
              ),
            }}
          />
        ) : helpers.length ? (
          <T msg={ms.ideasNeedAsk} values={{ ...tags, helpers: formatList(helpers, 'disjunction') }} />
        ) : (
          <T msg={ms.ideasNeedNobody} values={tags} />
        )}
      </AINote>
    </Popover>
  )
}

/** Subject ideas in the right column (AIComposer): five subjects with their checks, Use this. */
export function IdeasPanel() {
  const ca = useComposerAssist()!
  const c = useComposer()
  const t = useT()
  const outcome = useOutcome()
  const [subject, setSubject] = useVariantSubject()
  const s = ca.ideas.state
  const data: Ideas | undefined = s.phase === 'done' ? s.data : undefined
  // How many ideas of this answer were used ("Accepted 1 of 5" in the activity log).
  const used = useRef<{ id?: string; n: number }>({ n: 0 })
  // Using an idea is a content edit: it is saved as its own version, named after the idea.
  const pending = useRef<{ subject: string; label: string } | null>(null)
  useEffect(() => {
    const p = pending.current
    if (p && subject === p.subject) {
      pending.current = null
      void c.save({ label: p.label })
    }
  }, [subject, c])
  const close = () => {
    if (data?.requestId && (used.current.id !== data.requestId || !used.current.n)) outcome(data.requestId, 'rejected')
    ca.closePanel()
  }
  const use = (idea: string) => {
    const before = subject
    if (used.current.id !== data?.requestId) used.current = { id: data?.requestId, n: 0 }
    used.current.n += 1
    pending.current = { subject: idea, label: t(ms.subjectVersion, { subject: idea }) }
    setSubject(idea)
    if (data?.requestId) outcome(data.requestId, 'accepted', `${used.current.n} of ${data.ideas.length}`)
    toast({ kind: 'success', title: t(ms.subjectUpdated), detail: t(ms.subjectUpdatedDetail), undo: () => setSubject(before) })
  }
  const panel = ca.panel?.kind === 'ideas' ? ca.panel : null
  const language = langName(data?.language ?? versionLanguage(c.campaign) ?? c.campaign.sourceLanguage ?? 'en')
  const chars = panel ? textLength(panel.input.mjml) : 0
  const prices = ca.assist.status?.prices ?? { input: 0, output: 0 }
  const sentLine =
    s.phase === 'done' && data
      ? sentText(t, ms.sentSubjectIdeas, data.sent, { count: data.sent.characters, language })
      : sendingText(t, ms.sentSubjectIdeas, ca.assist.status, estimateMicros(chars, 300, prices), { count: chars, language })
  return (
    <aside className="cpanel right ai-panel" aria-label={t(ms.ideasTitle)}>
      <div className="ai-panel-h">
        <Sparkle />
        <span className="ai-sug-t">{t(ms.ideasTitle)}</span>
        {ca.assist.model ? <span className="ai-sug-m">{t(m.cardModel, { model: ca.assist.model })}</span> : null}
        <button type="button" className="ai-sug-x" aria-label={t(ms.closeSuggestion)} onClick={close}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
            <path d="M4 4l8 8M12 4l-8 8" />
          </svg>
        </button>
      </div>
      <div className="ai-now">
        <span className="k">{subject.trim() ? t(ms.ideasNow, { count: [...subject].length }) : t(ms.ideasNowEmpty)}</span>
        {subject.trim() ? <b>{subject}</b> : null}
      </div>
      <div className="ai-ideas">
        {s.phase === 'working' ? (
          <Writing />
        ) : s.phase === 'error' ? (
          <AINote tone={s.error.kind === 'rate' ? 'warn' : 'bad'}>{s.error.message}</AINote>
        ) : data?.refusal ? (
          <Refusal model={ca.assist.model} text={data.refusal} />
        ) : data && !data.ideas.length ? (
          <span className="k">{t(ms.ideasNone)}</span>
        ) : data ? (
          data.ideas.map((idea, i) => (
            <div key={`${i}-${idea.subject}`} className={`ai-idea${idea.verdict === 'warn' ? ' ai-warn-card' : ''}`}>
              <b>{idea.subject}</b>
              <div className="ai-idea-r">
                {idea.verdict === 'warn' ? (
                  <>
                    <Badge tone="warn">{t(ms.warn)}</Badge>
                    <span className="ai-phrase">{idea.phrase ? t(ms.trips, { phrase: idea.phrase }) : t(ms.tripsHeuristic, { heuristic: idea.heuristics[0] ?? '' })}</span>
                  </>
                ) : (
                  <>
                    <Badge tone="ok">{t(ms.pass)}</Badge>
                    <span className="k">{t(ms.characters, { count: idea.characters })}</span>
                  </>
                )}
                <Button size="sm" onClick={() => use(idea.subject)} aria-label={t(ms.useThisLabel, { subject: idea.subject })} disabledReason={subject === idea.subject ? t(ms.subjectUpdated) : undefined}>
                  {t(ms.useThis)}
                </Button>
              </div>
            </div>
          ))
        ) : null}
      </div>
      <div className="ai-panel-a">
        {s.phase === 'working' ? (
          <Button size="sm" onClick={close}>
            {t(m.cancel)}
          </Button>
        ) : (
          <>
            <Button size="sm" icon="refresh" onClick={() => ca.openIdeas({ ...(panel?.input ?? { campaignId: c.campaign.id, subject, mjml: c.currentMjml() }), subject })}>
              {t(m.tryAgain)}
            </Button>
            <Button size="sm" onClick={close}>
              {t(m.close)}
            </Button>
          </>
        )}
      </div>
      {ca.assist.status ? <div className="ai-panel-f">{sentLine}</div> : null}
    </aside>
  )
}
