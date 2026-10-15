import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { formatList, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useComposer } from '../context'
import { download, slugify } from '../format'
import { m } from './messages'
import { useTx } from './state'

export interface SavedCard {
  at: string
  /** Also saved as a reusable template or section (the composer's Save as template). */
  template?: string
}

function Card({ tone, badge, title, children, actions }: { tone: 'bad' | 'warn' | 'ok'; badge: string; title: string; children: ReactNode; actions: ReactNode }) {
  const border = tone === 'bad' ? 'var(--bad-line)' : tone === 'warn' ? '#FCD34D' : '#86EFAC'
  return (
    <div role={tone === 'ok' ? 'status' : 'alert'} className="card flex flex-col" style={{ padding: '16px 18px', gap: 10, borderColor: border }}>
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
 * Save states above the canvas, as the composer's (ComposerStates): offline,
 * a conflict with a teammate's save, a failed save, and a manual save.
 */
export function TxStateCards({ saved, onCloseSaved, onCompare }: { saved: SavedCard | null; onCloseSaved: () => void; onCompare: () => void }) {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const s = c.status
  const backup = () => download(`${slugify(c.name)}.mjml`, c.currentMjml() || tx.html, 'text/plain')
  const cards: ReactNode[] = []
  if (s.kind === 'offline') {
    cards.push(
      <Card
        key="offline"
        tone="bad"
        badge={t(m.notSaved)}
        title={t(m.offlineTitle)}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              {t(m.retryNow)}
            </button>
            <button type="button" className="btn sm" onClick={backup}>
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
    const both = s.theirs.theirChanges.filter((x) => s.theirs.yourChanges.includes(x))
    cards.push(
      <Card
        key="conflict"
        tone="warn"
        badge={t(m.conflictBadge)}
        title={s.theirs.at ? t(m.conflictTitleAt, { who, time: clock(s.theirs.at) }) : t(m.conflictTitle, { who })}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save({ force: true })}>
              {t(m.keepMine)}
            </button>
            <button type="button" className="btn sm" onClick={() => void c.reloadFromServer()}>
              {t(m.loadTheirs, { name: first })}
            </button>
            <button type="button" className="btn sm" onClick={onCompare}>
              {t(m.compare)}
            </button>
          </>
        }
      >
        {both.length
          ? t(m.conflictBoth, { parts: formatList(both), name: first })
          : s.theirs.theirChanges.length
            ? t(m.conflictTheirs, { parts: formatList(s.theirs.theirChanges), name: first })
            : t(m.conflictPlain, { name: first })}
      </Card>,
    )
  } else if (s.kind === 'error') {
    cards.push(
      <Card
        key="error"
        tone="bad"
        badge={t(m.notSaved)}
        title={t(m.errorTitle)}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              {t(m.retryNow)}
            </button>
            <button type="button" className="btn sm" onClick={backup}>
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
        badge={t(m.savedBadge)}
        title={t(m.savedTitle, { time: clock(saved.at) })}
        actions={
          <>
            <button type="button" className="btn sm" onClick={onCloseSaved}>
              {t(m.backToEditor)}
            </button>
            {tx.canPublish ? (
              <button
                type="button"
                className="btn sm pri"
                onClick={() => {
                  onCloseSaved()
                  tx.openPublish()
                }}
              >
                {t(m.publishVersion, { version: tx.template.versionsCount + 1 })}
              </button>
            ) : null}
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
