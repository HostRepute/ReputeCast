import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, monthDay } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'

/** Delete the workspace (WorkspaceDialogs, right): consequences, export first, typed name. */
export function DeleteWorkspaceDialog({
  settings,
  open,
  onOpenChange,
}: {
  settings: Pick<Schemas['WorkspaceSettings'], 'name' | 'danger'>
  open: boolean
  onOpenChange: (o: boolean) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const canExport = useCan('subscribers:export')
  const [typed, setTyped] = useState('')
  const del = $api.useMutation('post', '/api/v1/workspaces/{workspace}/delete')
  const d = settings.danger
  const match = typed === settings.name
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.deleteEyebrow)}</span>}
      title={<span className="text-bad-fg">{t(m.deleteDialogTitle, { name: settings.name })}</span>}
      description={<span className="text-[13.5px]">{t(m.deleteDescription, { days: count(d.restoreDays) })}</span>}
      className="card border-bad-line"
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button
            variant="danger-primary"
            loading={del.isPending}
            disabledReason={match ? undefined : t(m.typeToEnable)}
            onClick={() =>
              del.mutate(
                { params: { path: { workspace: slug } }, body: { confirm: typed } },
                {
                  onSuccess: (r) => {
                    toast({ kind: 'deleted', title: t(m.deleted, { name: settings.name }), detail: t(m.deletedDetail, { date: monthDay(r.restoreUntil) }) })
                    void qc.invalidateQueries({ queryKey: ['me'] })
                    void navigate({ to: '/' })
                  },
                  onError: (e) => toast({ kind: 'error', title: t(m.deleteFailed), detail: errorMessage(e) }),
                },
              )
            }
          >
            {t(m.deleteConfirm)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(m.deleteContacts, { contacts: d.contacts, contactsText: count(d.contacts), lists: d.lists, segments: d.segments })}</li>
        <li>{t(m.deleteCampaigns, { campaigns: d.campaigns, campaignsText: count(d.campaigns), automations: d.automations, templates: d.templates })}</li>
        <li>{t(m.deleteProviders, { count: d.providers })}</li>
        <li>{t(m.deleteKeys)}</li>
        <li>
          <T msg={m.deleteKept} values={{ b: (c) => <b>{c}</b> }} />
        </li>
      </ul>
      <div className="note i">
        <Badge tone="info">{t(m.recommended)}</Badge>
        <span className="col gap-1.5">
          <span>{t(m.exportBody)}</span>
          <span>
            {canExport ? (
              <ButtonAnchor size="sm" href={`/api/v1/workspaces/${slug}/export`} newTab={false} download="">
                {t(m.exportButton)}
              </ButtonAnchor>
            ) : (
              <Button size="sm" disabledReason={t(m.exportNeeds)}>
                {t(m.exportButton)}
              </Button>
            )}
          </span>
        </span>
      </div>
      <label className="lbl">
        <span>
          <T msg={m.typeName} values={{ name: settings.name, b: (c) => <b>{c}</b> }} />
        </span>
        <input className="in" type="text" value={typed} onChange={(e) => setTyped(e.target.value)} autoComplete="off" spellCheck={false} />
        <span className="k font-medium">{t(m.mustMatch)}</span>
      </label>
    </Dialog>
  )
}
