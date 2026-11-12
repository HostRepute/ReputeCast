import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'

/*
 * Lead ads from Meta (M66, G-41): the connection page, its dialogs and the
 * catalog card read these operations (internal/leads/api.go).
 */

export type Connection = Schemas['LeadAdsConnection']
export type LeadPage = Schemas['LeadAdsPage']
export type LeadForm = Schemas['LeadAdsForm']
export type ActivityItem = Schemas['LeadAdsActivityItem']
export type FormDetail = Schemas['LeadAdsFormDetail']
export type Question = Schemas['LeadAdsQuestion']
export type PageChoice = Schemas['LeadAdsPageChoice']
export type Setup = Schemas['LeadAdsSetup']
export type Outcome = ActivityItem['outcome']
export type Problem = NonNullable<Connection['problem']>

export const leadsBase = '/api/v1/workspaces/{workspace}/lead-ads' as const

/** A connection's page. While a form imports past leads, it reads again every 3 seconds. */
export function useConnection(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${leadsBase}/connections/{id}`,
    { params: { path: { workspace: slug, id } } },
    { refetchInterval: (q) => (q.state.data?.pages.some((p) => p.forms.some((f) => f.import.state === 'running')) ? 3000 : false) },
  )
}

/** The connection's last 50 leads. */
export function useActivity(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${leadsBase}/connections/{id}/activity`, { params: { path: { workspace: slug, id } } })
}

/** What the connect dialog's first step shows. */
export function useSetup(enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${leadsBase}/setup`, { params: { path: { workspace: slug } } }, { enabled })
}

/** The Pages the token reaches, read from Meta. */
export function usePageChoices(id: string | undefined) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${leadsBase}/connections/{id}/pages`, { params: { path: { workspace: slug, id: id ?? '' } } }, { enabled: !!id, staleTime: 0, gcTime: 0 })
}

/** A form's mapping dialog. */
export function useFormDetail(formId: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${leadsBase}/forms/{formId}`, { params: { path: { workspace: slug, formId } } }, { staleTime: 0 })
}

/** Refreshes every lead ads screen, the catalog and the automation options. */
export function useRefreshLeads() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey[1]
          return (
            typeof k === 'string' &&
            (k.startsWith(leadsBase) || k.startsWith('/api/v1/workspaces/{workspace}/integrations/') || k === '/api/v1/workspaces/{workspace}/automations/options')
          )
        },
      }),
    [qc],
  )
}

/** Every form of the chosen Pages, in the page's order. */
export function formsOf(c: Pick<Connection, 'pages'>): LeadForm[] {
  return c.pages.flatMap((p) => p.forms)
}
