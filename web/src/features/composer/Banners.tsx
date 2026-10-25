import { useId, useState } from 'react'
import { toast } from '@/components/ui/Toast'
import { andList, clock } from '@/lib/format'
import { versionLanguage } from '@/features/languages/composerParts'
import { useT } from '@/i18n'
import { useComposer } from './context'
import { sectionWord } from './editor/labels'
import type { JournalEntry } from './journal'
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

/** What the recovered-edits banner restores, and how. */
export interface Recovered {
  entry: JournalEntry
  restore(): Promise<boolean>
  discard(): void
}

/**
 * Edits this browser kept that never reached the server (ComposerStates ·
 * Recovered edits): restore them as a new version on top, or discard them.
 */
export function RecoveredBanner({ recovered }: { recovered: Recovered }) {
  const c = useComposer()
  const t = useT()
  const [busy, setBusy] = useState(false)
  const e = recovered.entry
  const from = clock(e.startedAt)
  const to = clock(e.updatedAt)
  const minutes = Math.max(1, Math.round((Date.parse(e.updatedAt) - Date.parse(e.startedAt)) / 60_000))
  const version = c.currentVersion?.() ?? c.campaign.version
  const onVersion = Boolean(versionLanguage(c.campaign))
  const body = onVersion
    ? t(mn.recoveredBodyVersionless)
    : e.baseVersion < version
      ? t(mn.recoveredBodyOlder, { base: e.baseVersion, name: c.campaign.updatedBy || t(mn.someone), version, next: version + 1 })
      : t(mn.recoveredBody, { version, next: version + 1 })
  return (
    <div className="bnr w" role="status">
      <HistoryIcon />
      <span className="bt">
        <b>{t(mn.recoveredTitle, { minutes, from, to })}</b>
        <span>{body}</span>
      </span>
      <span className="ba">
        <button
          type="button"
          className="btn sm pri"
          aria-busy={busy || undefined}
          disabled={busy || !c.canEdit}
          onClick={() => {
            setBusy(true)
            void recovered.restore().finally(() => setBusy(false))
          }}
        >
          {t(mn.restoreThem)}
        </button>
        <button type="button" className="btn sm danger" disabled={busy} onClick={recovered.discard}>
          {t(mn.discard)}
        </button>
      </span>
    </div>
  )
}

/** The banners under the top bar, in the order the boards stack them. */
export function ComposerBanners({ recovered }: { recovered?: Recovered | null }) {
  return (
    <>
      {recovered ? <RecoveredBanner recovered={recovered} /> : null}
      <ConflictBanner />
    </>
  )
}
