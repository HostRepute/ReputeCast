import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, ApiError, http, type Schemas } from '@/api/client'
import { useCan, useWorkspace } from '@/lib/session'

/** The assistant's API (internal/ai/assistant, B3). */
export const base = '/api/v1/workspaces/{workspace}/ai/assistant' as const

export type AssistantModel = Schemas['AIAssistantModel']
export type ThreadListItem = Schemas['AIThreadListItem']

/** A request key: a retry after a lost response returns the same thread or turn. */
export function requestKey(): string {
  return crypto.randomUUID().replaceAll('-', '')
}

/** The models a thread may use; 409 ai_assistant_off when the assistant is switched off. */
export function useAssistantModels() {
  const { slug } = useWorkspace()
  const can = useCan('ai:use')
  return $api.useQuery('get', `${base}/models`, { params: { path: { workspace: slug } } }, { enabled: can, staleTime: 30_000, retry: retryable })
}

export function useThread(thread: string | undefined) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${base}/threads/{thread}`,
    { params: { path: { workspace: slug, thread: thread ?? '' } } },
    { enabled: Boolean(thread), staleTime: 0, gcTime: 60_000, refetchOnWindowFocus: false, retry: retryable },
  )
}

export function useThreads(q: string, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/threads`, { params: { path: { workspace: slug }, query: { q: q || undefined, limit: 50 } } }, { enabled, staleTime: 10_000 })
}

/** 403, 404 and 409 (the assistant off, a thread expired) aren't worth retrying. */
function retryable(n: number, e: unknown): boolean {
  return n < 2 && !(e instanceof ApiError && [403, 404, 409].includes(e.status))
}

/** Refetches the thread list (after a new, renamed or deleted thread). */
export function useRefreshThreads() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => q.queryKey[1] === `${base}/threads`,
      }),
    [qc],
  )
}

export async function createThread(slug: string, body: Schemas['AIThreadInput']) {
  const { data } = await http.POST(`${base}/threads`, { params: { path: { workspace: slug } }, body })
  return data!
}

export async function sendMessage(slug: string, thread: string, text: string, key: string) {
  const { data } = await http.POST(`${base}/threads/{thread}/messages`, { params: { path: { workspace: slug, thread } }, body: { text, requestKey: key } })
  return data!
}

export async function stopTurn(slug: string, thread: string, turn: string) {
  const { data } = await http.POST(`${base}/threads/{thread}/turns/{turn}/stop`, { params: { path: { workspace: slug, thread, turn } } })
  return data!
}

export async function updateThread(slug: string, thread: string, body: Schemas['AIThreadUpdate']) {
  const { data } = await http.PATCH(`${base}/threads/{thread}`, { params: { path: { workspace: slug, thread } }, body })
  return data!
}

export async function deleteThread(slug: string, thread: string) {
  await http.DELETE(`${base}/threads/{thread}`, { params: { path: { workspace: slug, thread } } })
}

/** Runs a model's tool-calling check again (its own key's, or the workspace's); true when it passed. */
export async function testModelTools(slug: string, model: AssistantModel): Promise<boolean> {
  const p = { params: { path: { workspace: slug, account: model.accountId, model: model.id } } }
  const res = model.personal ? await http.POST('/api/v1/workspaces/{workspace}/ai/personal/accounts/{account}/models/{model}/test-tools', p) : await http.POST('/api/v1/workspaces/{workspace}/ai/accounts/{account}/models/{model}/test-tools', p)
  return res.data?.state === 'supported'
}

export async function attachContext(slug: string, thread: string, body: Schemas['AIThreadContextInput']) {
  const { data } = await http.POST(`${base}/threads/{thread}/contexts`, { params: { path: { workspace: slug, thread } }, body })
  return data!
}

export async function detachContext(slug: string, thread: string, context: string) {
  const { data } = await http.DELETE(`${base}/threads/{thread}/contexts/{context}`, { params: { path: { workspace: slug, thread, context } } })
  return data!
}

/** Allow once or Deny a writing MCP tool's call: the same decision again is fine; a closed call is a 409. */
export async function decideConfirmation(slug: string, thread: string, confirmation: string, decision: 'allow' | 'deny') {
  const { data } = await http.POST(`${base}/threads/{thread}/confirmations/{confirmation}`, { params: { path: { workspace: slug, thread, confirmation } }, body: { decision } })
  return data!
}

/** The MCP servers a thread may use (from list-ai-assistant-mcp-servers). */
export async function setThreadServers(slug: string, thread: string, serverIds: string[]) {
  const { data } = await http.PUT(`${base}/threads/{thread}/mcp-servers`, { params: { path: { workspace: slug, thread } }, body: { serverIds } })
  return data!
}
