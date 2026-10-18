import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { useBlockIds } from '@/features/polls/Block'
import { conditionalComponents } from './rules'

export type VisibilityReport = Schemas['VisibilityReport']
export type VisibilityPreview = Schemas['VisibilityPreview']
export type VisibilityCatalog = Schemas['VisibilityCatalog']
export type VisibilityBlockInfo = Schemas['VisibilityBlockInfo']
export type VisibilityVersion = Schemas['VisibilityVersion']

export interface ConditionalState {
  /** The campaign composer has conditional content (transactional templates don't). */
  enabled: boolean
  /** Conditional blocks on the canvas now. */
  blocks: number
  report: VisibilityReport | null
  counting: boolean
  error: string | null
  catalog: VisibilityCatalog | undefined
  /** The canvas shows every block, or the version one contact gets. */
  mode: 'all' | 'contact'
  setMode(m: 'all' | 'contact'): void
  contactId: string
  setContactId(id: string): void
  contacts: Schemas['VisibilityPreviewContact'][]
  /** The contact's version (contact mode). */
  preview: VisibilityPreview | null
  versionsOpen: boolean
  setVersionsOpen(open: boolean): void
  /** A contact the preview should show next ("Preview this version"); takePreview clears it. */
  pendingPreview: string
  requestPreview(contactId: string): void
  takePreview(): void
  recount(): void
  blockInfo(id: string): VisibilityBlockInfo | undefined
}

const Ctx = createContext<ConditionalState | null>(null)

/** The composer's conditional content; null outside the composer. */
export function useConditional(): ConditionalState | null {
  return useContext(Ctx)
}

const AS_KEY = (id: string) => `rc-conditional-as:${id}`

/**
 * Keeps the counts of an email's versions in step with the canvas: each
 * edit to a block's rule (or the content) recounts after a moment, with the
 * audience's live count; the server keeps the last count until the rules or
 * the audience change.
 */
export function ConditionalProvider({ children }: { children: ReactNode }) {
  const c = useComposer()
  const enabled = c.kind !== 'transactional'
  const canView = useCan('subscribers:view')
  const [report, setReport] = useState<VisibilityReport | null>(null)
  const [counting, setCounting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mode, setMode] = useState<'all' | 'contact'>('all')
  const [contactId, setContactIdState] = useState(() => {
    try {
      return window.sessionStorage.getItem(AS_KEY(c.campaign.id)) ?? ''
    } catch {
      return ''
    }
  })
  // The version of the contact it was worked out for.
  const [got, setGot] = useState<{ contactId: string; preview: VisibilityPreview } | null>(null)
  const [versionsOpen, setVersionsOpen] = useState(false)
  const [pendingPreview, setPendingPreview] = useState('')
  const [nonce, setNonce] = useState(0)
  const recountRef = useRef(false)
  const seq = useRef(0)
  // Copied conditional blocks and polls get ids of their own.
  useBlockIds(enabled ? c.editor : null, c.context.publicUrl)
  const root = c.editor?.getWrapper()
  // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers every canvas change
  const blocks = useMemo(() => (enabled && root ? conditionalComponents(root).length : 0), [enabled, root, c.rev])
  const catalog = $api.useQuery('get', '/api/v1/workspaces/{workspace}/visibility/catalog', { params: { path: { workspace: c.slug } } }, { enabled, staleTime: 60_000 })
  const contacts = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/visibility/preview-contacts',
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } },
    { enabled: enabled && canView && blocks > 0, staleTime: 60_000 },
  )

  const content = useCallback(() => {
    if (!c.ge) return { mjml: c.campaign.mjml, html: c.campaign.html }
    const mjml = c.currentMjml()
    return { mjml, html: c.compileCurrent()?.html ?? '' }
  }, [c])

  // Count the versions a moment after each change.
  useEffect(() => {
    if (!enabled) return
    const h = window.setTimeout(
      () => {
        const { mjml, html } = content()
        if (!mjml.includes('rc-visible')) {
          setReport(null)
          setError(null)
          return
        }
        const n = ++seq.current
        const recount = recountRef.current
        recountRef.current = false
        setCounting(true)
        void http
          .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/visibility', {
            params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
            body: { mjml, html, recount: recount || undefined },
          })
          .then(({ data }) => {
            if (n !== seq.current || !data) return
            setReport(data)
            setError(null)
          })
          .catch((e: unknown) => n === seq.current && setError(errorMessage(e)))
          .finally(() => n === seq.current && setCounting(false))
      },
      c.ge ? 1200 : 0,
    )
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas edits
  }, [enabled, c.rev, c.ge, c.slug, c.campaign.id, nonce])

  // The contact's version, when the canvas shows one.
  const items = contacts.data?.items ?? []
  const chosen = contactId || items[0]?.id || ''
  useEffect(() => {
    if (!enabled || mode !== 'contact' || !chosen) return
    let cancelled = false
    const h = window.setTimeout(() => {
      const { mjml, html } = content()
      void http
        .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/visibility/preview', {
          params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
          body: { contactId: chosen, mjml, html },
        })
        .then(({ data }) => !cancelled && data && setGot({ contactId: chosen, preview: data }))
        .catch(() => !cancelled && setGot(null))
    }, 400)
    return () => {
      cancelled = true
      window.clearTimeout(h)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas edits
  }, [enabled, mode, chosen, c.rev, c.slug, c.campaign.id])

  const setContactId = useCallback(
    (id: string) => {
      setContactIdState(id)
      try {
        window.sessionStorage.setItem(AS_KEY(c.campaign.id), id)
      } catch {
        /* storage unavailable */
      }
    },
    [c.campaign.id],
  )

  const preview = enabled && mode === 'contact' && got?.contactId === chosen ? got.preview : null
  const value: ConditionalState = {
    enabled,
    blocks,
    report,
    counting,
    error,
    catalog: catalog.data,
    mode,
    setMode,
    contactId: chosen,
    setContactId,
    contacts: items,
    preview,
    versionsOpen,
    setVersionsOpen,
    pendingPreview,
    requestPreview: setPendingPreview,
    takePreview: () => setPendingPreview(''),
    recount: () => {
      recountRef.current = true
      setNonce((n) => n + 1)
    },
    blockInfo: (id) => report?.blocks.find((b) => b.id === id),
  }
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
