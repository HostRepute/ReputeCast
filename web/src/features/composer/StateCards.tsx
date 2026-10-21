import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { t as tt, useT } from '@/i18n'
import { andList, clock } from '@/lib/format'
import { useComposer } from './context'
import { sectionWord } from './editor/labels'
import { download, slugify } from './format'
import { m } from './messages'

/** A part of the email a save changed, as the server names it ("subject", "hero"), in the app language. */
function partName(part: string): string {
  if (part === 'subject') return tt(m.conflictSubject)
  if (part === 'the email body') return tt(m.conflictBody)
  return sectionWord(part)
}

function Card({ tone, badge, title, children, actions }: { tone: 'bad' | 'warn' | 'ok'; badge: string; title: string; children: ReactNode; actions: ReactNode }) {
  const border = tone === 'bad' ? '#FCA5A5' : tone === 'warn' ? '#FCD34D' : '#86EFAC'
  return (
    <div role={tone === 'ok' ? 'status' : 'alert'} className="card flex flex-col" style={{ padding: '16px 18px', gap: 10, borderColor: border }}>
      {/* As on ComposerStates: a long title breaks beside the badge; phones wrap the row. */}
      <div className="flex items-center gap-2.5 max-md:flex-wrap">
        <Badge tone={tone} dot>
          {badge}
        </Badge>
        <span className="font-semibold">{title}</span>
      </div>
      <span className="k">{children}</span>
      <div className="flex flex-wrap gap-2">{actions}</div>
    </div>
  )
}

/**
 * Save states shown above the canvas (ComposerStates): offline, a conflict
 * with someone else's save, a failed save, and the confirmation of a manual
 * save.
 */
export function StateCards({ saved, onCloseSaved, onReview }: { saved: { version: number; at: string; template?: string } | null; onCloseSaved: () => void; onReview: () => void }) {
  const c = useComposer()
  const t = useT()
  const s = c.status
  const cards: ReactNode[] = []
  if (s.kind === 'offline') {
    cards.push(
      <Card
        key="offline"
        tone="bad"
        badge={t(m.notSaved)}
        title={t(m.offlineTitle, { version: s.version + 1, reason: s.message })}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              {t(m.retryNow)}
            </button>
            <button type="button" className="btn sm" onClick={() => download(`${slugify(c.name)}.mjml`, c.currentMjml(), 'text/plain')}>
              {t(m.downloadBackup)}
            </button>
          </>
        }
      >
        {t(m.offlineBody, { time: clock(s.since) })}
      </Card>,
    )
  } else if (s.kind === 'conflict') {
    const who = s.theirs.by || t(m.someone)
    const first = who.split(' ')[0] ?? who
    const where = s.theirs.theirChanges.filter((x) => s.theirs.yourChanges.includes(x))
    const version = s.theirs.version + 1
    cards.push(
      <Card
        key="conflict"
        tone="warn"
        badge={t(m.conflict)}
        title={s.theirs.at ? t(m.conflictTitleAt, { who, version: s.theirs.version, time: clock(s.theirs.at) }) : t(m.conflictTitle, { who, version: s.theirs.version })}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save({ force: true })}>
              {t(m.keepMine, { version })}
            </button>
            <button type="button" className="btn sm" onClick={() => void c.reloadFromServer()}>
              {t(m.loadTheirs, { name: first, version: s.theirs.version })}
            </button>
            <button type="button" className="btn sm" onClick={() => c.openDialog('compare')}>
              {t(m.compare)}
            </button>
          </>
        }
      >
        {where.length
          ? t(m.conflictBoth, { parts: andList(where.map(partName)), version, name: first })
          : s.theirs.theirChanges.length
            ? t(m.conflictTheirs, { parts: s.theirs.theirChanges.map(partName).join(', '), version, name: first })
            : t(m.conflictPlain, { version, name: first })}
      </Card>,
    )
  } else if (s.kind === 'error') {
    cards.push(
      <Card
        key="error"
        tone="bad"
        badge={t(m.notSaved)}
        title={t(m.errorTitle, { version: s.version + 1 })}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              {t(m.retryNow)}
            </button>
            <button type="button" className="btn sm" onClick={() => download(`${slugify(c.name)}.mjml`, c.currentMjml(), 'text/plain')}>
              {t(m.downloadBackup)}
            </button>
          </>
        }
      >
        {t(m.errorBody, { message: s.message })}
      </Card>,
    )
  }
  if (saved && s.kind !== 'offline' && s.kind !== 'conflict' && s.kind !== 'error') {
    cards.push(
      <Card
        key="saved"
        tone="ok"
        badge={t(m.saved)}
        title={t(m.savedTitle, { version: saved.version, time: clock(saved.at) })}
        actions={
          <>
            <button type="button" className="btn sm" onClick={onCloseSaved}>
              {t(m.backToEditor)}
            </button>
            <button type="button" className="btn sm pri" onClick={onReview}>
              {c.campaign.access.canSend ? t(m.reviewSend) : t(m.requestApproval)}
            </button>
          </>
        }
      >
        {saved.template ? t(m.savedAsTemplate, { name: saved.template }) : t(m.savedBody)}
      </Card>,
    )
  }
  if (!cards.length) return null
  return (
    <div className="mx-auto flex w-full flex-col gap-3" style={{ maxWidth: 600, marginBottom: 20 }}>
      {cards}
    </div>
  )
}
