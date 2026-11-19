import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { useCan, useWorkspace } from '@/lib/session'

/** The workspace's remote MCP servers (internal/mcpclient, B5). */
export const mcpBase = '/api/v1/workspaces/{workspace}/ai/mcp/servers' as const

export type MCPServer = Schemas['RemoteMCPServer']
export type MCPTool = Schemas['RemoteMCPTool']
export type MCPDecision = Schemas['RemoteMCPToolDecision']['decision']

export function useMCPServers() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', mcpBase, { params: { path: { workspace: slug } } })
}

/**
 * The servers the person may pick for a thread, with whether they still
 * have to sign in (list-ai-assistant-mcp-servers).
 */
export function useUsableMCPServers(enabled = true) {
  const { slug } = useWorkspace()
  const can = useCan('ai:use')
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/ai/assistant/mcp-servers', { params: { path: { workspace: slug } } }, { enabled: can && enabled, staleTime: 30_000 })
}

/** Refetches both lists after a change (the settings and the panel read them). */
export function useRefreshMCP() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const path = q.queryKey[1]
          return path === mcpBase || path === '/api/v1/workspaces/{workspace}/ai/assistant/mcp-servers'
        },
      }),
    [qc],
  )
}

export async function createServer(slug: string, body: Schemas['RemoteMCPServerInput']) {
  const { data } = await http.POST(mcpBase, { params: { path: { workspace: slug } }, body })
  return data!
}

export async function updateServer(slug: string, server: string, body: Schemas['RemoteMCPServerUpdate']) {
  const { data } = await http.PATCH(`${mcpBase}/{server}`, { params: { path: { workspace: slug, server } }, body })
  return data!
}

export async function deleteServer(slug: string, server: string) {
  await http.DELETE(`${mcpBase}/{server}`, { params: { path: { workspace: slug, server } } })
}

export async function syncServer(slug: string, server: string) {
  const { data } = await http.POST(`${mcpBase}/{server}/sync`, { params: { path: { workspace: slug, server } } })
  return data!
}

/** A decision at the hash the manager reviewed: a 409 mcp_tool_reviewed means the tool changed meanwhile. */
export async function decideTool(slug: string, server: string, tool: string, body: Schemas['RemoteMCPToolDecision']) {
  const { data } = await http.PUT(`${mcpBase}/{server}/tools/{tool}`, { params: { path: { workspace: slug, server, tool } }, body })
  return data!
}

/** Starts the person's own sign-in: the browser goes to the server's authorization page. */
export async function startSignIn(slug: string, server: string) {
  const { data } = await http.POST(`${mcpBase}/{server}/oauth`, { params: { path: { workspace: slug, server } } })
  window.location.assign(data!.authorizationUrl)
}

export async function disconnect(slug: string, server: string) {
  await http.DELETE(`${mcpBase}/{server}/oauth`, { params: { path: { workspace: slug, server } } })
}

/** "mcp__linear__create_issue" → the server's key and the tool's name. */
export function splitModelName(name: string): { key: string; tool: string } | undefined {
  const x = /^mcp__([a-z0-9]+)__(.+)$/.exec(name)
  return x && x[1] && x[2] ? { key: x[1], tool: x[2] } : undefined
}
