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
const kindLabel: Record<DiffItem['kind'], string> = { added: 'Added', changed: 'Changed', removed: 'Removed', unchanged: 'Unchanged' }

/** Version diff rows: Added, Changed, Removed, then what stayed the same. */
export function DiffRows({ items }: { items: DiffItem[] }) {
  return (
    <>
      {items.map((it, i) => (
        <div key={i} className={cn('chg', it.kind === 'added' && 'add', it.kind === 'removed' && 'rem')}>
          <Badge tone={kindTone[it.kind]}>{kindLabel[it.kind]}</Badge>
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
          <ErrorPanel title="Couldn't prepare the publish" actions={<Button onClick={() => preview.refetch()}>Retry</Button>}>
            {errorMessage(preview.error)} Nothing was published.
          </ErrorPanel>
        </div>
      ) : !p ? (
        <div className="flex flex-col gap-3 p-6" aria-busy="true">
          <RDialog.Title id="pub-title" className="vh">
            Publish
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
                    Live · v{from}
                  </Badge>
                  <span className="k">→</span>
                </>
              ) : null}
              <Badge tone="info">Draft · v{to}</Badge>
            </div>
            <RDialog.Title id="pub-title" className="m-0 text-[20px] font-bold tracking-[-0.02em]">
              Publish version {to} of "{p.name}"?
            </RDialog.Title>
            <span className="text-ink-2">
              {from !== undefined
                ? `${p.changes} ${p.changes === 1 ? 'change' : 'changes'} since v${from} was published${p.livePublishedAt ? ` on ${dayMonthDate(p.livePublishedAt)}` : ''}${p.livePublishedBy ? ` by ${p.livePublishedBy}` : ''}. Every step is re-validated before it goes live.`
                : 'This is the first version. Every step is validated before it goes live.'}
            </span>
          </div>
          {from !== undefined ? (
            <div className="flex flex-col">
              <div className="k px-4 pb-1 pt-3">
                Version diff · v{from} → v{to}
              </div>
              <DiffRows items={p.items} />
            </div>
          ) : null}
          {from !== undefined ? (
            <div className="flex flex-col gap-2.5 border-t border-line-2 px-4 py-3.5">
              <span className="font-semibold">
                {count(p.inFlow)} {p.inFlow === 1 ? 'contact is' : 'contacts are'} in this flow right now
              </span>
              <Grid min={300} gap={10} role="radiogroup" aria-label="Contacts already in the flow">
                <label className="opt">
                  <input type="radio" name="mig" checked={migration === 'finish'} onChange={() => setMigration('finish')} />
                  <span className="font-semibold">
                    Let them finish on v{from} (recommended)
                    <br />
                    <span className="k">
                      New entrants start on v{to}. Nobody gets a step twice or skips one. v{from} retires when the last contact leaves, {p.finishEstimate}.
                    </span>
                  </span>
                </label>
                <label className="opt">
                  <input type="radio" name="mig" checked={migration === 'move'} onChange={() => setMigration('move')} />
                  <span className="font-semibold">
                    Move them to v{to} at their current step
                    <br />
                    <span className="k">{p.moveImpact}</span>
                  </span>
                </label>
              </Grid>
            </div>
          ) : null}
          <div className="flex flex-col gap-1.5 border-t border-line-2 bg-ground-3 px-4 py-3">
            <span className="k">Pre-publish checks</span>
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
              <Button className="tall">Back to canvas</Button>
            </RDialog.Close>
            <Button
              className="tall"
              loading={saving}
              onClick={async () => {
                setSaving(true)
                try {
                  await onSaveDraft()
                  toast({ title: `Draft v${to} saved`, detail: 'Nothing changed for contacts in the flow.' })
                  onOpenChange(false)
                } finally {
                  setSaving(false)
                }
              }}
            >
              Save draft only
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
                        toast({ kind: 'info', title: `v${to} is waiting for approval`, detail: r.message || 'It goes live once approved.' })
                        return
                      }
                      toast({ title: `"${p.name}" v${to} is live`, detail: migration === 'move' ? 'Contacts moved to it at their current step.' : `Contacts in the flow finish on v${from}.` })
                      void navigate({ to: '/w/$workspace/automations', params: { workspace: slug } })
                    },
                    onError: (e) => toast({ kind: 'error', title: "Couldn't publish", detail: errorMessage(e) }),
                  },
                )
              }
            >
              Publish v{to}
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
          toast({ title: `v${version} restored as draft v${r.draftVersion}`, detail: 'Publish it when it looks right.' })
          onOpenChange(false)
          onRestored()
        },
        onError: (e) => {
          onRestoreFailed?.()
          toast({ kind: 'error', title: "Couldn't restore the version", detail: errorMessage(e) })
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
          {compare ? 'Changes since the live version' : `Version history · ${v?.name ?? ''}`}
        </RDialog.Title>
        <div className="flex gap-2">
          <Button size="sm" onClick={() => setCompare(!compare)}>
            {compare ? 'Back to versions' : 'Compare'}
          </Button>
          <RDialog.Close asChild>
            <Button size="sm" iconOnly icon="close" aria-label="Close" className="border-transparent" />
          </RDialog.Close>
        </div>
      </div>
      {compare ? (
        diff.data ? (
          <div className="flex flex-col">
            <div className="k px-4 pb-1">
              Version diff · v{diff.data.from} → v{diff.data.to}
            </div>
            {diff.data.items.length ? <DiffRows items={diff.data.items} /> : <div className="chg k">No changes.</div>}
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
          const retired = x.retiredAt && dayMonthDate(x.retiredAt) !== when ? ` · retired ${dayMonthDate(x.retiredAt)}` : ''
          const who = x.summary.startsWith('created from') ? '' : ` · ${x.publishedBy || x.createdBy}`
          const live = x.status === 'live'
          return (
            <div key={x.version} className="ver" style={live ? { background: 'var(--accent-tint)' } : undefined}>
              <span className="n w-[30px]">v{x.version}</span>
              <span className="flex-1">
                {live ? (
                  <>
                    <b>Live</b> ·{' '}
                  </>
                ) : null}
                {when}
                {who}
                {x.summary ? ` · ${x.summary}` : ''}
                {x.status === 'retiring' ? ` · retiring, ${count(x.inFlow)} still in flow` : retired}
              </span>
              {live ? (
                <Badge tone="ok">Live · {count(x.inFlow)} in flow</Badge>
              ) : confirm === x.version ? (
                <span className="flex items-center gap-1.5">
                  <span className="k">Replace the unpublished draft?</span>
                  <Button size="sm" variant="primary" loading={restore.isPending} onClick={() => doRestore(x.version)}>
                    Replace
                  </Button>
                  <Button size="sm" onClick={() => setConfirm(null)}>
                    Keep
                  </Button>
                </span>
              ) : (
                <Button size="sm" disabledReason={reason} onClick={() => (hasDraftChanges ? setConfirm(x.version) : doRestore(x.version))}>
                  Restore as draft
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
  const [reason, setReason] = useState('')
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="sm"
      title={`Pause "${name}"?`}
      description={`No new contacts enter. The ${count(inFlow)} ${inFlow === 1 ? 'contact' : 'contacts'} in the flow stay where they are; delays keep counting, and nothing is sent until you resume.`}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Keep it running</Button>
          <Button variant="primary" loading={pending} onClick={() => onPause(reason.trim())}>
            Pause automation
          </Button>
        </>
      }
    >
      <Field label="Reason" optional hint="Shown on the automations list, e.g. “beta is full”.">
        <Input value={reason} maxLength={200} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  )
}

/** The email as recipients see it (sandboxed, scripts off). */
export function EmailPreviewDialog({ email, onClose }: { email: { id: string; version: number } | null; onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automation-emails/{id}',
    { params: { path: { workspace: slug, id: email?.id ?? '' }, query: { version: email?.version ?? 0 } } },
    { enabled: Boolean(email) },
  )
  return (
    <Dialog open={Boolean(email)} onOpenChange={(o) => !o && onClose()} size="lg" title={q.data ? `${q.data.name} · v${q.data.version}` : 'Preview'}>
      {q.data ? (
        <>
          <div className="flex flex-col gap-0.5 rounded-lg border border-line px-3 py-2.5">
            <span className="font-semibold">{q.data.subject}</span>
            <span className="k">{q.data.preheader}</span>
          </div>
          <iframe title={`Preview of ${q.data.name}`} sandbox="" srcDoc={q.data.html} className="h-[480px] w-full rounded-lg border border-line bg-white" />
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
export function EmailEditDialog({ email, onClose, onSaved }: { email: { id: string; version: number } | null; onClose: () => void; onSaved: (version: number) => void }) {
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
      title={q.data ? `Composer · ${q.data.name}` : 'Composer'}
      description={q.data ? `Editing v${q.data.version}. Saving makes v${q.data.latestVersion + 1}; this step switches to it, other steps keep their version until you pick it.` : undefined}
      footer={
        <>
          <Button onClick={close}>Cancel</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={() =>
              email &&
              save.mutate(
                { params: { path: { workspace: slug, id: email.id } }, body: { subject: form.subject, preheader: form.preheader, html: form.html } },
                {
                  onSuccess: (r) => {
                    toast({ title: `Saved as v${r.version}`, detail: 'The step now sends it once you publish.' })
                    setEdits(null)
                    onSaved(r.version)
                  },
                  onError: (e) => toast({ kind: 'error', title: "Couldn't save the email", detail: errorMessage(e) }),
                },
              )
            }
          >
            Save as new version
          </Button>
        </>
      }
    >
      {q.data ? (
        <>
          <Field label="Subject">
            <Input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} />
          </Field>
          <Field label="Preheader" optional>
            <Input value={form.preheader} onChange={(e) => setForm({ ...form, preheader: e.target.value })} />
          </Field>
          <Field label="HTML" hint="Liquid tags like {{ first_name | default: &quot;there&quot; }} and {{ unsubscribe_url }} are filled in when it sends.">
            <Textarea className="mono min-h-[260px] text-[12px]" spellCheck={false} value={form.html} onChange={(e) => setForm({ ...form, html: e.target.value })} />
          </Field>
        </>
      ) : (
        <Skeleton w="100%" h={320} />
      )}
    </Dialog>
  )
}
