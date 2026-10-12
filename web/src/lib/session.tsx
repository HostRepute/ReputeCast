import { useQuery, useQueryClient } from '@tanstack/react-query'
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { http, type Schemas } from '@/api/client'
import { orList } from './format'

export type Me = Schemas['Me']
export type WorkspaceMe = Schemas['WorkspaceMe']
export type Permission = string

export const meQueryKey = ['me'] as const
export const workspaceMeKey = (slug: string) => ['workspace-me', slug] as const

/** Loads the signed-in person; `null` when signed out. */
export async function fetchMe(): Promise<Me | null> {
  try {
    const { data } = await http.GET('/api/v1/me')
    return data ?? null
  } catch (e) {
    if ((e as { status?: number }).status === 401) return null
    throw e
  }
}

export function useMe() {
  return useQuery({ queryKey: meQueryKey, queryFn: fetchMe, staleTime: 60_000 })
}

export async function fetchWorkspaceMe(slug: string): Promise<WorkspaceMe> {
  const { data } = await http.GET('/api/v1/workspaces/{workspace}/me', { params: { path: { workspace: slug } } })
  return data!
}

interface WorkspaceCtx {
  slug: string
  me: WorkspaceMe
}

const Ctx = createContext<WorkspaceCtx | null>(null)

export function WorkspaceProvider({ slug, me, children }: { slug: string; me: WorkspaceMe; children: ReactNode }) {
  const value = useMemo(() => ({ slug, me }), [slug, me])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

/** The current workspace (slug, role, permissions). Only inside /w/$workspace. */
export function useWorkspace(): WorkspaceCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useWorkspace must be used inside a workspace route')
  return v
}

/** Optional variant for components that also render outside a workspace. */
export function useOptionalWorkspace(): WorkspaceCtx | null {
  return useContext(Ctx)
}

/** Whether the caller holds a permission in the current workspace. */
export function useCan(perm: Permission): boolean {
  const ws = useOptionalWorkspace()
  return Boolean(ws?.me.permissions.includes(perm))
}

/** Plain-language action names for "X needs the … permission". */
const actions: Record<string, string> = {
  'campaigns:send': 'Sending',
  'campaigns:pause': 'Pausing a send',
  'campaigns:approve': 'Approving',
  'campaigns:edit': 'Editing campaigns',
  'subscribers:export': 'Exporting contacts',
  'subscribers:import': 'Importing contacts',
  'subscribers:delete': 'Erasing personal data',
  'providers:manage': 'Managing providers',
  'domains:manage': 'Managing domains',
  'team:invite': 'Inviting members',
  'team:manage': 'Managing the team',
  'segments:sql': 'Writing SQL segments',
  'apikeys:manage': 'Managing API keys',
  'mcp:manage': 'Managing MCP agents',
  'automations:publish': 'Publishing automations',
  'workspace:manage': 'Changing workspace settings',
  'webhooks:manage': 'Managing webhooks',
  'integrations:hostrepute': 'Connecting HostRepute',
  'forms:edit': 'Editing forms',
  'reports:share': 'Sharing reports',
}

/**
 * Explains a missing permission (principle 7): "Sending needs the
 * campaigns:send permission. Ask Marcus Chen or Linh Nguyen."
 */
export function deniedReason(perm: Permission, helpers: string[] = []): string {
  const action = actions[perm] ?? 'This'
  const ask = helpers.length ? ` Ask ${orList(helpers)}.` : ''
  return `${action} needs the ${perm} permission.${ask}`
}

/**
 * Returns undefined when allowed, otherwise the reason to show on a disabled
 * control. `helpers` come from the team list when known.
 */
export function usePermissionReason(perm: Permission, helpers?: string[]): string | undefined {
  const can = useCan(perm)
  return can ? undefined : deniedReason(perm, helpers)
}

/** Clears all cached data after sign-out or a workspace switch. */
export function useResetSession() {
  const qc = useQueryClient()
  return () => qc.clear()
}
