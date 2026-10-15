import { createContext, useContext } from 'react'
import type { Category, Checks, Data, Preview, Settings, Template, TestData, TxTab, TxView, Variable } from './types'

/** Template fields saved with the draft that aren't content. */
export interface DraftMeta {
  senderId: string
  replyTo: string
  trackOpens: boolean
  trackClicks: boolean
}

/** Fields of the template itself (PATCH), not versioned. */
export interface TemplateMetaPatch {
  name?: string
  key?: string
  description?: string
  categoryId?: string
}

/**
 * What the transactional editor adds to the composer: the template, its
 * variables and test data, the checks and the inbox preview. The shared
 * composer parts read ComposerState; the transactional ones read this too.
 */
export interface TxState {
  template: Template
  settings: Settings | undefined
  categories: Category[]
  /** Address of the transactional footer (Settings › Senders), when the member can read it. */
  address: string
  /** The product's name in footers: Settings › Transactional, or the workspace's name. */
  product: string
  kind: 'essential' | 'optional'
  variables: Variable[]
  setVariables(v: Variable[], label?: string): void
  testData: TestData[]
  setTestData(t: TestData[], label?: string): void
  /** The test data set the preview, the canvas and the API tab use ("" for the samples). */
  previewWith: string
  setPreviewWith(name: string): void
  previewData: Data
  meta: DraftMeta
  setMeta(patch: Partial<DraftMeta>): void
  plainText: string
  setPlainText(v: string): void
  /** An HTML-only template (imported HTML): edited as HTML, no canvas. */
  htmlMode: boolean
  html: string
  setHtml(v: string): void
  /** Turns an HTML-only template into MJML (one Raw HTML block). */
  convertToMjml(): void
  checks: Checks | undefined
  checksPending: boolean
  preview: Preview | undefined
  tab: TxTab
  setTab(t: TxTab): void
  view: TxView
  setView(v: TxView): void
  /** A variable row to open (from a check, or a click on the canvas). */
  focusVariable: string | null
  setFocusVariable(name: string | null): void
  /** Design tab: the selected component's Style or Settings. */
  designTab: 'style' | 'settings'
  setDesignTab(t: 'style' | 'settings'): void
  updateTemplate(patch: TemplateMetaPatch): Promise<boolean>
  /** Takes the server's view of the template (live version, ID, access); the content stays as edited. */
  mergeTemplate(t: Template): void
  /** After a publish: also takes the draft's variables when making some optional saved a new revision. */
  afterPublish(t: Template): void
  openPublish(): void
  openVersions(): void
  canEdit: boolean
  canPublish: boolean
  /** Renames a variable in its declaration, the test data and every place the template uses it. */
  renameVariable(from: string, to: string): void
  /** Removes every output of a variable from the subject, preheader, text and content. */
  removeUsages(name: string): void
}

export const TxCtx = createContext<TxState | null>(null)

export function useTx(): TxState {
  const v = useContext(TxCtx)
  if (!v) throw new Error('useTx must be used inside the transactional editor')
  return v
}
