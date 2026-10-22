import { useRef, useState, type RefObject } from 'react'
import { http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Popover } from '@/components/ui/Menu'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { aiBase } from './api'
import { m, ms } from './messages'
import { AINote } from './parts'
import { ConnectButton, Refusal, sentText, Sparkle, useAssist, useOutcome, useSuggest, type Assist } from './suggest'
import './ai.css'

export type SegmentDraft = Schemas['AISegmentDraft']
type Definition = Schemas['SegmentDefinition']

/**
 * Describe the segment (AISegmentReport): the description, Draft, and what
 * the model couldn't draft. The draft itself shows in the builder's
 * conditions, marked, until it's accepted or rejected.
 */
export function useSegmentAssist({ current, segmentId, fields, onDraft }: { current: () => Definition; segmentId?: string; fields: string[]; onDraft: (d: SegmentDraft) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const assist = useAssist()
  const outcome = useOutcome()
  const [text, setText] = useState('')
  const area = useRef<HTMLTextAreaElement>(null)
  const s = useSuggest<Schemas['AISegmentDraftInput'], SegmentDraft>(async (body, signal) => (await http.POST(`${aiBase}/segment-draft`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const run = async (description: string) => {
    if (!description.trim()) {
      area.current?.focus()
      return
    }
    const d = await s.start({ description: description.trim(), current: current(), segmentId: segmentId && segmentId !== 'new' ? segmentId : undefined })
    if (d && !d.refusal && !d.nothing && d.groups.length) onDraft(d)
  }
  const data = s.state.phase === 'done' ? s.state.data : undefined
  const off = assist.configured && !assist.featureOn('segment_draft')
  const card = assist.can ? (
    <DescribeCard
      assist={assist}
      text={text}
      setText={setText}
      area={area}
      onDraft={() => void run(text)}
      working={s.state.phase === 'working'}
      error={s.state.phase === 'error' ? s.state.error : undefined}
      nothing={data && !data.refusal && (data.nothing || !data.groups.length) ? (data.nothing ?? '') : undefined}
      refusal={data?.refusal}
      off={off}
      fields={fields}
      t={t}
    />
  ) : null
  return {
    card,
    assist,
    retry: () => void run(text),
    /** Accepted (as drafted, or edited in the builder) or rejected: the activity log's outcome. */
    decide: (d: SegmentDraft, how: 'accepted' | 'accepted_edits' | 'rejected') => outcome(d.requestId, how),
  }
}

function DescribeCard({
  assist,
  text,
  setText,
  area,
  onDraft,
  working,
  error,
  nothing,
  refusal,
  off,
  fields,
  t,
}: {
  assist: Assist
  text: string
  setText: (v: string) => void
  area: RefObject<HTMLTextAreaElement | null>
  onDraft: () => void
  working: boolean
  error?: { kind: string; message: string }
  nothing?: string
  refusal?: string
  off: boolean
  fields: string[]
  t: ReturnType<typeof useT>
}) {
  const title = (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <h2 className="h2 m-0 flex items-center gap-2">
        <Sparkle size={16} />
        {t(ms.describeTitle)}
      </h2>
      <span className="k">{t(ms.describeSub)}</span>
    </div>
  )
  if (!assist.configured && !assist.loading) {
    return (
      <section className="card ai-describe ai-describe-invite" aria-label={t(ms.describeTitle)}>
        {title}
        <p className="m-0 text-[13px] leading-normal text-ink-2">{t(ms.describeInvite)}</p>
        <div className="flex flex-wrap items-center gap-2.5">
          <ConnectButton assist={assist} />
          <span className="k">{t(m.connectWhereShort)}</span>
        </div>
      </section>
    )
  }
  return (
    <section className="card ai-describe" aria-label={t(ms.describeTitle)}>
      {title}
      {/* As on the board: the row wraps, so on phones Draft goes under the description. */}
      <div className="flex flex-wrap items-start gap-2">
        <textarea
          ref={area}
          className="in ai-describe-in basis-[420px]"
          rows={2}
          aria-label={t(ms.describeLabel)}
          placeholder={t(ms.describePlaceholder)}
          value={text}
          disabled={off}
          maxLength={500}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) onDraft()
          }}
        />
        <Button variant="primary" onClick={onDraft} loading={working} disabledReason={off ? t(ms.describeOff) : text.trim() ? undefined : t(ms.describeLabel)}>
          {t(ms.draftButton)}
        </Button>
      </div>
      {error ? <AINote tone={error.kind === 'rate' ? 'warn' : 'bad'}>{error.message}</AINote> : null}
      {refusal ? <Refusal model={assist.model} text={refusal} /> : null}
      {nothing !== undefined ? (
        <AINote tone="warn">
          <div className="flex flex-col items-start gap-2">
            <span>{nothing ? `${nothing} ${t(ms.nothingDrafted)}` : t(ms.nothingDrafted)}</span>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" onClick={() => area.current?.focus()}>
                {t(ms.changeDescription)}
              </Button>
              <Popover trigger={<Button size="sm">{t(ms.seeConditions)}</Button>} label={t(ms.conditionsYouCanUse)} className="ai-pop" align="start">
                <span className="font-semibold">{t(ms.conditionsYouCanUse)}</span>
                <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13px]">
                  {fields.map((f) => (
                    <li key={f}>{f}</li>
                  ))}
                </ul>
              </Popover>
            </div>
          </div>
        </AINote>
      ) : null}
      <span className="k leading-normal">{off ? t(ms.describeOff) : t(ms.describeSends)}</span>
    </section>
  )
}

/** The strip over a draft in the builder's conditions: Accept · Edit · Reject · Try again. */
export function DraftBanner({ draft, model, onAccept, onEdit, onReject, onRetry }: { draft: SegmentDraft; model: string; onAccept: () => void; onEdit: () => void; onReject: () => void; onRetry: () => void }) {
  const t = useT()
  return (
    <div className="ai-draft-strip" role="status">
      <Sparkle />
      <span className="font-semibold">{draft.removed ? t(ms.draftChanges, { removed: draft.removed, added: draft.added }) : t(ms.draftFrom)}</span>
      {model ? <span className="ai-sug-m">{t(m.cardModel, { model })}</span> : null}
      <span className="ml-auto flex flex-wrap gap-2">
        <Button size="sm" variant="primary" onClick={onAccept}>
          {t(m.accept)}
        </Button>
        <Button size="sm" onClick={onEdit}>
          {t(m.editSuggestion)}
        </Button>
        <Button size="sm" onClick={onReject}>
          {t(m.reject)}
        </Button>
        <Button size="sm" icon="refresh" onClick={onRetry}>
          {t(m.tryAgain)}
        </Button>
      </span>
    </div>
  )
}

/** "Added" or "Removed" on a drafted condition. */
export function DraftMark({ mark }: { mark: 'added' | 'removed' }) {
  const t = useT()
  return <span className={`ai-dmark ai-dmark-${mark}`}>{mark === 'added' ? t(ms.addedBadge) : t(ms.removedBadge)}</span>
}

/** The model's note, and what the draft sent. */
export function DraftFooter({ draft }: { draft: SegmentDraft }) {
  const t = useT()
  return (
    <>
      {draft.note ? (
        <AINote tone="info">
          <span className="font-semibold">{t(ms.modelNote)}</span>
          <div>{draft.note}</div>
        </AINote>
      ) : null}
      <span className="k leading-normal">{sentText(t, ms.sentSegment, draft.sent, { count: draft.sent.characters })}</span>
    </>
  )
}

/** Under the generated SQL while a draft is open. */
export function DraftSqlNote() {
  const t = useT()
  return <AINote tone="neutral">{t(ms.sqlAfterDraft)}</AINote>
}

export function draftCountLabels(t: ReturnType<typeof useT>, before: { count: number; empty: boolean }): { title: string; before: string } {
  return { title: t(ms.liveWithDraft), before: before.empty ? t(ms.beforeDraftNone, { count: before.count }) : t(ms.beforeDraft, { count: before.count }) }
}
