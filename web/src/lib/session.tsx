import { useQuery, useQueryClient } from '@tanstack/react-query'
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { http, type Schemas } from '@/api/client'
import { t, type MessageDescriptor } from '@/i18n'
import { orList } from './format'
import { m } from './session.messages'

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

/**
 * Explains a missing permission (principle 7): "Sending needs the
 * campaigns:send permission. Ask Marcus Chen or Linh Nguyen." `action`
 * names a narrower action than the permission's own ("Changing the route of
 * a send"), in the app's language.
 */
export function deniedReason(perm: Permission, helpers: string[] = [], action?: string): string {
  const named = (m as Record<string, MessageDescriptor | undefined>)[perm]
  const what = action ?? t(named ?? m.this)
  return helpers.length ? t(m.deniedAsk, { action: what, permission: perm, helpers: orList(helpers) }) : t(m.denied, { action: what, permission: perm })
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
