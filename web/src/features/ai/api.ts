import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { useCan, useWorkspace } from '@/lib/session'

export const aiBase = '/api/v1/workspaces/{workspace}/ai' as const

export type AISettings = Schemas['AISettings']
export type AIConnection = Schemas['AIConnectionView']
export type AIFeature = Schemas['AIFeatureState']['key']
export type AITest = Schemas['AITestResult']
export type AIStatus = Schemas['AIStatus']
export type AIModelOption = Schemas['AIModelOption']

/** The features in the settings' order. */
export const AI_FEATURES: AIFeature[] = ['subject_ideas', 'rewrite', 'translate', 'alt_text', 'accessibility_fix', 'segment_draft', 'report_summary']

/** Settings › Integrations › AI assistant. */
export function useAISettings() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', aiBase, { params: { path: { workspace: slug } } })
}

/**
 * Whether the assistant is ready for the signed-in person: only asked of
 * people who hold ai:use (the endpoint is theirs alone).
 */
export function useAIStatus() {
  const { slug } = useWorkspace()
  const can = useCan('ai:use')
  return $api.useQuery('get', `${aiBase}/status`, { params: { path: { workspace: slug } } }, { enabled: can, staleTime: 30_000 })
}

/** Refetches every AI query and the integrations catalog after a change. */
export function useRefreshAI() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey[1]
          return typeof k === 'string' && (k.startsWith(aiBase) || k.startsWith('/api/v1/workspaces/{workspace}/integrations'))
        },
      }),
    [qc],
  )
}

/** The problem's code, when an API call failed with one. */
export function problemCode(e: unknown): string | undefined {
  return e instanceof ApiError ? e.code : undefined
}

/** A value from the problem's meta (ai_test_failed's test, ai_rate_limited's retryAfter). */
export function problemMeta<T>(e: unknown, key: string): T | undefined {
  if (!(e instanceof ApiError)) return undefined
  return e.problem.meta?.[key] as T | undefined
}

/** A problem's reference, as the error screens show it ("request 9b41c2e0"). */
export function referenceOf(e: unknown): string | undefined {
  if (!(e instanceof ApiError)) return undefined
  const p = e.problem as { instance?: string; meta?: { requestId?: string; errorId?: string } }
  return p.meta?.errorId ?? p.meta?.requestId ?? p.instance
}
