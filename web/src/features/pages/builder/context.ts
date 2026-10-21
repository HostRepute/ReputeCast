import type { Component, Editor } from 'grapesjs'
import { createContext, useContext } from 'react'
import type { Schemas } from '@/api/client'
import type { BlockContext, PaletteId } from './blocks'
import type { PageDevice, PageEditor } from './editor/pageEditor'
import type { SaveStatus } from './save'

export type PageDetail = Schemas['PageDetail']
export type PageCheck = Schemas['PageCheck']
export type PageCanvas = Schemas['PageCanvas']
export type PageSettings = Schemas['PageSettings']
export type PageCounts = Schemas['PageCounts']

export type RightTab = 'block' | 'page' | 'checks'

/** The publish dialog's starting time (the Page tab's "Publish time"). */
export interface PublishWhen {
  when: 'now' | 'at'
  date: string
  time: string
  tz: string
}

export type BuilderDialog =
  | { kind: 'publish' }
  | { kind: 'approval' }
  | { kind: 'unpublish' }
  | { kind: 'versions' }
  | { kind: 'preview'; version?: number }
  | { kind: 'media'; pick: (url: string) => void }
  | { kind: 'compare' }
  | { kind: 'shortcuts' }
  | null

/** Someone else's newer draft, kept while a conflict is open. */
export interface ConflictInfo {
  html: string
  /** Blocks (by kind) both of you changed, and the ones only they changed. */
  both: string[]
  theirs: string[]
}

export interface Builder {
  slug: string
  page: PageDetail
  /** Replaces what's known about the page (after the settings, a publish or a reload). */
  setPage: (fn: (p: PageDetail) => PageDetail) => void
  /**
   * Takes a page detail the server answered with (the settings saved, a
   * publish): the canvas keeps its blocks, the next save goes on top of
   * the version it made.
   */
  applyDetail: (d: PageDetail) => void
  canvas: PageCanvas
  pe: PageEditor | null
  editor: Editor | null
  selected: Component | null
  select: (c: Component | null, opts?: { scroll?: boolean }) => void
  /** Bumps on every change of the canvas, so panels re-read it. */
  rev: number
  canEdit: boolean
  /** website:publish. */
  canPublishPerm: boolean
  device: PageDevice
  setDevice: (d: PageDevice) => void
  rightTab: RightTab
  setRightTab: (t: RightTab) => void
  status: SaveStatus
  save: (opts?: { manual?: boolean; force?: boolean; quiet?: boolean; background?: boolean }) => Promise<boolean>
  reloadFromServer: () => Promise<void>
  /** Reads the publish checks again (a form resumed, a product published elsewhere). */
  refreshChecks: () => Promise<void>
  conflict: ConflictInfo | null
  savedCard: { version: number; at: string } | null
  closeSavedCard: () => void
  dialog: BuilderDialog
  openDialog: (d: BuilderDialog) => void
  publishWhen: PublishWhen
  setPublishWhen: (w: PublishWhen) => void
  /** Adds a block from the palette next to the selection. */
  insertBlock: (id: PaletteId) => void
  /** The selected block's toolbar. */
  moveSelected: (dir: -1 | 1) => void
  duplicateSelected: () => void
  removeSelected: () => void
  /** Starts the publish flow (saving first). */
  startPublish: () => void
  /** A field of the Block tab to focus once it's shown ("Pick another form"). */
  focusField: string | null
  setFocusField: (id: string | null) => void
  /** The workspace's time zone. */
  tz: string
  /** What new blocks start with (the palette's markup). */
  blockContext: BlockContext
}

export const BuilderCtx = createContext<Builder | null>(null)

export function useBuilder(): Builder {
  const v = useContext(BuilderCtx)
  if (!v) throw new Error('useBuilder must be used inside the page builder')
  return v
}
