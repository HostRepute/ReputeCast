import type { QueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { vi } from 'vitest'
import { meQueryKey } from '@/lib/session'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import type { WorkspaceMe } from '@/lib/session'
import type { ComposerSearch } from './Composer'
import { ComposerPage } from './ComposerPage'
import type { Campaign } from './context'
import { testContext } from './testing'

/*
 * The whole composer page for tests (saves, autosave, approvals, previews):
 * the campaign and the composer's context come from the API as in the app,
 * the canvas is a headless editor (the test file mocks './editor/grapes'
 * with headlessComposerEditor), and API calls the test doesn't answer get a
 * 404, as an endpoint outside the test would.
 */

type Handler = (request: Request) => Response | Promise<Response>

export const workspaceBase = '/api/v1/workspaces/acme-newsletter'

/** Linh, who edits campaigns, unless a test says otherwise. */
export const composerMe = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'campaigns:send', 'subscribers:view'] })

const seedMe = (qc: QueryClient) =>
  qc.setQueryData(meQueryKey, { user: { id: 'u-linh', email: 'linh@acme.io', name: 'Linh Nguyen' }, workspaces: [], instance: {} } as never)

/** jsdom lacks the observers the composer's frames and panels use. */
function stubObservers() {
  class Observer {
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return []
    }
  }
  vi.stubGlobal('ResizeObserver', Observer)
  vi.stubGlobal('IntersectionObserver', Observer)
}

/** The page with its search params kept in state, as the route keeps them in the address. */
function Page({ id, initial }: { id: string; initial: ComposerSearch }) {
  const [search, setSearch] = useState(initial)
  return <ComposerPage campaignId={id} search={search} onSearch={setSearch} />
}

/**
 * Renders /campaigns/:id/edit for `campaign()` (read on every GET, so a test
 * can change what the server has). `routes` answer the calls the test is
 * about ("PUT /api/v1/workspaces/acme-newsletter/campaigns/…/content").
 */
export async function renderComposerPage(
  campaign: () => Campaign,
  { routes = {}, me = composerMe, search = {} }: { routes?: Record<string, Handler>; me?: WorkspaceMe; search?: ComposerSearch } = {},
) {
  stubObservers()
  const id = campaign().id
  const own: Record<string, Handler> = {
    [`GET ${workspaceBase}/campaigns/${id}`]: () => json(campaign()),
    [`GET ${workspaceBase}/composer`]: () => json(testContext()),
    ...routes,
  }
  const unmock = mockApi(new Proxy(own, { get: (t, k: string) => t[k] ?? (() => json({ title: 'Not Found', status: 404, detail: 'Not part of this test.' }, 404)) }))
  const utils = await renderInWorkspace(<Page id={id} initial={search} />, { me, seed: seedMe, path: `/w/acme-newsletter/campaigns/${id}/edit` })
  return {
    ...utils,
    unmock: () => {
      unmock()
      vi.unstubAllGlobals()
    },
  }
}
