import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { atPath } from '@/features/composer/editor/labels'
import { splitMjml } from '@/features/composer/editor/mjmlDoc'
import { m } from './messages'

export type Report = Schemas['AccessibilityReport']
export type Issue = Schemas['AccessibilityIssue']
export type FixBody = Omit<Schemas['AccessibilityFixInput'], 'mjml'>

/** A fix made in this session, which Undo can take back while the canvas hasn't changed since. */
export interface FixedEntry {
  id: number
  summary: string
  before: string
  after: string
}

export interface AccessibilityState {
  /** The campaign composer checks; the transactional mode has its own Checks tab. */
  enabled: boolean
  report?: Report
  checking: boolean
  /** When the edit that started the running check was made. */
  editedAt?: Date
  error?: { message: string; reference?: string }
  /** The last check that worked, for the error state. */
  last?: { at: string; count: number }
  fixed: FixedEntry[]
  /** The issue being fixed. */
  busy?: string
  policy?: Schemas['AccessibilityPolicy']
  fix(issue: Issue, body: FixBody, summary: string, toastTitle?: string): Promise<boolean>
  /** Changes the canvas the way a fix does, for edits that aren't fixes (Text role). */
  transform(body: FixBody): Promise<boolean>
  undo(entry: FixedEntry): void
  recheck(): void
}

const Ctx = createContext<AccessibilityState | null>(null)

/** The composer's accessibility checks; null outside the campaign composer. */
export function useAccessibility(): AccessibilityState | null {
  return useContext(Ctx)
}

/** A problem's reference, as the error screens show it ("request 9b41c2e0"). */
function referenceOf(e: unknown): string | undefined {
  if (!(e instanceof ApiError)) return undefined
  const p = e.problem as { instance?: string; meta?: { requestId?: string; errorId?: string } }
  return p.meta?.errorId ?? p.meta?.requestId ?? p.instance
}

/**
 * Runs the accessibility checks on the composer's content as it changes
 * (debounced, like the content checks) and applies the panel's fixes to the
 * canvas. `language` is the version on the canvas (the source when unset).
 */
export function AccessibilityProvider({ language, children }: { language?: string; children: ReactNode }) {
  const c = useComposer()
  const t = useT()
  const enabled = c.kind !== 'transactional' && !c.isEmpty
  const [report, setReport] = useState<Report>()
  const [checking, setChecking] = useState(false)
  const [editedAt, setEditedAt] = useState<Date>()
  const [error, setError] = useState<AccessibilityState['error']>()
  const [last, setLast] = useState<AccessibilityState['last']>()
  const [fixed, setFixed] = useState<FixedEntry[]>([])
  const [busy, setBusy] = useState<string>()
  const [nonce, setNonce] = useState(0)
  const seq = useRef(0)
  const nextId = useRef(1)
  const policy = $api.useQuery('get', '/api/v1/workspaces/{workspace}/accessibility/policy', { params: { path: { workspace: c.slug } } }, { enabled, staleTime: 60_000 })
  const { ge, slug, canEdit, currentMjml, campaign } = c

  useEffect(() => {
    if (!enabled) return
    const h = window.setTimeout(
      () => {
        const mjml = ge ? currentMjml() : campaign.mjml
        const html = mjml.trim() ? undefined : campaign.html
        const n = ++seq.current
        setEditedAt(new Date())
        setChecking(true)
        void http
          .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/accessibility', {
            params: { path: { workspace: slug, campaignId: campaign.id } },
            body: { mjml: mjml || undefined, html, lang: language },
          })
          .then(({ data }) => {
            if (n !== seq.current || !data) return
            setReport(data)
            setError(undefined)
            setLast({ at: data.checkedAt, count: data.issues.length })
          })
          .catch((e: unknown) => {
            if (n === seq.current) setError({ message: errorMessage(e), reference: referenceOf(e) })
          })
          .finally(() => {
            if (n === seq.current) setChecking(false)
          })
      },
      ge ? 900 : 0,
    )
    return () => window.clearTimeout(h)
    // rev covers canvas edits; ge starts the first run once the editor is ready.
  }, [enabled, c.rev, ge, c.subject, c.preheader, slug, campaign.id, campaign.mjml, campaign.html, currentMjml, language, nonce])

  const recheck = useCallback(() => setNonce((x) => x + 1), [])

  const undo = useCallback(
    (entry: FixedEntry) => {
      if (!ge) return
      if (ge.getBody() !== entry.after) {
        toast({ kind: 'info', title: t(m.undoGone), detail: t(m.undoGoneDetail) })
        return
      }
      ge.setBody(entry.before)
      setFixed((list) => list.filter((x) => x.id !== entry.id))
    },
    [ge, t],
  )

  /** Applies a fix to the canvas through the server; the body before and after. */
  const apply = useCallback(
    async (body: FixBody): Promise<{ before: string; after: string } | null> => {
      if (!ge || !canEdit) return null
      if (ge.isEditingText()) await ge.flushEditing()
      const before = ge.getBody()
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/accessibility/fix', {
        params: { path: { workspace: slug } },
        body: { ...body, mjml: currentMjml() },
      })
      if (!data) return null
      ge.setBody(splitMjml(data.mjml).body)
      // The canvas is rebuilt: select the same element again.
      const target = atPath(ge.editor.getWrapper()?.findType('mj-body')[0], body.path)
      if (target) ge.editor.select(target)
      return { before, after: ge.getBody() }
    },
    [ge, canEdit, slug, currentMjml],
  )

  const transform = useCallback(
    async (body: FixBody): Promise<boolean> => {
      try {
        return (await apply(body)) !== null
      } catch (e) {
        toast({ kind: 'error', title: t(m.fixFailed), detail: errorMessage(e) })
        return false
      }
    },
    [apply, t],
  )

  const fix = useCallback(
    async (issue: Issue, body: FixBody, summary: string, toastTitle?: string): Promise<boolean> => {
      if (!ge || !canEdit) return false
      setBusy(issue.id)
      try {
        const done = await apply({ ...body, path: body.path ?? issue.path })
        if (!done) return false
        const { before, after } = done
        const entry: FixedEntry = { id: nextId.current++, summary, before, after }
        setFixed((list) => [entry, ...list])
        toast({ kind: 'success', title: toastTitle ?? summary, undo: () => undo(entry) })
        return true
      } catch (e) {
        toast({ kind: 'error', title: t(m.fixFailed), detail: errorMessage(e) })
        return false
      } finally {
        setBusy(undefined)
      }
    },
    [ge, canEdit, apply, undo, t],
  )

  const value = useMemo<AccessibilityState>(
    () => ({ enabled, report: enabled ? report : undefined, checking, editedAt, error, last, fixed, busy, policy: policy.data, fix, transform, undo, recheck }),
    [enabled, report, checking, editedAt, error, last, fixed, busy, policy.data, fix, transform, undo, recheck],
  )
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
