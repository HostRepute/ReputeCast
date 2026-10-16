import type { Component, Editor } from 'grapesjs'
import { createContext, useContext } from 'react'
import type { Schemas } from '@/api/client'
import type { ComposerEditor, DeviceId } from './editor/grapes'

export type Campaign = Schemas['ComposerCampaign']
export type ComposerCtxData = Schemas['ComposerContext']
export type Checks = Schemas['ComposerChecks']
export type Issue = Schemas['ContentCheckIssue']
export type ABTest = Schemas['ComposerABTest']
export type View = 'visual' | 'mjml' | 'html' | 'preview'

export type SaveStatus =
  | { kind: 'new' }
  | { kind: 'saved'; at: string; version: number }
  | { kind: 'dirty'; at?: string; version: number }
  | { kind: 'saving'; version: number }
  | { kind: 'offline'; since: string; version: number; message: string }
  | { kind: 'conflict'; version: number; theirs: { version: number; by: string; at: string; theirChanges: string[]; yourChanges: string[] } }
  | { kind: 'error'; version: number; message: string }

export interface ComposerState {
  /**
   * What is being edited. In 'transactional' mode (features/composer/transactional)
   * `campaign` describes a transactional template, so its id isn't a campaign's:
   * shared parts must not call campaign endpoints with it.
   */
  kind?: 'campaign' | 'transactional'
  /** The View switcher's items, when they differ from Visual · MJML · HTML (the transactional mode adds Text); selectView picks one. */
  views?: { key: string; label: string; on: boolean }[]
  selectView?(key: string): void
  slug: string
  campaign: Campaign
  context: ComposerCtxData
  /** The live editor, once GrapesJS is loaded and the canvas is not empty. */
  ge: ComposerEditor | null
  editor: Editor | null
  selected: Component | null
  view: View
  device: DeviceId
  variant: string
  setView(v: View): void
  setDevice(d: DeviceId): void
  setVariant(v: string): void

  name: string
  subject: string
  preheader: string
  abTest: ABTest | null
  setName(v: string): void
  commitName(): void
  setSubject(v: string): void
  setPreheader(v: string): void
  setABTest(t: ABTest | null): void
  /** The head (mj-head inner MJML) kept next to the canvas. */
  head: string
  setHead(h: string): void

  status: SaveStatus
  /** Saves a version now. `manual` shows the saved card unless `quiet`; `force` overrides a conflict. */
  save(opts?: { manual?: boolean; quiet?: boolean; force?: boolean; background?: boolean; source?: 'autosave' | 'manual' | 'import' | 'template'; label?: string }): Promise<boolean>
  /** Full MJML of the current content (head with title and preview + canvas). */
  currentMjml(): string
  compileCurrent(): { html: string; errors: { line: number; message: string; tagName: string }[] } | null
  /** Replaces the whole content (template, import, code view, restore); empty string resets to the gallery. */
  loadContent(mjml: string, how: { source: 'template' | 'import' | 'code' | 'restore' | 'reload'; label?: string }): void
  /** Loads the latest saved version (after a restore, or to take someone else's save). */
  reloadFromServer(): Promise<void>
  /** Content settings changed on the server (minify): re-run the checks. */
  onSettingsChanged(): void
  /** The code view applies or keeps its draft before another view opens; false blocks the switch. */
  registerLeaveCheck(fn: () => boolean): () => void
  /** The code view has edits not applied to the canvas yet (leaving would lose them). */
  setCodeDirty(dirty: boolean): void
  isEmpty: boolean
  /** Dark-mode check on the mobile canvas. */
  darkCanvas: boolean
  setDarkCanvas(v: boolean): void
  /** Where the asset manager puts the chosen image. */
  assetTarget: { component: Component; attr: 'src' | 'background-url' } | null
  setAssetTarget(t: { component: Component; attr: 'src' | 'background-url' } | null): void
  /** Where the preview puts "Preview as" and its actions (in the header). */
  previewSlot: HTMLElement | null
  setPreviewSlot(el: HTMLElement | null): void

  checks: Checks | undefined
  checksPending: boolean
  rightTab: 'style' | 'settings' | 'deliverability'
  setRightTab(t: 'style' | 'settings' | 'deliverability'): void
  leftTab: 'blocks' | 'layers' | 'templates'
  setLeftTab(t: 'blocks' | 'layers' | 'templates'): void
  openDialog(d: ComposerDialog): void
  /** Bumped on every canvas change; components re-read the editor when it changes. */
  rev: number
  canEdit: boolean
  /** A language picked that has no version (internal/languages): the canvas gives way to "No … version". */
  missingLanguage?: string
}

export type ComposerDialog = 'versions' | 'assets' | 'import' | 'sendTest' | 'saveTemplate' | 'shortcuts' | 'compare' | null

export const ComposerCtx = createContext<ComposerState | null>(null)

export function useComposer(): ComposerState {
  const v = useContext(ComposerCtx)
  if (!v) throw new Error('useComposer must be used inside the composer')
  return v
}
