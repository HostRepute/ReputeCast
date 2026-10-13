import type { ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { cn } from '@/lib/cn'
import { useSetupAction, type Setup, type SetupStep } from './api'

/** What every step gets from the wizard. */
export interface StepProps {
  setup: Setup
  /** Moves to a step (and records it on the setup). */
  go: (step: SetupStep) => void
  /** Records the current step and leaves the wizard. */
  saveLater: () => void
  /** Why the person can't change the setup (providers:manage), if so. */
  reason?: string
  /**
   * The setup is finished: its streams and contacts steps show what it did,
   * and moving on doesn't change routing again (that's on Routing now).
   */
  finished: boolean
}

/** A numbered card of a step: "1. How to grant access". */
export function SectionCard({ title, aside, children, className }: { title: ReactNode; aside?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={cn('card flex flex-col gap-3 px-5 py-4.5', className)}>
      {aside ? (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="h2">{title}</h3>
          {aside}
        </div>
      ) : (
        <h3 className="h2">{title}</h3>
      )}
      {children}
    </section>
  )
}

/** A status row of a card: badge, text, and something on the right. */
export function StatusRow({
  badge,
  children,
  aside,
  className,
  lg,
}: {
  badge: ReactNode
  children: ReactNode
  aside?: ReactNode
  className?: string
  lg?: boolean
}) {
  return (
    <div className={cn('cn-row', lg && 'lg', className)}>
      {badge}
      <span className="min-w-0">{children}</span>
      {aside ? <span className="ml-auto flex shrink-0 items-center gap-1.5">{aside}</span> : null}
    </div>
  )
}

/** Done / Failed / Skipped / Pending badge of a row. */
export function StateBadge({ state }: { state: 'done' | 'failed' | 'skipped' | 'pending' | 'yes' | 'partial' | 'no' }) {
  const map: Record<typeof state, [Tone, string]> = {
    done: ['ok', 'Done'],
    failed: ['bad', 'Failed'],
    skipped: ['neutral', 'Skipped'],
    pending: ['warn', 'Pending'],
    yes: ['ok', 'Yes'],
    partial: ['warn', 'Partial'],
    no: ['bad', 'No'],
  }
  const [tone, label] = map[state]
  return <Badge tone={tone}>{label}</Badge>
}

/** The most useful sentence of an API error: the field messages of a 422, else the detail. */
export function problemMessage(e: unknown): string {
  if (e instanceof ApiError) {
    const errs = e.problem.errors?.map((x) => x.message).filter(Boolean)
    if (errs && errs.length && e.status === 422) return errs.join(' ')
  }
  return errorMessage(e)
}

/** Shows a failed action as an error toast. */
export function failed(title: string, retry?: () => void) {
  return (e: unknown) => toast({ kind: 'error', title, detail: problemMessage(e), retry })
}

/** Copies text with a toast either way. */
export async function copy(text: string, what: string) {
  if (await copyText(text)) toast({ title: `Copied ${what}` })
  else toast({ kind: 'error', title: `Couldn't copy ${what}`, detail: 'Your browser blocked the clipboard. Select the text and copy it instead.' })
}

/** The setup's actions; each one puts the setup it returns in the cache. */
export function useSetupActions(workspace: string, setupId: string) {
  const path = { params: { path: { workspace, setupId } } }
  const connect = useSetupAction(workspace, setupId, async (body: Schemas['ConnectInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/connection', { ...path, body })).data)
  const mailbox = useSetupAction(workspace, setupId, async (body: Schemas['ConnectMailboxInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/mailbox', { ...path, body })).data)
  const webhook = useSetupAction(workspace, setupId, async (body: Schemas['ConnectWebhookInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/webhook', { ...path, body })).data)
  const polling = useSetupAction(workspace, setupId, async () => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/polling', path)).data)
  const region = useSetupAction(workspace, setupId, async (body: Schemas['ConnectRegionInput']) => (await http.PUT('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/region', { ...path, body })).data)
  const provision = useSetupAction(workspace, setupId, async () => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/provision', path)).data)
  const production = useSetupAction(workspace, setupId, async (body: Schemas['ConnectProductionAccessInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/production-access', { ...path, body })).data)
  const domain = useSetupAction(workspace, setupId, async (body: Schemas['ConnectDomainInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/domain', { ...path, body })).data)
  const streams = useSetupAction(workspace, setupId, async (body: Schemas['ConnectStreamsInput']) => (await http.PUT('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/streams', { ...path, body })).data)
  const finish = useSetupAction(workspace, setupId, async (body: Schemas['ConnectFinishInput']) => (await http.POST('/api/v1/workspaces/{workspace}/connect/setups/{setupId}/finish', { ...path, body })).data)
  const step = useSetupAction(workspace, setupId, async (body: Schemas['ConnectStepInput']) => (await http.PATCH('/api/v1/workspaces/{workspace}/connect/setups/{setupId}', { ...path, body })).data)
  return { connect, mailbox, webhook, polling, region, provision, production, domain, streams, finish, step }
}

export type SetupActions = ReturnType<typeof useSetupActions>
