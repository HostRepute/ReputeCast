import { useId, useState } from 'react'
import { toast } from '@/components/ui/Toast'
import { andList } from '@/lib/format'
import { useT } from '@/i18n'
import { useComposer } from './context'
import { sectionWord } from './editor/labels'
import { mn } from './banners.messages'

/** A part of the email a save changed, as the server names it ("subject", "hero"), in the app language. */
export function partName(part: string, t: ReturnType<typeof useT>): string {
  if (part === 'subject') return t(mn.partSubject)
  if (part === 'the email body') return t(mn.partBody)
  return sectionWord(part)
}

/** The conflict banner's icon (ComposerStates): two branches meeting. */
function BranchIcon() {
  return (
    <svg className="bi" width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="4" cy="3.5" r="1.6" />
      <circle cx="4" cy="12.5" r="1.6" />
      <circle cx="12" cy="8" r="1.6" />
      <path d="M4 5.1v5.8M5.4 4.3c3 .7 5 1.8 5.3 2.4" />
    </svg>
  )
}

/** The recovered-edits banner's icon (ComposerStates): a clock turning back. */
export function HistoryIcon() {
  return (
    <svg className="bi" width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2.5 8a5.5 5.5 0 1 0 1.6-3.9" />
      <path d="M2.5 2.5v3h3" />
      <path d="M8 5.2V8l2 1.3" />
    </svg>
  )
}

/**
 * Someone else saved while you were editing (ComposerStates · Conflict):
 * keep yours on top, compare section by section, or load theirs after
 * keeping yours in the version history. Nothing is lost either way.
 */
export function ConflictBanner() {
  const c = useComposer()
  const t = useT()
  const id = useId()
  const [busy, setBusy] = useState<'mine' | 'theirs' | null>(null)
  const s = c.status
  if (s.kind !== 'conflict') return null
  const who = s.theirs.by || t(mn.someone)
  const first = who.split(' ')[0] ?? who
  const both = s.theirs.theirChanges.filter((x) => s.theirs.yourChanges.includes(x))
  const next = s.theirs.version + 1
  const keepMine = async () => {
    setBusy('mine')
    try {
      await c.save({ force: true, manual: true, quiet: true })
    } finally {
      setBusy(null)
    }
  }
  const loadTheirs = async () => {
    setBusy('theirs')
    try {
      // Yours go into the history first; then the editor opens theirs.
      if (await c.save({ stash: true })) {
        await c.reloadFromServer()
        toast({ kind: 'success', title: t(mn.loadTheirsDone, { name: first, version: s.theirs.version }), detail: t(mn.loadTheirsDoneDetail, { kept: next }) })
      }
    } catch (e) {
      toast({ kind: 'error', title: t(mn.loadTheirsFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setBusy(null)
    }
  }
  return (
    <div className="bnr w" role="alert">
      <BranchIcon />
      <span className="bt">
        <b>{t(mn.conflictTitle, { who, version: s.theirs.version })}</b>
        <span>
          {both.length
            ? t(mn.conflictBoth, { parts: andList(both.map((p) => partName(p, t))) })
            : s.theirs.theirChanges.length
              ? t(mn.conflictTheirs, { name: first, parts: andList(s.theirs.theirChanges.map((p) => partName(p, t))) })
              : t(mn.conflictPlain)}
        </span>
      </span>
      <div className="choices" role="group" aria-label={t(mn.resolveGroup)}>
        <div className="choice">
          <button type="button" className="btn sm pri" aria-describedby={`${id}-mine`} aria-busy={busy === 'mine' || undefined} disabled={busy !== null} onClick={() => void keepMine()}>
            {t(mn.keepMine)}
          </button>
          <span id={`${id}-mine`}>{t(mn.keepMineText, { next, version: s.theirs.version, name: first })}</span>
        </div>
        <div className="choice">
          <button type="button" className="btn sm" aria-describedby={`${id}-cmp`} disabled={busy !== null} onClick={() => c.openDialog('compare')}>
            {t(mn.compare)}
          </button>
          <span id={`${id}-cmp`}>{t(mn.compareText)}</span>
        </div>
        <div className="choice">
          <button type="button" className="btn sm" aria-describedby={`${id}-theirs`} aria-busy={busy === 'theirs' || undefined} disabled={busy !== null} onClick={() => void loadTheirs()}>
            {t(mn.loadTheirs)}
          </button>
          <span id={`${id}-theirs`}>{t(mn.loadTheirsText, { next, version: s.theirs.version, name: first })}</span>
        </div>
      </div>
    </div>
  )
}

/** The banners under the top bar, in the order the boards stack them. */
export function ComposerBanners() {
  return <ConflictBanner />
}
