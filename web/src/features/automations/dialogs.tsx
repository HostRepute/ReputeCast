import { useNavigate } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { count, dayMonthDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { CheckoutItemsButton } from '@/features/commerce/composer'
import { useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { mf } from './flow.messages'

type DiffItem = Schemas['AutomationDiffItem']

/** A modal shaped like the publish board (card, 60px from the top). */
function BoardDialog({ open, onOpenChange, labelledBy, children, className }: { open: boolean; onOpenChange: (o: boolean) => void; labelledBy: string; children: ReactNode; className?: string }) {
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className="fe-ovl">
          <RDialog.Content className={cn('fe-dlg', className)} aria-labelledby={labelledBy} aria-describedby={undefined}>
            {children}
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

const kindTone: Record<DiffItem['kind'], Tone> = { added: 'ok', changed: 'warn', removed: 'bad', unchanged: 'neutral' }
const kindLabel: Record<DiffItem['kind'], MessageDescriptor> = { added: mf.kindAdded, changed: mf.kindChanged, removed: mf.kindRemoved, unchanged: mf.kindUnchanged }

/** Version diff rows: Added, Changed, Removed, then what stayed the same. */
export function DiffRows({ items }: { items: DiffItem[] }) {
  const t = useT()
  return (
    <>
      {items.map((it, i) => (
        <div key={i} className={cn('chg', it.kind === 'added' && 'add', it.kind === 'removed' && 'rem')}>
          <Badge tone={kindTone[it.kind]}>{t(kindLabel[it.kind])}</Badge>
          <div className="flex flex-1 flex-col gap-0.5">
            <span className={it.kind === 'unchanged' ? undefined : 'font-semibold'}>{it.title}</span>
            {it.detail ? <span className="k">{it.detail}</span> : null}
          </div>
        </div>
      ))}
    </>
  )
}

/**
 * AutomationPublish: the diff since the live version, what happens to the
 * contacts already in the flow, and the pre-publish checks.
 */
export function PublishDialog({
  open,
  onOpenChange,
  automationId,
  onSaveDraft,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  automationId: string
  onSaveDraft: () => Promise<void>
}) {
  return (
    <BoardDialog open={open} onOpenChange={onOpenChange} labelledBy="pub-title">
      <PublishContent onOpenChange={onOpenChange} automationId={automationId} onSaveDraft={onSaveDraft} />
    </BoardDialog>
  )
}

/** Mounted while the dialog is open, so the migration choice starts at "finish". */
function PublishContent({ onOpenChange, automationId, onSaveDraft }: { onOpenChange: (o: boolean) => void; automationId: string; onSaveDraft: () => Promise<void> }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const reason = usePermissionReason('automations:publish')
  const preview = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}/publish-preview',
    { params: { path: { workspace: slug, id: automationId } } },
    { staleTime: 0 },
  )
  const publish = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/publish')
  const [migration, setMigration] = useState<'finish' | 'move'>('finish')
  const [saving, setSaving] = useState(false)
  const p = preview.data
  const from = p?.fromVersion
  const to = p?.toVersion ?? 0
  return (
    <>
      {preview.isError ? (
        <div className="p-5">
          <ErrorPanel title={t(mf.prepareFailed)} actions={<Button onClick={() => preview.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(preview.error)} {t(mf.nothingPublished)}
          </ErrorPanel>
        </div>
      ) : !p ? (
        <div className="flex flex-col gap-3 p-6" aria-busy="true">
          <RDialog.Title id="pub-title" className="vh">
            {t(mf.publish)}
          </RDialog.Title>
          <Skeleton w={160} h={22} r={999} />
          <Skeleton w={420} h={22} />
          <Skeleton w="80%" h={14} />
          <Skeleton w="100%" h={120} />
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-1.5 border-b border-line-2 px-[22px] pb-3.5 pt-5">
            <div className="flex flex-wrap items-center gap-2.5">
              {from !== undefined ? (
                <>
                  <Badge tone="ok" dot>
                    {t(mf.liveBadge, { version: from })}
                  </Badge>
                  <span className="k">→</span>
                </>
              ) : null}
              <Badge tone="info">{t(mf.draftBadge, { version: to })}</Badge>
            </div>
            <RDialog.Title id="pub-title" className="m-0 text-[20px] font-bold tracking-[-0.02em]">
              {t(mf.publishTitle, { version: to, name: p.name })}
            </RDialog.Title>
            <span className="text-ink-2">
              {from !== undefined
                ? t(mf.publishLead, {
                    count: p.changes,
                    from,
                    hasDate: p.livePublishedAt ? 'yes' : 'no',
                    date: p.livePublishedAt ? dayMonthDate(p.livePublishedAt) : '',
                    hasBy: p.livePublishedBy ? 'yes' : 'no',
                    name: p.livePublishedBy ?? '',
                  })
                : t(mf.publishLeadFirst)}
            </span>
          </div>
          {from !== undefined ? (
            <div className="flex flex-col">
              <div className="k px-4 pb-1 pt-3">{t(mf.versionDiff, { from, to })}</div>
              <DiffRows items={p.items} />
            </div>
          ) : null}
          {from !== undefined ? (
            <div className="flex flex-col gap-2.5 border-t border-line-2 px-4 py-3.5">
              <span className="font-semibold">{t(mf.inFlowRightNow, { count: p.inFlow })}</span>
              <Grid min={300} gap={10} role="radiogroup" aria-label={t(mf.migrationLabel)}>
                <label className="opt">
                  <input type="radio" name="mig" checked={migration === 'finish'} onChange={() => setMigration('finish')} />
                  <span className="font-semibold">
                    {t(mf.finishOn, { version: from })}
                    <br />
                    <span className="k">{t(mf.finishDetail, { from, to, estimate: p.finishEstimate })}</span>
                  </span>
                </label>
                <label className="opt">
                  <input type="radio" name="mig" checked={migration === 'move'} onChange={() => setMigration('move')} />
                  <span className="font-semibold">
                    {t(mf.moveTo, { version: to })}
                    <br />
                    <span className="k">{p.moveImpact}</span>
                  </span>
                </label>
              </Grid>
            </div>
          ) : null}
          <div className="flex flex-col gap-1.5 border-t border-line-2 bg-ground-3 px-4 py-3">
            <span className="k">{t(mf.checks)}</span>
            <div className="flex flex-wrap gap-2">
              {p.checks.map((c) => (
                <Badge key={c.label} tone={c.status === 'ok' ? 'ok' : c.status === 'warn' ? 'warn' : 'bad'}>
                  {c.label}
                </Badge>
              ))}
            </div>
          </div>
          <div className="flex flex-wrap justify-end gap-2.5 border-t border-line-2 px-4 pb-[18px] pt-3.5">
            <RDialog.Close asChild>
              <Button className="tall">{t(mf.backToCanvas)}</Button>
            </RDialog.Close>
            <Button
              className="tall"
              loading={saving}
              onClick={async () => {
                setSaving(true)
                try {
                  await onSaveDraft()
                  toast({ title: t(mf.savedToast, { version: to }), detail: t(mf.draftSavedDetail) })
                  onOpenChange(false)
                } finally {
                  setSaving(false)
                }
              }}
            >
              {t(mf.saveDraftOnly)}
            </Button>
            <Button
              variant="primary"
              className="tall px-[18px]"
              loading={publish.isPending}
              disabledReason={reason ?? (p.canPublish ? undefined : p.blocker)}
              onClick={() =>
                publish.mutate(
                  { params: { path: { workspace: slug, id: automationId } }, body: { version: to, migration } },
                  {
                    onSuccess: (r) => {
                      onOpenChange(false)
                      if (r.status === 'approval_required') {
                        toast({ kind: 'info', title: t(mf.waitingApproval, { version: to }), detail: r.message || t(mf.waitingApprovalDetail) })
                        return
                      }
                      toast({
                        title: t(mf.isLive, { name: p.name, version: to }),
                        detail: migration === 'move' ? t(mf.movedDetail) : t(mf.finishDetailToast, { version: from ?? '' }),
                      })
                      void navigate({ to: '/w/$workspace/automations', params: { workspace: slug } })
                    },
                    onError: (e) => toast({ kind: 'error', title: t(mf.publishFailed), detail: errorMessage(e) }),
                  },
                )
              }
            >
              {t(mf.publishVersion, { version: to })}
            </Button>
          </div>
        </>
      )}
    </>
  )
}

/** The version history card (AutomationsStates › Error): restore or compare. */
export function VersionHistoryDialog({
  open,
  onOpenChange,
  automationId,
  hasDraftChanges,
  onBeforeRestore,
  onRestoreFailed,
  onRestored,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  automationId: string
  hasDraftChanges: boolean
  /** Called before the draft is replaced (the editor stops saving its copy). */
  onBeforeRestore?: () => void
  onRestoreFailed?: () => void
  onRestored: () => void
}) {
  return (
    <BoardDialog open={open} onOpenChange={onOpenChange} labelledBy="vh-title">
      <VersionHistoryContent
        onOpenChange={onOpenChange}
        automationId={automationId}
        hasDraftChanges={hasDraftChanges}
        onBeforeRestore={onBeforeRestore}
        onRestoreFailed={onRestoreFailed}
        onRestored={onRestored}
      />
    </BoardDialog>
  )
}

function VersionHistoryContent({
  onOpenChange,
  automationId,
  hasDraftChanges,
  onBeforeRestore,
  onRestoreFailed,
  onRestored,
}: {
  onOpenChange: (o: boolean) => void
  automationId: string
  hasDraftChanges: boolean
  onBeforeRestore?: () => void
  onRestoreFailed?: () => void
  onRestored: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('automations:edit')
  const versions = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/versions', { params: { path: { workspace: slug, id: automationId } } }, { staleTime: 0 })
  const [compare, setCompare] = useState(false)
  const [confirm, setConfirm] = useState<number | null>(null)
  const diff = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/diff', { params: { path: { workspace: slug, id: automationId } } }, { enabled: compare })
  const restore = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/versions/{version}/restore')
  const doRestore = (version: number) => {
    onBeforeRestore?.()
    restore.mutate(
      { params: { path: { workspace: slug, id: automationId, version } } },
      {
        onSuccess: (r) => {
          toast({ title: t(mf.restored, { version, draft: r.draftVersion }), detail: t(mf.publishWhenReady) })
          onOpenChange(false)
          onRestored()
        },
        onError: (e) => {
          onRestoreFailed?.()
          toast({ kind: 'error', title: t(mf.restoreFailed), detail: errorMessage(e) })
        },
      },
    )
  }
  const v = versions.data
  const published = v?.versions.filter((x) => x.status !== 'draft') ?? []
  return (
    <>
      <div className="flex items-center justify-between gap-2 px-3.5 py-3">
        <RDialog.Title id="vh-title" className="m-0 text-[13px] font-semibold">
          {compare ? t(mf.changesSinceLive) : t(mf.historyTitle, { name: v?.name ?? '' })}
        </RDialog.Title>
        <div className="flex gap-2">
          <Button size="sm" onClick={() => setCompare(!compare)}>
            {compare ? t(mf.backToVersions) : t(mf.compare)}
          </Button>
          <RDialog.Close asChild>
            <Button size="sm" iconOnly icon="close" aria-label={t(common.close)} className="border-transparent" />
          </RDialog.Close>
        </div>
      </div>
      {compare ? (
        diff.data ? (
          <div className="flex flex-col">
            <div className="k px-4 pb-1">{t(mf.versionDiff, { from: diff.data.from, to: diff.data.to })}</div>
            {diff.data.items.length ? <DiffRows items={diff.data.items} /> : <div className="chg k">{t(mf.noChanges)}</div>}
          </div>
        ) : (
          <div className="px-4 pb-4">
            <Skeleton w="100%" h={90} />
          </div>
        )
      ) : versions.isPending ? (
        <div className="flex flex-col" aria-busy="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="ver">
              <Skeleton w={24} h={14} />
              <Skeleton w={300} h={14} />
              <Skeleton w={110} h={28} className="ml-auto" />
            </div>
          ))}
        </div>
      ) : (
        published.map((x) => {
          const when = dayMonthDate(x.publishedAt ?? x.createdAt)
          const retired = x.retiredAt && dayMonthDate(x.retiredAt) !== when ? ` · ${t(mf.retired, { date: dayMonthDate(x.retiredAt) })}` : ''
          const who = x.summary.startsWith('created from') ? '' : ` · ${x.publishedBy || x.createdBy}`
          const live = x.status === 'live'
          return (
            <div key={x.version} className="ver" style={live ? { background: 'var(--accent-tint)' } : undefined}>
              <span className="n w-[30px]">v{x.version}</span>
              <span className="flex-1">
                {live ? (
                  <>
                    <b>{t(mf.liveWord)}</b> ·{' '}
                  </>
                ) : null}
                {when}
                {who}
                {x.summary ? ` · ${x.summary}` : ''}
                {x.status === 'retiring' ? ` · ${t(mf.retiring, { count: count(x.inFlow) })}` : retired}
              </span>
              {live ? (
                <Badge tone="ok">{t(mf.liveInFlow, { count: count(x.inFlow) })}</Badge>
              ) : confirm === x.version ? (
                <span className="flex items-center gap-1.5">
                  <span className="k">{t(mf.replaceDraft)}</span>
                  <Button size="sm" variant="primary" loading={restore.isPending} onClick={() => doRestore(x.version)}>
                    {t(mf.replace)}
                  </Button>
                  <Button size="sm" onClick={() => setConfirm(null)}>
                    {t(mf.keep)}
                  </Button>
                </span>
              ) : (
                <Button size="sm" disabledReason={reason} onClick={() => (hasDraftChanges ? setConfirm(x.version) : doRestore(x.version))}>
                  {t(mf.restoreAsDraft)}
                </Button>
              )}
            </div>
          )
        })
      )}
    </>
  )
}

/** Pausing keeps contacts where they are; nothing is sent until it resumes. */
export function PauseDialog({ open, onOpenChange, name, inFlow, onPause, pending }: { open: boolean; onOpenChange: (o: boolean) => void; name: string; inFlow: number; onPause: (reason: string) => void; pending: boolean }) {
  const t = useT()
  const [reason, setReason] = useState('')
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="sm"
      title={t(mf.pauseTitle, { name })}
      description={t(mf.pauseBody, { count: inFlow })}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(mf.keepRunning)}</Button>
          <Button variant="primary" loading={pending} onClick={() => onPause(reason.trim())}>
            {t(mf.pauseAutomation)}
          </Button>
        </>
      }
    >
      <Field label={t(mf.reason)} optional hint={t(mf.reasonHint)}>
        <Input value={reason} maxLength={200} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  )
}

/** Puts a block before the email's footer (its unsubscribe line), else at the end of the body. */
function insertBlock(html: string, block: string): string {
  const unsub = html.search(/<[^<>]*\{\{\s*unsubscribe_url/i)
  const at = unsub >= 0 ? html.lastIndexOf('<table', unsub) : -1
  if (at > 0) return html.slice(0, at) + block + html.slice(at)
  const end = html.search(/<\/body>/i)
  return end >= 0 ? html.slice(0, end) + block + html.slice(end) : html + block
}

/** The email as recipients see it (sandboxed, scripts off). */
export function EmailPreviewDialog({ email, onClose }: { email: { id: string; version: number } | null; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automation-emails/{id}',
    { params: { path: { workspace: slug, id: email?.id ?? '' }, query: { version: email?.version ?? 0 } } },
    { enabled: Boolean(email) },
  )
  return (
    <Dialog open={Boolean(email)} onOpenChange={(o) => !o && onClose()} size="lg" title={q.data ? `${q.data.name} · v${q.data.version}` : t(mf.preview)}>
      {q.data ? (
        <>
          <div className="flex flex-col gap-0.5 rounded-lg border border-line px-3 py-2.5">
            <span className="font-semibold">{q.data.subject}</span>
            <span className="k">{q.data.preheader}</span>
          </div>
          <iframe title={t(mf.previewOf, { name: q.data.name })} sandbox="" srcDoc={q.data.html} className="h-[480px] w-full rounded-lg border border-line bg-white" />
        </>
      ) : (
        <Skeleton w="100%" h={480} />
      )}
    </Dialog>
  )
}

/**
 * Email content editor: subject, preheader and HTML. Saving makes the next
 * version; this step moves to it, other steps keep theirs until changed.
 */
export function EmailEditDialog({ email, onClose, onSaved, store }: { email: { id: string; version: number } | null; onClose: () => void; onSaved: (version: number) => void; store?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automation-emails/{id}',
    { params: { path: { workspace: slug, id: email?.id ?? '' }, query: { version: email?.version ?? 0 } } },
    { enabled: Boolean(email) },
  )
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/automation-emails/{id}')
  // Edits made in this dialog; until then the fields show the loaded version.
  const [edits, setEdits] = useState<{ subject: string; preheader: string; html: string } | null>(null)
  const form = edits ?? { subject: q.data?.subject ?? '', preheader: q.data?.preheader ?? '', html: q.data?.html ?? '' }
  const setForm = (f: typeof form) => setEdits(f)
  const close = () => {
    setEdits(null)
    onClose()
  }
  return (
    <Dialog
      open={Boolean(email)}
      onOpenChange={(o) => !o && close()}
      size="lg"
      title={q.data ? t(mf.composerOf, { name: q.data.name }) : t(mf.composer)}
      description={q.data ? t(mf.editingLead, { version: q.data.version, next: q.data.latestVersion + 1 }) : undefined}
      footer={
        <>
          <Button onClick={close}>{t(common.cancel)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={() =>
              email &&
              save.mutate(
                { params: { path: { workspace: slug, id: email.id } }, body: { subject: form.subject, preheader: form.preheader, html: form.html } },
                {
                  onSuccess: (r) => {
                    toast({ title: t(mf.savedAs, { version: r.version }), detail: t(mf.savedAsDetail) })
                    setEdits(null)
                    onSaved(r.version)
                  },
                  onError: (e) => toast({ kind: 'error', title: t(mf.emailSaveFailed), detail: errorMessage(e) }),
                },
              )
            }
          >
            {t(mf.saveAsNew)}
          </Button>
        </>
      }
    >
      {q.data ? (
        <>
          <Field label={t(mf.subject)}>
            <Input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} />
          </Field>
          <Field label={t(mf.preheader)} optional>
            <Input value={form.preheader} onChange={(e) => setForm({ ...form, preheader: e.target.value })} />
          </Field>
          {/* The Checkout items block, in flows a store trigger starts (X20). */}
          {store ? <CheckoutItemsButton store={store} onInsert={(block) => setForm({ ...form, html: insertBlock(form.html, block) })} /> : null}
          <Field label={t(mf.html)} hint={t(mf.htmlHint, { example: '{{ first_name | default: "there" }}', unsubscribe: '{{ unsubscribe_url }}' })}>
            <Textarea className="mono min-h-[260px] text-[12px]" spellCheck={false} value={form.html} onChange={(e) => setForm({ ...form, html: e.target.value })} />
          </Field>
        </>
      ) : (
        <Skeleton w="100%" h={320} />
      )}
    </Dialog>
  )
}
