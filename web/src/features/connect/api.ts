import { useMutation, useQueryClient } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'

export type Setup = Schemas['ConnectSetup']
export type SetupStep = Setup['step']
export type Catalog = Schemas['ConnectCatalog']

/** The wizard's steps after the catalog, in order. */
export const STEPS: SetupStep[] = ['connection', 'domain', 'test', 'streams', 'contacts', 'done']

/** "Step 2 of 6": the catalog is step 1. */
export const stepNumber: Record<SetupStep, number> = { connection: 2, domain: 3, test: 4, streams: 5, contacts: 6, done: 7 }

export const setupPath = '/api/v1/workspaces/{workspace}/connect/setups/{setupId}' as const

export function setupInit(workspace: string, setupId: string) {
  return { params: { path: { workspace, setupId } } }
}

export function setupKey(workspace: string, setupId: string) {
  return $api.queryOptions('get', setupPath, setupInit(workspace, setupId)).queryKey
}

/** A setup; polls while provisioning runs. */
export function useSetup(workspace: string, setupId: string) {
  return $api.useQuery('get', setupPath, setupInit(workspace, setupId), {
    refetchInterval: (q) => (q.state.data?.connection.working ? 2000 : false),
  })
}

/** Runs a setup action and puts the setup it returns in the cache. */
export function useSetupAction<A>(workspace: string, setupId: string, run: (args: A) => Promise<Setup | undefined>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: run,
    onSuccess: (data) => {
      if (data) qc.setQueryData(setupKey(workspace, setupId), data)
      // The catalog's counts and the providers pages change with a setup.
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/connect/catalog'] })
    },
  })
}
