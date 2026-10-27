import { useMemo } from 'react'
import { useNow } from '@/lib/hooks'
import { useComposer } from './context'
import { inboxTime } from './format'
import { useT } from '@/i18n'
import { useVersionName } from '@/features/languages/composerParts'
import { mz } from '@/features/languages/messages'
import { FeedTokensMenu, SubjectPreviews, useDigestMode } from '@/features/digests/composer/DigestMode'
import { dc } from '@/features/digests/composer/messages'
import { IdeasButton } from '@/features/ai/composer/ideas'
import { m } from './messages'
import { mh } from './chrome.messages'

/** Subject of the variant being edited (A is the campaign itself). */
export function useVariantSubject(): [string, (v: string) => void] {
  const c = useComposer()
  if (c.variant === 'A' || !c.abTest) return [c.subject, c.setSubject]
  const v = c.abTest.variants.find((x) => x.key === c.variant)
  return [
    v?.subject ?? '',
    (s: string) => {
      if (!c.abTest) return
      c.setABTest({ ...c.abTest, variants: c.abTest.variants.map((x) => (x.key === c.variant ? { ...x, subject: s } : x)) })
    },
  ]
}

/**
 * Subject and preheader with their lengths, variant B of a subject test,
 * and the inbox preview (Gmail on desktop, iPhone Mail on mobile): the
 * Inbox strip's panel (Composer board).
 */
export function SubjectBar() {
  const c = useComposer()
  const t = useT()
  const [subject, setSubject] = useVariantSubject()
  const now = useNow(60_000)
  const digest = useDigestMode()
  // A digest's template reads as its next issue: the subject with that issue's posts, sent at its time.
  const shown = digest?.render ? { subject: digest.render.subject, preheader: digest.render.preheader } : { subject, preheader: c.preheader }
  const time = inboxTime(digest?.render?.sendAt ?? digest?.digest?.next?.sendAt ?? digest?.digest?.nextSend ?? c.campaign.scheduledAt ?? now)
  const sender = c.campaign.sender?.name ?? c.context.senders[0]?.name ?? t(m.senderFallback)
  const heuristics = useMemo(() => c.checks?.subjectHeuristics ?? [], [c.checks])
  const spam = heuristics.length > 0 && subject.trim() !== ''
  const empty = !subject.trim() && !c.preheader.trim()
  const mobile = c.device === 'mobile'
  const language = useVersionName()
  const ab = c.abTest && c.abTest.kind === 'subject' && !digest && !language ? c.abTest : null
  const variantB = ab?.variants.find((v) => v.key === 'B')
  const subjectLabel = language
    ? t(mz.subjectIn, { language })
    : digest && !spam
      ? t(dc.subjectLabel)
      : spam
        ? t(m.subjectSpam, {
            count: heuristics.length,
            // The checks name each heuristic in English ("all caps"); the label names it in the app's language.
            list: heuristics.map((h) => t(m.subjectHeuristic, { h: h.replace(/[^a-z]+/gi, '_').toLowerCase(), name: h })).join(', '),
          })
        : ab
          ? t(m.subjectVariant, { variant: 'A' })
          : t(m.subject)
  const editingA = !ab || c.variant === 'A'
  const aValue = ab ? c.subject : subject
  const setA = ab ? c.setSubject : setSubject
  const setB = (v: string) => ab && c.setABTest({ ...ab, variants: ab.variants.map((x) => (x.key === 'B' ? { ...x, subject: v } : x)) })
  return (
    <div className="ip-in">
      <div className="ip-row">
        <label className="lbl">
          <span className="flex items-center justify-between gap-2" style={spam && editingA ? { color: 'var(--warn-fg)' } : undefined}>
            {subjectLabel}
            {digest && c.canEdit ? <FeedTokensMenu onInsert={(token) => setSubject(`${subject}${subject && !subject.endsWith(' ') ? ' ' : ''}${token}`)} /> : null}
          </span>
          <span className="flex gap-2">
            <input
              className={digest ? 'in mono min-w-0 flex-1' : 'in min-w-0 flex-1'}
              value={aValue}
              placeholder={t(m.subjectPlaceholder)}
              onChange={(e) => setA(e.target.value.replace(/[\r\n]+/g, ' '))}
              readOnly={!c.canEdit}
              maxLength={998}
              aria-invalid={(spam && editingA) || undefined}
              style={spam && editingA ? { borderColor: '#F59E0B', background: '#FFFBEB' } : digest ? { fontSize: 12 } : undefined}
            />
            {digest ? null : <IdeasButton />}
          </span>
          {digest ? null : <span className="k">{t(mh.subjectCount, { count: [...aValue].length })}</span>}
        </label>
        {ab ? (
          <label className="lbl">
            {t(m.subjectVariant, { variant: 'B' })}
            <input className="in" value={variantB?.subject ?? ''} placeholder={t(m.subjectPlaceholder)} onChange={(e) => setB(e.target.value.replace(/[\r\n]+/g, ' '))} readOnly={!c.canEdit} maxLength={998} />
            <span className="k">
              {t(mh.abLine, { pct: ab.testPercent, metric: ab.metric === 'human_opens' ? 'opens' : ab.metric, hours: ab.durationHours })} ·{' '}
              <button type="button" className="linkish" onClick={() => c.setView('preview')}>
                {t(mh.testSettings)}
              </button>
            </span>
          </label>
        ) : null}
      </div>
      <label className="lbl">
        {language ? t(mz.preheaderIn, { language }) : t(m.preheader)}
        <input
          className={digest ? 'in mono' : 'in'}
          style={digest ? { fontSize: 12 } : undefined}
          value={c.preheader}
          placeholder={t(m.preheaderPlaceholder)}
          onChange={(e) => c.setPreheader(e.target.value.replace(/[\r\n]+/g, ' '))}
          readOnly={!c.canEdit}
          maxLength={500}
        />
        {digest ? null : <span className="k">{t(mh.preheaderCount, { count: [...c.preheader].length })}</span>}
      </label>
      <div className="lbl">
        <span id="inbox-label">
          {language
            ? t(mz.inboxIn, { client: mobile ? 'iPhone Mail' : 'Gmail', language })
            : digest?.render
              ? // The inbox preview follows "Preview with".
                digest.preview === 'next'
                ? t(dc.inboxNext, { number: digest.render.number, client: mobile ? 'iPhone Mail' : 'Gmail' })
                : digest.preview === 'last'
                  ? t(dc.inboxLast, { number: digest.render.number, client: mobile ? 'iPhone Mail' : 'Gmail' })
                  : t(dc.inboxItems, { count: digest.render.count, client: mobile ? 'iPhone Mail' : 'Gmail' })
              : t(mh.inboxPreviewOn, { mobile: mobile ? 'yes' : 'no' })}
        </span>
        {empty ? (
          <div className="gmail empty" aria-labelledby="inbox-label">
            <span className="gf">{sender}</span>
            <span className="gs">{t(m.inboxEmpty)}</span>
            <span className="gt">{time}</span>
          </div>
        ) : mobile ? (
          <div className="gmail phone" aria-labelledby="inbox-label">
            <span className="gf">{sender}</span>
            <span className="gt">{time}</span>
            <span className="gs">
              <b>{shown.subject}</b>
              {shown.preheader ? <span> · {shown.preheader}</span> : null}
            </span>
          </div>
        ) : (
          <div className="gmail" aria-labelledby="inbox-label">
            <span className="gf">{sender}</span>
            <span className="gs">
              <b>{shown.subject}</b>
              {shown.preheader ? <span> - {shown.preheader}</span> : null}
            </span>
            <span className="gt">{time}</span>
          </div>
        )}
      </div>
      {digest ? <SubjectPreviews subject={subject} onSubject={setSubject} /> : null}
    </div>
  )
}
