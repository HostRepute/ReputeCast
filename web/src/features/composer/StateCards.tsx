import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { clock } from '@/lib/format'
import { useComposer } from './context'
import { download, slugify } from './format'

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
  const s = c.status
  const cards: ReactNode[] = []
  if (s.kind === 'offline') {
    cards.push(
      <Card
        key="offline"
        tone="bad"
        badge="Not saved"
        title={`Couldn't save version ${s.version + 1}: ${s.message}`}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              Retry now
            </button>
            <button type="button" className="btn sm" onClick={() => download(`${slugify(c.name)}.mjml`, c.currentMjml(), 'text/plain')}>
              Download MJML as backup
            </button>
          </>
        }
      >
        Your edits since {clock(s.since)} are kept in this browser. ReputeCast retries every 10 seconds and saves as soon as the connection is back. Don't close the tab.
      </Card>,
    )
  } else if (s.kind === 'conflict') {
    const who = s.theirs.by || 'Someone'
    const first = who.split(' ')[0]
    const where = s.theirs.theirChanges.filter((x) => s.theirs.yourChanges.includes(x))
    cards.push(
      <Card
        key="conflict"
        tone="warn"
        badge="Conflict"
        title={`${who} saved version ${s.theirs.version}${s.theirs.at ? ` at ${clock(s.theirs.at)}` : ''} while you were editing`}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save({ force: true })}>
              Keep mine as v{s.theirs.version + 1}
            </button>
            <button type="button" className="btn sm" onClick={() => void c.reloadFromServer()}>
              Load {first}'s v{s.theirs.version}
            </button>
            <button type="button" className="btn sm" onClick={() => c.openDialog('compare')}>
              Compare
            </button>
          </>
        }
      >
        {where.length
          ? `Both of you changed the ${where.join(' and ')}. `
          : s.theirs.theirChanges.length
            ? `${first} changed the ${s.theirs.theirChanges.join(', ')}. `
            : ''}
        Keeping yours makes a new version {s.theirs.version + 1} on top of {first}'s; nothing is lost either way.
      </Card>,
    )
  } else if (s.kind === 'error') {
    cards.push(
      <Card
        key="error"
        tone="bad"
        badge="Not saved"
        title={`Couldn't save version ${s.version + 1}`}
        actions={
          <>
            <button type="button" className="btn sm pri" onClick={() => void c.save()}>
              Retry now
            </button>
            <button type="button" className="btn sm" onClick={() => download(`${slugify(c.name)}.mjml`, c.currentMjml(), 'text/plain')}>
              Download MJML as backup
            </button>
          </>
        }
      >
        {s.message} Your edits are still on the canvas.
      </Card>,
    )
  }
  if (saved && s.kind !== 'offline' && s.kind !== 'conflict' && s.kind !== 'error') {
    cards.push(
      <Card
        key="saved"
        tone="ok"
        badge="Saved"
        title={`Version ${saved.version} saved at ${clock(saved.at)} · autosave every 30 s and on every structural change`}
        actions={
          <>
            <button type="button" className="btn sm" onClick={onCloseSaved}>
              Back to editor
            </button>
            <button type="button" className="btn sm pri" onClick={onReview}>
              {c.campaign.access.canSend ? 'Review & send' : 'Request approval'}
            </button>
          </>
        }
      >
        {saved.template ? `Saved as a reusable template too: "${saved.template}" is now in Templates.` : 'Every version stays in the history, so you can restore any of them.'}
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
