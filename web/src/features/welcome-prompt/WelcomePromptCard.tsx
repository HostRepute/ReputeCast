import { useQueryClient } from '@tanstack/react-query'
import { $api, errorMessage } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'

export type WelcomePromptContext = { context: 'health' } | { context: 'form'; id: string } | { context: 'import'; id: string }

/**
 * "New subscribers of Product Updates get no welcome email yet": offers the
 * Welcome series starter for a list nothing welcomes yet (ListHealth,
 * FormBuilder, MigrationDone). Only people who can start an automation see
 * it; "Not now" hides it for that list, for the whole workspace.
 */
export function WelcomePromptCard({ where, className, raised }: { where: WelcomePromptContext; className?: string; raised?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('automations:edit')
  const qc = useQueryClient()
  const params = { path: { workspace: slug }, query: where.context === 'health' ? { context: where.context } : { context: where.context, id: where.id } }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/welcome-prompt', { params }, { enabled: can, staleTime: 30_000 })
  const dismiss = $api.useMutation('post', '/api/v1/workspaces/{workspace}/welcome-prompt/dismiss')
  const p = q.data?.prompt
  if (!can || !p) return null
  const covered = p.coveredLists
  const onDismiss = async () => {
    try {
      await dismiss.mutateAsync({ params: { path: { workspace: slug } }, body: { listId: p.listId } })
      toast({ title: t(m.dismissed, { list: p.listName }), detail: t(m.dismissedDetail) })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/welcome-prompt'] })
    } catch (e) {
      toast({ kind: 'error', title: t(m.dismissFailed), detail: errorMessage(e) })
    }
  }
  return (
    <section
      className={cn('card flex flex-wrap items-center gap-3.5 border-accent-line bg-accent-tint px-4.5 py-3.5', raised && 'shadow-[0_1px_2px_rgba(11,18,32,.06)]', className)}
      aria-label={t(m.label)}
    >
      <svg width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.6" aria-hidden="true" className="flex-none">
        <circle cx="4" cy="3" r="2" />
        <circle cx="4" cy="13" r="2" />
        <circle cx="12" cy="8" r="2" />
        <path d="M4 5v6M6 8h4" />
      </svg>
      <span className="flex min-w-0 flex-[1_1_320px] flex-col gap-0.5">
        <span className="font-semibold">{t(m.title, { list: p.listName })}</span>
        <span className="k text-ink-nav">
          {t(m.body)}
          {covered.length > 0 ? ` ${t(m.covered, { count: covered.length, lists: formatList(covered) })}` : ''}
        </span>
      </span>
      <span className="flex flex-wrap gap-2">
        {p.draftId ? (
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: p.draftId }}>
            {t(m.start)}
          </ButtonLink>
        ) : (
          <ButtonLink
            size="sm"
            variant="primary"
            to="/w/$workspace/automations/$automationId"
            params={{ workspace: slug, automationId: 'new' }}
            search={{ template: 'welcome-series', list: p.listId }}
          >
            {t(m.start)}
          </ButtonLink>
        )}
        <Button size="sm" loading={dismiss.isPending} onClick={() => void onDismiss()}>
          {t(m.notNow)}
        </Button>
      </span>
    </section>
  )
}
