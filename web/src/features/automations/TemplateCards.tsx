import { useNavigate } from '@tanstack/react-router'
import { Fragment } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Grid } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { usePermissionReason, useWorkspace } from '@/lib/session'

type Template = Schemas['AutomationStarterTemplate']

/** Creates a draft from a template (or a blank canvas) and opens it. */
export function useCreateAutomation() {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations')
  return {
    pending: create.isPending,
    start: (template?: Template['key']) =>
      create.mutate(
        { params: { path: { workspace: slug } }, body: template ? { template } : {} },
        {
          onSuccess: (res) => void navigate({ to: '/w/$workspace/automations/$automationId', params: { workspace: slug, automationId: res.id } }),
          onError: (e) => toast({ kind: 'error', title: "Couldn't create the automation", detail: errorMessage(e) }),
        },
      ),
  }
}

/**
 * The dot strip of a template: one dot per step on the main path. The list
 * colours only branches; the empty state also marks webhooks and the end.
 */
function MiniStrip({ preview, full }: { preview: string[]; full?: boolean }) {
  const color = (t: string) => {
    if (t === 'branch') return '#B45309'
    if (!full) return undefined
    if (t === 'webhook') return '#1D4ED8'
    if (t === 'goal' || t === 'exit') return '#0B1220'
    return undefined
  }
  return (
    <div className="mini" aria-hidden="true">
      {preview.map((t, i) => (
        <Fragment key={i}>
          {i > 0 ? <i style={{ width: full ? 26 : 30 }} /> : null}
          <b style={{ background: color(t) }} />
        </Fragment>
      ))}
    </div>
  )
}

/** The list section's template cards. */
export function StarterTemplates({ templates }: { templates: Template[] }) {
  const reason = usePermissionReason('automations:edit')
  const { start, pending } = useCreateAutomation()
  return (
    <section className="flex flex-col gap-3" aria-labelledby="starter-templates">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="starter-templates">
          Starter templates
        </h2>
        <span className="k">Each opens as a draft on the canvas; every step is editable</span>
      </div>
      <Grid min={260} gap={14}>
        {templates.map((t) => (
          <button
            key={t.key}
            type="button"
            className="tpl"
            aria-disabled={reason || pending ? true : undefined}
            title={reason}
            onClick={() => !reason && !pending && start(t.key)}
          >
            <MiniStrip preview={t.preview} />
            <span className="font-semibold">{t.name}</span>
            <span className="k">{t.summary}</span>
          </button>
        ))}
      </Grid>
    </section>
  )
}

/** The four cards of the empty state and the "New automation" dialog. */
export function TemplateGrid({ templates }: { templates: Template[] }) {
  const reason = usePermissionReason('automations:edit')
  const { start, pending } = useCreateAutomation()
  const disabled = Boolean(reason) || pending
  return (
    <Grid min={240} gap={14}>
      {templates.map((t) => (
        <button key={t.key} type="button" className="tpl lg" aria-disabled={disabled || undefined} title={reason} onClick={() => !disabled && start(t.key)}>
          <MiniStrip preview={t.preview} full />
          <span className="font-semibold">{t.name}</span>
          <span className="k">{t.outline}</span>
          <Badge tone="neutral" className="self-start">
            {t.steps} steps
          </Badge>
        </button>
      ))}
      <button type="button" className={cn('tpl lg blank')} aria-disabled={disabled || undefined} title={reason} onClick={() => !disabled && start()}>
        <span className="font-semibold">Blank canvas</span>
        <span className="k">Pick a trigger and build from scratch</span>
      </button>
    </Grid>
  )
}

export function EmptyAutomations({ templates, routeLabel }: { templates: Template[]; routeLabel: string }) {
  return (
    <section className="card flex flex-col gap-4 border-dashed p-7" aria-labelledby="automations-empty">
      <div className="flex flex-col gap-1">
        <h2 id="automations-empty" className="m-0 text-[18px] font-bold">
          Start with a template or a blank canvas
        </h2>
        <span className="text-ink-2">
          Flows send on the Automations route ({routeLabel}) and fail over automatically, so a provider outage never stalls a welcome email.
        </span>
      </div>
      <TemplateGrid templates={templates} />
    </section>
  )
}
