import type { Component } from 'grapesjs'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { http, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'
import { aiBase } from '../api'
import { useAssist, useSuggest, type Assist, type Suggest } from '../suggest'

export type RewriteAction = 'shorten' | 'friendlier' | 'formal' | 'direct' | 'instruction'
export type IdeasInput = Schemas['AISubjectIdeasInput']
export type Ideas = Schemas['AISubjectIdeas']
export type RewriteInput = Schemas['AIRewriteInput']
export type Rewrite = Schemas['AIRewrite']
export type Translation = Schemas['AITranslation']
export type TranslateInput = Schemas['AITranslateInput']

/** The text block a rewrite is about: the component, its HTML when asked, and the instruction. */
export type RewriteTarget = { action: RewriteAction; component: Component; html: string; instruction?: string; input: RewriteInput | null }

/** What the composer's right column shows instead of the inspector. */
export type AssistPanel = { kind: 'ideas'; input: IdeasInput } | ({ kind: 'rewrite' } & RewriteTarget) | null

export interface ComposerAssist {
  assist: Assist
  panel: AssistPanel
  ideas: Suggest<IdeasInput, Ideas>
  rewriting: Suggest<RewriteInput, Rewrite>
  /** Subject ideas: opens the panel and asks the model. */
  openIdeas(input: IdeasInput): void
  /** A rewrite of a text block; "With an instruction…" waits for the instruction. */
  rewrite(target: Omit<RewriteTarget, 'input'>, input: RewriteInput | null): void
  closePanel(): void
  /** The language being translated with AI: the compare view's AI mode. */
  translating: string | null
  translation: Suggest<TranslateInput, Translation>
  /** Translate a language version with AI (the compare view's AI mode); null leaves it. */
  translate(language: string | null): void
}

const Ctx = createContext<ComposerAssist | null>(null)

/** The assistant inside the campaign composer; null elsewhere (the template editor has none). */
export function useComposerAssist(): ComposerAssist | null {
  return useContext(Ctx)
}

/** The composer's assistant state: the panel open, its request, and the translation under review. */
export function useComposerAssistState(campaignId: string): ComposerAssist {
  const { slug } = useWorkspace()
  const assist = useAssist()
  const [panel, setPanel] = useState<AssistPanel>(null)
  const [translating, setTranslating] = useState<string | null>(null)
  const ideas = useSuggest<IdeasInput, Ideas>(async (body, signal) => (await http.POST(`${aiBase}/subject-ideas`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const rewriting = useSuggest<RewriteInput, Rewrite>(async (body, signal) => (await http.POST(`${aiBase}/rewrite`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const translation = useSuggest<TranslateInput, Translation>(async (body, signal) => (await http.POST(`${aiBase}/translate`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const { start: startIdeas, cancel: cancelIdeas } = ideas
  const { start: startTranslation, cancel: cancelTranslation } = translation
  const translate = useCallback(
    (language: string | null) => {
      setTranslating(language)
      if (language) void startTranslation({ campaignId, language })
      else cancelTranslation()
    },
    [campaignId, startTranslation, cancelTranslation],
  )
  const { start: startRewrite, cancel: cancelRewrite } = rewriting
  const openIdeas = useCallback(
    (input: IdeasInput) => {
      cancelRewrite()
      setPanel({ kind: 'ideas', input })
      void startIdeas(input)
    },
    [startIdeas, cancelRewrite],
  )
  const rewrite = useCallback(
    (target: Omit<RewriteTarget, 'input'>, input: RewriteInput | null) => {
      cancelIdeas()
      cancelRewrite()
      setPanel({ kind: 'rewrite', ...target, input })
      if (input) void startRewrite(input)
    },
    [startRewrite, cancelIdeas, cancelRewrite],
  )
  const closePanel = useCallback(() => {
    cancelIdeas()
    cancelRewrite()
    setPanel(null)
  }, [cancelIdeas, cancelRewrite])
  return useMemo(
    () => ({ assist, panel, ideas, rewriting, openIdeas, rewrite, closePanel, translating, translation, translate }),
    [assist, panel, ideas, rewriting, openIdeas, rewrite, closePanel, translating, translation, translate],
  )
}

export function ComposerAssistScope({ value, children }: { value: ComposerAssist; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
