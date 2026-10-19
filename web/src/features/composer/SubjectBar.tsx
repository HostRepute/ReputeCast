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

/** Subject and preheader with the inbox preview (Gmail on desktop, iPhone Mail on mobile). */
export function SubjectBar() {
  const c = useComposer()
  const t = useT()
  const [subject, setSubject] = useVariantSubject()
  const now = useNow(60_000)
  const digest = useDigestMode()
  // A digest's template reads as its next issue: the subject with that issue's posts, sent at its time.
  const shown = digest?.render ? { subject: digest.render.subject, preheader: digest.render.preheader } : { subject, preheader: c.preheader }
  const time = inboxTime(digest?.render?.sendAt ?? digest?.digest?.next?.sendAt ?? digest?.digest?.nextSend ?? c.campaign.scheduledAt ?? now)
  const sender = c.campaign.sender?.name ?? c.context.senders[0]?.name ?? 'Your name'
  const heuristics = useMemo(() => c.checks?.subjectHeuristics ?? [], [c.checks])
  const spam = heuristics.length > 0 && subject.trim() !== ''
  const empty = !subject.trim() && !c.preheader.trim()
  const mobile = c.device === 'mobile'
  const language = useVersionName()
  const subjectLabel = language
    ? t(mz.subjectIn, { language })
    : digest && !spam
      ? t(dc.subjectLabel)
      : spam
    ? `Subject · ${heuristics.length} spam ${heuristics.length === 1 ? 'heuristic' : 'heuristics'}: ${heuristics.join(', ')}`
    : c.abTest && c.abTest.kind === 'subject'
      ? `Subject · Variant ${c.variant}`
      : 'Subject'
  return (
    <div className="cmp-subj">
      <label style={{ flex: digest ? '3 1 380px' : '2 1 320px' }}>
        <span className="k flex items-center justify-between gap-2" style={spam ? { color: 'var(--warn-fg)' } : undefined}>
          {subjectLabel}
          {digest && c.canEdit ? <FeedTokensMenu onInsert={(token) => setSubject(`${subject}${subject && !subject.endsWith(' ') ? ' ' : ''}${token}`)} /> : null}
        </span>
        <span className="flex gap-2">
          <input
            className={digest ? 'in mono min-w-0 flex-1' : 'in min-w-0 flex-1'}
            value={subject}
            placeholder="What is this email about?"
            onChange={(e) => setSubject(e.target.value.replace(/[\r\n]+/g, ' '))}
            readOnly={!c.canEdit}
            maxLength={998}
            aria-invalid={spam || undefined}
            style={spam ? { borderColor: '#F59E0B', background: '#FFFBEB' } : digest ? { fontSize: 12 } : undefined}
          />
          {digest ? null : <IdeasButton />}
        </span>
      </label>
      <label style={{ flex: digest ? '2 1 260px' : '2 1 320px' }}>
        <span className="k">{language ? t(mz.preheaderIn, { language }) : 'Preheader'}</span>
        <input
          className={digest ? 'in mono' : 'in'}
          style={digest ? { fontSize: 12 } : undefined}
          value={c.preheader}
          placeholder="Shown after the subject in most inboxes"
          onChange={(e) => c.setPreheader(e.target.value.replace(/[\r\n]+/g, ' '))}
          readOnly={!c.canEdit}
          maxLength={500}
        />
      </label>
      <div style={{ flex: digest ? '3 1 340px' : '3 1 360px' }}>
        <span className="k" id="inbox-label">
          {language
            ? t(mz.inboxIn, { client: mobile ? 'iPhone Mail' : 'Gmail', language })
            : digest?.render
              ? // The inbox preview follows "Preview with".
                digest.preview === 'next'
                ? t(dc.inboxNext, { number: digest.render.number, client: mobile ? 'iPhone Mail' : 'Gmail' })
                : digest.preview === 'last'
                  ? t(dc.inboxLast, { number: digest.render.number, client: mobile ? 'iPhone Mail' : 'Gmail' })
                  : t(dc.inboxItems, { count: digest.render.count, client: mobile ? 'iPhone Mail' : 'Gmail' })
              : `Inbox preview · ${mobile ? 'iPhone Mail' : 'Gmail'}`}
        </span>
        {empty ? (
          <div className="cmp-inbox empty" aria-labelledby="inbox-label">
            <span style={{ fontWeight: 700, width: 120 }}>{sender}</span>
            <span>Subject and preheader appear here as you type</span>
            <span className="k" style={{ marginLeft: 'auto' }}>
              {time}
            </span>
          </div>
        ) : mobile ? (
          <div className="cmp-inbox" style={{ flexDirection: 'column', alignItems: 'stretch', justifyContent: 'center', gap: 0, fontSize: 12, lineHeight: 1.2 }} aria-labelledby="inbox-label">
            <span style={{ fontWeight: 700 }}>
              {sender}{' '}
              <span className="k" style={{ float: 'right' }}>
                {time}
              </span>
            </span>
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {shown.subject}
              {shown.preheader ? ` · ${shown.preheader}` : ''}
            </span>
          </div>
        ) : (
          <div className="cmp-inbox" aria-labelledby="inbox-label">
            <span style={{ fontWeight: 700, width: digest ? 96 : sender.length > 6 ? 120 : 60, overflow: 'hidden', textOverflow: 'ellipsis', flex: digest ? 'none' : undefined }}>{sender}</span>
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', minWidth: digest ? 0 : undefined }}>
              <b style={{ fontWeight: 600 }}>{shown.subject}</b>
              {shown.preheader ? <span style={{ color: 'var(--ink-3)' }}> - {shown.preheader}</span> : null}
            </span>
            <span className="k" style={{ marginLeft: 'auto' }}>
              {time}
            </span>
          </div>
        )}
      </div>
      {digest ? <SubjectPreviews subject={subject} onSubject={setSubject} /> : null}
    </div>
  )
}
