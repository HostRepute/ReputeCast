import type { Component, Editor } from 'grapesjs'
import { createContext, useContext } from 'react'
import type { Schemas } from '@/api/client'
import type { ComposerEditor, DeviceId } from './editor/grapes'

export type Campaign = Schemas['CampaignDetail']
export type ComposerCtxData = Schemas['ComposerContext']
export type Checks = Schemas['ComposerChecks']
export type Issue = Schemas['ContentIssue']
export type ABTest = Schemas['ABTest']
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
  save(opts?: { manual?: boolean; force?: boolean; source?: 'autosave' | 'manual' | 'import' | 'template'; label?: string }): Promise<boolean>
  /** Full MJML of the current content (head with title and preview + canvas). */
  currentMjml(): string
  compileCurrent(): { html: string; errors: { line: number; message: string; tagName: string }[] } | null
  /** Replaces the whole content (template, import, restore); empty string resets to the gallery. */
  loadContent(mjml: string, how: { source: 'template' | 'import' | 'restore' | 'reload'; label?: string }): void
  isEmpty: boolean

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
}

export type ComposerDialog = 'versions' | 'assets' | 'import' | 'sendTest' | 'saveTemplate' | 'shortcuts' | 'compare' | null

export const ComposerCtx = createContext<ComposerState | null>(null)

export function useComposer(): ComposerState {
  const v = useContext(ComposerCtx)
  if (!v) throw new Error('useComposer must be used inside the composer')
  return v
}
